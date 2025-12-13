package types

import "encoding/json"

// Message represents either a User or Assistant message in a conversation.
type Message struct {
	ID        string      `json:"id"`
	SessionID string      `json:"sessionID"`
	Role      string      `json:"role"` // "user" | "assistant"
	Time      MessageTime `json:"time"`

	// User-specific fields
	Agent   string              `json:"agent,omitempty"`
	Model   *ModelRef           `json:"model,omitempty"`
	System  *string             `json:"system,omitempty"`
	Tools   map[string]bool     `json:"tools,omitempty"`
	Summary *UserMessageSummary `json:"-"` // Summary with title and diffs (for user messages)

	// Assistant-specific fields
	ParentID   string        `json:"parentID,omitempty"`   // Links to the user message that prompted this
	ModelID    string        `json:"modelID,omitempty"`
	ProviderID string        `json:"providerID,omitempty"`
	Mode       string        `json:"mode,omitempty"`       // Agent name (e.g., "Coder", "Build")
	Path       *MessagePath  `json:"path,omitempty"`       // Current working directory and root
	IsSummary  bool          `json:"-"`                    // True if this is a summary/compaction message (for assistant messages)
	Finish     *string       `json:"finish,omitempty"`
	Cost       float64       `json:"cost"`                 // Required by TUI
	Tokens     *TokenUsage   `json:"tokens,omitempty"`
	Error      *MessageError `json:"error,omitempty"`
}

// MarshalJSON implements custom JSON marshaling to handle the summary field
// differently based on message role.
func (m Message) MarshalJSON() ([]byte, error) {
	type Alias Message
	aux := struct {
		Alias
		Summary any `json:"summary,omitempty"`
	}{
		Alias: Alias(m),
	}

	// Set the appropriate summary field based on role
	if m.Role == "user" && m.Summary != nil {
		aux.Summary = m.Summary
	} else if m.Role == "assistant" && m.IsSummary {
		aux.Summary = true
	}

	return json.Marshal(aux)
}

// UnmarshalJSON implements custom JSON unmarshaling to handle the summary field
// differently based on message role.
func (m *Message) UnmarshalJSON(data []byte) error {
	type Alias Message
	aux := struct {
		*Alias
		Summary json.RawMessage `json:"summary,omitempty"`
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Parse summary based on role
	if len(aux.Summary) > 0 {
		if m.Role == "user" {
			var summary UserMessageSummary
			if err := json.Unmarshal(aux.Summary, &summary); err == nil {
				m.Summary = &summary
			}
		} else if m.Role == "assistant" {
			var isSummary bool
			if err := json.Unmarshal(aux.Summary, &isSummary); err == nil {
				m.IsSummary = isSummary
			}
		}
	}

	return nil
}

// MessagePath contains the current working directory and project root.
type MessagePath struct {
	Cwd  string `json:"cwd"`
	Root string `json:"root"`
}

// UserMessageSummary contains summary information for a user message.
// Uses FileDiff from session.go for diffs.
type UserMessageSummary struct {
	Title string     `json:"title,omitempty"`
	Body  string     `json:"body,omitempty"`
	Diffs []FileDiff `json:"diffs,omitempty"`
}

// MessageTime contains timestamps for a message.
type MessageTime struct {
	Created int64  `json:"created"`
	Updated *int64 `json:"updated,omitempty"`
}

// ModelRef references a specific model from a provider.
type ModelRef struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
}

// TokenUsage contains token usage statistics for a message.
// Note: All fields are required by TUI, do not use omitempty.
type TokenUsage struct {
	Input     int        `json:"input"`
	Output    int        `json:"output"`
	Reasoning int        `json:"reasoning"`
	Cache     CacheUsage `json:"cache"`
}

// CacheUsage contains cache hit/write statistics.
type CacheUsage struct {
	Read  int `json:"read"`
	Write int `json:"write"`
}

// MessageError represents an error that occurred during message processing.
// Format: {"name": "UnknownError", "data": {"message": "..."}}
type MessageError struct {
	Name string           `json:"name"` // "UnknownError" | "ProviderAuthError" | "MessageOutputLengthError"
	Data MessageErrorData `json:"data"`
}

// MessageErrorData contains the error details.
type MessageErrorData struct {
	Message    string `json:"message"`
	ProviderID string `json:"providerID,omitempty"` // For ProviderAuthError
}

// NewUnknownError creates a new UnknownError.
func NewUnknownError(message string) *MessageError {
	return &MessageError{
		Name: "UnknownError",
		Data: MessageErrorData{Message: message},
	}
}

// NewProviderAuthError creates a new ProviderAuthError.
func NewProviderAuthError(providerID, message string) *MessageError {
	return &MessageError{
		Name: "ProviderAuthError",
		Data: MessageErrorData{Message: message, ProviderID: providerID},
	}
}
