// Package permission provides permission control for tool execution.
package permission

// PermissionAction represents the action to take for a permission check.
type PermissionAction string

const (
	ActionAllow PermissionAction = "allow"
	ActionDeny  PermissionAction = "deny"
	ActionAsk   PermissionAction = "ask"
)

// PermissionType represents the type of permission being checked.
type PermissionType string

const (
	PermBash        PermissionType = "bash"
	PermEdit        PermissionType = "edit"
	PermWebFetch    PermissionType = "webfetch"
	PermExternalDir PermissionType = "external_directory"
	PermDoomLoop    PermissionType = "doom_loop"
)

// Request represents a request for permission.
type Request struct {
	ID        string         `json:"id"`
	Type      PermissionType `json:"type"`
	Pattern   []string       `json:"pattern,omitempty"`
	SessionID string         `json:"sessionID"`
	MessageID string         `json:"messageID"`
	CallID    string         `json:"callID,omitempty"`
	Title     string         `json:"title"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// Response represents a user's response to a permission request.
type Response struct {
	RequestID string `json:"requestID"`
	Action    string `json:"action"` // "once" | "always" | "reject"
}

// RejectedError is returned when permission is denied.
type RejectedError struct {
	SessionID string
	Type      PermissionType
	CallID    string
	Metadata  map[string]any
	Message   string
}

func (e *RejectedError) Error() string {
	return e.Message
}

// IsRejectedError checks if an error is a permission rejection.
func IsRejectedError(err error) bool {
	_, ok := err.(*RejectedError)
	return ok
}

// AgentPermissions represents the permission configuration for an agent.
type AgentPermissions struct {
	Edit        PermissionAction            `json:"edit"`
	WebFetch    PermissionAction            `json:"webfetch"`
	ExternalDir PermissionAction            `json:"external_directory"`
	DoomLoop    PermissionAction            `json:"doom_loop"`
	Bash        map[string]PermissionAction `json:"bash"` // pattern -> action
}

// DefaultAgentPermissions returns default (ask everything) permissions.
func DefaultAgentPermissions() AgentPermissions {
	return AgentPermissions{
		Edit:        ActionAsk,
		WebFetch:    ActionAsk,
		ExternalDir: ActionAsk,
		DoomLoop:    ActionAsk,
		Bash:        map[string]PermissionAction{},
	}
}
