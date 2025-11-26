package session

import (
	"context"
	"encoding/json"
	"io"
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

	// Emit step start
	stepStartPart := &types.TextPart{
		ID:   generatePartID(),
		Type: "step-start",
	}
	_ = stepStartPart // We'll add step tracking later

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
		if accInput, ok := accumulatedToolInputs[id]; ok && toolPart.Input == nil {
			var input map[string]any
			if err := json.Unmarshal([]byte(accInput), &input); err == nil {
				toolPart.Input = input
			}
		}
		toolPart.State = "running"
		p.savePart(ctx, state.message.ID, toolPart)
	}

	// Determine finish reason from accumulated state
	if finishReason == "" {
		if len(currentToolParts) > 0 {
			finishReason = "tool_use"
		} else {
			finishReason = "stop"
		}
	}

	return finishReason, nil
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
				ID:   generatePartID(),
				Type: "text",
				Text: msg.Content,
				Time: types.PartTime{Start: &now},
			}
			state.parts = append(state.parts, *currentTextPart)
			*accumulatedContent = msg.Content
			callback(state.message, state.parts)
		} else if len(msg.Content) > len(*accumulatedContent) {
			// Append delta
			delta := msg.Content[len(*accumulatedContent):]
			(*currentTextPart).Text = msg.Content
			*accumulatedContent = msg.Content

			// Publish delta event
			event.Publish(event.Event{
				Type: event.PartUpdated,
				Data: event.PartUpdatedData{
					SessionID: state.message.SessionID,
					MessageID: state.message.ID,
					Part:      *currentTextPart,
					Delta:     &delta,
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
				ID:   generatePartID(),
				Type: "reasoning",
				Text: msg.ReasoningContent,
				Time: types.PartTime{Start: &now},
			}
			state.parts = append(state.parts, *currentReasoningPart)
			callback(state.message, state.parts)
		} else {
			(*currentReasoningPart).Text = msg.ReasoningContent
			callback(state.message, state.parts)
		}
	}

	// Handle tool calls
	for _, tc := range msg.ToolCalls {
		toolPart, exists := currentToolParts[tc.ID]
		if !exists {
			// New tool call
			now := time.Now().UnixMilli()
			toolPart = &types.ToolPart{
				ID:         generatePartID(),
				Type:       "tool",
				ToolCallID: tc.ID,
				ToolName:   tc.Function.Name,
				State:      "pending",
				Time:       types.PartTime{Start: &now},
			}
			currentToolParts[tc.ID] = toolPart
			accumulatedToolInputs[tc.ID] = ""
			state.parts = append(state.parts, toolPart)
			callback(state.message, state.parts)
		}

		// Accumulate arguments
		if tc.Function.Arguments != "" {
			accumulatedToolInputs[tc.ID] = tc.Function.Arguments
			var input map[string]any
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err == nil {
				toolPart.Input = input
			}

			event.Publish(event.Event{
				Type: event.PartUpdated,
				Data: event.PartUpdatedData{
					SessionID: state.message.SessionID,
					MessageID: state.message.ID,
					Part:      toolPart,
				},
			})

			callback(state.message, state.parts)
		}
	}

	// Check for response metadata (token usage)
	if msg.ResponseMeta != nil {
		if state.message.Tokens == nil {
			state.message.Tokens = &types.TokenUsage{}
		}

		if msg.ResponseMeta.Usage != nil {
			state.message.Tokens.Input = msg.ResponseMeta.Usage.PromptTokens
			state.message.Tokens.Output = msg.ResponseMeta.Usage.CompletionTokens
		}

		// Check finish reason
		if msg.ResponseMeta.FinishReason != "" {
			finishReason = msg.ResponseMeta.FinishReason
		}
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
