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

func TestInitializeRequest(t *testing.T) {
	req := InitializeRequest{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ClientCapabilities{
			Roots: &RootsCapability{ListChanged: false},
		},
		ClientInfo: ClientInfo{
			Name:    "opencode",
			Version: "1.0.0",
		},
	}

	assert.Equal(t, "2024-11-05", req.ProtocolVersion)
	assert.NotNil(t, req.Capabilities.Roots)
	assert.Equal(t, "opencode", req.ClientInfo.Name)
}

func TestCallToolRequest(t *testing.T) {
	args := json.RawMessage(`{"key": "value"}`)
	req := CallToolRequest{
		Name:      "test_tool",
		Arguments: args,
	}

	assert.Equal(t, "test_tool", req.Name)
	assert.NotNil(t, req.Arguments)
}

func TestCallToolResponse(t *testing.T) {
	resp := CallToolResponse{
		Content: []Content{
			{Type: "text", Text: "Hello, World!"},
			{Type: "image", MimeType: "image/png", Data: "base64data"},
		},
		IsError: false,
	}

	assert.Len(t, resp.Content, 2)
	assert.Equal(t, "text", resp.Content[0].Type)
	assert.Equal(t, "Hello, World!", resp.Content[0].Text)
	assert.False(t, resp.IsError)
}

func TestContent(t *testing.T) {
	textContent := Content{Type: "text", Text: "Hello"}
	assert.Equal(t, "text", textContent.Type)
	assert.Equal(t, "Hello", textContent.Text)

	imageContent := Content{Type: "image", MimeType: "image/png", Data: "data"}
	assert.Equal(t, "image", imageContent.Type)
	assert.Equal(t, "image/png", imageContent.MimeType)
}

func TestJSONRPCRequest(t *testing.T) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
		Params:  map[string]string{"key": "value"},
	}

	assert.Equal(t, "2.0", req.JSONRPC)
	assert.Equal(t, int64(1), req.ID)
	assert.Equal(t, "test", req.Method)
}

func TestJSONRPCResponse(t *testing.T) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  json.RawMessage(`{"success": true}`),
	}

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, int64(1), resp.ID)
	assert.NotNil(t, resp.Result)
	assert.Nil(t, resp.Error)
}

func TestJSONRPCError(t *testing.T) {
	err := JSONRPCError{
		Code:    -32600,
		Message: "Invalid Request",
		Data:    "Additional info",
	}

	assert.Equal(t, -32600, err.Code)
	assert.Equal(t, "Invalid Request", err.Message)
}

func TestNewHTTPTransport(t *testing.T) {
	transport, err := NewHTTPTransport("http://localhost:8080", nil)
	assert.NoError(t, err)
	assert.NotNil(t, transport)

	// Test Close
	err = transport.Close()
	assert.NoError(t, err)
}

func TestNewHTTPTransport_EmptyURL(t *testing.T) {
	_, err := NewHTTPTransport("", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "URL is required")
}

func TestNewHTTPTransport_WithHeaders(t *testing.T) {
	headers := map[string]string{
		"Authorization": "Bearer token",
		"X-Custom":      "value",
	}
	transport, err := NewHTTPTransport("http://localhost:8080", headers)
	assert.NoError(t, err)
	assert.NotNil(t, transport)
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

func TestServerCapabilities(t *testing.T) {
	caps := ServerCapabilities{
		Tools:     &ToolCapability{ListChanged: true},
		Resources: &ResourceCapability{Subscribe: true, ListChanged: true},
		Prompts:   &PromptCapability{ListChanged: false},
	}

	assert.True(t, caps.Tools.ListChanged)
	assert.True(t, caps.Resources.Subscribe)
	assert.False(t, caps.Prompts.ListChanged)
}

func TestGetPromptRequest(t *testing.T) {
	req := GetPromptRequest{
		Name: "test_prompt",
		Arguments: map[string]string{
			"arg1": "value1",
		},
	}

	assert.Equal(t, "test_prompt", req.Name)
	assert.Equal(t, "value1", req.Arguments["arg1"])
}

func TestPromptMessage(t *testing.T) {
	msg := PromptMessage{
		Role:    "user",
		Content: Content{Type: "text", Text: "Hello"},
	}

	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "Hello", msg.Content.Text)
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
