package provider

import (
	"encoding/json"
	"testing"

	"github.com/cloudwego/eino/schema"

	"github.com/opencode-ai/opencode/pkg/types"
)

func TestParseModelString(t *testing.T) {
	tests := []struct {
		input          string
		wantProvider   string
		wantModel      string
	}{
		{"anthropic/claude-3-opus", "anthropic", "claude-3-opus"},
		{"openai/gpt-4o", "openai", "gpt-4o"},
		{"bedrock/anthropic.claude-3", "bedrock", "anthropic.claude-3"},
		{"claude-3-opus", "", "claude-3-opus"}, // No provider prefix
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			provider, model := ParseModelString(tt.input)
			if provider != tt.wantProvider {
				t.Errorf("ParseModelString(%q) provider = %q, want %q", tt.input, provider, tt.wantProvider)
			}
			if model != tt.wantModel {
				t.Errorf("ParseModelString(%q) model = %q, want %q", tt.input, model, tt.wantModel)
			}
		})
	}
}

func TestModelPriority(t *testing.T) {
	tests := []struct {
		modelID       string
		wantHigherThan string
	}{
		{"gpt-5-turbo", "claude-sonnet-4-latest"},
		{"claude-sonnet-4-20250514", "gpt-4o-2024"},
		{"claude-opus-4", "gpt-4o"},
		{"gpt-4o-latest", "claude-3-5-sonnet"},
	}

	for _, tt := range tests {
		t.Run(tt.modelID+" > "+tt.wantHigherThan, func(t *testing.T) {
			high := modelPriority(tt.modelID)
			low := modelPriority(tt.wantHigherThan)
			if high <= low {
				t.Errorf("modelPriority(%q) = %d, should be > modelPriority(%q) = %d",
					tt.modelID, high, tt.wantHigherThan, low)
			}
		})
	}
}

func TestConvertToEinoTools(t *testing.T) {
	tools := []ToolInfo{
		{
			Name:        "read_file",
			Description: "Reads a file",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {"type": "string", "description": "File path"},
					"limit": {"type": "integer", "description": "Max lines"}
				},
				"required": ["path"]
			}`),
		},
		{
			Name:        "bash",
			Description: "Runs a command",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"command": {"type": "string", "description": "Command to run"},
					"timeout": {"type": "number", "description": "Timeout in ms"}
				},
				"required": ["command"]
			}`),
		},
	}

	result := ConvertToEinoTools(tools)

	if len(result) != 2 {
		t.Fatalf("Expected 2 tools, got %d", len(result))
	}

	if result[0].Name != "read_file" {
		t.Errorf("Expected tool name 'read_file', got %s", result[0].Name)
	}
	if result[0].Desc != "Reads a file" {
		t.Errorf("Expected description 'Reads a file', got %s", result[0].Desc)
	}

	if result[1].Name != "bash" {
		t.Errorf("Expected tool name 'bash', got %s", result[1].Name)
	}
}

func TestParseJSONSchemaToParams(t *testing.T) {
	schemaJSON := json.RawMessage(`{
		"type": "object",
		"properties": {
			"stringParam": {"type": "string", "description": "A string"},
			"intParam": {"type": "integer", "description": "An integer"},
			"numParam": {"type": "number", "description": "A number"},
			"boolParam": {"type": "boolean", "description": "A boolean"},
			"arrayParam": {"type": "array", "description": "An array"},
			"objectParam": {"type": "object", "description": "An object"}
		},
		"required": ["stringParam", "intParam"]
	}`)

	params := parseJSONSchemaToParams(schemaJSON)

	if params == nil {
		t.Fatal("Expected non-nil params")
	}

	// Check string param
	if p, ok := params["stringParam"]; !ok {
		t.Error("Missing stringParam")
	} else {
		if p.Type != schema.String {
			t.Errorf("stringParam type = %v, want String", p.Type)
		}
		if !p.Required {
			t.Error("stringParam should be required")
		}
	}

	// Check integer param
	if p, ok := params["intParam"]; !ok {
		t.Error("Missing intParam")
	} else {
		if p.Type != schema.Integer {
			t.Errorf("intParam type = %v, want Integer", p.Type)
		}
		if !p.Required {
			t.Error("intParam should be required")
		}
	}

	// Check number param
	if p, ok := params["numParam"]; !ok {
		t.Error("Missing numParam")
	} else {
		if p.Type != schema.Number {
			t.Errorf("numParam type = %v, want Number", p.Type)
		}
		if p.Required {
			t.Error("numParam should not be required")
		}
	}

	// Check boolean param
	if p, ok := params["boolParam"]; !ok {
		t.Error("Missing boolParam")
	} else if p.Type != schema.Boolean {
		t.Errorf("boolParam type = %v, want Boolean", p.Type)
	}

	// Check array param
	if p, ok := params["arrayParam"]; !ok {
		t.Error("Missing arrayParam")
	} else if p.Type != schema.Array {
		t.Errorf("arrayParam type = %v, want Array", p.Type)
	}

	// Check object param
	if p, ok := params["objectParam"]; !ok {
		t.Error("Missing objectParam")
	} else if p.Type != schema.Object {
		t.Errorf("objectParam type = %v, want Object", p.Type)
	}
}

func TestParseJSONSchemaToParams_InvalidJSON(t *testing.T) {
	result := parseJSONSchemaToParams(json.RawMessage(`invalid json`))
	if result != nil {
		t.Error("Expected nil for invalid JSON")
	}
}

func TestParseJSONSchemaToParams_EmptySchema(t *testing.T) {
	result := parseJSONSchemaToParams(json.RawMessage(`{}`))
	if result == nil {
		t.Error("Expected non-nil map for empty schema")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty map, got %d entries", len(result))
	}
}

func TestConvertFromEinoMessage(t *testing.T) {
	tests := []struct {
		name      string
		einoMsg   *schema.Message
		wantRole  string
	}{
		{
			name:     "user message",
			einoMsg:  &schema.Message{Role: schema.User, Content: "Hello"},
			wantRole: "user",
		},
		{
			name:     "assistant message",
			einoMsg:  &schema.Message{Role: schema.Assistant, Content: "Hi there"},
			wantRole: "assistant",
		},
		{
			name:     "system message",
			einoMsg:  &schema.Message{Role: schema.System, Content: "You are helpful"},
			wantRole: "system",
		},
		{
			name:     "tool message",
			einoMsg:  &schema.Message{Role: schema.Tool, Content: "result"},
			wantRole: "tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertFromEinoMessage(tt.einoMsg, "session-123")
			if result.Role != tt.wantRole {
				t.Errorf("Role = %q, want %q", result.Role, tt.wantRole)
			}
			if result.SessionID != "session-123" {
				t.Errorf("SessionID = %q, want 'session-123'", result.SessionID)
			}
		})
	}
}

func TestConvertToEinoMessages(t *testing.T) {
	messages := []*types.Message{
		{ID: "msg1", Role: "user"},
		{ID: "msg2", Role: "assistant"},
		{ID: "msg3", Role: "system"},
	}

	parts := map[string][]types.Part{
		"msg1": {&types.TextPart{ID: "p1", Type: "text", Text: "Hello"}},
		"msg2": {
			&types.TextPart{ID: "p2", Type: "text", Text: "Hi there"},
			&types.ToolPart{
				ID:         "p3",
				Type:       "tool_use",
				ToolCallID: "call-123",
				ToolName:   "read_file",
				Input:      map[string]any{"path": "/test.txt"},
			},
		},
	}

	result := ConvertToEinoMessages(messages, parts)

	if len(result) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(result))
	}

	// Check user message
	if result[0].Role != schema.User {
		t.Errorf("Message 0 role = %v, want User", result[0].Role)
	}
	if result[0].Content != "Hello" {
		t.Errorf("Message 0 content = %q, want 'Hello'", result[0].Content)
	}

	// Check assistant message with tool call
	if result[1].Role != schema.Assistant {
		t.Errorf("Message 1 role = %v, want Assistant", result[1].Role)
	}
	if result[1].Content != "Hi there" {
		t.Errorf("Message 1 content = %q, want 'Hi there'", result[1].Content)
	}
	if len(result[1].ToolCalls) != 1 {
		t.Fatalf("Message 1 should have 1 tool call, got %d", len(result[1].ToolCalls))
	}
	if result[1].ToolCalls[0].ID != "call-123" {
		t.Errorf("Tool call ID = %q, want 'call-123'", result[1].ToolCalls[0].ID)
	}
	if result[1].ToolCalls[0].Function.Name != "read_file" {
		t.Errorf("Tool call name = %q, want 'read_file'", result[1].ToolCalls[0].Function.Name)
	}

	// Check system message
	if result[2].Role != schema.System {
		t.Errorf("Message 2 role = %v, want System", result[2].Role)
	}
}

func TestConvertToEinoMessages_Empty(t *testing.T) {
	result := ConvertToEinoMessages(nil, nil)
	if result == nil {
		t.Error("Expected non-nil slice")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got %d", len(result))
	}
}
