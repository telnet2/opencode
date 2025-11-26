# ADK-Go SDK Evaluation for OpenCode Server

## Executive Summary

This document evaluates Google's Agent Development Kit (ADK) for Go (`google.golang.org/adk`) as a potential replacement for the Vercel AI SDK in the OpenCode server implementation. While ADK-Go provides a solid foundation for building AI agents, it has significant gaps compared to the Vercel AI SDK's comprehensive feature set.

**Recommendation**: ADK-Go is **not a direct replacement** for the Vercel AI SDK, but can serve as **inspiration for architecture patterns** and may be useful for specific components. A custom Go implementation leveraging direct provider SDKs is recommended.

---

## 1. Feature Comparison Matrix

| Feature | Vercel AI SDK | ADK-Go | Gap Analysis |
|---------|---------------|--------|--------------|
| **Multi-Provider Support** | ✅ 10+ providers bundled | ⚠️ Gemini only | Major gap - need custom provider implementations |
| **Streaming Text Generation** | ✅ `streamText()` | ✅ `GenerateContent(stream=true)` | Conceptually similar |
| **Non-Streaming Generation** | ✅ `generateText()` | ✅ `GenerateContent(stream=false)` | Equivalent |
| **Tool/Function Calling** | ✅ `tool()` with execute | ✅ `functiontool.New()` | Similar approach |
| **Model Middleware** | ✅ `wrapLanguageModel()` | ⚠️ Callbacks only | Different pattern, achievable |
| **Provider Options** | ✅ Per-provider config | ⚠️ Generic config | Need custom handling |
| **Cache Control** | ✅ Built-in ephemeral cache | ❌ Not supported | Need to implement |
| **MCP Client** | ✅ `@ai-sdk/mcp` | ✅ `mcptoolset` | Good parity |
| **Session Management** | ❌ External | ✅ Built-in | ADK advantage |
| **Agent Orchestration** | ❌ External | ✅ Built-in | ADK advantage |
| **JSON Schema for Tools** | ✅ `jsonSchema()` | ✅ Auto-inference | Similar |
| **Error Handling** | ✅ Typed errors | ✅ Go errors | Different patterns |
| **Token Usage Tracking** | ✅ Built-in | ✅ UsageMetadata | Equivalent |

---

## 2. Vercel AI SDK Features Used in OpenCode

### 2.1 Core Text Generation

```typescript
// OpenCode uses streamText for main chat loop
import { streamText, generateText, type ModelMessage } from "ai"

const result = await streamText({
  model: wrapLanguageModel({ model: provider.language, middleware: [...] }),
  messages: [...],
  tools: {...},
  maxOutputTokens: 32000,
  abortSignal: abort,
  providerOptions: {...},
  temperature: 0.7,
  topP: 0.9,
  stopWhen: stepCountIs(1),
  onError(error) { ... },
  experimental_repairToolCall(input) { ... },
})
```

### 2.2 Tool Definition

```typescript
import { tool, jsonSchema } from "ai"

tools[item.id] = tool({
  id: item.id,
  description: item.description,
  inputSchema: jsonSchema(schema),
  async execute(args, options) {
    // Tool execution logic
    return result
  },
  toModelOutput(result) {
    return { type: "text", value: result.output }
  },
})
```

### 2.3 Multi-Provider Support

```typescript
// OpenCode supports 10+ providers via @ai-sdk/*
import { createAnthropic } from "@ai-sdk/anthropic"
import { createOpenAI } from "@ai-sdk/openai"
import { createGoogleGenerativeAI } from "@ai-sdk/google"
// ... and more

const sdk = providerFactory({ apiKey, baseURL, ... })
const model = sdk.languageModel(modelID)
```

### 2.4 MCP Integration

```typescript
import { experimental_createMCPClient } from "@ai-sdk/mcp"
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js"

const mcpClient = await experimental_createMCPClient({
  name: "opencode",
  transport: new StdioClientTransport({ command, args, env }),
})
const tools = await mcpClient.tools()
```

---

## 3. ADK-Go Architecture Overview

### 3.1 Core Components

```
google.golang.org/adk/
├── agent/           # Agent interface and implementations
│   ├── agent.go     # Base Agent interface
│   ├── llmagent/    # LLM-powered agent
│   ├── remoteagent/ # A2A remote agents
│   └── workflowagents/  # Sequential, parallel, loop agents
├── model/           # LLM interface
│   ├── llm.go       # model.LLM interface
│   └── gemini/      # Gemini implementation
├── tool/            # Tool interfaces
│   ├── tool.go      # tool.Tool interface
│   ├── functiontool/ # Function wrapper
│   └── mcptoolset/  # MCP integration
├── session/         # Session management
├── runner/          # Agent execution
├── memory/          # Agent memory
└── server/          # HTTP server
    ├── adkrest/     # REST API
    └── adka2a/      # Agent-to-Agent protocol
```

### 3.2 Key Interfaces

```go
// Agent interface
type Agent interface {
    Name() string
    Description() string
    Run(InvocationContext) iter.Seq2[*session.Event, error]
    SubAgents() []Agent
}

// LLM interface
type LLM interface {
    Name() string
    GenerateContent(ctx context.Context, req *LLMRequest, stream bool) iter.Seq2[*LLMResponse, error]
}

// Tool interface
type Tool interface {
    Name() string
    Description() string
    IsLongRunning() bool
}
```

---

## 4. How to Implement OpenCode Features in Go

### 4.1 Streaming Text Generation

**Vercel AI SDK Approach:**
```typescript
const result = await streamText({
  model: languageModel,
  messages: messages,
  tools: tools,
})

for await (const chunk of result.textStream) {
  // Handle streaming chunk
}
```

**Go Implementation with ADK-Go Pattern:**

```go
package streaming

import (
    "context"
    "iter"

    "google.golang.org/genai"
)

// LLMRequest represents the request to the language model
type LLMRequest struct {
    Model       string
    Messages    []*Message
    Tools       []Tool
    MaxTokens   int
    Temperature float64
    TopP        float64
}

// LLMResponse represents streaming response chunks
type LLMResponse struct {
    Content       *genai.Content
    UsageMetadata *UsageMetadata
    Partial       bool       // True for intermediate chunks
    TurnComplete  bool       // True when generation is complete
    FinishReason  string
}

// Provider interface for different LLM providers
type Provider interface {
    Name() string
    GenerateContent(ctx context.Context, req *LLMRequest, stream bool) iter.Seq2[*LLMResponse, error]
}

// StreamText is the main streaming function similar to Vercel's streamText
func StreamText(ctx context.Context, opts StreamOptions) (*StreamResult, error) {
    provider := opts.Provider

    req := &LLMRequest{
        Model:       opts.Model,
        Messages:    opts.Messages,
        Tools:       opts.Tools,
        MaxTokens:   opts.MaxOutputTokens,
        Temperature: opts.Temperature,
        TopP:        opts.TopP,
    }

    // Create streaming iterator
    stream := provider.GenerateContent(ctx, req, true)

    return &StreamResult{
        stream:   stream,
        provider: provider,
    }, nil
}

// StreamResult wraps the streaming response
type StreamResult struct {
    stream   iter.Seq2[*LLMResponse, error]
    provider Provider
}

// TextStream returns an iterator for streaming text chunks
func (r *StreamResult) TextStream() iter.Seq2[string, error] {
    return func(yield func(string, error) bool) {
        for resp, err := range r.stream {
            if err != nil {
                yield("", err)
                return
            }

            // Extract text from response
            if resp.Content != nil {
                for _, part := range resp.Content.Parts {
                    if part.Text != "" {
                        if !yield(part.Text, nil) {
                            return
                        }
                    }
                }
            }
        }
    }
}

// Usage example
func ExampleStreamText() {
    ctx := context.Background()

    result, err := StreamText(ctx, StreamOptions{
        Provider:        anthropicProvider,
        Model:          "claude-sonnet-4",
        Messages:       messages,
        Tools:          tools,
        MaxOutputTokens: 32000,
    })
    if err != nil {
        // Handle error
    }

    for text, err := range result.TextStream() {
        if err != nil {
            // Handle error
            break
        }
        fmt.Print(text)
    }
}
```

### 4.2 Multi-Provider Architecture

**Go Implementation:**

```go
package provider

import (
    "context"
    "fmt"
    "iter"

    "github.com/anthropics/anthropic-sdk-go"
    "github.com/openai/openai-go"
)

// Provider is the interface all LLM providers must implement
type Provider interface {
    ID() string
    Name() string
    Models() []ModelInfo
    GenerateContent(ctx context.Context, req *LLMRequest, stream bool) iter.Seq2[*LLMResponse, error]
}

// ProviderConfig holds configuration for a provider
type ProviderConfig struct {
    APIKey     string
    BaseURL    string
    Headers    map[string]string
    Options    map[string]any
}

// Registry manages provider instances
type Registry struct {
    providers map[string]Provider
}

func NewRegistry() *Registry {
    return &Registry{
        providers: make(map[string]Provider),
    }
}

func (r *Registry) Register(p Provider) {
    r.providers[p.ID()] = p
}

func (r *Registry) Get(id string) (Provider, bool) {
    p, ok := r.providers[id]
    return p, ok
}

// AnthropicProvider implements Provider for Anthropic/Claude
type AnthropicProvider struct {
    client *anthropic.Client
    config ProviderConfig
}

func NewAnthropicProvider(cfg ProviderConfig) (*AnthropicProvider, error) {
    client := anthropic.NewClient(
        anthropic.WithAPIKey(cfg.APIKey),
        anthropic.WithBaseURL(cfg.BaseURL),
    )

    return &AnthropicProvider{
        client: client,
        config: cfg,
    }, nil
}

func (p *AnthropicProvider) ID() string   { return "anthropic" }
func (p *AnthropicProvider) Name() string { return "Anthropic" }

func (p *AnthropicProvider) GenerateContent(ctx context.Context, req *LLMRequest, stream bool) iter.Seq2[*LLMResponse, error] {
    return func(yield func(*LLMResponse, error) bool) {
        // Convert messages to Anthropic format
        anthropicMessages := convertToAnthropicMessages(req.Messages)

        // Convert tools to Anthropic format
        anthropicTools := convertToAnthropicTools(req.Tools)

        if stream {
            // Streaming request
            stream, err := p.client.Messages.Stream(ctx, anthropic.MessagesStreamParams{
                Model:     req.Model,
                Messages:  anthropicMessages,
                Tools:     anthropicTools,
                MaxTokens: int64(req.MaxTokens),
            })
            if err != nil {
                yield(nil, err)
                return
            }
            defer stream.Close()

            for event := range stream.Events() {
                resp := convertAnthropicStreamEvent(event)
                if !yield(resp, nil) {
                    return
                }
            }
        } else {
            // Non-streaming request
            resp, err := p.client.Messages.Create(ctx, anthropic.MessagesParams{
                Model:     req.Model,
                Messages:  anthropicMessages,
                Tools:     anthropicTools,
                MaxTokens: int64(req.MaxTokens),
            })
            if err != nil {
                yield(nil, err)
                return
            }
            yield(convertAnthropicResponse(resp), nil)
        }
    }
}

// OpenAIProvider implements Provider for OpenAI
type OpenAIProvider struct {
    client *openai.Client
    config ProviderConfig
}

func NewOpenAIProvider(cfg ProviderConfig) (*OpenAIProvider, error) {
    client := openai.NewClient(
        openai.WithAPIKey(cfg.APIKey),
    )

    return &OpenAIProvider{
        client: client,
        config: cfg,
    }, nil
}

func (p *OpenAIProvider) ID() string   { return "openai" }
func (p *OpenAIProvider) Name() string { return "OpenAI" }

func (p *OpenAIProvider) GenerateContent(ctx context.Context, req *LLMRequest, stream bool) iter.Seq2[*LLMResponse, error] {
    return func(yield func(*LLMResponse, error) bool) {
        // Convert messages to OpenAI format
        openaiMessages := convertToOpenAIMessages(req.Messages)

        // Convert tools to OpenAI format
        openaiTools := convertToOpenAITools(req.Tools)

        if stream {
            stream := p.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
                Model:     req.Model,
                Messages:  openaiMessages,
                Tools:     openaiTools,
                MaxTokens: openai.Int(int64(req.MaxTokens)),
            })

            for stream.Next() {
                chunk := stream.Current()
                resp := convertOpenAIStreamChunk(chunk)
                if !yield(resp, nil) {
                    return
                }
            }

            if err := stream.Err(); err != nil {
                yield(nil, err)
            }
        } else {
            resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
                Model:     req.Model,
                Messages:  openaiMessages,
                Tools:     openaiTools,
                MaxTokens: openai.Int(int64(req.MaxTokens)),
            })
            if err != nil {
                yield(nil, err)
                return
            }
            yield(convertOpenAIResponse(resp), nil)
        }
    }
}
```

### 4.3 Tool System

**Go Implementation:**

```go
package tool

import (
    "context"
    "encoding/json"
    "fmt"
)

// Tool defines the interface for all tools
type Tool interface {
    ID() string
    Description() string
    Parameters() json.RawMessage // JSON Schema
    Execute(ctx context.Context, args json.RawMessage, opts ExecuteOptions) (*Result, error)
}

// ExecuteOptions provides context for tool execution
type ExecuteOptions struct {
    SessionID   string
    MessageID   string
    CallID      string
    Agent       string
    AbortSignal context.Context

    // Metadata callback for real-time updates
    OnMetadata func(title string, metadata map[string]any)
}

// Result represents the output of a tool execution
type Result struct {
    Title       string         `json:"title"`
    Output      string         `json:"output"`
    Metadata    map[string]any `json:"metadata,omitempty"`
    Attachments []Attachment   `json:"attachments,omitempty"`
}

// Attachment represents a file attachment
type Attachment struct {
    Filename  string `json:"filename"`
    MediaType string `json:"mediaType"`
    URL       string `json:"url"`
}

// Registry manages tool registration
type Registry struct {
    tools map[string]Tool
}

func NewRegistry() *Registry {
    return &Registry{
        tools: make(map[string]Tool),
    }
}

func (r *Registry) Register(tool Tool) {
    r.tools[tool.ID()] = tool
}

func (r *Registry) Get(id string) (Tool, bool) {
    t, ok := r.tools[id]
    return t, ok
}

func (r *Registry) All() map[string]Tool {
    return r.tools
}

// ToProviderFormat converts tools to provider-specific format
func (r *Registry) ToProviderFormat(providerID string) ([]any, error) {
    var result []any

    for _, tool := range r.tools {
        switch providerID {
        case "anthropic":
            result = append(result, map[string]any{
                "name":        tool.ID(),
                "description": tool.Description(),
                "input_schema": json.RawMessage(tool.Parameters()),
            })
        case "openai":
            result = append(result, map[string]any{
                "type": "function",
                "function": map[string]any{
                    "name":        tool.ID(),
                    "description": tool.Description(),
                    "parameters":  json.RawMessage(tool.Parameters()),
                },
            })
        default:
            // Generic format
            result = append(result, map[string]any{
                "name":        tool.ID(),
                "description": tool.Description(),
                "parameters":  json.RawMessage(tool.Parameters()),
            })
        }
    }

    return result, nil
}

// FunctionTool wraps a Go function as a Tool
type FunctionTool[TArgs, TResult any] struct {
    id          string
    description string
    handler     func(ctx context.Context, args TArgs, opts ExecuteOptions) (*TResult, error)
    schema      json.RawMessage
}

func NewFunctionTool[TArgs, TResult any](
    id string,
    description string,
    schema json.RawMessage,
    handler func(ctx context.Context, args TArgs, opts ExecuteOptions) (*TResult, error),
) *FunctionTool[TArgs, TResult] {
    return &FunctionTool[TArgs, TResult]{
        id:          id,
        description: description,
        handler:     handler,
        schema:      schema,
    }
}

func (t *FunctionTool[TArgs, TResult]) ID() string {
    return t.id
}

func (t *FunctionTool[TArgs, TResult]) Description() string {
    return t.description
}

func (t *FunctionTool[TArgs, TResult]) Parameters() json.RawMessage {
    return t.schema
}

func (t *FunctionTool[TArgs, TResult]) Execute(ctx context.Context, args json.RawMessage, opts ExecuteOptions) (*Result, error) {
    var typedArgs TArgs
    if err := json.Unmarshal(args, &typedArgs); err != nil {
        return nil, fmt.Errorf("failed to unmarshal args: %w", err)
    }

    result, err := t.handler(ctx, typedArgs, opts)
    if err != nil {
        return nil, err
    }

    // Convert result to generic Result type
    output, err := json.Marshal(result)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal result: %w", err)
    }

    return &Result{
        Output: string(output),
    }, nil
}
```

### 4.4 MCP Integration

**Go Implementation (leveraging ADK-Go's mcptoolset):**

```go
package mcp

import (
    "context"
    "fmt"
    "os/exec"
    "sync"

    mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
    "google.golang.org/adk/tool/mcptoolset"
)

// Config represents MCP server configuration
type Config struct {
    Type        string            `json:"type"` // "local" or "remote"
    Command     []string          `json:"command,omitempty"`
    URL         string            `json:"url,omitempty"`
    Headers     map[string]string `json:"headers,omitempty"`
    Environment map[string]string `json:"environment,omitempty"`
    Enabled     *bool             `json:"enabled,omitempty"`
    Timeout     int               `json:"timeout,omitempty"`
}

// Status represents the connection status of an MCP server
type Status struct {
    Status string `json:"status"` // "connected", "disabled", "failed"
    Error  string `json:"error,omitempty"`
}

// Client wraps an MCP connection
type Client struct {
    name    string
    config  Config
    toolset *mcptoolset.Set
    status  Status
    mu      sync.RWMutex
}

// Manager handles multiple MCP connections
type Manager struct {
    clients map[string]*Client
    mu      sync.RWMutex
}

func NewManager() *Manager {
    return &Manager{
        clients: make(map[string]*Client),
    }
}

// Connect establishes connection to an MCP server
func (m *Manager) Connect(ctx context.Context, name string, cfg Config) (*Status, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if cfg.Enabled != nil && !*cfg.Enabled {
        status := Status{Status: "disabled"}
        m.clients[name] = &Client{name: name, config: cfg, status: status}
        return &status, nil
    }

    var transport mcpsdk.Transport
    var err error

    switch cfg.Type {
    case "local":
        if len(cfg.Command) == 0 {
            return nil, fmt.Errorf("command required for local MCP server")
        }
        cmd := exec.Command(cfg.Command[0], cfg.Command[1:]...)
        for k, v := range cfg.Environment {
            cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
        }
        transport = &mcpsdk.CommandTransport{Command: cmd}

    case "remote":
        if cfg.URL == "" {
            return nil, fmt.Errorf("URL required for remote MCP server")
        }
        transport, err = newHTTPTransport(cfg.URL, cfg.Headers)
        if err != nil {
            return nil, fmt.Errorf("failed to create HTTP transport: %w", err)
        }

    default:
        return nil, fmt.Errorf("unknown MCP type: %s", cfg.Type)
    }

    // Create toolset using ADK-Go's mcptoolset
    toolset, err := mcptoolset.New(mcptoolset.Config{
        Transport: transport,
    })
    if err != nil {
        status := Status{Status: "failed", Error: err.Error()}
        m.clients[name] = &Client{name: name, config: cfg, status: status}
        return &status, nil
    }

    status := Status{Status: "connected"}
    m.clients[name] = &Client{
        name:    name,
        config:  cfg,
        toolset: toolset,
        status:  status,
    }

    return &status, nil
}

// Tools returns all tools from connected MCP servers
func (m *Manager) Tools(ctx context.Context) (map[string]Tool, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    result := make(map[string]Tool)

    for name, client := range m.clients {
        if client.toolset == nil {
            continue
        }

        tools, err := client.toolset.Tools(nil) // ReadonlyContext
        if err != nil {
            continue
        }

        for _, tool := range tools {
            key := fmt.Sprintf("mcp__%s__%s", name, tool.Name())
            result[key] = &mcpToolWrapper{
                client: client,
                tool:   tool,
            }
        }
    }

    return result, nil
}

// mcpToolWrapper wraps an MCP tool to implement our Tool interface
type mcpToolWrapper struct {
    client *Client
    tool   tool.Tool
}

func (w *mcpToolWrapper) ID() string {
    return w.tool.Name()
}

func (w *mcpToolWrapper) Description() string {
    return w.tool.Description()
}

func (w *mcpToolWrapper) Parameters() json.RawMessage {
    // Get parameters from the MCP tool
    // This depends on the underlying tool implementation
    return nil
}

func (w *mcpToolWrapper) Execute(ctx context.Context, args json.RawMessage, opts ExecuteOptions) (*Result, error) {
    // Execute via MCP protocol
    // Implementation depends on mcptoolset internals
    return nil, fmt.Errorf("not implemented")
}
```

### 4.5 Session Management

**Go Implementation:**

```go
package session

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "time"
)

// Session represents a conversation session
type Session struct {
    ID        string       `json:"id"`
    ProjectID string       `json:"projectID"`
    Directory string       `json:"directory"`
    ParentID  *string      `json:"parentID,omitempty"`
    Title     string       `json:"title"`
    Version   string       `json:"version"`
    Summary   Summary      `json:"summary"`
    Time      TimeInfo     `json:"time"`
}

// Summary holds session statistics
type Summary struct {
    Additions int `json:"additions"`
    Deletions int `json:"deletions"`
    Files     int `json:"files"`
}

// TimeInfo holds timestamps
type TimeInfo struct {
    Created int64 `json:"created"`
    Updated int64 `json:"updated"`
}

// Message represents a conversation message
type Message struct {
    ID         string       `json:"id"`
    SessionID  string       `json:"sessionID"`
    Role       string       `json:"role"` // "user" or "assistant"
    ParentID   *string      `json:"parentID,omitempty"`
    Time       MessageTime  `json:"time"`

    // User-specific
    Agent  string             `json:"agent,omitempty"`
    Model  *ModelRef          `json:"model,omitempty"`
    System *string            `json:"system,omitempty"`
    Tools  map[string]bool    `json:"tools,omitempty"`

    // Assistant-specific
    ModelID    string       `json:"modelID,omitempty"`
    ProviderID string       `json:"providerID,omitempty"`
    Mode       string       `json:"mode,omitempty"`
    Finish     *string      `json:"finish,omitempty"`
    Cost       float64      `json:"cost,omitempty"`
    Tokens     *TokenUsage  `json:"tokens,omitempty"`
}

// Part represents a message component
type Part interface {
    PartType() string
    PartID() string
}

// TextPart represents text content
type TextPart struct {
    ID        string `json:"id"`
    Type      string `json:"type"` // "text"
    Text      string `json:"text"`
    Synthetic bool   `json:"synthetic,omitempty"`
}

// ToolPart represents a tool invocation
type ToolPart struct {
    ID       string         `json:"id"`
    Type     string         `json:"type"` // "tool"
    CallID   string         `json:"callID"`
    Tool     string         `json:"tool"`
    State    ToolState      `json:"state"`
}

// ToolState represents tool execution state
type ToolState struct {
    Status   string         `json:"status"` // "pending", "running", "completed", "error"
    Input    map[string]any `json:"input,omitempty"`
    Output   string         `json:"output,omitempty"`
    Error    string         `json:"error,omitempty"`
    Title    string         `json:"title,omitempty"`
    Metadata map[string]any `json:"metadata,omitempty"`
    Time     ToolTime       `json:"time"`
}

// Store provides persistent session storage
type Store struct {
    basePath string
    mu       sync.RWMutex
}

func NewStore(basePath string) *Store {
    return &Store{basePath: basePath}
}

func (s *Store) Create(ctx context.Context, session *Session) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    session.Time.Created = time.Now().UnixMilli()
    session.Time.Updated = session.Time.Created

    return s.save(session)
}

func (s *Store) Get(ctx context.Context, sessionID string) (*Session, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    path := s.sessionPath(sessionID)
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("session not found: %s", sessionID)
    }

    var session Session
    if err := json.Unmarshal(data, &session); err != nil {
        return nil, err
    }

    return &session, nil
}

func (s *Store) Update(ctx context.Context, session *Session) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    session.Time.Updated = time.Now().UnixMilli()
    return s.save(session)
}

func (s *Store) Delete(ctx context.Context, sessionID string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    return os.Remove(s.sessionPath(sessionID))
}

func (s *Store) List(ctx context.Context, projectID string) ([]*Session, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    dir := filepath.Join(s.basePath, "session", projectID)
    entries, err := os.ReadDir(dir)
    if err != nil {
        return nil, err
    }

    var sessions []*Session
    for _, entry := range entries {
        if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
            continue
        }

        data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
        if err != nil {
            continue
        }

        var session Session
        if err := json.Unmarshal(data, &session); err != nil {
            continue
        }

        sessions = append(sessions, &session)
    }

    return sessions, nil
}

func (s *Store) save(session *Session) error {
    dir := filepath.Join(s.basePath, "session", session.ProjectID)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }

    data, err := json.MarshalIndent(session, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(s.sessionPath(session.ID), data, 0644)
}

func (s *Store) sessionPath(sessionID string) string {
    // Note: In production, you'd need to look up the projectID
    return filepath.Join(s.basePath, "session", sessionID+".json")
}
```

---

## 5. Gaps and Custom Implementation Requirements

### 5.1 Provider-Specific Features Not in ADK-Go

| Feature | Required For | Implementation Approach |
|---------|--------------|------------------------|
| **Anthropic Beta Headers** | Extended thinking, tool streaming | Custom Anthropic provider |
| **OpenAI Responses API** | o1/o3 reasoning models | Custom OpenAI provider |
| **Cache Control** | Token cost optimization | Provider-specific headers |
| **Azure Cognitive Services** | Enterprise deployments | Custom Azure provider |
| **AWS Bedrock Credentials** | Bedrock deployments | AWS credential chain |
| **Vertex AI** | Google Cloud | Custom Vertex provider |

### 5.2 Missing Abstractions

1. **Model Middleware**: ADK-Go uses callbacks instead of middleware. Need wrapper pattern:

```go
type ModelMiddleware func(next Provider) Provider

func WrapProvider(provider Provider, middleware ...ModelMiddleware) Provider {
    for i := len(middleware) - 1; i >= 0; i-- {
        provider = middleware[i](provider)
    }
    return provider
}

// Example: Cache control middleware
func CacheControlMiddleware(next Provider) Provider {
    return &cacheProvider{next: next}
}
```

2. **Provider Options**: Need unified options handling:

```go
type ProviderOptions struct {
    Anthropic  *AnthropicOptions
    OpenAI     *OpenAIOptions
    Google     *GoogleOptions
    Bedrock    *BedrockOptions
    // ... etc
}

func ApplyProviderOptions(req *LLMRequest, opts ProviderOptions) {
    // Apply provider-specific options
}
```

---

## 6. Recommendations

### 6.1 What to Use from ADK-Go

1. **Architecture Patterns**
   - Agent interface design
   - Iterator-based streaming (`iter.Seq2`)
   - Tool interface pattern
   - Session/Event model

2. **MCP Integration**
   - Use `mcptoolset` directly or as reference
   - MCP SDK Go integration patterns

3. **Server Components**
   - REST API handler patterns from `adkrest`
   - Event streaming patterns from `adka2a`

### 6.2 What to Build Custom

1. **Multi-Provider LLM Interface**
   - Custom implementations for Anthropic, OpenAI, Google, Azure, Bedrock
   - Provider-specific option handling
   - Cache control support

2. **Tool System**
   - Extend ADK-Go's pattern with OpenCode-specific requirements
   - Permission checking integration
   - Real-time metadata updates

3. **Session Management**
   - File-based storage (matching TypeScript implementation)
   - Message/Part storage
   - Event bus integration

### 6.3 Implementation Priority

1. **Phase 1**: Core Provider Abstraction
   - Base `Provider` interface
   - Anthropic implementation (primary)
   - OpenAI implementation
   - Streaming support

2. **Phase 2**: Tool System
   - Tool interface
   - Registry
   - Built-in tools (read, write, edit, bash, glob, grep)

3. **Phase 3**: Session Management
   - Storage layer
   - Message handling
   - Event bus

4. **Phase 4**: MCP Integration
   - Leverage ADK-Go's mcptoolset
   - Custom transport implementations

---

## 7. Code Examples for Migration

### 7.1 Migrating streamText

**TypeScript (Vercel AI SDK):**
```typescript
const result = await streamText({
  model: wrapLanguageModel({ model, middleware }),
  messages,
  tools,
  maxOutputTokens: 32000,
  providerOptions: { anthropic: { thinking: { type: "enabled" } } },
})
```

**Go Equivalent:**
```go
result, err := streaming.StreamText(ctx, streaming.StreamOptions{
    Provider:        anthropicProvider,
    Model:          "claude-sonnet-4",
    Messages:       messages,
    Tools:          tools,
    MaxOutputTokens: 32000,
    ProviderOptions: provider.Options{
        Anthropic: &provider.AnthropicOptions{
            Thinking: &provider.ThinkingConfig{Type: "enabled"},
        },
    },
})
```

### 7.2 Migrating Tool Definitions

**TypeScript:**
```typescript
const readTool = tool({
  id: "read",
  description: "Read a file",
  inputSchema: jsonSchema({
    type: "object",
    properties: { file_path: { type: "string" } },
    required: ["file_path"],
  }),
  async execute(args) {
    return { output: await fs.readFile(args.file_path, "utf-8") }
  },
})
```

**Go Equivalent:**
```go
type ReadArgs struct {
    FilePath string `json:"file_path"`
}

readTool := tool.NewFunctionTool[ReadArgs, tool.Result](
    "read",
    "Read a file",
    json.RawMessage(`{
        "type": "object",
        "properties": {"file_path": {"type": "string"}},
        "required": ["file_path"]
    }`),
    func(ctx context.Context, args ReadArgs, opts tool.ExecuteOptions) (*tool.Result, error) {
        content, err := os.ReadFile(args.FilePath)
        if err != nil {
            return nil, err
        }
        return &tool.Result{Output: string(content)}, nil
    },
)
```

---

## 8. Conclusion

ADK-Go provides valuable architectural patterns but is not a drop-in replacement for the Vercel AI SDK. The recommended approach is:

1. **Use ADK-Go as reference** for Go-idiomatic patterns
2. **Build custom provider implementations** using direct Go SDKs
3. **Leverage ADK-Go's mcptoolset** for MCP integration
4. **Implement OpenCode-specific features** (session storage, event bus, tool permissions)

This hybrid approach provides the best balance of leveraging existing work while meeting OpenCode's specific requirements.

---

*Document Version: 1.0*
*Last Updated: 2025-11-26*
