package mcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	assert.NotNil(t, client)
	assert.Equal(t, 0, client.ServerCount())
}

func TestClient_ServerCount(t *testing.T) {
	client := NewClient()
	assert.Equal(t, 0, client.ServerCount())
}

func TestClient_ConnectedCount(t *testing.T) {
	client := NewClient()
	assert.Equal(t, 0, client.ConnectedCount())
}

func TestClient_Status_Empty(t *testing.T) {
	client := NewClient()
	status := client.Status()
	assert.Empty(t, status)
}

func TestClient_Close(t *testing.T) {
	client := NewClient()

	// Should not panic on empty client
	err := client.Close()
	assert.NoError(t, err)
}

func TestClient_GetServer_NotFound(t *testing.T) {
	client := NewClient()
	_, err := client.GetServer("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server not found")
}

func TestClient_RemoveServer_NotFound(t *testing.T) {
	client := NewClient()
	err := client.RemoveServer("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server not found")
}

func TestClient_Tools_Empty(t *testing.T) {
	client := NewClient()
	tools := client.Tools()
	assert.Empty(t, tools)
}

func TestSanitizeToolName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with-dash", "with_dash"},
		{"with_underscore", "with_underscore"},
		{"with.dot", "with_dot"},
		{"with space", "with_space"},
		{"CamelCase", "CamelCase"},
		{"with123numbers", "with123numbers"},
		{"special!@#chars", "special___chars"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeToolName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig(t *testing.T) {
	config := Config{
		Enabled: true,
		Type:    TransportTypeRemote,
		URL:     "http://localhost:8080",
		Headers: map[string]string{
			"Authorization": "Bearer token",
		},
		Timeout: 5000,
	}

	assert.True(t, config.Enabled)
	assert.Equal(t, TransportTypeRemote, config.Type)
	assert.Equal(t, "http://localhost:8080", config.URL)
	assert.Equal(t, "Bearer token", config.Headers["Authorization"])
	assert.Equal(t, 5000, config.Timeout)
}

func TestConfig_Local(t *testing.T) {
	config := Config{
		Enabled: true,
		Type:    TransportTypeLocal,
		Command: []string{"mcp-server", "--port", "8080"},
		Environment: map[string]string{
			"DEBUG": "true",
		},
	}

	assert.Equal(t, TransportTypeLocal, config.Type)
	assert.Len(t, config.Command, 3)
	assert.Equal(t, "mcp-server", config.Command[0])
	assert.Equal(t, "true", config.Environment["DEBUG"])
}

func TestTool(t *testing.T) {
	schema := json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}}}`)
	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: schema,
	}

	assert.Equal(t, "test_tool", tool.Name)
	assert.Equal(t, "A test tool", tool.Description)
	assert.NotNil(t, tool.InputSchema)
}

func TestResource(t *testing.T) {
	resource := Resource{
		URI:         "file:///path/to/file",
		Name:        "test_file",
		Description: "A test file",
		MimeType:    "text/plain",
	}

	assert.Equal(t, "file:///path/to/file", resource.URI)
	assert.Equal(t, "test_file", resource.Name)
	assert.Equal(t, "text/plain", resource.MimeType)
}

func TestPrompt(t *testing.T) {
	prompt := Prompt{
		Name:        "test_prompt",
		Description: "A test prompt",
		Arguments: []PromptArgument{
			{Name: "arg1", Description: "First argument", Required: true},
			{Name: "arg2", Description: "Second argument", Required: false},
		},
	}

	assert.Equal(t, "test_prompt", prompt.Name)
	assert.Len(t, prompt.Arguments, 2)
	assert.True(t, prompt.Arguments[0].Required)
	assert.False(t, prompt.Arguments[1].Required)
}

func TestServerStatus(t *testing.T) {
	errMsg := "connection failed"
	status := ServerStatus{
		Name:      "test_server",
		Status:    StatusFailed,
		ToolCount: 5,
		Error:     &errMsg,
	}

	assert.Equal(t, "test_server", status.Name)
	assert.Equal(t, StatusFailed, status.Status)
	assert.Equal(t, 5, status.ToolCount)
	assert.NotNil(t, status.Error)
	assert.Equal(t, "connection failed", *status.Error)
}

func TestStatus_Constants(t *testing.T) {
	assert.Equal(t, Status("connected"), StatusConnected)
	assert.Equal(t, Status("disabled"), StatusDisabled)
	assert.Equal(t, Status("failed"), StatusFailed)
	assert.Equal(t, Status("connecting"), StatusConnecting)
	assert.Equal(t, Status("disconnected"), StatusDisconnected)
}

func TestTransportType_Constants(t *testing.T) {
	assert.Equal(t, TransportType("remote"), TransportTypeRemote)
	assert.Equal(t, TransportType("local"), TransportTypeLocal)
	assert.Equal(t, TransportType("stdio"), TransportTypeStdio)
}

func TestProtocolVersion(t *testing.T) {
	assert.Equal(t, "2024-11-05", ProtocolVersion)
}

func TestServerInfo(t *testing.T) {
	info := ServerInfo{
		Name:    "test-server",
		Version: "1.0.0",
	}
	assert.Equal(t, "test-server", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
}

func TestContent(t *testing.T) {
	textContent := Content{Type: "text", Text: "Hello"}
	assert.Equal(t, "text", textContent.Type)
	assert.Equal(t, "Hello", textContent.Text)

	imageContent := Content{Type: "image", MimeType: "image/png", Data: "data"}
	assert.Equal(t, "image", imageContent.Type)
	assert.Equal(t, "image/png", imageContent.MimeType)
}

func TestResourceContent(t *testing.T) {
	content := ResourceContent{
		URI:      "file:///test.txt",
		MimeType: "text/plain",
		Text:     "file contents",
	}

	assert.Equal(t, "file:///test.txt", content.URI)
	assert.Equal(t, "text/plain", content.MimeType)
	assert.Equal(t, "file contents", content.Text)
}

func TestReadResourceResponse(t *testing.T) {
	resp := ReadResourceResponse{
		Contents: []ResourceContent{
			{
				URI:      "file:///test.txt",
				MimeType: "text/plain",
				Text:     "content",
			},
		},
	}

	assert.Len(t, resp.Contents, 1)
	assert.Equal(t, "file:///test.txt", resp.Contents[0].URI)
}

func TestPromptArgument(t *testing.T) {
	arg := PromptArgument{
		Name:        "test_arg",
		Description: "A test argument",
		Required:    true,
	}

	assert.Equal(t, "test_arg", arg.Name)
	assert.Equal(t, "A test argument", arg.Description)
	assert.True(t, arg.Required)
}
