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
			if toolPart.State.Status == "running" {
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
	t, ok := p.toolRegistry.Get(toolPart.Tool)
	if !ok {
		return p.failTool(ctx, state, toolPart, callback,
			fmt.Sprintf("Tool not found: %s", toolPart.Tool))
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
	inputJSON, err := json.Marshal(toolPart.State.Input)
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
		CallID:    toolPart.CallID,
		Agent:     agent.Name,
		WorkDir:   "",
		AbortCh:   abortCh,
		Extra: map[string]any{
			"model": state.message.ModelID,
		},
	}

	// Set metadata callback for real-time updates
	toolCtx.OnMetadata = func(title string, meta map[string]any) {
		toolPart.State.Title = title
		if toolPart.State.Metadata == nil {
			toolPart.State.Metadata = make(map[string]any)
		}
		for k, v := range meta {
			toolPart.State.Metadata[k] = v
		}

		// Publish event (SDK compatible: uses MessagePartUpdated)
		event.Publish(event.Event{
			Type: event.MessagePartUpdated,
			Data: event.MessagePartUpdatedData{
				Part: toolPart,
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
	toolPart.State.Status = "completed"
	toolPart.State.Output = result.Output
	toolPart.State.Title = result.Title
	toolPart.State.Time.End = &now

	if result.Metadata != nil {
		if toolPart.State.Metadata == nil {
			toolPart.State.Metadata = make(map[string]any)
		}
		for k, v := range result.Metadata {
			toolPart.State.Metadata[k] = v
		}
	}

	// Handle attachments - convert to types.FilePart and add to state
	if len(result.Attachments) > 0 {
		toolPart.State.Attachments = make([]types.FilePart, len(result.Attachments))
		for i, att := range result.Attachments {
			toolPart.State.Attachments[i] = types.FilePart{
				ID:        generatePartID(),
				SessionID: state.message.SessionID,
				MessageID: state.message.ID,
				Type:      "file",
				Filename:  att.Filename,
				Mime:      att.MediaType,
				URL:       att.URL,
			}
		}
	}

	// Save updated part
	p.savePart(ctx, state.message.ID, toolPart)

	// Publish event (SDK compatible: uses MessagePartUpdated)
	event.Publish(event.Event{
		Type: event.MessagePartUpdated,
		Data: event.MessagePartUpdatedData{
			Part: toolPart,
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
	toolPart.State.Status = "error"
	toolPart.State.Error = errMsg
	toolPart.State.Time.End = &now

	p.savePart(ctx, state.message.ID, toolPart)

	// Publish event (SDK compatible: uses MessagePartUpdated)
	event.Publish(event.Event{
		Type: event.MessagePartUpdated,
		Data: event.MessagePartUpdatedData{
			Part: toolPart,
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

	switch toolPart.Tool {
	case "Bash":
		permType = permission.PermBash
		if cmd, ok := toolPart.State.Input["command"].(string); ok {
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
		if path, ok := toolPart.State.Input["filePath"].(string); ok {
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
		CallID:    toolPart.CallID,
		Title:     fmt.Sprintf("Allow %s?", toolPart.Tool),
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
	inputJSON, _ := json.Marshal(toolPart.State.Input)
	inputStr := string(inputJSON)

	for _, part := range state.parts {
		if tp, ok := part.(*types.ToolPart); ok {
			if tp.Tool == toolPart.Tool && tp.State.Status == "completed" {
				otherInput, _ := json.Marshal(tp.State.Input)
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
		return fmt.Errorf("doom loop detected: %s called %d times with same input", toolPart.Tool, count)

	case "ask", "":
		if p.permissionChecker == nil {
			return nil
		}

		// Request permission from user
		req := permission.Request{
			Type:      permission.PermDoomLoop,
			Pattern:   []string{toolPart.Tool},
			SessionID: state.message.SessionID,
			MessageID: state.message.ID,
			CallID:    toolPart.CallID,
			Title:     fmt.Sprintf("Allow repeated %s call?", toolPart.Tool),
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
