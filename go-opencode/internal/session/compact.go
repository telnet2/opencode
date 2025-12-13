package session

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"

	"github.com/opencode-ai/opencode/internal/event"
	"github.com/opencode-ai/opencode/internal/provider"
	"github.com/opencode-ai/opencode/pkg/types"
)

// CompactionConfig controls message compaction behavior.
type CompactionConfig struct {
	// MinMessagesToKeep is the minimum number of recent messages to keep.
	MinMessagesToKeep int

	// SummaryMaxTokens is the maximum tokens for the summary.
	SummaryMaxTokens int

	// ContextThreshold is the percentage of context usage that triggers compaction.
	ContextThreshold float64
}

// DefaultCompactionConfig returns the default compaction configuration.
var DefaultCompactionConfig = CompactionConfig{
	MinMessagesToKeep: 4,
	SummaryMaxTokens:  2000,
	ContextThreshold:  0.75,
}

// compactMessages summarizes old messages to free context.
func (p *Processor) compactMessages(
	ctx context.Context,
	sessionID string,
	messages []*types.Message,
) error {
	if len(messages) <= DefaultCompactionConfig.MinMessagesToKeep {
		return nil
	}

	// Update session compacting flag
	session, err := p.findSession(ctx, sessionID)
	if err != nil {
		return err
	}

	now := time.Now().UnixMilli()
	session.Time.Compacting = &now
	p.storage.Put(ctx, []string{"session", session.ProjectID, session.ID}, session)

	defer func() {
		session.Time.Compacting = nil
		p.storage.Put(ctx, []string{"session", session.ProjectID, session.ID}, session)
	}()

	// Determine which messages to compact
	compactEnd := len(messages) - DefaultCompactionConfig.MinMessagesToKeep
	toCompact := messages[:compactEnd]

	// Build summary prompt
	summaryPrompt := buildSummaryPrompt(ctx, p, toCompact)

	// Get default model for summarization
	model, err := p.providerRegistry.DefaultModel()
	if err != nil {
		return err
	}

	prov, err := p.providerRegistry.Get(model.ProviderID)
	if err != nil {
		return err
	}

	// Generate summary
	systemMsg := &schema.Message{
		Role:    schema.System,
		Content: "You are a conversation summarizer. Create a concise summary of the conversation that preserves key context for continuing the discussion.",
	}

	userMsg := &schema.Message{
		Role:    schema.User,
		Content: summaryPrompt,
	}

	// Create streaming request
	stream, err := prov.CreateCompletion(ctx, &provider.CompletionRequest{
		Model:     model.ID,
		Messages:  []*schema.Message{systemMsg, userMsg},
		MaxTokens: DefaultCompactionConfig.SummaryMaxTokens,
	})
	if err != nil {
		return err
	}
	defer stream.Close()

	// Collect response
	var summary strings.Builder
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		summary.WriteString(msg.Content)
	}

	// Mark compacted messages as summarized
	for _, msg := range toCompact {
		// Update message to indicate it was summarized
		// In a full implementation, we might add a "summarized" field
		p.storage.Put(ctx, []string{"message", sessionID, msg.ID}, msg)
	}

	// Create compaction marker in session
	// This would be used to inject the summary into future prompts
	session.Summary.Diffs = append(session.Summary.Diffs, types.FileDiff{
		File:   "__compaction__",
		Before: "",
		After:  summary.String(),
	})
	p.storage.Put(ctx, []string{"session", session.ProjectID, session.ID}, session)

	return nil
}

// buildSummaryPrompt creates a prompt for summarizing messages.
func buildSummaryPrompt(ctx context.Context, p *Processor, messages []*types.Message) string {
	var prompt strings.Builder

	prompt.WriteString("Please summarize the following conversation, focusing on:\n")
	prompt.WriteString("1. Key decisions and outcomes\n")
	prompt.WriteString("2. Files that were modified\n")
	prompt.WriteString("3. Important context for continuing the work\n\n")
	prompt.WriteString("---\n\n")

	for _, msg := range messages {
		if msg.Role == "user" {
			prompt.WriteString("USER:\n")
		} else {
			prompt.WriteString("ASSISTANT:\n")
		}

		// Load parts for the message
		parts, err := p.loadParts(ctx, msg.ID)
		if err != nil {
			continue
		}

		for _, part := range parts {
			switch pt := part.(type) {
			case *types.TextPart:
				prompt.WriteString(pt.Text)
				prompt.WriteString("\n")
			case *types.ToolPart:
				prompt.WriteString(fmt.Sprintf("[Tool: %s]\n", pt.Tool))
				if pt.State.Output != "" {
					// Truncate long outputs
					output := pt.State.Output
					if len(output) > 500 {
						output = output[:500] + "..."
					}
					prompt.WriteString(output)
					prompt.WriteString("\n")
				}
			}
		}

		prompt.WriteString("\n")
	}

	return prompt.String()
}

// estimateTokens provides a rough estimate of token count.
func estimateTokens(text string) int {
	// Rough estimate: ~4 characters per token
	return len(text) / 4
}

// compactionSystemPrompt is the system prompt for generating summaries.
const compactionSystemPrompt = `You are a conversation summarizer. Create a concise summary of the conversation that preserves key context for continuing the discussion.

Focus on:
1. What was accomplished
2. Current work in progress
3. Files involved
4. Next steps
5. Any key user requests or constraints

Be concise but detailed enough that work can continue seamlessly.`

// processCompaction handles a compaction request by summarizing the conversation.
func (p *Processor) processCompaction(
	ctx context.Context,
	sessionID string,
	messages []*types.Message,
	compactionPart *types.CompactionPart,
	callback ProcessCallback,
) error {
	// Find session
	session, err := p.findSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Get the last user message (which contains the compaction part)
	lastMsg := messages[len(messages)-1]

	// Get provider and model from the user message
	providerID := p.defaultProviderID
	modelID := p.defaultModelID
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

	// Set compacting flag on session
	now := time.Now().UnixMilli()
	session.Time.Compacting = &now
	p.storage.Put(ctx, []string{"session", session.ProjectID, session.ID}, session)

	defer func() {
		session.Time.Compacting = nil
		p.storage.Put(ctx, []string{"session", session.ProjectID, session.ID}, session)
	}()

	// Build summary prompt from all messages except the compaction request itself
	summaryPrompt := buildSummaryPrompt(ctx, p, messages[:len(messages)-1])
	summaryPrompt += "\n\nSummarize our conversation above. This summary will be the only context available when the conversation continues, so preserve critical information including: what was accomplished, current work in progress, files involved, next steps, and any key user requests or constraints. Be concise but detailed enough that work can continue seamlessly."

	// Create assistant message with summary flag
	assistantMsg := &types.Message{
		ID:         generatePartID(),
		SessionID:  sessionID,
		Role:       "assistant",
		ParentID:   lastMsg.ID,
		ProviderID: providerID,
		ModelID:    modelID,
		Mode:       lastMsg.Agent,
		IsSummary:  true, // Mark as summary message
		Path: &types.MessagePath{
			Cwd:  session.Directory,
			Root: session.Directory,
		},
		Time: types.MessageTime{
			Created: now,
		},
		Tokens: &types.TokenUsage{Input: 0, Output: 0},
	}

	// Save initial message
	if err := p.storage.Put(ctx, []string{"message", sessionID, assistantMsg.ID}, assistantMsg); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Notify callback
	callback(assistantMsg, nil)

	// Publish message created event
	event.PublishSync(event.Event{
		Type: event.MessageCreated,
		Data: event.MessageCreatedData{Info: assistantMsg},
	})

	// Create text part for streaming the summary
	textPart := &types.TextPart{
		ID:        generatePartID(),
		SessionID: sessionID,
		MessageID: assistantMsg.ID,
		Type:      "text",
		Text:      "",
	}

	// Save initial part
	if err := p.storage.Put(ctx, []string{"part", assistantMsg.ID, textPart.ID}, textPart); err != nil {
		return fmt.Errorf("failed to save part: %w", err)
	}

	// Publish part created event
	event.PublishSync(event.Event{
		Type: event.MessagePartUpdated,
		Data: event.MessagePartUpdatedData{Part: textPart},
	})

	// Generate summary using LLM
	systemMsg := &schema.Message{
		Role:    schema.System,
		Content: compactionSystemPrompt,
	}

	userMsg := &schema.Message{
		Role:    schema.User,
		Content: summaryPrompt,
	}

	stream, err := prov.CreateCompletion(ctx, &provider.CompletionRequest{
		Model:     model.ID,
		Messages:  []*schema.Message{systemMsg, userMsg},
		MaxTokens: DefaultCompactionConfig.SummaryMaxTokens,
	})
	if err != nil {
		return fmt.Errorf("failed to create completion: %w", err)
	}
	defer stream.Close()

	// Stream the response
	var fullText strings.Builder
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("stream error: %w", err)
		}

		fullText.WriteString(msg.Content)
		textPart.Text = fullText.String()

		// Save updated part
		p.storage.Put(ctx, []string{"part", assistantMsg.ID, textPart.ID}, textPart)

		// Publish streaming update with delta
		event.PublishSync(event.Event{
			Type: event.MessagePartUpdated,
			Data: event.MessagePartUpdatedData{
				Part:  textPart,
				Delta: msg.Content,
			},
		})
	}

	// Update message with final token counts
	// (In a full implementation, we'd get actual token counts from the provider)
	assistantMsg.Tokens = &types.TokenUsage{
		Input:  estimateTokens(summaryPrompt),
		Output: estimateTokens(fullText.String()),
	}
	p.storage.Put(ctx, []string{"message", sessionID, assistantMsg.ID}, assistantMsg)

	// Publish message updated event
	event.PublishSync(event.Event{
		Type: event.MessageUpdated,
		Data: event.MessageUpdatedData{Info: assistantMsg},
	})

	// Publish session.compacted event
	event.PublishSync(event.Event{
		Type: event.SessionCompacted,
		Data: event.SessionCompactedData{SessionID: sessionID},
	})

	// If auto-compaction, add a "Continue if you have next steps" message
	if compactionPart.Auto {
		continueMsg := &types.Message{
			ID:        generatePartID(),
			SessionID: sessionID,
			Role:      "user",
			Agent:     lastMsg.Agent,
			Model:     lastMsg.Model,
			Time: types.MessageTime{
				Created: time.Now().UnixMilli(),
			},
		}
		p.storage.Put(ctx, []string{"message", sessionID, continueMsg.ID}, continueMsg)

		continuePart := &types.TextPart{
			ID:        generatePartID(),
			SessionID: sessionID,
			MessageID: continueMsg.ID,
			Type:      "text",
			Text:      "Continue if you have next steps",
		}
		p.storage.Put(ctx, []string{"part", continueMsg.ID, continuePart.ID}, continuePart)

		event.PublishSync(event.Event{
			Type: event.MessageCreated,
			Data: event.MessageCreatedData{Info: continueMsg},
		})
		event.PublishSync(event.Event{
			Type: event.MessagePartUpdated,
			Data: event.MessagePartUpdatedData{Part: continuePart},
		})
	}

	return nil
}
