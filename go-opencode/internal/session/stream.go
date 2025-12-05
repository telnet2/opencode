package session

import (
	"context"
	"encoding/json"
	"fmt"
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

	fmt.Printf("[stream] Starting to receive chunks\n")
	chunkCount := 0

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[stream] Context cancelled\n")
			return "error", ctx.Err()
		default:
		}

		msg, err := stream.Recv()
		if err == io.EOF {
			fmt.Printf("[stream] Received EOF after %d chunks\n", chunkCount)
			break
		}
		if err != nil {
			fmt.Printf("[stream] Error receiving chunk: %v\n", err)
			return "error", err
		}
		chunkCount++
		fmt.Printf("[stream] Chunk %d: content=%q, toolCalls=%d, responseMeta=%v\n",
			chunkCount, truncate(msg.Content, 50), len(msg.ToolCalls), msg.ResponseMeta != nil)

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
			finishReason = "tool_use"
		} else {
			finishReason = "stop"
		}
	}

	fmt.Printf("[stream] Finished with reason=%s, parts=%d, tokens=%v\n",
		finishReason, len(state.parts), state.message.Tokens)

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
			callback(state.message, state.parts)
		} else {
			// Check if this is accumulated content (longer than previous) or delta content (shorter)
			var delta string
			if len(msg.Content) > len(*accumulatedContent) {
				// Accumulated mode: extract delta from difference
				delta = msg.Content[len(*accumulatedContent):]
				(*currentTextPart).Text = msg.Content
				*accumulatedContent = msg.Content
			} else {
				// Delta mode: append delta directly
				delta = msg.Content
				*accumulatedContent += msg.Content
				(*currentTextPart).Text = *accumulatedContent
			}

			// Publish delta event (SDK compatible: uses MessagePartUpdated)
			event.Publish(event.Event{
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
	for _, tc := range msg.ToolCalls {
		toolPart, exists := currentToolParts[tc.ID]
		if !exists {
			// New tool call
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
			currentToolParts[tc.ID] = toolPart
			accumulatedToolInputs[tc.ID] = ""
			state.parts = append(state.parts, toolPart)
			callback(state.message, state.parts)
		}

		// Accumulate arguments
		if tc.Function.Arguments != "" {
			accumulatedToolInputs[tc.ID] = tc.Function.Arguments
			toolPart.State.Raw = tc.Function.Arguments
			var input map[string]any
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err == nil {
				toolPart.State.Input = input
			}

			// Publish tool part update (SDK compatible: uses MessagePartUpdated)
			event.Publish(event.Event{
				Type: event.MessagePartUpdated,
				Data: event.MessagePartUpdatedData{
					Part: toolPart,
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
