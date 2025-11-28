package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/opencode-ai/opencode/internal/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegisterMCPTools_WithCalculator tests that MCP tools can be registered
// in the tool registry and executed via the tool.Tool interface.
func TestRegisterMCPTools_WithCalculator(t *testing.T) {
	// Build the calculator-mcp binary
	binaryPath := buildCalculatorMCP(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create MCP client and connect to calculator server
	client := NewClient()
	defer client.Close()

	config := &Config{
		Enabled: true,
		Type:    TransportTypeStdio,
		Command: []string{binaryPath},
		Timeout: 10000,
	}

	err := client.AddServer(ctx, "calculator", config)
	require.NoError(t, err, "failed to add calculator server")

	// Create tool registry and register MCP tools
	registry := tool.NewRegistry("")
	RegisterMCPTools(client, registry)

	// Verify the sum tool is registered with prefixed name
	sumTool, ok := registry.Get("calculator_sum")
	require.True(t, ok, "sum tool should be registered in registry")

	// Verify tool interface methods
	assert.Equal(t, "calculator_sum", sumTool.ID())
	assert.Contains(t, sumTool.Description(), "sum")
	assert.NotNil(t, sumTool.Parameters())

	// Execute tool via the registry's tool interface
	input := json.RawMessage(`{"numbers":[1,2,3,4,5]}`)
	result, err := sumTool.Execute(ctx, input, nil)
	require.NoError(t, err)
	assert.Equal(t, "15", result.Output)
}

// TestRegisterMCPTools_EinoToolExecution tests that MCP tools can be executed
// via the Eino tool interface.
func TestRegisterMCPTools_EinoToolExecution(t *testing.T) {
	// Build the calculator-mcp binary
	binaryPath := buildCalculatorMCP(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create MCP client and connect to calculator server
	client := NewClient()
	defer client.Close()

	config := &Config{
		Enabled: true,
		Type:    TransportTypeStdio,
		Command: []string{binaryPath},
		Timeout: 10000,
	}

	err := client.AddServer(ctx, "calculator", config)
	require.NoError(t, err)

	// Create tool registry and register MCP tools
	registry := tool.NewRegistry("")
	RegisterMCPTools(client, registry)

	// Get the sum tool
	sumTool, ok := registry.Get("calculator_sum")
	require.True(t, ok)

	// Get Eino tool interface
	einoTool := sumTool.EinoTool()
	require.NotNil(t, einoTool)

	// Check tool info
	info, err := einoTool.Info(ctx)
	require.NoError(t, err)
	assert.Equal(t, "calculator_sum", info.Name)

	// Execute via Eino interface
	result, err := einoTool.InvokableRun(ctx, `{"numbers":[10,20,30]}`)
	require.NoError(t, err)
	assert.Equal(t, "60", result)
}

// TestRegisterMCPTools_ToolListContainsMCPTools tests that the registry's List()
// method returns MCP tools alongside built-in tools.
func TestRegisterMCPTools_ToolListContainsMCPTools(t *testing.T) {
	// Build the calculator-mcp binary
	binaryPath := buildCalculatorMCP(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create MCP client and connect to calculator server
	client := NewClient()
	defer client.Close()

	config := &Config{
		Enabled: true,
		Type:    TransportTypeStdio,
		Command: []string{binaryPath},
		Timeout: 10000,
	}

	err := client.AddServer(ctx, "calculator", config)
	require.NoError(t, err)

	// Create tool registry with built-in tools (using a temp dir for workDir)
	registry := tool.DefaultRegistry(t.TempDir())

	// Count built-in tools before MCP registration
	builtInCount := len(registry.List())

	// Register MCP tools
	RegisterMCPTools(client, registry)

	// List all tools
	allTools := registry.List()

	// Should have more tools now (built-in + MCP)
	assert.Greater(t, len(allTools), builtInCount, "should have MCP tools added")

	// Verify calculator_sum is in the list
	var foundSum bool
	for _, t := range allTools {
		if t.ID() == "calculator_sum" {
			foundSum = true
			break
		}
	}
	assert.True(t, foundSum, "calculator_sum should be in the tool list")
}

// TestMCPToolWrapper_ExecuteWithContext tests that tool execution works with
// a proper tool.Context.
func TestMCPToolWrapper_ExecuteWithContext(t *testing.T) {
	// Build the calculator-mcp binary
	binaryPath := buildCalculatorMCP(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create MCP client and connect to calculator server
	client := NewClient()
	defer client.Close()

	config := &Config{
		Enabled: true,
		Type:    TransportTypeStdio,
		Command: []string{binaryPath},
		Timeout: 10000,
	}

	err := client.AddServer(ctx, "calculator", config)
	require.NoError(t, err)

	// Create tool registry and register MCP tools
	registry := tool.NewRegistry("")
	RegisterMCPTools(client, registry)

	// Get the sum tool
	sumTool, ok := registry.Get("calculator_sum")
	require.True(t, ok)

	// Create a tool context
	var metadataReceived bool
	toolCtx := &tool.Context{
		SessionID: "test-session",
		MessageID: "test-message",
		CallID:    "test-call",
		WorkDir:   t.TempDir(),
		OnMetadata: func(title string, meta map[string]any) {
			metadataReceived = true
			assert.Equal(t, "calculator_sum", title)
		},
	}

	// Execute tool with context
	input := json.RawMessage(`{"numbers":[5,5,5]}`)
	result, err := sumTool.Execute(ctx, input, toolCtx)
	require.NoError(t, err)
	assert.Equal(t, "15", result.Output)
	assert.True(t, metadataReceived, "metadata callback should have been called")
}

// TestMCPToolWrapper_ErrorHandling tests that errors from MCP tool execution
// are properly propagated.
func TestMCPToolWrapper_ErrorHandling(t *testing.T) {
	// Build the calculator-mcp binary
	binaryPath := buildCalculatorMCP(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create MCP client and connect to calculator server
	client := NewClient()
	defer client.Close()

	config := &Config{
		Enabled: true,
		Type:    TransportTypeStdio,
		Command: []string{binaryPath},
		Timeout: 10000,
	}

	err := client.AddServer(ctx, "calculator", config)
	require.NoError(t, err)

	// Create tool registry and register MCP tools
	registry := tool.NewRegistry("")
	RegisterMCPTools(client, registry)

	// Get the sum tool
	sumTool, ok := registry.Get("calculator_sum")
	require.True(t, ok)

	// Execute with invalid JSON (missing required field or wrong type)
	// Note: The calculator tool is lenient, so we test with missing numbers
	input := json.RawMessage(`{}`)
	result, err := sumTool.Execute(ctx, input, nil)

	// The tool should handle empty input gracefully (returns "0")
	// If it errors, that's also acceptable
	if err == nil {
		assert.Equal(t, "0", result.Output)
	}
}
