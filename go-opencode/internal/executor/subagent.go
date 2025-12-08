// Package executor provides task execution implementations.
package executor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/opencode-ai/opencode/internal/agent"
	"github.com/opencode-ai/opencode/internal/event"
	"github.com/opencode-ai/opencode/internal/permission"
	"github.com/opencode-ai/opencode/internal/provider"
	"github.com/opencode-ai/opencode/internal/session"
	"github.com/opencode-ai/opencode/internal/storage"
	"github.com/opencode-ai/opencode/internal/tool"
	"github.com/opencode-ai/opencode/pkg/types"
)

// SubagentExecutor implements tool.TaskExecutor to run subagent tasks.
type SubagentExecutor struct {
	storage           *storage.Storage
	providerRegistry  *provider.Registry
	toolRegistry      *tool.Registry
	permissionChecker *permission.Checker
	agentRegistry     *agent.Registry
	workDir           string

	// Default provider and model settings
	defaultProviderID string
	defaultModelID    string
}

// SubagentExecutorConfig holds configuration for creating a SubagentExecutor.
type SubagentExecutorConfig struct {
	Storage           *storage.Storage
	ProviderRegistry  *provider.Registry
	ToolRegistry      *tool.Registry
	PermissionChecker *permission.Checker
	AgentRegistry     *agent.Registry
	WorkDir           string
	DefaultProviderID string
	DefaultModelID    string
}

// NewSubagentExecutor creates a new SubagentExecutor.
func NewSubagentExecutor(cfg SubagentExecutorConfig) *SubagentExecutor {
	return &SubagentExecutor{
		storage:           cfg.Storage,
		providerRegistry:  cfg.ProviderRegistry,
		toolRegistry:      cfg.ToolRegistry,
		permissionChecker: cfg.PermissionChecker,
		agentRegistry:     cfg.AgentRegistry,
		workDir:           cfg.WorkDir,
		defaultProviderID: cfg.DefaultProviderID,
		defaultModelID:    cfg.DefaultModelID,
	}
}

// ExecuteSubtask implements tool.TaskExecutor.ExecuteSubtask.
// It creates a child session, runs the subagent, and returns the result.
func (e *SubagentExecutor) ExecuteSubtask(
	ctx context.Context,
	parentSessionID string,
	agentName string,
	prompt string,
	opts tool.TaskOptions,
) (*tool.TaskResult, error) {
	// Get the agent configuration
	agentConfig, err := e.agentRegistry.Get(agentName)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %s: %w", agentName, err)
	}

	// Verify it can be used as a subagent
	if !agentConfig.IsSubagent() {
		return nil, fmt.Errorf("agent %s cannot be used as subagent (mode: %s)", agentName, agentConfig.Mode)
	}

	// Create a child session
	childSession, err := e.createChildSession(ctx, parentSessionID, agentName)
	if err != nil {
		return nil, fmt.Errorf("failed to create child session: %w", err)
	}

	// Convert agent.Agent to session.Agent
	sessionAgent := convertToSessionAgent(agentConfig)

	// Resolve model from options
	providerID, modelID := e.resolveModel(opts.Model)

	// Create user message with the prompt
	userMsg, err := e.createUserMessage(ctx, childSession, prompt, providerID, modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to create user message: %w", err)
	}

	// Create and run processor
	processor := session.NewProcessor(
		e.providerRegistry,
		e.toolRegistry,
		e.storage,
		e.permissionChecker,
		providerID,
		modelID,
	)

	// Collect response parts
	var responseParts []types.Part
	var responseMsg *types.Message

	// Run the processing loop
	err = processor.Process(ctx, childSession.ID, sessionAgent, func(msg *types.Message, parts []types.Part) {
		responseMsg = msg
		responseParts = parts
	})

	if err != nil {
		return &tool.TaskResult{
			Output:    fmt.Sprintf("Error executing subtask: %s", err.Error()),
			SessionID: childSession.ID,
			Error:     err.Error(),
			Metadata: map[string]any{
				"parentSessionID": parentSessionID,
				"userMessageID":   userMsg.ID,
			},
		}, nil
	}

	// Extract text content from response
	output := extractTextContent(responseParts)

	return &tool.TaskResult{
		Output:    output,
		SessionID: childSession.ID,
		AgentID:   agentName,
		Metadata: map[string]any{
			"parentSessionID":    parentSessionID,
			"assistantMessageID": responseMsg.ID,
			"userMessageID":      userMsg.ID,
		},
	}, nil
}

// createChildSession creates a new session as a child of the parent session.
func (e *SubagentExecutor) createChildSession(ctx context.Context, parentSessionID string, agentName string) (*types.Session, error) {
	now := time.Now().UnixMilli()
	sessionID := ulid.Make().String()

	// Get parent session to inherit directory
	var parentSession types.Session
	var directory string

	// Try to find parent session
	projects, err := e.storage.List(ctx, []string{"session"})
	if err == nil {
		for _, projectID := range projects {
			if err := e.storage.Get(ctx, []string{"session", projectID, parentSessionID}, &parentSession); err == nil {
				directory = parentSession.Directory
				break
			}
		}
	}

	// Use work directory if parent not found
	if directory == "" {
		directory = e.workDir
	}

	// Create project ID from directory
	projectID := hashDirectory(directory)

	sess := &types.Session{
		ID:        sessionID,
		ProjectID: projectID,
		Directory: directory,
		Title:     fmt.Sprintf("Subtask: %s", agentName),
		ParentID:  &parentSessionID,
		Version:   "1",
		Summary: types.SessionSummary{
			Additions: 0,
			Deletions: 0,
			Files:     0,
		},
		Time: types.SessionTime{
			Created: now,
			Updated: now,
		},
	}

	if err := e.storage.Put(ctx, []string{"session", projectID, sess.ID}, sess); err != nil {
		return nil, fmt.Errorf("failed to save child session: %w", err)
	}

	// Publish session created event
	event.PublishSync(event.Event{
		Type: event.SessionCreated,
		Data: event.SessionCreatedData{Info: sess},
	})

	return sess, nil
}

// createUserMessage creates a user message with the prompt.
func (e *SubagentExecutor) createUserMessage(
	ctx context.Context,
	sess *types.Session,
	prompt string,
	providerID string,
	modelID string,
) (*types.Message, error) {
	now := time.Now().UnixMilli()
	msgID := ulid.Make().String()

	msg := &types.Message{
		ID:         msgID,
		SessionID:  sess.ID,
		Role:       "user",
		ProviderID: providerID,
		ModelID:    modelID,
		Model: &types.ModelRef{
			ProviderID: providerID,
			ModelID:    modelID,
		},
		Path: &types.MessagePath{
			Cwd:  sess.Directory,
			Root: sess.Directory,
		},
		Time: types.MessageTime{
			Created: now,
		},
	}

	// Save message
	if err := e.storage.Put(ctx, []string{"message", sess.ID, msg.ID}, msg); err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// Create text part for the prompt
	partID := ulid.Make().String()
	textPart := &types.TextPart{
		ID:        partID,
		SessionID: sess.ID,
		MessageID: msg.ID,
		Type:      "text",
		Text:      prompt,
	}

	// Save part
	if err := e.storage.Put(ctx, []string{"part", msg.ID, partID}, textPart); err != nil {
		return nil, fmt.Errorf("failed to save text part: %w", err)
	}

	// Publish message created event
	event.PublishSync(event.Event{
		Type: event.MessageCreated,
		Data: event.MessageCreatedData{Info: msg},
	})

	// Publish part updated event
	event.PublishSync(event.Event{
		Type: event.MessagePartUpdated,
		Data: event.MessagePartUpdatedData{Part: textPart},
	})

	return msg, nil
}

// resolveModel resolves provider and model IDs from the options.
func (e *SubagentExecutor) resolveModel(modelOption string) (providerID, modelID string) {
	providerID = e.defaultProviderID
	modelID = e.defaultModelID

	// Handle model override from options
	switch modelOption {
	case "sonnet":
		modelID = "claude-sonnet-4-20250514"
	case "opus":
		modelID = "claude-opus-4-20250514"
	case "haiku":
		modelID = "claude-haiku-3-20240307"
	default:
		// Keep defaults
	}

	return providerID, modelID
}

// convertToSessionAgent converts agent.Agent to session.Agent.
func convertToSessionAgent(a *agent.Agent) *session.Agent {
	// Build enabled/disabled tool lists from the map
	var enabledTools []string
	var disabledTools []string

	hasWildcard := false
	wildcardEnabled := false

	for tool, enabled := range a.Tools {
		if tool == "*" {
			hasWildcard = true
			wildcardEnabled = enabled
			continue
		}
		if enabled {
			enabledTools = append(enabledTools, tool)
		} else {
			disabledTools = append(disabledTools, tool)
		}
	}

	// If wildcard is enabled but not explicitly set, we treat it as all enabled
	// The DisabledTools list will handle exceptions
	if hasWildcard && wildcardEnabled {
		enabledTools = nil // Empty means all enabled
	}

	// Convert bash permission to simple string
	bashPerm := "ask"
	if a.Permission.Bash != nil {
		if action, ok := a.Permission.Bash["*"]; ok {
			bashPerm = string(action)
		}
	}

	// Convert write/edit permission
	writePerm := "ask"
	if a.Permission.Edit != "" {
		writePerm = string(a.Permission.Edit)
	}

	// Convert doom loop permission
	doomLoopPerm := "ask"
	if a.Permission.DoomLoop != "" {
		doomLoopPerm = string(a.Permission.DoomLoop)
	}

	return &session.Agent{
		Name:          a.Name,
		Prompt:        a.Prompt,
		Temperature:   a.Temperature,
		TopP:          a.TopP,
		MaxSteps:      50, // Default max steps for subagents
		Tools:         enabledTools,
		DisabledTools: disabledTools,
		Permission: session.AgentPermission{
			DoomLoop: doomLoopPerm,
			Bash:     bashPerm,
			Write:    writePerm,
		},
	}
}

// extractTextContent extracts text content from response parts.
func extractTextContent(parts []types.Part) string {
	var texts []string
	for _, part := range parts {
		switch p := part.(type) {
		case *types.TextPart:
			if p.Text != "" {
				texts = append(texts, p.Text)
			}
		}
	}
	return strings.Join(texts, "\n")
}

// hashDirectory creates a project ID from a directory path.
func hashDirectory(directory string) string {
	h := sha256.New()
	h.Write([]byte(directory))
	return hex.EncodeToString(h.Sum(nil))[:16]
}
