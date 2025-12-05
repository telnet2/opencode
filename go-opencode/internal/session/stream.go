package session

import (
	"context"
	"encoding/json"
	"fmt"
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

	// Emit step-start part at the beginning of inference
	stepStartPart := &types.StepStartPart{
		ID:        generatePartID(),
		SessionID: state.message.SessionID,
		MessageID: state.message.ID,
		Type:      "step-start",
	}
	state.parts = append(state.parts, stepStartPart)
	p.savePart(ctx, state.message.ID, stepStartPart)
	event.Publish(event.Event{
		Type: event.MessagePartUpdated,
		Data: event.MessagePartUpdatedData{Part: stepStartPart},
	})
	callback(state.message, state.parts)

	fmt.Printf("[stream] Starting to receive chunks\n")
	chunkCount := 0
	var lastChunkTime time.Time
	var lastEventTime time.Time // For throttling event publishing

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
		now := time.Now()
		var delta time.Duration
		if !lastChunkTime.IsZero() {
			delta = now.Sub(lastChunkTime)
		}
		lastChunkTime = now
		fmt.Printf("[stream] Chunk %d (+%v): content=%q, toolCalls=%d, responseMeta=%v\n",
			chunkCount, delta, truncate(msg.Content, 50), len(msg.ToolCalls), msg.ResponseMeta != nil)

		// Process the message chunk
		finishReason = p.processMessageChunk(ctx, msg, state, callback,
			&currentTextPart, &currentReasoningPart, currentToolParts,
			&accumulatedContent, accumulatedToolInputs, &lastEventTime)

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
	fmt.Printf("[stream] Finalizing %d tool parts\n", len(currentToolParts))
	for id, toolPart := range currentToolParts {
		fmt.Printf("[stream] Finalizing toolPart: id=%s, tool=%s, callID=%s, currentStatus=%s\n",
			id, toolPart.Tool, toolPart.CallID, toolPart.State.Status)
		if accInput, ok := accumulatedToolInputs[id]; ok && toolPart.State.Input == nil {
			var input map[string]any
			if err := json.Unmarshal([]byte(accInput), &input); err == nil {
				toolPart.State.Input = input
			}
		}
		toolPart.State.Status = "running"
		fmt.Printf("[stream] Set toolPart status to 'running': tool=%s, ptr=%p\n", toolPart.Tool, toolPart)
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
	event.Publish(event.Event{
		Type: event.MessagePartUpdated,
		Data: event.MessagePartUpdatedData{Part: stepFinishPart},
	})
	callback(state.message, state.parts)

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

// MinEventInterval is the minimum time between streaming events.
// This ensures the TUI has time to process each event before the next arrives.
// Set to slightly above TUI's 16ms batching window to prevent batching.
const MinEventInterval = 20 * time.Millisecond

// throttledPublish publishes an event with optional throttling to prevent TUI batching.
func throttledPublish(e event.Event, lastEventTime *time.Time) {
	if lastEventTime != nil && !lastEventTime.IsZero() {
		elapsed := time.Since(*lastEventTime)
		if elapsed < MinEventInterval {
			sleepTime := MinEventInterval - elapsed
			fmt.Printf("[stream] THROTTLE sleep=%v (elapsed=%v)\n", sleepTime, elapsed)
			time.Sleep(sleepTime)
		}
	}
	event.Publish(e)
	if lastEventTime != nil {
		*lastEventTime = time.Now()
	}
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
	lastEventTime *time.Time,
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
			// This ensures the TUI receives and displays the first text chunk
			// Note: Uses throttledPublish to prevent TUI batching
			throttledPublish(event.Event{
				Type: event.MessagePartUpdated,
				Data: event.MessagePartUpdatedData{
					Part:  *currentTextPart,
					Delta: msg.Content, // First chunk IS the delta
				},
			}, lastEventTime)

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
			// Note: Uses throttledPublish to prevent TUI batching
			throttledPublish(event.Event{
				Type: event.MessagePartUpdated,
				Data: event.MessagePartUpdatedData{
					Part:  *currentTextPart,
					Delta: delta,
				},
			}, lastEventTime)

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
			fmt.Printf("[stream] Skipping tool call with no Index and no ID\n")
			continue
		}

		// Determine lookup key - use index string or ID
		var lookupKey string
		if toolIndex >= 0 {
			lookupKey = fmt.Sprintf("idx:%d", toolIndex)
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
			fmt.Printf("[stream] Created new ToolPart: tool=%s, callID=%s, index=%d\n", toolPart.Tool, toolPart.CallID, toolIndex)
			currentToolParts[lookupKey] = toolPart
			accumulatedToolInputs[lookupKey] = ""
			state.parts = append(state.parts, toolPart)
			fmt.Printf("[stream] Added toolPart to state.parts, total parts=%d\n", len(state.parts))
			callback(state.message, state.parts)
		}

		// Accumulate arguments (delta chunks have arguments but no ID/Name)
		if tc.Function.Arguments != "" && toolPart != nil {
			// Append arguments (eino sends deltas, not accumulated)
			accumulatedToolInputs[lookupKey] += tc.Function.Arguments
			toolPart.State.Raw = accumulatedToolInputs[lookupKey]
			fmt.Printf("[stream] Tool %s accumulated args: %s\n", toolPart.Tool, truncate(accumulatedToolInputs[lookupKey], 100))

			// Try to parse accumulated JSON
			var input map[string]any
			if err := json.Unmarshal([]byte(accumulatedToolInputs[lookupKey]), &input); err == nil {
				toolPart.State.Input = input
				fmt.Printf("[stream] Tool %s parsed input: %v\n", toolPart.Tool, input)
			}

			// Publish tool part update (SDK compatible: uses MessagePartUpdated)
			// Note: Must use async Publish so SSE select loop can process events
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
