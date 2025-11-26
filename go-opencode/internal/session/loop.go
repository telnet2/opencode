package session

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/cloudwego/eino/schema"
	"github.com/oklog/ulid/v2"

	"github.com/opencode-ai/opencode/internal/event"
	"github.com/opencode-ai/opencode/internal/provider"
	"github.com/opencode-ai/opencode/pkg/types"
)

const (
	// MaxSteps is the maximum number of agentic loop iterations.
	MaxSteps = 50
	// MaxRetries is the maximum number of retries for API errors.
	MaxRetries = 3
	// RetryInitialInterval is the initial interval for exponential backoff.
	RetryInitialInterval = time.Second
	// RetryMaxInterval is the maximum interval for exponential backoff.
	RetryMaxInterval = 30 * time.Second
	// RetryMaxElapsedTime is the maximum total time for retries.
	RetryMaxElapsedTime = 2 * time.Minute
	// MaxContextTokens is the threshold for triggering context compaction.
	MaxContextTokens = 150000
)

// newRetryBackoff creates a new exponential backoff with jitter for API retries.
// Uses cenkalti/backoff for better retry behavior including jitter to prevent
// thundering herd problems and context-aware cancellation.
func newRetryBackoff(ctx context.Context) backoff.BackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = RetryInitialInterval
	b.MaxInterval = RetryMaxInterval
	b.MaxElapsedTime = RetryMaxElapsedTime
	b.RandomizationFactor = 0.5 // Add jitter
	b.Multiplier = 2.0
	b.Reset()
	return backoff.WithContext(backoff.WithMaxRetries(b, MaxRetries), ctx)
}

// runLoop executes the agentic loop.
func (p *Processor) runLoop(
	ctx context.Context,
	sessionID string,
	state *sessionState,
	agent *Agent,
	callback ProcessCallback,
) error {
	// Load session
	var session types.Session
	if err := p.storage.Get(ctx, []string{"session", sessionID}, &session); err != nil {
		// Try to find session in any project
		session, err := p.findSession(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("session not found: %w", err)
		}
		_ = session
	}

	// Load messages
	messages, err := p.loadMessages(ctx, sessionID)
	if err != nil {
		return err
	}

	if len(messages) == 0 {
		return fmt.Errorf("no messages in session")
	}

	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != "user" {
		return fmt.Errorf("expected user message, got %s", lastMsg.Role)
	}

	// Get provider and model
	providerID := "anthropic"
	modelID := "claude-sonnet-4-20250514"

	if lastMsg.Model != nil {
		providerID = lastMsg.Model.ProviderID
		modelID = lastMsg.Model.ModelID
	}

	prov, err := p.providerRegistry.Get(providerID)
	if err != nil {
		return fmt.Errorf("provider not found: %w", err)
	}

	model, err := p.providerRegistry.GetModel(providerID, modelID)
	if err != nil {
		return fmt.Errorf("model not found: %w", err)
	}

	// Create assistant message
	now := time.Now().UnixMilli()
	assistantMsg := &types.Message{
		ID:         generatePartID(),
		SessionID:  sessionID,
		Role:       "assistant",
		ProviderID: providerID,
		ModelID:    modelID,
		Time: types.MessageTime{
			Created: now,
		},
	}
	state.message = assistantMsg

	// Save initial message
	if err := p.storage.Put(ctx, []string{"message", sessionID, assistantMsg.ID}, assistantMsg); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Notify callback
	callback(assistantMsg, nil)

	// Publish event
	event.Publish(event.Event{
		Type: event.MessageCreated,
		Data: event.MessageCreatedData{Message: assistantMsg},
	})

	// Get agent config
	if agent == nil {
		agent = DefaultAgent()
	}

	maxSteps := agent.MaxSteps
	if maxSteps <= 0 {
		maxSteps = MaxSteps
	}

	// Run loop
	step := 0
	retryBackoff := newRetryBackoff(ctx)

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			assistantMsg.Error = &types.MessageError{
				Type:    "abort",
				Message: "Processing aborted",
			}
			p.saveMessage(ctx, sessionID, assistantMsg)
			return ctx.Err()
		default:
		}

		// Check step limit
		if step >= maxSteps {
			assistantMsg.Error = &types.MessageError{
				Type:    "max_steps",
				Message: "Maximum steps reached",
			}
			p.saveMessage(ctx, sessionID, assistantMsg)
			return fmt.Errorf("max steps exceeded")
		}

		// Check for context overflow and compact if needed
		if p.shouldCompact(messages) {
			if err := p.compactMessages(ctx, sessionID, messages); err != nil {
				// Log but don't fail
			}
			// Reload messages
			messages, _ = p.loadMessages(ctx, sessionID)
		}

		// Build completion request
		req, err := p.buildCompletionRequest(ctx, sessionID, messages, assistantMsg, agent, model)
		if err != nil {
			return fmt.Errorf("failed to build request: %w", err)
		}

		// Call LLM with streaming
		stream, err := prov.CreateCompletion(ctx, req)
		if err != nil {
			// Use exponential backoff with jitter for retries
			nextInterval := retryBackoff.NextBackOff()
			if nextInterval == backoff.Stop {
				assistantMsg.Error = &types.MessageError{
					Type:    "api",
					Message: err.Error(),
				}
				p.saveMessage(ctx, sessionID, assistantMsg)
				return err
			}
			time.Sleep(nextInterval)
			continue
		}

		// Process stream
		finishReason, err := p.processStream(ctx, stream, state, callback)
		stream.Close()

		if err != nil {
			// Use exponential backoff with jitter for retries
			nextInterval := retryBackoff.NextBackOff()
			if nextInterval == backoff.Stop {
				assistantMsg.Error = &types.MessageError{
					Type:    "api",
					Message: err.Error(),
				}
				p.saveMessage(ctx, sessionID, assistantMsg)
				return err
			}
			time.Sleep(nextInterval)
			continue
		}

		// Reset backoff on success
		retryBackoff.Reset()

		// Check finish reason
		switch finishReason {
		case "stop", "end_turn":
			// Normal completion
			finish := "stop"
			assistantMsg.Finish = &finish
			p.saveMessage(ctx, sessionID, assistantMsg)
			return nil

		case "tool_use", "tool_calls":
			// Execute tools and continue loop
			if err := p.executeToolCalls(ctx, state, agent, callback); err != nil {
				// Tool execution errors don't stop the loop
				// The error is captured in the tool part
			}
			step++
			continue

		case "max_tokens", "length":
			// Output limit reached
			finish := "max_tokens"
			assistantMsg.Finish = &finish
			assistantMsg.Error = &types.MessageError{
				Type:    "output_length",
				Message: "Output length limit reached",
			}
			p.saveMessage(ctx, sessionID, assistantMsg)
			return nil

		case "error":
			// Use exponential backoff with jitter for retries
			nextInterval := retryBackoff.NextBackOff()
			if nextInterval == backoff.Stop {
				return fmt.Errorf("stream error: max retries exceeded")
			}
			time.Sleep(nextInterval)
			continue

		default:
			// Unknown finish reason, treat as stop
			assistantMsg.Finish = &finishReason
			p.saveMessage(ctx, sessionID, assistantMsg)
			return nil
		}
	}
}

// findSession finds a session by ID across all projects.
func (p *Processor) findSession(ctx context.Context, sessionID string) (*types.Session, error) {
	projects, err := p.storage.List(ctx, []string{"session"})
	if err != nil {
		return nil, err
	}

	for _, projectID := range projects {
		var session types.Session
		if err := p.storage.Get(ctx, []string{"session", projectID, sessionID}, &session); err == nil {
			return &session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", sessionID)
}

// loadMessages loads all messages for a session.
func (p *Processor) loadMessages(ctx context.Context, sessionID string) ([]*types.Message, error) {
	var messages []*types.Message
	err := p.storage.Scan(ctx, []string{"message", sessionID}, func(key string, data json.RawMessage) error {
		var msg types.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			return err
		}
		messages = append(messages, &msg)
		return nil
	})
	return messages, err
}

// saveMessage saves an assistant message.
func (p *Processor) saveMessage(ctx context.Context, sessionID string, msg *types.Message) error {
	now := time.Now().UnixMilli()
	msg.Time.Updated = &now

	if err := p.storage.Put(ctx, []string{"message", sessionID, msg.ID}, msg); err != nil {
		return err
	}

	event.Publish(event.Event{
		Type: event.MessageUpdated,
		Data: event.MessageUpdatedData{Message: msg},
	})

	return nil
}

// savePart saves a part for a message.
func (p *Processor) savePart(ctx context.Context, messageID string, part types.Part) error {
	return p.storage.Put(ctx, []string{"part", messageID, part.PartID()}, part)
}

// shouldCompact checks if messages should be compacted.
func (p *Processor) shouldCompact(messages []*types.Message) bool {
	totalTokens := 0
	for _, msg := range messages {
		if msg.Tokens != nil {
			totalTokens += msg.Tokens.Input + msg.Tokens.Output
		}
	}
	return totalTokens > MaxContextTokens
}

// buildCompletionRequest builds an LLM completion request.
func (p *Processor) buildCompletionRequest(
	ctx context.Context,
	sessionID string,
	messages []*types.Message,
	currentMsg *types.Message,
	agent *Agent,
	model *types.Model,
) (*provider.CompletionRequest, error) {
	// Build system prompt
	session, _ := p.findSession(ctx, sessionID)
	systemPrompt := NewSystemPrompt(session, agent, currentMsg.ProviderID, currentMsg.ModelID)

	// Convert messages to Eino format
	var einoMessages []*schema.Message

	// Add system message
	einoMessages = append(einoMessages, &schema.Message{
		Role:    schema.System,
		Content: systemPrompt.Build(),
	})

	// Add conversation history
	for _, msg := range messages {
		// Skip errored messages without content
		if msg.Error != nil && !p.hasUsableContent(ctx, msg) {
			continue
		}

		// Load parts for this message
		parts, err := p.loadParts(ctx, msg.ID)
		if err != nil {
			continue
		}

		einoMsg := p.convertMessage(msg, parts)
		einoMessages = append(einoMessages, einoMsg)
	}

	// Get enabled tools
	tools, err := p.resolveTools(agent, model)
	if err != nil {
		return nil, err
	}

	// Build request
	maxTokens := model.MaxOutputTokens
	if maxTokens <= 0 {
		maxTokens = 8192
	}

	req := &provider.CompletionRequest{
		Model:       model.ID,
		Messages:    einoMessages,
		Tools:       tools,
		MaxTokens:   maxTokens,
		Temperature: agent.Temperature,
		TopP:        agent.TopP,
	}

	return req, nil
}

// loadParts loads all parts for a message.
func (p *Processor) loadParts(ctx context.Context, messageID string) ([]types.Part, error) {
	var parts []types.Part
	err := p.storage.Scan(ctx, []string{"part", messageID}, func(key string, data json.RawMessage) error {
		part, err := types.UnmarshalPart(data)
		if err != nil {
			return err
		}
		parts = append(parts, part)
		return nil
	})
	return parts, err
}

// hasUsableContent checks if a message has content worth including.
func (p *Processor) hasUsableContent(ctx context.Context, msg *types.Message) bool {
	parts, err := p.loadParts(ctx, msg.ID)
	if err != nil {
		return false
	}
	return len(parts) > 0
}

// convertMessage converts a types.Message to schema.Message.
func (p *Processor) convertMessage(msg *types.Message, parts []types.Part) *schema.Message {
	role := schema.Assistant
	switch msg.Role {
	case "user":
		role = schema.User
	case "system":
		role = schema.System
	case "tool":
		role = schema.Tool
	}

	// Build content from parts
	var content string
	var toolCalls []schema.ToolCall
	var toolCallID string

	for _, part := range parts {
		switch pt := part.(type) {
		case *types.TextPart:
			content += pt.Text
		case *types.ToolPart:
			if msg.Role == "assistant" {
				inputJSON, _ := json.Marshal(pt.Input)
				toolCalls = append(toolCalls, schema.ToolCall{
					ID: pt.ToolCallID,
					Function: schema.FunctionCall{
						Name:      pt.ToolName,
						Arguments: string(inputJSON),
					},
				})
			} else {
				// Tool result
				toolCallID = pt.ToolCallID
				if pt.Output != nil {
					content = *pt.Output
				} else if pt.Error != nil {
					content = "Error: " + *pt.Error
				}
			}
		}
	}

	einoMsg := &schema.Message{
		Role:      role,
		Content:   content,
		ToolCalls: toolCalls,
	}

	if toolCallID != "" {
		einoMsg.ToolCallID = toolCallID
	}

	return einoMsg
}

// resolveTools returns tools enabled for the agent.
func (p *Processor) resolveTools(agent *Agent, model *types.Model) ([]*schema.ToolInfo, error) {
	if !model.SupportsTools {
		return nil, nil
	}

	allTools := p.toolRegistry.List()

	var result []*schema.ToolInfo

	for _, t := range allTools {
		if !agent.ToolEnabled(t.ID()) {
			continue
		}

		params := parseJSONSchemaToParams(t.Parameters())
		result = append(result, &schema.ToolInfo{
			Name:        t.ID(),
			Desc:        t.Description(),
			ParamsOneOf: schema.NewParamsOneOfByParams(params),
		})
	}

	return result, nil
}

// parseJSONSchemaToParams converts JSON Schema to Eino ParameterInfo.
func parseJSONSchemaToParams(schemaJSON json.RawMessage) map[string]*schema.ParameterInfo {
	var jsonSchema struct {
		Properties map[string]struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"properties"`
		Required []string `json:"required"`
	}

	if err := json.Unmarshal(schemaJSON, &jsonSchema); err != nil {
		return nil
	}

	requiredSet := make(map[string]bool)
	for _, r := range jsonSchema.Required {
		requiredSet[r] = true
	}

	params := make(map[string]*schema.ParameterInfo)
	for name, prop := range jsonSchema.Properties {
		paramType := schema.String
		switch prop.Type {
		case "integer":
			paramType = schema.Integer
		case "number":
			paramType = schema.Number
		case "boolean":
			paramType = schema.Boolean
		case "array":
			paramType = schema.Array
		case "object":
			paramType = schema.Object
		}

		params[name] = &schema.ParameterInfo{
			Type:     paramType,
			Desc:     prop.Description,
			Required: requiredSet[name],
		}
	}

	return params
}

// generatePartID generates a new ULID for parts.
func generatePartID() string {
	return ulid.Make().String()
}

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}

// processStream is defined in stream.go

// Stub for io.EOF check - the actual implementation is in stream.go
var _ = io.EOF
