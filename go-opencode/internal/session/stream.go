package session

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"

	"github.com/opencode-ai/opencode/internal/event"
	"github.com/opencode-ai/opencode/internal/provider"
	"github.com/opencode-ai/opencode/pkg/types"
)

// processStream processes events from the LLM stream.
func (p *Processor) processStream(
	ctx context.Context,
	stream *provider.CompletionStream,
	state *sessionState,
	callback ProcessCallback,
) (string, error) {
	var currentTextPart *types.TextPart
	var currentReasoningPart *types.ReasoningPart
	var currentToolParts map[string]*types.ToolPart
	var finishReason string
	var accumulatedContent string
	var accumulatedToolInputs map[string]string

	currentToolParts = make(map[string]*types.ToolPart)
	accumulatedToolInputs = make(map[string]string)

	// Track token usage across stream events
	// Eino claude framework sends usage in two events:
	// - MessageStartEvent (first): contains PromptTokens and cache info
	// - MessageDeltaEvent (last): contains CompletionTokens only
	// We need to merge both to get complete usage.
	var inputTokens, completionTokens, cachedTokens int
	var hasUsage bool

	// Emit step-start part at the beginning of inference
	stepStartPart := &types.StepStartPart{
		ID:        generatePartID(),
		SessionID: state.message.SessionID,
		MessageID: state.message.ID,
		Type:      "step-start",
	}
	state.parts = append(state.parts, stepStartPart)
	p.savePart(ctx, state.message.ID, stepStartPart)
	event.PublishSync(event.Event{
		Type: event.MessagePartUpdated,
		Data: event.MessagePartUpdatedData{Part: stepStartPart},
	})
	callback(state.message, state.parts)

	for {
		select {
		case <-ctx.Done():
			return "error", ctx.Err()
		default:
		}

		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "error", err
		}

		// Process the message chunk
		finishReason = p.processMessageChunk(ctx, msg, state, callback,
			&currentTextPart, &currentReasoningPart, currentToolParts,
			&accumulatedContent, accumulatedToolInputs)

		// Collect token usage - merge from different stream events
		// MessageStartEvent has PromptTokens, MessageDeltaEvent has CompletionTokens
		if msg.ResponseMeta != nil && msg.ResponseMeta.Usage != nil {
			usage := msg.ResponseMeta.Usage
			hasUsage = true
			// Take the max of each field (first event has input, last has output)
			if usage.PromptTokens > inputTokens {
				inputTokens = usage.PromptTokens
			}
			if usage.CompletionTokens > completionTokens {
				completionTokens = usage.CompletionTokens
			}
			if usage.PromptTokenDetails.CachedTokens > cachedTokens {
				cachedTokens = usage.PromptTokenDetails.CachedTokens
			}
		}

		if finishReason != "" {
			break
		}
	}

	// Finalize any open parts
	if currentTextPart != nil {
		now := time.Now().UnixMilli()
		currentTextPart.Time.End = &now
		p.savePart(ctx, state.message.ID, currentTextPart)
	}

	if currentReasoningPart != nil {
		now := time.Now().UnixMilli()
		currentReasoningPart.Time.End = &now
		p.savePart(ctx, state.message.ID, currentReasoningPart)
	}

	// Finalize tool parts
	for id, toolPart := range currentToolParts {
		if accInput, ok := accumulatedToolInputs[id]; ok && toolPart.State.Input == nil {
			var input map[string]any
			if err := json.Unmarshal([]byte(accInput), &input); err == nil {
				toolPart.State.Input = input
			}
		}
		toolPart.State.Status = "running"
		p.savePart(ctx, state.message.ID, toolPart)
	}

	// Determine finish reason from accumulated state
	if finishReason == "" {
		if len(currentToolParts) > 0 {
			finishReason = "tool-calls" // SDK compatible: TypeScript uses "tool-calls"
		} else {
			finishReason = "stop"
		}
	}

	// Normalize finish reason to SDK-compatible format
	// TypeScript uses "tool-calls" but some providers return "tool_use"
	if finishReason == "tool_use" {
		finishReason = "tool-calls"
	}

	// Apply merged token usage after stream completes
	if hasUsage {
		if state.message.Tokens == nil {
			state.message.Tokens = &types.TokenUsage{}
		}
		state.message.Tokens.Input = inputTokens
		state.message.Tokens.Output = completionTokens
		state.message.Tokens.Cache.Read = cachedTokens
	}

	// Emit step-finish part at the end of inference with cost and token info
	stepFinishPart := &types.StepFinishPart{
		ID:        generatePartID(),
		SessionID: state.message.SessionID,
		MessageID: state.message.ID,
		Type:      "step-finish",
		Reason:    finishReason,
		Cost:      state.message.Cost,
		Tokens:    state.message.Tokens,
	}
	state.parts = append(state.parts, stepFinishPart)
	p.savePart(ctx, state.message.ID, stepFinishPart)
	event.PublishSync(event.Event{
		Type: event.MessagePartUpdated,
		Data: event.MessagePartUpdatedData{Part: stepFinishPart},
	})
	callback(state.message, state.parts)

	return finishReason, nil
}

// truncate truncates a string to the specified length.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// processMessageChunk handles a single message chunk from the stream.
func (p *Processor) processMessageChunk(
	ctx context.Context,
	msg *schema.Message,
	state *sessionState,
	callback ProcessCallback,
	currentTextPart **types.TextPart,
	currentReasoningPart **types.ReasoningPart,
	currentToolParts map[string]*types.ToolPart,
	accumulatedContent *string,
	accumulatedToolInputs map[string]string,
) string {
	var finishReason string

	// Handle text content
	if msg.Content != "" {
		// Check if this is new content (delta)
		if *currentTextPart == nil {
			// Start new text part
			now := time.Now().UnixMilli()
			*currentTextPart = &types.TextPart{
				ID:        generatePartID(),
				SessionID: state.message.SessionID,
				MessageID: state.message.ID,
				Type:      "text",
				Text:      msg.Content,
				Time:      types.PartTime{Start: &now},
			}
			state.parts = append(state.parts, *currentTextPart)
			*accumulatedContent = msg.Content

			// Publish delta event for FIRST chunk (SDK compatible)
			event.PublishSync(event.Event{
				Type: event.MessagePartUpdated,
				Data: event.MessagePartUpdatedData{
					Part:  *currentTextPart,
					Delta: msg.Content, // First chunk IS the delta
				},
			})

			callback(state.message, state.parts)
		} else {
			// Check if this is accumulated content (starts with previous) or delta content (new chunk only)
			var delta string
			if strings.HasPrefix(msg.Content, *accumulatedContent) {
				// Accumulated mode: new content STARTS WITH all previous content
				delta = msg.Content[len(*accumulatedContent):]
				(*currentTextPart).Text = msg.Content
				*accumulatedContent = msg.Content
			} else {
				// Delta mode: new content is just the new part
				delta = msg.Content
				*accumulatedContent += msg.Content
				(*currentTextPart).Text = *accumulatedContent
			}

			// Publish delta event (SDK compatible: uses MessagePartUpdated)
			event.PublishSync(event.Event{
				Type: event.MessagePartUpdated,
				Data: event.MessagePartUpdatedData{
					Part:  *currentTextPart,
					Delta: delta,
				},
			})

			callback(state.message, state.parts)
		}
	}

	// Handle reasoning content (extended thinking)
	if msg.ReasoningContent != "" {
		if *currentReasoningPart == nil {
			now := time.Now().UnixMilli()
			*currentReasoningPart = &types.ReasoningPart{
				ID:        generatePartID(),
				SessionID: state.message.SessionID,
				MessageID: state.message.ID,
				Type:      "reasoning",
				Text:      msg.ReasoningContent,
				Time:      types.PartTime{Start: &now},
			}
			state.parts = append(state.parts, *currentReasoningPart)
			callback(state.message, state.parts)
		} else {
			(*currentReasoningPart).Text = msg.ReasoningContent
			callback(state.message, state.parts)
		}
	}

	// Handle tool calls
	// The eino streaming model uses Index to track tool calls:
	// - Start event: Index=N, ID="toolu_xxx", Name="Read"
	// - Delta events: Index=N, ID="", Name="", Arguments='{"partial...'
	for _, tc := range msg.ToolCalls {
		// Use Index to track tool calls (eino streaming model)
		var toolIndex int
		if tc.Index != nil {
			toolIndex = *tc.Index
		} else if tc.ID != "" {
			// Fallback: use ID-based tracking if Index not available
			toolIndex = -1 // Will use ID map
		} else {
			continue
		}

		// Determine lookup key - use index string or ID
		var lookupKey string
		if toolIndex >= 0 {
			lookupKey = "idx:" + string(rune('0'+toolIndex))
		} else {
			lookupKey = tc.ID
		}

		toolPart, exists := currentToolParts[lookupKey]

		// New tool call (has ID and Name)
		if !exists && tc.ID != "" && tc.Function.Name != "" {
			now := time.Now().UnixMilli()
			toolPart = &types.ToolPart{
				ID:        generatePartID(),
				SessionID: state.message.SessionID,
				MessageID: state.message.ID,
				Type:      "tool",
				CallID:    tc.ID,
				Tool:      tc.Function.Name,
				State: types.ToolState{
					Status: "pending",
					Input:  make(map[string]any),
					Raw:    "",
					Time:   &types.ToolTime{Start: now},
				},
			}
			currentToolParts[lookupKey] = toolPart
			accumulatedToolInputs[lookupKey] = ""
			state.parts = append(state.parts, toolPart)
			callback(state.message, state.parts)
		}

		// Accumulate arguments (delta chunks have arguments but no ID/Name)
		if tc.Function.Arguments != "" && toolPart != nil {
			// Append arguments (eino sends deltas, not accumulated)
			accumulatedToolInputs[lookupKey] += tc.Function.Arguments
			toolPart.State.Raw = accumulatedToolInputs[lookupKey]

			// Try to parse accumulated JSON
			var input map[string]any
			if err := json.Unmarshal([]byte(accumulatedToolInputs[lookupKey]), &input); err == nil {
				toolPart.State.Input = input
			}

			// Publish tool part update (SDK compatible: uses MessagePartUpdated)
			event.PublishSync(event.Event{
				Type: event.MessagePartUpdated,
				Data: event.MessagePartUpdatedData{
					Part: toolPart,
				},
			})

			callback(state.message, state.parts)
		}
	}

	// Check for finish reason in response metadata
	// Note: Token usage is collected in processStream and applied after the stream completes
	if msg.ResponseMeta != nil && msg.ResponseMeta.FinishReason != "" {
		finishReason = msg.ResponseMeta.FinishReason
	}

	return finishReason
}

// StreamEvent represents different types of stream events.
type StreamEvent interface {
	streamEvent()
}

// TextStartEvent indicates the start of text content.
type TextStartEvent struct{}

func (TextStartEvent) streamEvent() {}

// TextDeltaEvent contains a text delta.
type TextDeltaEvent struct {
	Text string
}

func (TextDeltaEvent) streamEvent() {}

// TextEndEvent indicates the end of text content.
type TextEndEvent struct{}

func (TextEndEvent) streamEvent() {}

// ReasoningStartEvent indicates the start of reasoning content.
type ReasoningStartEvent struct{}

func (ReasoningStartEvent) streamEvent() {}

// ReasoningDeltaEvent contains a reasoning delta.
type ReasoningDeltaEvent struct {
	Text string
}

func (ReasoningDeltaEvent) streamEvent() {}

// ReasoningEndEvent indicates the end of reasoning content.
type ReasoningEndEvent struct{}

func (ReasoningEndEvent) streamEvent() {}

// ToolCallStartEvent indicates the start of a tool call.
type ToolCallStartEvent struct {
	ID   string
	Name string
}

func (ToolCallStartEvent) streamEvent() {}

// ToolCallDeltaEvent contains input delta for a tool call.
type ToolCallDeltaEvent struct {
	ID    string
	Delta string
}

func (ToolCallDeltaEvent) streamEvent() {}

// ToolCallEndEvent indicates completion of a tool call.
type ToolCallEndEvent struct {
	ID    string
	Input json.RawMessage
}

func (ToolCallEndEvent) streamEvent() {}

// FinishEvent indicates stream completion.
type FinishEvent struct {
	Reason string
	Error  error
}

func (FinishEvent) streamEvent() {}
