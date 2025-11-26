package types

// Message represents either a User or Assistant message in a conversation.
type Message struct {
	ID        string       `json:"id"`
	SessionID string       `json:"sessionID"`
	Role      string       `json:"role"` // "user" | "assistant"
	Time      MessageTime  `json:"time"`

	// User-specific fields
	Agent  string          `json:"agent,omitempty"`
	Model  *ModelRef       `json:"model,omitempty"`
	System *string         `json:"system,omitempty"`
	Tools  map[string]bool `json:"tools,omitempty"`

	// Assistant-specific fields
	ModelID    string        `json:"modelID,omitempty"`
	ProviderID string        `json:"providerID,omitempty"`
	Mode       string        `json:"mode,omitempty"`
	Finish     *string       `json:"finish,omitempty"`
	Cost       float64       `json:"cost,omitempty"`
	Tokens     *TokenUsage   `json:"tokens,omitempty"`
	Error      *MessageError `json:"error,omitempty"`
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
type TokenUsage struct {
	Input     int        `json:"input"`
	Output    int        `json:"output"`
	Reasoning int        `json:"reasoning,omitempty"`
	Cache     CacheUsage `json:"cache,omitempty"`
}

// CacheUsage contains cache hit/write statistics.
type CacheUsage struct {
	Read  int `json:"read"`
	Write int `json:"write"`
}

// MessageError represents an error that occurred during message processing.
type MessageError struct {
	Type    string `json:"type"` // "api" | "auth" | "output_length"
	Message string `json:"message"`
}
