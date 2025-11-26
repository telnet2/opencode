package event

import "github.com/opencode-ai/opencode/pkg/types"

// SessionCreatedData is the data for session.created events.
type SessionCreatedData struct {
	Session *types.Session `json:"session"`
}

// SessionUpdatedData is the data for session.updated events.
type SessionUpdatedData struct {
	Session *types.Session `json:"session"`
}

// SessionDeletedData is the data for session.deleted events.
type SessionDeletedData struct {
	SessionID string `json:"sessionID"`
}

// MessageCreatedData is the data for message.created events.
type MessageCreatedData struct {
	Message *types.Message `json:"message"`
}

// MessageUpdatedData is the data for message.updated events.
type MessageUpdatedData struct {
	Message *types.Message `json:"message"`
}

// MessageRemovedData is the data for message.removed events.
type MessageRemovedData struct {
	SessionID string `json:"sessionID"`
	MessageID string `json:"messageID"`
}

// PartUpdatedData is the data for part.updated events.
type PartUpdatedData struct {
	SessionID string     `json:"sessionID"`
	MessageID string     `json:"messageID"`
	Part      types.Part `json:"part"`
	Delta     *string    `json:"delta,omitempty"` // For streaming text
}

// FileEditedData is the data for file.edited events.
type FileEditedData struct {
	SessionID string `json:"sessionID"`
	File      string `json:"file"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// PermissionRequiredData is the data for permission.required events.
type PermissionRequiredData struct {
	ID             string   `json:"id"`
	SessionID      string   `json:"sessionID"`
	PermissionType string   `json:"permissionType"` // "bash" | "edit" | "external_directory"
	Pattern        []string `json:"pattern"`
	Title          string   `json:"title"`
}

// PermissionResolvedData is the data for permission.resolved events.
type PermissionResolvedData struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	Granted   bool   `json:"granted"`
}
