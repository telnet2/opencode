package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/opencode-ai/opencode/internal/event"
	"github.com/opencode-ai/opencode/internal/permission"
	"github.com/opencode-ai/opencode/internal/tool"
	"github.com/opencode-ai/opencode/pkg/types"
	"github.com/sergi/go-diff/diffmatchpatch"
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
		WorkDir: func() string {
			if state.message.Path != nil {
				return state.message.Path.Cwd
			}
			return ""
		}(),
		AbortCh: abortCh,
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
		event.PublishSync(event.Event{
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

	// Record diff for edit-like tools when metadata contains before/after
	p.recordDiff(state, toolPart)

	// Save updated part
	p.savePart(ctx, state.message.ID, toolPart)

	// Publish event (SDK compatible: uses MessagePartUpdated)
	event.PublishSync(event.Event{
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
	event.PublishSync(event.Event{
		Type: event.MessagePartUpdated,
		Data: event.MessagePartUpdatedData{
			Part: toolPart,
		},
	})

	callback(state.message, state.parts)
	return errors.New(errMsg)
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

// recordDiff captures file diffs from tool metadata and updates session summary/state.
func (p *Processor) recordDiff(state *sessionState, toolPart *types.ToolPart) error {
	if toolPart.State.Metadata == nil {
		toolPart.State.Metadata = make(map[string]any)
	}

	pathVal, ok := toolPart.State.Metadata["file"].(string)
	if !ok || pathVal == "" {
		return nil
	}

	before, okBefore := toolPart.State.Metadata["before"].(string)
	after, okAfter := toolPart.State.Metadata["after"].(string)
	if !okBefore || !okAfter {
		return nil
	}

	root := ""
	if state.message.Path != nil {
		root = state.message.Path.Root
	}
	relPath := pathVal
	if root != "" {
		if rp, err := filepath.Rel(root, pathVal); err == nil {
			relPath = rp
		}
	}

	diffText, additions, deletions, err := computeDiff(before, after, relPath)
	if err != nil {
		return err
	}

	fileDiff := types.FileDiff{
		File:      relPath,
		Additions: additions,
		Deletions: deletions,
		Before:    before,
		After:     after,
	}

	// Load session to update summary
	session, err := p.loadSession(state.message.SessionID)
	if err != nil {
		return err
	}

	// Replace existing diff for same path, then append
	var filtered []types.FileDiff
	for _, d := range session.Summary.Diffs {
		if d.File != relPath {
			filtered = append(filtered, d)
		}
	}
	filtered = append(filtered, fileDiff)
	session.Summary.Diffs = filtered

	// Recompute summary totals
	adds, dels, files := 0, 0, len(session.Summary.Diffs)
	for _, d := range session.Summary.Diffs {
		adds += d.Additions
		dels += d.Deletions
	}
	session.Summary.Additions = adds
	session.Summary.Deletions = dels
	session.Summary.Files = files
	session.Time.Updated = time.Now().UnixMilli()

	if err := p.saveSession(session); err != nil {
		return err
	}

	// Publish updated session diff
	event.PublishSync(event.Event{
		Type: event.SessionDiff,
		Data: event.SessionDiffData{SessionID: session.ID, Diff: session.Summary.Diffs},
	})

	// Attach diff text to metadata for consumers (non-breaking)
	toolPart.State.Metadata["diff"] = diffText
	if toolPart.Metadata == nil {
		toolPart.Metadata = map[string]any{}
	}
	toolPart.Metadata["diff"] = diffText
	return nil
}

func computeDiff(before, after, path string) (string, int, int, error) {
	dmp := diffmatchpatch.New()

	// Compute line-based diff for accurate line counting
	a, b, lineArray := dmp.DiffLinesToChars(before, after)
	diffs := dmp.DiffMain(a, b, false)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)

	// Count additions and deletions by lines
	additions, deletions := 0, 0
	for _, d := range diffs {
		switch d.Type {
		case diffmatchpatch.DiffInsert:
			lines := countLines(d.Text)
			additions += lines
		case diffmatchpatch.DiffDelete:
			lines := countLines(d.Text)
			deletions += lines
		}
	}

	// Generate proper unified diff text for display
	diffText := generateUnifiedDiff(diffs, path)

	return diffText, additions, deletions, nil
}

// countLines counts the number of lines in text
func countLines(text string) int {
	if text == "" {
		return 0
	}
	lines := strings.Count(text, "\n")
	// If text doesn't end with newline, count it as a line
	if !strings.HasSuffix(text, "\n") {
		lines++
	}
	return lines
}

// generateUnifiedDiff creates a proper unified diff format from diffs with context lines
func generateUnifiedDiff(diffs []diffmatchpatch.Diff, path string) string {
	if len(diffs) == 0 {
		return ""
	}

	// Check if there are any actual changes
	hasChanges := false
	for _, d := range diffs {
		if d.Type != diffmatchpatch.DiffEqual {
			hasChanges = true
			break
		}
	}
	if !hasChanges {
		return ""
	}

	// Convert diffs to lines with their types
	type diffLine struct {
		text     string
		diffType diffmatchpatch.Operation
	}
	var allLines []diffLine

	for _, d := range diffs {
		text := d.Text
		lines := strings.Split(text, "\n")
		// Handle trailing newline - if text ends with \n, the last split element is empty
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		for _, line := range lines {
			allLines = append(allLines, diffLine{text: line, diffType: d.Type})
		}
	}

	// Find ranges of changes with context (3 lines before and after)
	const contextLines = 3
	type hunk struct {
		startOld, countOld int
		startNew, countNew int
		lines              []diffLine
	}

	var hunks []hunk
	var currentHunk *hunk
	oldLineNum := 1
	newLineNum := 1

	for i, line := range allLines {
		isChange := line.diffType != diffmatchpatch.DiffEqual

		if isChange {
			// Start a new hunk or extend current one
			if currentHunk == nil {
				// Calculate start positions including context
				contextStart := i - contextLines
				if contextStart < 0 {
					contextStart = 0
				}

				// Calculate old/new line numbers at context start
				startOld := 1
				startNew := 1
				for j := 0; j < contextStart; j++ {
					switch allLines[j].diffType {
					case diffmatchpatch.DiffEqual:
						startOld++
						startNew++
					case diffmatchpatch.DiffDelete:
						startOld++
					case diffmatchpatch.DiffInsert:
						startNew++
					}
				}

				currentHunk = &hunk{
					startOld: startOld,
					startNew: startNew,
				}

				// Add context lines before the change
				for j := contextStart; j < i; j++ {
					currentHunk.lines = append(currentHunk.lines, allLines[j])
				}
			}
			currentHunk.lines = append(currentHunk.lines, line)
		} else if currentHunk != nil {
			// Check if we should end the hunk or continue with context
			// Look ahead to see if there's another change within context range
			nextChangeIdx := -1
			for j := i + 1; j < len(allLines) && j <= i+contextLines*2; j++ {
				if allLines[j].diffType != diffmatchpatch.DiffEqual {
					nextChangeIdx = j
					break
				}
			}

			if nextChangeIdx != -1 && nextChangeIdx <= i+contextLines*2 {
				// Another change is close, include this line and continue
				currentHunk.lines = append(currentHunk.lines, line)
			} else {
				// Add remaining context lines and close hunk
				for j := i; j < len(allLines) && j < i+contextLines; j++ {
					if allLines[j].diffType == diffmatchpatch.DiffEqual {
						currentHunk.lines = append(currentHunk.lines, allLines[j])
					} else {
						break
					}
				}

				// Calculate counts
				for _, l := range currentHunk.lines {
					switch l.diffType {
					case diffmatchpatch.DiffEqual:
						currentHunk.countOld++
						currentHunk.countNew++
					case diffmatchpatch.DiffDelete:
						currentHunk.countOld++
					case diffmatchpatch.DiffInsert:
						currentHunk.countNew++
					}
				}

				hunks = append(hunks, *currentHunk)
				currentHunk = nil
			}
		}

		// Track line numbers
		switch line.diffType {
		case diffmatchpatch.DiffEqual:
			oldLineNum++
			newLineNum++
		case diffmatchpatch.DiffDelete:
			oldLineNum++
		case diffmatchpatch.DiffInsert:
			newLineNum++
		}
	}

	// Close any remaining hunk
	if currentHunk != nil {
		for _, l := range currentHunk.lines {
			switch l.diffType {
			case diffmatchpatch.DiffEqual:
				currentHunk.countOld++
				currentHunk.countNew++
			case diffmatchpatch.DiffDelete:
				currentHunk.countOld++
			case diffmatchpatch.DiffInsert:
				currentHunk.countNew++
			}
		}
		hunks = append(hunks, *currentHunk)
	}

	// Build output
	var buf strings.Builder

	// Write file headers
	buf.WriteString("Index: ")
	buf.WriteString(path)
	buf.WriteString("\n")
	buf.WriteString("===================================================================\n")
	buf.WriteString("--- ")
	buf.WriteString(path)
	buf.WriteString("\n")
	buf.WriteString("+++ ")
	buf.WriteString(path)
	buf.WriteString("\n")

	// Write each hunk
	for _, h := range hunks {
		buf.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", h.startOld, h.countOld, h.startNew, h.countNew))
		for _, line := range h.lines {
			switch line.diffType {
			case diffmatchpatch.DiffEqual:
				buf.WriteString(" ")
			case diffmatchpatch.DiffDelete:
				buf.WriteString("-")
			case diffmatchpatch.DiffInsert:
				buf.WriteString("+")
			}
			buf.WriteString(line.text)
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

func (p *Processor) loadSession(sessionID string) (*types.Session, error) {
	projects, err := p.storage.List(context.Background(), []string{"session"})
	if err != nil {
		return nil, err
	}

	for _, projectID := range projects {
		var session types.Session
		if err := p.storage.Get(context.Background(), []string{"session", projectID, sessionID}, &session); err == nil {
			return &session, nil
		}
	}
	return nil, fmt.Errorf("session %s not found", sessionID)
}

func (p *Processor) saveSession(session *types.Session) error {
	return p.storage.Put(context.Background(), []string{"session", session.ProjectID, session.ID}, session)
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
