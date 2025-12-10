package headless

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/opencode-ai/opencode/internal/event"
	"github.com/opencode-ai/opencode/pkg/types"
)

// Printer handles event output in various formats for headless mode.
type Printer struct {
	mu           sync.Mutex
	writer       io.Writer
	format       OutputFormat
	quiet        bool
	verbose      bool
	unsubscribe  func()
	sessionID    string
	startTime    time.Time
	result       *Result
	toolCalls    []ToolCall
	currentTool  *ToolCall
	lastTextDelta string
}

// NewPrinter creates a new event printer.
func NewPrinter(writer io.Writer, format OutputFormat, quiet, verbose bool) *Printer {
	return &Printer{
		writer:    writer,
		format:    format,
		quiet:     quiet,
		verbose:   verbose,
		startTime: time.Now(),
		result: &Result{
			Status:   "running",
			ExitCode: ExitSuccess,
		},
		toolCalls: make([]ToolCall, 0),
	}
}

// Subscribe starts listening to events.
func (p *Printer) Subscribe() {
	p.unsubscribe = event.SubscribeAll(p.handleEvent)
}

// Unsubscribe stops listening to events.
func (p *Printer) Unsubscribe() {
	if p.unsubscribe != nil {
		p.unsubscribe()
		p.unsubscribe = nil
	}
}

// SetSessionID sets the session ID for the printer.
func (p *Printer) SetSessionID(sessionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sessionID = sessionID
	p.result.SessionID = sessionID
}

// GetResult returns the current result.
func (p *Printer) GetResult() *Result {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Finalize result
	p.result.DurationMS = time.Since(p.startTime).Milliseconds()
	p.result.ToolCalls = p.toolCalls

	return p.result
}

// SetResult updates the result with final values.
func (p *Printer) SetResult(status string, exitCode ExitCode, finalMessage string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.result.Status = status
	p.result.ExitCode = exitCode
	p.result.FinalMessage = finalMessage
	if err != nil {
		p.result.Error = err.Error()
	}
	p.result.DurationMS = time.Since(p.startTime).Milliseconds()
}

// SetTokens updates token usage in the result.
func (p *Printer) SetTokens(tokens *types.TokenUsage) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.result.Tokens = tokens
}

// SetModel updates the model in the result.
func (p *Printer) SetModel(model string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.result.Model = model
}

// IncrementSteps increments the step counter.
func (p *Printer) IncrementSteps() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.result.Steps++
}

// PrintFinalResult prints the final JSON result (for json format).
func (p *Printer) PrintFinalResult() {
	if p.format != OutputJSON {
		return
	}

	result := p.GetResult()
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return
	}
	fmt.Fprintln(p.writer, string(data))
}

// handleEvent processes incoming events and outputs them according to format.
func (p *Printer) handleEvent(e event.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.format {
	case OutputText:
		p.handleTextEvent(e)
	case OutputJSON:
		// JSON format only outputs final result, but we still track events
		p.trackEvent(e)
	case OutputJSONL:
		p.handleJSONLEvent(e)
	}
}

// handleTextEvent outputs events in human-readable text format.
func (p *Printer) handleTextEvent(e event.Event) {
	if p.quiet {
		// In quiet mode, only output final text
		if e.Type == event.MessagePartUpdated {
			if data, ok := e.Data.(event.MessagePartUpdatedData); ok {
				if data.Delta != "" {
					fmt.Fprint(p.writer, data.Delta)
				}
			}
		}
		return
	}

	switch e.Type {
	case event.SessionCreated:
		if data, ok := e.Data.(event.SessionCreatedData); ok && data.Info != nil {
			fmt.Fprintf(p.writer, "[session:%s] Starting...\n", truncateID(data.Info.ID))
		}

	case event.SessionStatus:
		if data, ok := e.Data.(event.SessionStatusData); ok {
			if data.Status.Type == "idle" {
				duration := time.Since(p.startTime)
				fmt.Fprintf(p.writer, "\n[done] Session completed in %s", formatDuration(duration))
				if p.result.Tokens != nil {
					fmt.Fprintf(p.writer, " (input: %d tokens, output: %d tokens)",
						p.result.Tokens.Input, p.result.Tokens.Output)
				}
				fmt.Fprintln(p.writer)
			}
		}

	case event.MessageCreated:
		if data, ok := e.Data.(event.MessageCreatedData); ok && data.Info != nil {
			if data.Info.Role == "user" {
				// Don't print user messages in text mode
			} else if data.Info.Role == "assistant" && p.verbose {
				fmt.Fprintf(p.writer, "[assistant] Thinking...\n")
			}
		}

	case event.MessagePartUpdated:
		if data, ok := e.Data.(event.MessagePartUpdatedData); ok {
			switch part := data.Part.(type) {
			case *types.TextPart:
				if data.Delta != "" {
					fmt.Fprint(p.writer, data.Delta)
					p.lastTextDelta = data.Delta
				}
			case *types.ToolPart:
				p.handleToolPartText(part)
			}
		}

	case event.PermissionUpdated:
		if data, ok := e.Data.(event.PermissionUpdatedData); ok {
			if p.verbose {
				fmt.Fprintf(p.writer, "[permission] %s: %s (auto-approved)\n",
					data.PermissionType, data.Title)
			}
		}

	case event.FileEdited:
		if data, ok := e.Data.(event.FileEditedData); ok {
			if p.verbose {
				fmt.Fprintf(p.writer, "[file] Edited: %s\n", data.File)
			}
		}

	case event.SessionError:
		if data, ok := e.Data.(event.SessionErrorData); ok && data.Error != nil {
			fmt.Fprintf(p.writer, "[error] %s\n", data.Error.Data.Message)
		}
	}
}

// handleToolPartText outputs tool information in text format.
func (p *Printer) handleToolPartText(part *types.ToolPart) {
	switch part.State.Status {
	case "pending":
		if p.verbose {
			fmt.Fprintf(p.writer, "\n[tool:%s] Starting...\n", part.Tool)
		}
	case "running":
		// Show brief tool info
		toolInfo := formatToolInfo(part)
		if toolInfo != "" {
			fmt.Fprintf(p.writer, "\n[tool:%s] %s\n", part.Tool, toolInfo)
		}
	case "completed":
		if p.verbose && part.State.Output != "" {
			output := part.State.Output
			if len(output) > 200 {
				output = output[:200] + "..."
			}
			fmt.Fprintf(p.writer, "[tool:%s] Done\n", part.Tool)
		}
	case "error":
		fmt.Fprintf(p.writer, "[tool:%s] Error: %s\n", part.Tool, part.State.Error)
	}
}

// handleJSONLEvent outputs events in JSONL format.
func (p *Printer) handleJSONLEvent(e event.Event) {
	// Track event for result
	p.trackEvent(e)

	// Filter events if not verbose
	if !p.verbose && !isImportantEvent(e.Type) {
		return
	}

	evt := &Event{
		Type:      string(e.Type),
		Timestamp: time.Now(),
		Data:      e.Data,
	}

	data, err := json.Marshal(evt)
	if err != nil {
		return
	}
	fmt.Fprintln(p.writer, string(data))
}

// trackEvent tracks events for the final result.
func (p *Printer) trackEvent(e event.Event) {
	switch e.Type {
	case event.MessageUpdated:
		if data, ok := e.Data.(event.MessageUpdatedData); ok && data.Info != nil {
			if data.Info.Role == "assistant" && data.Info.Tokens != nil {
				p.result.Tokens = data.Info.Tokens
			}
		}

	case event.MessagePartUpdated:
		if data, ok := e.Data.(event.MessagePartUpdatedData); ok {
			switch part := data.Part.(type) {
			case *types.TextPart:
				// Track final message
				if data.Delta == "" && part.Text != "" {
					p.result.FinalMessage = part.Text
				}
			case *types.ToolPart:
				p.trackToolCall(part)
			}
		}

	case event.SessionDiff:
		if data, ok := e.Data.(event.SessionDiffData); ok {
			p.result.Diffs = make([]FileDiff, len(data.Diff))
			for i, diff := range data.Diff {
				p.result.Diffs[i] = FileDiff{
					File:      diff.File,
					Additions: diff.Additions,
					Deletions: diff.Deletions,
				}
			}
		}
	}
}

// trackToolCall tracks tool call information for the result.
func (p *Printer) trackToolCall(part *types.ToolPart) {
	if part.State.Status == "completed" || part.State.Status == "error" {
		call := ToolCall{
			Tool:   part.Tool,
			Input:  part.State.Input,
			Output: truncateOutput(part.State.Output, 500),
			Error:  part.State.Error,
		}
		p.toolCalls = append(p.toolCalls, call)
	}
}

// Helper functions

func truncateID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func truncateOutput(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}

func formatToolInfo(part *types.ToolPart) string {
	if part.State.Input == nil {
		return ""
	}

	input := part.State.Input

	switch part.Tool {
	case "read":
		if path, ok := input["file_path"].(string); ok {
			return fmt.Sprintf("Reading %s", path)
		}
	case "write":
		if path, ok := input["file_path"].(string); ok {
			return fmt.Sprintf("Writing %s", path)
		}
	case "edit":
		if path, ok := input["file_path"].(string); ok {
			return fmt.Sprintf("Editing %s", path)
		}
	case "bash":
		if cmd, ok := input["command"].(string); ok {
			cmd = strings.Split(cmd, "\n")[0]
			if len(cmd) > 60 {
				cmd = cmd[:60] + "..."
			}
			return fmt.Sprintf("$ %s", cmd)
		}
	case "glob":
		if pattern, ok := input["pattern"].(string); ok {
			return fmt.Sprintf("Searching: %s", pattern)
		}
	case "grep":
		if pattern, ok := input["pattern"].(string); ok {
			return fmt.Sprintf("Grepping: %s", pattern)
		}
	case "web_fetch":
		if url, ok := input["url"].(string); ok {
			return fmt.Sprintf("Fetching: %s", url)
		}
	}

	return ""
}

func isImportantEvent(eventType event.EventType) bool {
	switch eventType {
	case event.SessionCreated,
		event.SessionStatus,
		event.SessionError,
		event.SessionDiff,
		event.MessageCreated,
		event.MessagePartUpdated,
		event.PermissionUpdated,
		event.FileEdited:
		return true
	default:
		return false
	}
}
