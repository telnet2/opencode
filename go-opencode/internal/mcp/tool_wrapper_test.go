package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/opencode-ai/opencode/internal/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPToolWrapper_ImplementsInterface(t *testing.T) {
	mcpTool := Tool{
		Name:        "test_server_test_tool",
		Description: "A test tool",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"input":{"type":"string"}}}`),
	}

	wrapper := NewMCPToolWrapper(mcpTool, nil)

	// Verify interface compliance at compile time
	var _ tool.Tool = wrapper

	assert.Equal(t, "test_server_test_tool", wrapper.ID())
	assert.Equal(t, "A test tool", wrapper.Description())
	assert.NotNil(t, wrapper.Parameters())
}

func TestMCPToolWrapper_ID(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		want     string
	}{
		{
			name:     "simple name",
			toolName: "calculator_sum",
			want:     "calculator_sum",
		},
		{
			name:     "prefixed name",
			toolName: "server_name_tool_name",
			want:     "server_name_tool_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapper := NewMCPToolWrapper(Tool{Name: tt.toolName}, nil)
			assert.Equal(t, tt.want, wrapper.ID())
		})
	}
}

func TestMCPToolWrapper_Description(t *testing.T) {
	wrapper := NewMCPToolWrapper(Tool{
		Name:        "test",
		Description: "Test tool description",
	}, nil)

	assert.Equal(t, "Test tool description", wrapper.Description())
}

func TestMCPToolWrapper_Parameters(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"numbers":{"type":"array","description":"Numbers to add"}}}`)
	wrapper := NewMCPToolWrapper(Tool{
		Name:        "test",
		InputSchema: schema,
	}, nil)

	params := wrapper.Parameters()
	assert.NotNil(t, params)
	assert.JSONEq(t, string(schema), string(params))
}

func TestMCPToolWrapper_EinoTool(t *testing.T) {
	mcpTool := Tool{
		Name:        "test_tool",
		Description: "Test description",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"num":{"type":"integer","description":"A number"}},"required":["num"]}`),
	}

	wrapper := NewMCPToolWrapper(mcpTool, nil)
	einoTool := wrapper.EinoTool()

	info, err := einoTool.Info(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "test_tool", info.Name)
	assert.Equal(t, "Test description", info.Desc)
	assert.NotNil(t, info.ParamsOneOf)
}

func TestParseInputSchemaToParams(t *testing.T) {
	tests := []struct {
		name           string
		schema         json.RawMessage
		expectedParams []string
		expectedTypes  map[string]string
	}{
		{
			name: "string param",
			schema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {"type": "string", "description": "The name"}
				},
				"required": ["name"]
			}`),
			expectedParams: []string{"name"},
			expectedTypes:  map[string]string{"name": "string"},
		},
		{
			name: "integer param",
			schema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"count": {"type": "integer", "description": "The count"}
				}
			}`),
			expectedParams: []string{"count"},
			expectedTypes:  map[string]string{"count": "integer"},
		},
		{
			name: "array param",
			schema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"numbers": {"type": "array", "description": "Numbers to sum"}
				},
				"required": ["numbers"]
			}`),
			expectedParams: []string{"numbers"},
			expectedTypes:  map[string]string{"numbers": "array"},
		},
		{
			name:           "empty schema",
			schema:         json.RawMessage(`{}`),
			expectedParams: []string{},
			expectedTypes:  map[string]string{},
		},
		{
			name:           "invalid schema",
			schema:         json.RawMessage(`invalid`),
			expectedParams: []string{},
			expectedTypes:  map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := parseInputSchemaToParams(tt.schema)

			if len(tt.expectedParams) == 0 {
				assert.Empty(t, params)
				return
			}

			for _, expectedName := range tt.expectedParams {
				assert.Contains(t, params, expectedName)
			}
		})
	}
}

func TestRegisterMCPTools_NilClient(t *testing.T) {
	registry := tool.NewRegistry("")

	// Should not panic with nil client
	RegisterMCPTools(nil, registry)

	// Registry should still be empty
	assert.Empty(t, registry.List())
}

func TestRegisterMCPTools_NilRegistry(t *testing.T) {
	client := NewClient()
	defer client.Close()

	// Should not panic with nil registry
	RegisterMCPTools(client, nil)
}

func TestRegisterMCPTools_NoServers(t *testing.T) {
	client := NewClient()
	defer client.Close()
	registry := tool.NewRegistry("")

	// Register with no connected servers
	RegisterMCPTools(client, registry)

	// Registry should be empty (no tools from MCP)
	// Only contains tools that were already registered
	assert.Empty(t, registry.List())
}
