package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/opencode-ai/opencode/internal/event"
	"github.com/opencode-ai/opencode/internal/permission"
	"github.com/opencode-ai/opencode/internal/tool"
	"github.com/opencode-ai/opencode/pkg/types"
)

// executeToolCalls executes all pending tool calls in the state.
func (p *Processor) executeToolCalls(
	ctx context.Context,
	state *sessionState,
	agent *Agent,
	callback ProcessCallback,
) error {
	// Find all running tool parts
	var pendingTools []*types.ToolPart
	for _, part := range state.parts {
		if toolPart, ok := part.(*types.ToolPart); ok {
			if toolPart.State == "running" {
				pendingTools = append(pendingTools, toolPart)
			}
		}
	}

	// Execute each tool
	for _, toolPart := range pendingTools {
		err := p.executeSingleTool(ctx, state, agent, toolPart, callback)
		if err != nil {
			// Error is captured in tool part, don't stop processing
			continue
		}
	}

	return nil
}

// executeSingleTool executes a single tool call.
func (p *Processor) executeSingleTool(
	ctx context.Context,
	state *sessionState,
	agent *Agent,
	toolPart *types.ToolPart,
	callback ProcessCallback,
) error {
	// Get the tool from registry
	t, ok := p.toolRegistry.Get(toolPart.ToolName)
	if !ok {
		return p.failTool(ctx, state, toolPart, callback,
			fmt.Sprintf("Tool not found: %s", toolPart.ToolName))
	}

	// Check permissions
	if err := p.checkToolPermission(ctx, state, agent, toolPart); err != nil {
		return p.failTool(ctx, state, toolPart, callback, err.Error())
	}

	// Check for doom loop
	if err := p.checkDoomLoop(ctx, state, agent, toolPart); err != nil {
		return p.failTool(ctx, state, toolPart, callback, err.Error())
	}

	// Prepare input JSON
	inputJSON, err := json.Marshal(toolPart.Input)
	if err != nil {
		return p.failTool(ctx, state, toolPart, callback,
			fmt.Sprintf("Failed to marshal input: %v", err))
	}

	// Create tool context
	abortCh := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(abortCh)
	}()

	toolCtx := &tool.Context{
		SessionID: state.message.SessionID,
		MessageID: state.message.ID,
		CallID:    toolPart.ToolCallID,
		Agent:     agent.Name,
		WorkDir:   "",
		AbortCh:   abortCh,
		Extra: map[string]any{
			"model": state.message.ModelID,
		},
	}

	// Set metadata callback for real-time updates
	toolCtx.OnMetadata = func(title string, meta map[string]any) {
		toolPart.Title = &title
		if toolPart.Metadata == nil {
			toolPart.Metadata = make(map[string]any)
		}
		for k, v := range meta {
			toolPart.Metadata[k] = v
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

	// Execute tool
	result, err := t.Execute(ctx, inputJSON, toolCtx)
	if err != nil {
		return p.failTool(ctx, state, toolPart, callback, err.Error())
	}

	// Update tool part with result
	now := time.Now().UnixMilli()
	toolPart.State = "completed"
	toolPart.Output = &result.Output
	toolPart.Title = &result.Title
	toolPart.Time.End = &now

	if result.Metadata != nil {
		if toolPart.Metadata == nil {
			toolPart.Metadata = make(map[string]any)
		}
		for k, v := range result.Metadata {
			toolPart.Metadata[k] = v
		}
	}

	// Handle attachments
	if len(result.Attachments) > 0 {
		if toolPart.Metadata == nil {
			toolPart.Metadata = make(map[string]any)
		}
		toolPart.Metadata["attachments"] = result.Attachments
	}

	// Save updated part
	p.savePart(ctx, state.message.ID, toolPart)

	// Publish event
	event.Publish(event.Event{
		Type: event.PartUpdated,
		Data: event.PartUpdatedData{
			SessionID: state.message.SessionID,
			MessageID: state.message.ID,
			Part:      toolPart,
		},
	})

	callback(state.message, state.parts)
	return nil
}

// failTool marks a tool as failed with an error.
func (p *Processor) failTool(
	ctx context.Context,
	state *sessionState,
	toolPart *types.ToolPart,
	callback ProcessCallback,
	errMsg string,
) error {
	now := time.Now().UnixMilli()
	toolPart.State = "error"
	toolPart.Error = &errMsg
	toolPart.Time.End = &now

	p.savePart(ctx, state.message.ID, toolPart)

	event.Publish(event.Event{
		Type: event.PartUpdated,
		Data: event.PartUpdatedData{
			SessionID: state.message.SessionID,
			MessageID: state.message.ID,
			Part:      toolPart,
		},
	})

	callback(state.message, state.parts)
	return fmt.Errorf(errMsg)
}

// checkToolPermission checks if the tool execution is permitted.
func (p *Processor) checkToolPermission(
	ctx context.Context,
	state *sessionState,
	agent *Agent,
	toolPart *types.ToolPart,
) error {
	if p.permissionChecker == nil {
		return nil
	}

	var permType permission.PermissionType
	var action permission.PermissionAction
	var pattern []string

	switch toolPart.ToolName {
	case "Bash":
		permType = permission.PermBash
		if cmd, ok := toolPart.Input["command"].(string); ok {
			pattern = []string{cmd}
		}
		switch agent.Permission.Bash {
		case "allow":
			action = permission.ActionAllow
		case "deny":
			action = permission.ActionDeny
		default:
			action = permission.ActionAsk
		}

	case "Write", "Edit":
		permType = permission.PermEdit
		if path, ok := toolPart.Input["file_path"].(string); ok {
			pattern = []string{path}
		}
		switch agent.Permission.Write {
		case "allow":
			action = permission.ActionAllow
		case "deny":
			action = permission.ActionDeny
		default:
			action = permission.ActionAsk
		}

	default:
		// Other tools don't require permission
		return nil
	}

	req := permission.Request{
		Type:      permType,
		Pattern:   pattern,
		SessionID: state.message.SessionID,
		MessageID: state.message.ID,
		CallID:    toolPart.ToolCallID,
		Title:     fmt.Sprintf("Allow %s?", toolPart.ToolName),
	}

	return p.permissionChecker.Check(ctx, req, action)
}

// checkDoomLoop detects and handles repetitive tool calls.
func (p *Processor) checkDoomLoop(
	ctx context.Context,
	state *sessionState,
	agent *Agent,
	toolPart *types.ToolPart,
) error {
	// Count identical tool calls
	count := 0
	inputJSON, _ := json.Marshal(toolPart.Input)
	inputStr := string(inputJSON)

	for _, part := range state.parts {
		if tp, ok := part.(*types.ToolPart); ok {
			if tp.ToolName == toolPart.ToolName && tp.State == "completed" {
				otherInput, _ := json.Marshal(tp.Input)
				if string(otherInput) == inputStr {
					count++
				}
			}
		}
	}

	// Threshold for doom loop detection
	if count < 3 {
		return nil
	}

	// Check permission policy
	switch agent.Permission.DoomLoop {
	case "allow":
		return nil

	case "deny":
		return fmt.Errorf("doom loop detected: %s called %d times with same input", toolPart.ToolName, count)

	case "ask", "":
		if p.permissionChecker == nil {
			return nil
		}

		// Request permission from user
		req := permission.Request{
			Type:      permission.PermDoomLoop,
			Pattern:   []string{toolPart.ToolName},
			SessionID: state.message.SessionID,
			MessageID: state.message.ID,
			CallID:    toolPart.ToolCallID,
			Title:     fmt.Sprintf("Allow repeated %s call?", toolPart.ToolName),
		}

		return p.permissionChecker.Ask(ctx, req)
	}

	return nil
}

// waitForPermission waits for a permission response.
func (p *Processor) waitForPermission(ctx context.Context, requestID string) (bool, error) {
	// This is handled by the permission checker's Ask method
	// which blocks until a response is received
	return true, nil
}

// ToolState represents the current state of tool execution.
type ToolState string

const (
	ToolStatePending   ToolState = "pending"
	ToolStateRunning   ToolState = "running"
	ToolStateCompleted ToolState = "completed"
	ToolStateError     ToolState = "error"
)
