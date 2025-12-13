package main

import (
    "context"
    "errors"
    "fmt"
    "time"

    opencode "github.com/sst/opencode-sdk-go"
    "github.com/sst/opencode-sdk-go/option"
)

// SimpleClient wraps the Go SDK with session management and streaming events.
type SimpleClient struct {
    api       *opencode.Client
    sessionID string
    cancel    context.CancelFunc
}

func newSimpleClient(cfg ResolvedConfig, renderer *Renderer) (*SimpleClient, error) {
    opts := []option.RequestOption{option.WithBaseURL(cfg.URL)}
    if cfg.APIKey != "" {
        opts = append(opts, option.WithHeader("authorization", fmt.Sprintf("Bearer %s", cfg.APIKey)))
    }

    client := opencode.NewClient(opts...)
    ctx := context.Background()

    // health check via list
    if _, err := client.Session.List(ctx, opencode.SessionListParams{Directory: opencode.F(cfg.Directory)}); err != nil {
        return nil, fmt.Errorf("failed to reach server: %w", err)
    }

    sessionID := cfg.Session
    cached := loadSessionState(cfg)
    if sessionID == "" && cached != nil {
        sessionID = cached.SessionID
    }

    if sessionID == "" {
        created, err := client.Session.New(ctx, opencode.SessionNewParams{Directory: opencode.F(cfg.Directory)})
        if err != nil {
            return nil, fmt.Errorf("failed to create session: %w", err)
        }
        sessionID = created.ID
        cached = &SessionStateEntry{SessionID: sessionID, Model: cfg.Model, Provider: cfg.Provider, Agent: cfg.Agent, UpdatedAt: time.Now().UnixMilli()}
        persistSessionState(cfg, *cached)
    }

    renderer.Trace("session", map[string]any{"sessionID": sessionID})

    streamCtx, cancel := context.WithCancel(context.Background())
    stream := client.Event.ListStreaming(streamCtx, opencode.EventListParams{Directory: opencode.F(cfg.Directory)})
    go func() {
        for stream.Next() {
            evt := stream.Current()
            handleEvent(evt, sessionID, renderer)
        }
        if err := stream.Err(); err != nil {
            renderer.Trace("event stream closed", map[string]any{"error": err.Error()})
        }
    }()

    return &SimpleClient{api: client, sessionID: sessionID, cancel: cancel}, nil
}

func (c *SimpleClient) Close() {
    if c.cancel != nil {
        c.cancel()
    }
}

func (c *SimpleClient) SendPrompt(ctx context.Context, text string, cfg ResolvedConfig) (*opencode.SessionPromptResponse, error) {
    if c.sessionID == "" {
        return nil, errors.New("no active session")
    }
    parts := []opencode.SessionPromptParamsPartUnion{
        opencode.SessionPromptParamsPart{
            Type: opencode.F(opencode.SessionPromptParamsPartsTypeText),
            Text: opencode.F(text),
        },
    }
    params := opencode.SessionPromptParams{
        Parts:     opencode.F(parts),
        Directory: opencode.F(cfg.Directory),
    }
    if cfg.Agent != "" {
        params.Agent = opencode.F(cfg.Agent)
    }
    if cfg.Model != "" && cfg.Provider != "" {
        params.Model = opencode.F(opencode.SessionPromptParamsModel{ModelID: opencode.F(cfg.Model), ProviderID: opencode.F(cfg.Provider)})
    }

    resp, err := c.api.Session.Prompt(ctx, c.sessionID, params)
    if err != nil {
        return nil, err
    }

    persistSessionState(cfg, SessionStateEntry{
        SessionID: c.sessionID,
        Model:     cfg.Model,
        Provider:  cfg.Provider,
        Agent:     cfg.Agent,
        UpdatedAt: time.Now().UnixMilli(),
    })

    return resp, nil
}

func handleEvent(evt opencode.EventListResponse, sessionID string, renderer *Renderer) {
    switch v := evt.AsUnion().(type) {
    case opencode.EventListResponseEventMessageUpdated:
        if v.Properties.Info.SessionID == sessionID {
            renderer.RenderMessage(v.Properties.Info.ID, v.Properties.Info.Role == opencode.MessageRoleAssistant, nil)
        }
    case opencode.EventListResponseEventMessagePartUpdated:
        if v.Properties.Part.SessionID == sessionID {
            renderer.RenderPart(v.Properties.Part)
        }
    }
}
