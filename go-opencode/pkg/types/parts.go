package types

import "encoding/json"

// Part represents a component of an assistant message.
// SDK compatible: all parts must have sessionID and messageID fields.
type Part interface {
	PartType() string
	PartID() string
	PartSessionID() string
	PartMessageID() string
}

// PartTime contains timing information for a message part.
type PartTime struct {
	Start *int64 `json:"start,omitempty"`
	End   *int64 `json:"end,omitempty"`
}

// ToolTime contains timing information for tool execution.
// SDK compatible: matches TypeScript ToolStateCompleted.time structure.
type ToolTime struct {
	Start     int64  `json:"start"`
	End       *int64 `json:"end,omitempty"`
	Compacted *int64 `json:"compacted,omitempty"`
}

// ToolState represents the state of a tool execution.
// SDK compatible: matches TypeScript ToolState discriminated union.
type ToolState struct {
	Status      string         `json:"status"`                // "pending" | "running" | "completed" | "error"
	Input       map[string]any `json:"input"`
	Raw         string         `json:"raw,omitempty"`         // Only for pending state
	Output      string         `json:"output,omitempty"`      // Only for completed state
	Error       string         `json:"error,omitempty"`       // Only for error state
	Title       string         `json:"title,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Time        *ToolTime      `json:"time,omitempty"`
	Attachments []FilePart     `json:"attachments,omitempty"` // Only for completed state
}

// TextPart represents a text content part.
// SDK compatible: includes sessionID and messageID fields.
type TextPart struct {
	ID        string         `json:"id"`
	SessionID string         `json:"sessionID"` // SDK compatible
	MessageID string         `json:"messageID"` // SDK compatible
	Type      string         `json:"type"`      // always "text"
	Text      string         `json:"text"`
	Time      PartTime       `json:"time,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

func (p *TextPart) PartType() string      { return "text" }
func (p *TextPart) PartID() string        { return p.ID }
func (p *TextPart) PartSessionID() string { return p.SessionID }
func (p *TextPart) PartMessageID() string { return p.MessageID }

// ReasoningPart represents extended thinking/reasoning content.
// SDK compatible: includes sessionID and messageID fields.
type ReasoningPart struct {
	ID        string   `json:"id"`
	SessionID string   `json:"sessionID"` // SDK compatible
	MessageID string   `json:"messageID"` // SDK compatible
	Type      string   `json:"type"`      // always "reasoning"
	Text      string   `json:"text"`
	Time      PartTime `json:"time,omitempty"`
}

func (p *ReasoningPart) PartType() string      { return "reasoning" }
func (p *ReasoningPart) PartID() string        { return p.ID }
func (p *ReasoningPart) PartSessionID() string { return p.SessionID }
func (p *ReasoningPart) PartMessageID() string { return p.MessageID }

// ToolPart represents a tool call and its result.
// SDK compatible: includes sessionID and messageID fields.
// SDK compatible: uses nested State object to match TypeScript ToolPart.state structure.
type ToolPart struct {
	ID        string         `json:"id"`
	SessionID string         `json:"sessionID"` // SDK compatible
	MessageID string         `json:"messageID"` // SDK compatible
	Type      string         `json:"type"`      // always "tool"
	CallID    string         `json:"callID"`    // SDK compatible: TypeScript uses callID
	Tool      string         `json:"tool"`      // SDK compatible: TypeScript uses tool
	State     ToolState      `json:"state"`     // SDK compatible: nested state object
	Metadata  map[string]any `json:"metadata,omitempty"` // Top-level metadata (separate from state.metadata)
}

func (p *ToolPart) PartType() string      { return "tool" }
func (p *ToolPart) PartID() string        { return p.ID }
func (p *ToolPart) PartSessionID() string { return p.SessionID }
func (p *ToolPart) PartMessageID() string { return p.MessageID }

// FilePart represents a file attachment.
// SDK compatible: includes sessionID and messageID fields.
type FilePart struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"` // SDK compatible
	MessageID string `json:"messageID"` // SDK compatible
	Type      string `json:"type"`      // always "file"
	Filename  string `json:"filename,omitempty"`
	Mime      string `json:"mime"` // SDK compatible: TypeScript uses mime
	URL       string `json:"url"`
}

func (p *FilePart) PartType() string      { return "file" }
func (p *FilePart) PartID() string        { return p.ID }
func (p *FilePart) PartSessionID() string { return p.SessionID }
func (p *FilePart) PartMessageID() string { return p.MessageID }

// StepStartPart marks the beginning of an inference step.
// SDK compatible: includes sessionID and messageID fields.
type StepStartPart struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"` // SDK compatible
	MessageID string `json:"messageID"` // SDK compatible
	Type      string `json:"type"`      // always "step-start"
	Snapshot  string `json:"snapshot,omitempty"`
}

func (p *StepStartPart) PartType() string      { return "step-start" }
func (p *StepStartPart) PartID() string        { return p.ID }
func (p *StepStartPart) PartSessionID() string { return p.SessionID }
func (p *StepStartPart) PartMessageID() string { return p.MessageID }

// StepFinishPart marks the end of an inference step with cost and token info.
// SDK compatible: includes sessionID and messageID fields.
type StepFinishPart struct {
	ID        string      `json:"id"`
	SessionID string      `json:"sessionID"` // SDK compatible
	MessageID string      `json:"messageID"` // SDK compatible
	Type      string      `json:"type"`      // always "step-finish"
	Reason    string      `json:"reason"`    // e.g., "stop", "tool-calls"
	Snapshot  string      `json:"snapshot,omitempty"`
	Cost      float64     `json:"cost"`
	Tokens    *TokenUsage `json:"tokens,omitempty"`
}

func (p *StepFinishPart) PartType() string      { return "step-finish" }
func (p *StepFinishPart) PartID() string        { return p.ID }
func (p *StepFinishPart) PartSessionID() string { return p.SessionID }
func (p *StepFinishPart) PartMessageID() string { return p.MessageID }

// CompactionPart represents a request to compact/summarize the conversation.
// SDK compatible: includes sessionID and messageID fields.
type CompactionPart struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"` // SDK compatible
	MessageID string `json:"messageID"` // SDK compatible
	Type      string `json:"type"`      // always "compaction"
	Auto      bool   `json:"auto"`      // Whether this was triggered automatically
}

func (p *CompactionPart) PartType() string      { return "compaction" }
func (p *CompactionPart) PartID() string        { return p.ID }
func (p *CompactionPart) PartSessionID() string { return p.SessionID }
func (p *CompactionPart) PartMessageID() string { return p.MessageID }

// SnapshotPart marks a git snapshot point.
// SDK compatible: includes sessionID and messageID fields.
type SnapshotPart struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"` // SDK compatible
	MessageID string `json:"messageID"` // SDK compatible
	Type      string `json:"type"`      // always "snapshot"
	Snapshot  string `json:"snapshot"`  // Git commit hash
}

func (p *SnapshotPart) PartType() string      { return "snapshot" }
func (p *SnapshotPart) PartID() string        { return p.ID }
func (p *SnapshotPart) PartSessionID() string { return p.SessionID }
func (p *SnapshotPart) PartMessageID() string { return p.MessageID }

// PatchPart represents a code patch.
// SDK compatible: includes sessionID and messageID fields.
type PatchPart struct {
	ID        string   `json:"id"`
	SessionID string   `json:"sessionID"` // SDK compatible
	MessageID string   `json:"messageID"` // SDK compatible
	Type      string   `json:"type"`      // always "patch"
	Hash      string   `json:"hash"`      // Patch hash
	Files     []string `json:"files"`     // Affected files
}

func (p *PatchPart) PartType() string      { return "patch" }
func (p *PatchPart) PartID() string        { return p.ID }
func (p *PatchPart) PartSessionID() string { return p.SessionID }
func (p *PatchPart) PartMessageID() string { return p.MessageID }

// AgentPart represents an agent invocation.
// SDK compatible: includes sessionID and messageID fields.
type AgentPart struct {
	ID        string          `json:"id"`
	SessionID string          `json:"sessionID"` // SDK compatible
	MessageID string          `json:"messageID"` // SDK compatible
	Type      string          `json:"type"`      // always "agent"
	Name      string          `json:"name"`      // Agent name
	Source    *AgentPartSource `json:"source,omitempty"`
}

// AgentPartSource contains the source text reference.
type AgentPartSource struct {
	Value string `json:"value"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

func (p *AgentPart) PartType() string      { return "agent" }
func (p *AgentPart) PartID() string        { return p.ID }
func (p *AgentPart) PartSessionID() string { return p.SessionID }
func (p *AgentPart) PartMessageID() string { return p.MessageID }

// SubtaskPart represents a subtask delegation.
// SDK compatible: includes sessionID and messageID fields.
type SubtaskPart struct {
	ID          string `json:"id"`
	SessionID   string `json:"sessionID"`   // SDK compatible
	MessageID   string `json:"messageID"`   // SDK compatible
	Type        string `json:"type"`        // always "subtask"
	Prompt      string `json:"prompt"`      // Task prompt
	Description string `json:"description"` // Task description
	Agent       string `json:"agent"`       // Agent to use
}

func (p *SubtaskPart) PartType() string      { return "subtask" }
func (p *SubtaskPart) PartID() string        { return p.ID }
func (p *SubtaskPart) PartSessionID() string { return p.SessionID }
func (p *SubtaskPart) PartMessageID() string { return p.MessageID }

// RetryPart represents a retry attempt after an error.
// SDK compatible: includes sessionID and messageID fields.
type RetryPart struct {
	ID        string         `json:"id"`
	SessionID string         `json:"sessionID"` // SDK compatible
	MessageID string         `json:"messageID"` // SDK compatible
	Type      string         `json:"type"`      // always "retry"
	Attempt   int            `json:"attempt"`   // Retry attempt number
	Error     *APIError      `json:"error"`     // Error that caused the retry
	Time      RetryPartTime  `json:"time"`
}

// RetryPartTime contains the time of the retry.
type RetryPartTime struct {
	Created int64 `json:"created"`
}

// APIError represents an API error (used by RetryPart).
type APIError struct {
	Name string         `json:"name"` // Always "APIError"
	Data APIErrorData   `json:"data"`
}

// APIErrorData contains API error details.
type APIErrorData struct {
	Status    int    `json:"status,omitempty"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable,omitempty"`
}

func (p *RetryPart) PartType() string      { return "retry" }
func (p *RetryPart) PartID() string        { return p.ID }
func (p *RetryPart) PartSessionID() string { return p.SessionID }
func (p *RetryPart) PartMessageID() string { return p.MessageID }

// RawPart is used for JSON unmarshaling of parts.
type RawPart struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"-"`
}

// UnmarshalPart unmarshals a JSON part into the appropriate type.
func UnmarshalPart(data []byte) (Part, error) {
	var raw RawPart
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	switch raw.Type {
	case "text":
		var p TextPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return &p, nil
	case "reasoning":
		var p ReasoningPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return &p, nil
	case "tool":
		var p ToolPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return &p, nil
	case "file":
		var p FilePart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return &p, nil
	case "step-start":
		var p StepStartPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return &p, nil
	case "step-finish":
		var p StepFinishPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return &p, nil
	case "compaction":
		var p CompactionPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return &p, nil
	case "snapshot":
		var p SnapshotPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return &p, nil
	case "patch":
		var p PatchPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return &p, nil
	case "agent":
		var p AgentPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return &p, nil
	case "subtask":
		var p SubtaskPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return &p, nil
	case "retry":
		var p RetryPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return &p, nil
	default:
		// Return raw part for unknown types
		var p TextPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return &p, nil
	}
}
