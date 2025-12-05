package session

import (
	"context"
	"io"
	"strings"

	"github.com/cloudwego/eino/schema"

	"github.com/opencode-ai/opencode/internal/event"
	"github.com/opencode-ai/opencode/internal/provider"
	"github.com/opencode-ai/opencode/pkg/types"
)

const titleSystemPrompt = `You are a title generator. You output ONLY a thread title. Nothing else.

Generate a brief title that would help the user find this conversation later.

Rules:
- A single line, ≤50 characters
- No explanations
- Use -ing verbs for actions (Debugging, Implementing, Analyzing)
- Keep exact: technical terms, numbers, filenames
- Remove: the, this, my, a, an
- Always output something meaningful

Examples:
"debug 500 errors in production" → Debugging production 500 errors
"refactor user service" → Refactoring user service
"implement rate limiting" → Implementing rate limiting`

const defaultTitlePrefix = "New Session"

// isDefaultTitle checks if a title is the default "New Session" title.
func isDefaultTitle(title string) bool {
	return title == defaultTitlePrefix || strings.HasPrefix(title, defaultTitlePrefix)
}

// ensureTitle generates a title for the session if it's still using the default title.
// Should only be called on the first user message.
func (p *Processor) ensureTitle(
	ctx context.Context,
	session *types.Session,
	userContent string,
) {
	// Skip if session has a parent (child session)
	if session.ParentID != nil && *session.ParentID != "" {
		return
	}

	// Skip if title is not the default
	if !isDefaultTitle(session.Title) {
		return
	}

	// Get the default model for title generation
	model, err := p.providerRegistry.DefaultModel()
	if err != nil {
		return
	}

	prov, err := p.providerRegistry.Get(model.ProviderID)
	if err != nil {
		return
	}

	// Create title generation request
	stream, err := prov.CreateCompletion(ctx, &provider.CompletionRequest{
		Model: model.ID,
		Messages: []*schema.Message{
			{Role: schema.System, Content: titleSystemPrompt},
			{Role: schema.User, Content: "Generate a title for this conversation:\n\n" + userContent},
		},
		MaxTokens: 50, // Short title
	})
	if err != nil {
		return
	}
	defer stream.Close()

	// Collect response
	var title strings.Builder
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return
		}
		title.WriteString(msg.Content)
	}

	// Clean up title
	titleText := strings.TrimSpace(title.String())
	// Get first non-empty line
	for _, line := range strings.Split(titleText, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			titleText = line
			break
		}
	}

	// Truncate if too long
	if len(titleText) > 100 {
		titleText = titleText[:97] + "..."
	}

	if titleText == "" {
		return
	}

	// Update session title
	session.Title = titleText
	p.storage.Put(ctx, []string{"session", session.ProjectID, session.ID}, session)

	// Publish session.updated event
	event.PublishSync(event.Event{
		Type: event.SessionUpdated,
		Data: event.SessionUpdatedData{Info: session},
	})
}
