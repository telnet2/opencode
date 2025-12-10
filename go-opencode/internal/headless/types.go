package headless

import (
	"time"

	"github.com/opencode-ai/opencode/pkg/types"
)

// OutputFormat defines the output format for headless mode.
type OutputFormat string

const (
	// OutputText is human-readable streaming text output.
	OutputText OutputFormat = "text"
	// OutputJSON is final JSON result summary.
	OutputJSON OutputFormat = "json"
	// OutputJSONL is streaming JSONL events.
	OutputJSONL OutputFormat = "jsonl"
)

// ExitCode defines exit codes for headless mode.
type ExitCode int

const (
	// ExitSuccess indicates successful completion.
	ExitSuccess ExitCode = 0
	// ExitError indicates a general/unknown error.
	ExitError ExitCode = 1
	// ExitTimeout indicates timeout exceeded.
	ExitTimeout ExitCode = 2
	// ExitPermissionDenied indicates tool execution was blocked.
	ExitPermissionDenied ExitCode = 3
	// ExitProviderError indicates model/provider error (auth, rate limit).
	ExitProviderError ExitCode = 4
	// ExitInvalidInput indicates bad prompt or missing required flags.
	ExitInvalidInput ExitCode = 5
	// ExitSessionNotFound indicates session not found when continuing.
	ExitSessionNotFound ExitCode = 6
)

// Config holds configuration for headless mode execution.
type Config struct {
	// Prompt is the instruction to execute.
	Prompt string
	// WorkDir is the working directory.
	WorkDir string
	// AutoApprove enables automatic approval of all tool executions.
	AutoApprove bool
	// OutputFormat specifies the output format (text, json, jsonl).
	OutputFormat OutputFormat
	// Timeout is the maximum execution time.
	Timeout time.Duration
	// MaxSteps is the maximum number of agentic loop iterations.
	MaxSteps int
	// ReadStdin indicates whether to read prompt from stdin.
	ReadStdin bool
	// NoSave disables session persistence (ephemeral mode).
	NoSave bool
	// SessionID is an existing session ID to continue.
	SessionID string
	// ContinueLast continues the last session.
	ContinueLast bool
	// Files is a list of files to attach as context.
	Files []string
	// SystemPrompt is a custom system prompt file path.
	SystemPrompt string
	// Quiet suppresses progress output, only shows result.
	Quiet bool
	// Verbose shows all events (with jsonl format).
	Verbose bool
	// Model overrides the default model (format: provider/model).
	Model string
	// Agent specifies which agent to use.
	Agent string
	// Title is an optional session title.
	Title string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		OutputFormat: OutputText,
		Timeout:      30 * time.Minute,
		MaxSteps:     50,
		AutoApprove:  false,
		NoSave:       false,
		Quiet:        false,
		Verbose:      false,
	}
}

// ToolCall represents a tool call in the result.
type ToolCall struct {
	Tool         string `json:"tool"`
	Input        any    `json:"input,omitempty"`
	Output       string `json:"output,omitempty"`
	Error        string `json:"error,omitempty"`
	DurationMS   int64  `json:"duration_ms,omitempty"`
	LinesChanged int    `json:"lines_changed,omitempty"`
}

// FileDiff represents a file change in the result.
type FileDiff struct {
	File      string `json:"file"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// Result holds the final result of a headless execution.
type Result struct {
	SessionID    string          `json:"session_id"`
	Status       string          `json:"status"` // "success", "error", "timeout"
	Model        string          `json:"model"`
	DurationMS   int64           `json:"duration_ms"`
	Tokens       *types.TokenUsage `json:"tokens,omitempty"`
	Steps        int             `json:"steps"`
	ToolCalls    []ToolCall      `json:"tool_calls,omitempty"`
	Diffs        []FileDiff      `json:"diffs,omitempty"`
	FinalMessage string          `json:"final_message,omitempty"`
	Error        string          `json:"error,omitempty"`
	ExitCode     ExitCode        `json:"exit_code"`
}

// Event represents a JSONL event for streaming output.
type Event struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"ts"`
	Data      any       `json:"data"`
}

// NewEvent creates a new event with the current timestamp.
func NewEvent(eventType string, data any) *Event {
	return &Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}
}
