package types

import "encoding/json"

// Part represents a component of an assistant message.
type Part interface {
	PartType() string
	PartID() string
}

// PartTime contains timing information for a message part.
type PartTime struct {
	Start *int64 `json:"start,omitempty"`
	End   *int64 `json:"end,omitempty"`
}

// TextPart represents a text content part.
type TextPart struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"` // always "text"
	Text     string         `json:"text"`
	Time     PartTime       `json:"time,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (p *TextPart) PartType() string { return "text" }
func (p *TextPart) PartID() string   { return p.ID }

// ReasoningPart represents extended thinking/reasoning content.
type ReasoningPart struct {
	ID   string   `json:"id"`
	Type string   `json:"type"` // always "reasoning"
	Text string   `json:"text"`
	Time PartTime `json:"time,omitempty"`
}

func (p *ReasoningPart) PartType() string { return "reasoning" }
func (p *ReasoningPart) PartID() string   { return p.ID }

// ToolPart represents a tool call and its result.
type ToolPart struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"` // always "tool"
	ToolCallID string         `json:"toolCallID"`
	ToolName   string         `json:"toolName"`
	Input      map[string]any `json:"input"`
	State      string         `json:"state"` // "pending" | "running" | "completed" | "error"
	Output     *string        `json:"output,omitempty"`
	Error      *string        `json:"error,omitempty"`
	Title      *string        `json:"title,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Time       PartTime       `json:"time,omitempty"`
}

func (p *ToolPart) PartType() string { return "tool" }
func (p *ToolPart) PartID() string   { return p.ID }

// FilePart represents a file attachment.
type FilePart struct {
	ID        string `json:"id"`
	Type      string `json:"type"` // always "file"
	Filename  string `json:"filename"`
	MediaType string `json:"mediaType"`
	URL       string `json:"url"`
}

func (p *FilePart) PartType() string { return "file" }
func (p *FilePart) PartID() string   { return p.ID }

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
	default:
		// Return raw part for unknown types
		var p TextPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return &p, nil
	}
}
