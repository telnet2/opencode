package event

import "github.com/opencode-ai/opencode/pkg/types"

// SessionCreatedData is the data for session.created events.
// SDK compatible: uses "info" field for session object.
type SessionCreatedData struct {
	Info *types.Session `json:"info"`
}

// SessionUpdatedData is the data for session.updated events.
// SDK compatible: uses "info" field for session object.
type SessionUpdatedData struct {
	Info *types.Session `json:"info"`
}

// SessionDeletedData is the data for session.deleted events.
// SDK compatible: uses "info" field for session object.
type SessionDeletedData struct {
	Info *types.Session `json:"info"`
}

// SessionIdleData is the data for session.idle events.
type SessionIdleData struct {
	SessionID string `json:"sessionID"`
}

// SessionErrorData is the data for session.error events.
type SessionErrorData struct {
	SessionID string              `json:"sessionID,omitempty"`
	Error     *types.MessageError `json:"error,omitempty"`
}

// MessageCreatedData is the data for message.created events.
// SDK compatible: uses "info" field for message object.
type MessageCreatedData struct {
	Info *types.Message `json:"info"`
}

// MessageUpdatedData is the data for message.updated events.
// SDK compatible: uses "info" field for message object.
type MessageUpdatedData struct {
	Info *types.Message `json:"info"`
}

// MessageRemovedData is the data for message.removed events.
type MessageRemovedData struct {
	SessionID string `json:"sessionID"`
	MessageID string `json:"messageID"`
}

// MessagePartUpdatedData is the data for message.part.updated events.
// SDK compatible: uses "part" and "delta" fields.
type MessagePartUpdatedData struct {
	Part  types.Part `json:"part"`
	Delta string     `json:"delta,omitempty"` // For streaming text
}

// Deprecated: Use MessagePartUpdatedData instead
type PartUpdatedData = MessagePartUpdatedData

// MessagePartRemovedData is the data for message.part.removed events.
type MessagePartRemovedData struct {
	SessionID string `json:"sessionID"`
	MessageID string `json:"messageID"`
	PartID    string `json:"partID"`
}

// FileEditedData is the data for file.edited events.
type FileEditedData struct {
	File string `json:"file"`
}

// PermissionUpdatedData is the data for permission.updated events.
// SDK compatible format for permission requests.
type PermissionUpdatedData struct {
	ID             string   `json:"id"`
	SessionID      string   `json:"sessionID"`
	PermissionType string   `json:"permissionType"` // "bash" | "edit" | "external_directory"
	Pattern        []string `json:"pattern"`
	Title          string   `json:"title"`
}

// Deprecated: Use PermissionUpdatedData instead
type PermissionRequiredData = PermissionUpdatedData

// PermissionRepliedData is the data for permission.replied events.
type PermissionRepliedData struct {
	PermissionID string `json:"permissionID"`
	SessionID    string `json:"sessionID"`
	Response     string `json:"response"` // "once" | "always" | "reject"
}

// Deprecated: Use PermissionRepliedData instead
type PermissionResolvedData struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	Granted   bool   `json:"granted"`
}

// ClientToolRequestData is the data for client-tool.request events.
type ClientToolRequestData struct {
	ClientID string `json:"clientID"`
	Request  any    `json:"request"` // ExecutionRequest from clienttool package
}

// ClientToolRegisteredData is the data for client-tool.registered events.
type ClientToolRegisteredData struct {
	ClientID string   `json:"clientID"`
	ToolIDs  []string `json:"toolIDs"`
}

// ClientToolUnregisteredData is the data for client-tool.unregistered events.
type ClientToolUnregisteredData struct {
	ClientID string   `json:"clientID"`
	ToolIDs  []string `json:"toolIDs"`
}

// ClientToolStatusData is the data for client-tool.executing/completed/failed events.
type ClientToolStatusData struct {
	SessionID string `json:"sessionID"`
	MessageID string `json:"messageID"`
	CallID    string `json:"callID"`
	Tool      string `json:"tool"`
	ClientID  string `json:"clientID"`
	Error     string `json:"error,omitempty"`
	Success   bool   `json:"success,omitempty"`
}
