package session

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"

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
		Path:   "__compaction__",
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
				prompt.WriteString(fmt.Sprintf("[Tool: %s]\n", pt.ToolName))
				if pt.Output != nil {
					// Truncate long outputs
					output := *pt.Output
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

// CompactionPart represents a summary of compacted messages.
type CompactionPart struct {
	ID      string `json:"id"`
	Type    string `json:"type"` // always "compaction"
	Summary string `json:"summary"`
	Count   int    `json:"count"` // Number of messages summarized
}

func (p *CompactionPart) PartType() string { return "compaction" }
func (p *CompactionPart) PartID() string   { return p.ID }

// estimateTokens provides a rough estimate of token count.
func estimateTokens(text string) int {
	// Rough estimate: ~4 characters per token
	return len(text) / 4
}

