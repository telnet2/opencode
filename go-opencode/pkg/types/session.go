// Package types provides the core data types for the OpenCode server.
package types

// Session represents a conversation session with the LLM.
type Session struct {
	ID           string          `json:"id"`
	ProjectID    string          `json:"projectID"`
	Directory    string          `json:"directory"`
	ParentID     *string         `json:"parentID,omitempty"`
	Title        string          `json:"title"`
	Version      string          `json:"version"`
	Summary      SessionSummary  `json:"summary"`
	Share        *SessionShare   `json:"share,omitempty"`
	Time         SessionTime     `json:"time"`
	Revert       *SessionRevert  `json:"revert,omitempty"`
	CustomPrompt *CustomPrompt   `json:"customPrompt,omitempty"`
}

// SessionSummary contains statistics about code changes in a session.
type SessionSummary struct {
	Additions int        `json:"additions"`
	Deletions int        `json:"deletions"`
	Files     int        `json:"files"`
	Diffs     []FileDiff `json:"diffs,omitempty"`
}

// FileDiff represents a diff for a single file.
type FileDiff struct {
	Path      string `json:"path"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Before    string `json:"before,omitempty"`
	After     string `json:"after,omitempty"`
}

// SessionTime contains timestamps for a session.
type SessionTime struct {
	Created    int64  `json:"created"`
	Updated    int64  `json:"updated"`
	Compacting *int64 `json:"compacting,omitempty"`
}

// SessionShare contains sharing information for a session.
type SessionShare struct {
	URL string `json:"url"`
}

// SessionRevert contains information about session revert state.
type SessionRevert struct {
	MessageID string  `json:"messageID"`
	PartID    *string `json:"partID,omitempty"`
	Snapshot  *string `json:"snapshot,omitempty"`
	Diff      *string `json:"diff,omitempty"`
}

// CustomPrompt represents a custom system prompt configuration.
type CustomPrompt struct {
	Type      string            `json:"type"` // "file" | "inline"
	Value     string            `json:"value"`
	LoadedAt  *int64            `json:"loadedAt,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

// SessionError represents an error that occurred during session processing.
// SDK compatible with the error union type.
type SessionError struct {
	Name string         `json:"name"` // "ProviderAuthError" | "UnknownError" | "MessageOutputLengthError" | "MessageAbortedError" | "APIError"
	Data map[string]any `json:"data,omitempty"`
}

// Common error types for SDK compatibility
const (
	ErrorNameProviderAuth        = "ProviderAuthError"
	ErrorNameUnknown             = "UnknownError"
	ErrorNameMessageOutputLength = "MessageOutputLengthError"
	ErrorNameMessageAborted      = "MessageAbortedError"
	ErrorNameAPI                 = "APIError"
)

// Project represents a project (worktree) that can contain sessions.
type Project struct {
	ID       string       `json:"id"`
	Worktree string       `json:"worktree"`
	VCS      *string      `json:"vcs,omitempty"` // "git" or nil
	Time     ProjectTime  `json:"time"`
}

// ProjectTime contains timestamps for a project.
type ProjectTime struct {
	Created     int64  `json:"created"`
	Initialized *int64 `json:"initialized,omitempty"`
}

// TUIControlRequest represents a pending TUI control request.
type TUIControlRequest struct {
	Path string `json:"path"`
	Body any    `json:"body"`
}
