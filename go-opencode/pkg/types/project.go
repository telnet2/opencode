package types

// Project represents a workspace project.
// SDK compatible: matches OpenAPI Project schema.
type Project struct {
	ID       string      `json:"id"`
	Worktree string      `json:"worktree"`
	VCS      string      `json:"vcs,omitempty"` // "git" or empty
	Time     ProjectTime `json:"time"`
}

// ProjectTime contains project timestamps.
type ProjectTime struct {
	Created     int64  `json:"created"`
	Initialized *int64 `json:"initialized,omitempty"`
}
