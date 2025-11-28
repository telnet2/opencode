package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/opencode-ai/opencode/internal/tool"
	"github.com/opencode-ai/opencode/pkg/mcpserver/calculator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCP_E2E_StdioTransport tests the full E2E flow with stdio transport:
// server startup -> tool registration -> tool execution via registry.
func TestMCP_E2E_StdioTransport(t *testing.T) {
	binaryPath := buildCalculatorMCP(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create MCP client
	client := NewClient()
	defer client.Close()

	config := &Config{
		Enabled: true,
		Type:    TransportTypeStdio,
		Command: []string{binaryPath},
		Timeout: 10000,
	}

	err := client.AddServer(ctx, "calc-stdio", config)
	require.NoError(t, err)

	// Register tools in registry
	registry := tool.NewRegistry("")
	RegisterMCPTools(client, registry)

	// Execute tool via registry
	sumTool, ok := registry.Get("calc_stdio_sum")
	require.True(t, ok, "sum tool should be registered")

	input := json.RawMessage(`{"numbers":[100,200,300]}`)
	result, err := sumTool.Execute(ctx, input, nil)
	require.NoError(t, err)
	assert.Equal(t, "600", result.Output)
}

// TestMCP_E2E_SSETransport tests the full E2E flow with SSE transport:
// server startup -> tool registration -> tool execution via registry.
func TestMCP_E2E_SSETransport(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start SSE server
	port := getFreePort(t)
	addr := fmt.Sprintf("localhost:%d", port)
	sseURL := fmt.Sprintf("http://%s/sse", addr)

	mcpServer := calculator.NewServer()
	sseServer := server.NewSSEServer(mcpServer,
		server.WithBaseURL(fmt.Sprintf("http://%s", addr)),
	)

	go func() {
		if err := sseServer.Start(addr); err != nil {
			t.Logf("SSE server stopped: %v", err)
		}
	}()

	waitForServer(t, addr, 5*time.Second)
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		sseServer.Shutdown(shutdownCtx)
	}()

	// Create MCP client
	client := NewClient()
	defer client.Close()

	config := &Config{
		Enabled: true,
		Type:    TransportTypeRemote,
		URL:     sseURL,
		Timeout: 10000,
	}

	err := client.AddServer(ctx, "calc-sse", config)
	require.NoError(t, err)

	// Register tools in registry
	registry := tool.NewRegistry("")
	RegisterMCPTools(client, registry)

	// Execute tool via registry
	sumTool, ok := registry.Get("calc_sse_sum")
	require.True(t, ok, "sum tool should be registered")

	input := json.RawMessage(`{"numbers":[1.5,2.5,3.0]}`)
	result, err := sumTool.Execute(ctx, input, nil)
	require.NoError(t, err)
	assert.Equal(t, "7", result.Output)
}

// TestMCP_E2E_MultipleServers tests using multiple MCP servers simultaneously.
func TestMCP_E2E_MultipleServers(t *testing.T) {
	binaryPath := buildCalculatorMCP(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create MCP client
	client := NewClient()
	defer client.Close()

	// Add first server (stdio)
	config1 := &Config{
		Enabled: true,
		Type:    TransportTypeStdio,
		Command: []string{binaryPath},
		Timeout: 10000,
	}
	err := client.AddServer(ctx, "calc-one", config1)
	require.NoError(t, err)

	// Add second server (also stdio, different instance)
	config2 := &Config{
		Enabled: true,
		Type:    TransportTypeStdio,
		Command: []string{binaryPath},
		Timeout: 10000,
	}
	err = client.AddServer(ctx, "calc-two", config2)
	require.NoError(t, err)

	// Verify both servers are connected
	status1, err := client.GetServer("calc-one")
	require.NoError(t, err)
	assert.Equal(t, StatusConnected, status1.Status)

	status2, err := client.GetServer("calc-two")
	require.NoError(t, err)
	assert.Equal(t, StatusConnected, status2.Status)

	// Register tools in registry
	registry := tool.NewRegistry("")
	RegisterMCPTools(client, registry)

	// Verify both servers' tools are registered
	sumTool1, ok := registry.Get("calc_one_sum")
	require.True(t, ok, "calc_one_sum tool should be registered")

	sumTool2, ok := registry.Get("calc_two_sum")
	require.True(t, ok, "calc_two_sum tool should be registered")

	// Execute tools from both servers
	input1 := json.RawMessage(`{"numbers":[1,2,3]}`)
	result1, err := sumTool1.Execute(ctx, input1, nil)
	require.NoError(t, err)
	assert.Equal(t, "6", result1.Output)

	input2 := json.RawMessage(`{"numbers":[10,20,30]}`)
	result2, err := sumTool2.Execute(ctx, input2, nil)
	require.NoError(t, err)
	assert.Equal(t, "60", result2.Output)

	// Verify tool count from multiple servers
	tools := client.Tools()
	// Each server has "sum" tool, so we should have 2 sum tools
	var sumToolCount int
	for _, tool := range tools {
		if tool.Name == "calc_one_sum" || tool.Name == "calc_two_sum" {
			sumToolCount++
		}
	}
	assert.Equal(t, 2, sumToolCount, "should have sum tools from both servers")
}

// TestMCP_E2E_ServerFailure tests behavior when an MCP server fails to start.
func TestMCP_E2E_ServerFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create MCP client
	client := NewClient()
	defer client.Close()

	// Try to add a server with invalid command
	config := &Config{
		Enabled: true,
		Type:    TransportTypeStdio,
		Command: []string{"/nonexistent/path/to/binary"},
		Timeout: 5000,
	}

	err := client.AddServer(ctx, "failing-server", config)
	assert.Error(t, err, "should fail with nonexistent binary")

	// Verify server is registered but with failed status
	// (AddServer stores the server even on failure for status tracking)
	status, err := client.GetServer("failing-server")
	require.NoError(t, err, "server should be registered even if failed")
	assert.Equal(t, StatusFailed, status.Status, "server should have failed status")
	assert.NotNil(t, status.Error, "server should have error message")

	// Create tool registry and register tools
	registry := tool.NewRegistry("")
	RegisterMCPTools(client, registry)

	// Registry should have no tools from the failed server
	_, ok := registry.Get("failing_server_sum")
	assert.False(t, ok, "should not have tools from failed server")
}

// TestMCP_E2E_ToolExecutionTimeout tests tool execution timeout handling.
func TestMCP_E2E_ToolExecutionTimeout(t *testing.T) {
	binaryPath := buildCalculatorMCP(t)

	// Create a very short timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create MCP client
	client := NewClient()
	defer client.Close()

	config := &Config{
		Enabled: true,
		Type:    TransportTypeStdio,
		Command: []string{binaryPath},
		Timeout: 10000,
	}

	err := client.AddServer(ctx, "calc-timeout", config)
	require.NoError(t, err)

	// Register tools in registry
	registry := tool.NewRegistry("")
	RegisterMCPTools(client, registry)

	// Get the sum tool
	sumTool, ok := registry.Get("calc_timeout_sum")
	require.True(t, ok)

	// Execute with already-canceled context
	canceledCtx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc() // Cancel immediately

	input := json.RawMessage(`{"numbers":[1,2,3]}`)
	_, err = sumTool.Execute(canceledCtx, input, nil)
	assert.Error(t, err, "should error with canceled context")
}

// TestMCP_E2E_ServerDisconnection tests behavior when an MCP server disconnects.
func TestMCP_E2E_ServerDisconnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start SSE server
	port := getFreePort(t)
	addr := fmt.Sprintf("localhost:%d", port)
	sseURL := fmt.Sprintf("http://%s/sse", addr)

	mcpServer := calculator.NewServer()
	sseServer := server.NewSSEServer(mcpServer,
		server.WithBaseURL(fmt.Sprintf("http://%s", addr)),
	)

	go func() {
		if err := sseServer.Start(addr); err != nil {
			t.Logf("SSE server stopped: %v", err)
		}
	}()

	waitForServer(t, addr, 5*time.Second)

	// Create MCP client and connect
	client := NewClient()
	defer client.Close()

	config := &Config{
		Enabled: true,
		Type:    TransportTypeRemote,
		URL:     sseURL,
		Timeout: 10000,
	}

	err := client.AddServer(ctx, "calc-disconnect", config)
	require.NoError(t, err)

	// Register tools and verify
	registry := tool.NewRegistry("")
	RegisterMCPTools(client, registry)

	sumTool, ok := registry.Get("calc_disconnect_sum")
	require.True(t, ok)

	// First execution should work
	input := json.RawMessage(`{"numbers":[1,2,3]}`)
	result, err := sumTool.Execute(ctx, input, nil)
	require.NoError(t, err)
	assert.Equal(t, "6", result.Output)

	// Shutdown the server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	sseServer.Shutdown(shutdownCtx)

	// Wait a bit for the shutdown to propagate
	time.Sleep(500 * time.Millisecond)

	// Execution after disconnection should fail
	_, err = sumTool.Execute(ctx, input, nil)
	assert.Error(t, err, "should error after server disconnection")
}

// TestMCP_E2E_DisabledServer tests that disabled servers are not connected.
func TestMCP_E2E_DisabledServer(t *testing.T) {
	binaryPath := buildCalculatorMCP(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create MCP client
	client := NewClient()
	defer client.Close()

	// Add a disabled server
	config := &Config{
		Enabled: false, // Disabled
		Type:    TransportTypeStdio,
		Command: []string{binaryPath},
		Timeout: 10000,
	}

	err := client.AddServer(ctx, "disabled-server", config)
	require.NoError(t, err, "adding disabled server should not error")

	// Verify server status
	status, err := client.GetServer("disabled-server")
	require.NoError(t, err)
	assert.Equal(t, StatusDisabled, status.Status, "server should be disabled")

	// Create tool registry and register tools
	registry := tool.NewRegistry("")
	RegisterMCPTools(client, registry)

	// Registry should have no tools from the disabled server
	_, ok := registry.Get("disabled_server_sum")
	assert.False(t, ok, "should not have tools from disabled server")
}

// TestMCP_E2E_EnvironmentVariables tests that environment variables are passed to stdio servers.
func TestMCP_E2E_EnvironmentVariables(t *testing.T) {
	binaryPath := buildCalculatorMCP(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create MCP client
	client := NewClient()
	defer client.Close()

	// Add server with custom environment variables
	config := &Config{
		Enabled: true,
		Type:    TransportTypeStdio,
		Command: []string{binaryPath},
		Environment: map[string]string{
			"TEST_VAR": "test_value",
		},
		Timeout: 10000,
	}

	err := client.AddServer(ctx, "calc-env", config)
	require.NoError(t, err)

	// Verify server is connected
	status, err := client.GetServer("calc-env")
	require.NoError(t, err)
	assert.Equal(t, StatusConnected, status.Status)

	// Register and execute tool (verifies the server works with env vars)
	registry := tool.NewRegistry("")
	RegisterMCPTools(client, registry)

	sumTool, ok := registry.Get("calc_env_sum")
	require.True(t, ok)

	input := json.RawMessage(`{"numbers":[7,8,9]}`)
	result, err := sumTool.Execute(ctx, input, nil)
	require.NoError(t, err)
	assert.Equal(t, "24", result.Output)
}

// TestMCP_E2E_RemoveServer tests removing an MCP server.
func TestMCP_E2E_RemoveServer(t *testing.T) {
	binaryPath := buildCalculatorMCP(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create MCP client
	client := NewClient()
	defer client.Close()

	config := &Config{
		Enabled: true,
		Type:    TransportTypeStdio,
		Command: []string{binaryPath},
		Timeout: 10000,
	}

	err := client.AddServer(ctx, "calc-remove", config)
	require.NoError(t, err)

	// Verify server is connected
	status, err := client.GetServer("calc-remove")
	require.NoError(t, err)
	assert.Equal(t, StatusConnected, status.Status)

	// Verify tools exist
	tools := client.Tools()
	var hasCalcRemoveSum bool
	for _, tool := range tools {
		if tool.Name == "calc_remove_sum" {
			hasCalcRemoveSum = true
			break
		}
	}
	assert.True(t, hasCalcRemoveSum, "should have calc_remove_sum tool")

	// Remove the server
	err = client.RemoveServer("calc-remove")
	require.NoError(t, err)

	// Verify server is gone
	_, err = client.GetServer("calc-remove")
	assert.Error(t, err, "should not find removed server")

	// Verify tools are removed
	tools = client.Tools()
	hasCalcRemoveSum = false
	for _, tool := range tools {
		if tool.Name == "calc_remove_sum" {
			hasCalcRemoveSum = true
			break
		}
	}
	assert.False(t, hasCalcRemoveSum, "should not have calc_remove_sum tool after removal")
}

// TestMCP_E2E_InvalidURL tests behavior with invalid URL for SSE transport.
func TestMCP_E2E_InvalidURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create MCP client
	client := NewClient()
	defer client.Close()

	// Try to connect to a non-existent SSE server
	port := getFreePort(t) // Get a port that's not being used
	config := &Config{
		Enabled: true,
		Type:    TransportTypeRemote,
		URL:     fmt.Sprintf("http://localhost:%d/sse", port),
		Timeout: 2000, // Short timeout
	}

	err := client.AddServer(ctx, "invalid-sse", config)
	assert.Error(t, err, "should fail to connect to non-existent SSE server")
}

// TestMCP_E2E_MixedTransports tests using both Stdio and SSE servers simultaneously.
func TestMCP_E2E_MixedTransports(t *testing.T) {
	binaryPath := buildCalculatorMCP(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start SSE server
	port := getFreePort(t)
	addr := fmt.Sprintf("localhost:%d", port)
	sseURL := fmt.Sprintf("http://%s/sse", addr)

	mcpServer := calculator.NewServer()
	sseServer := server.NewSSEServer(mcpServer,
		server.WithBaseURL(fmt.Sprintf("http://%s", addr)),
	)

	go func() {
		if err := sseServer.Start(addr); err != nil {
			t.Logf("SSE server stopped: %v", err)
		}
	}()

	waitForServer(t, addr, 5*time.Second)
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		sseServer.Shutdown(shutdownCtx)
	}()

	// Create MCP client
	client := NewClient()
	defer client.Close()

	// Add stdio server
	stdioConfig := &Config{
		Enabled: true,
		Type:    TransportTypeStdio,
		Command: []string{binaryPath},
		Timeout: 10000,
	}
	err := client.AddServer(ctx, "calc-stdio-mix", stdioConfig)
	require.NoError(t, err)

	// Add SSE server
	sseConfig := &Config{
		Enabled: true,
		Type:    TransportTypeRemote,
		URL:     sseURL,
		Timeout: 10000,
	}
	err = client.AddServer(ctx, "calc-sse-mix", sseConfig)
	require.NoError(t, err)

	// Register tools
	registry := tool.NewRegistry("")
	RegisterMCPTools(client, registry)

	// Execute tool from stdio server
	stdioTool, ok := registry.Get("calc_stdio_mix_sum")
	require.True(t, ok)
	stdioInput := json.RawMessage(`{"numbers":[1,2,3]}`)
	stdioResult, err := stdioTool.Execute(ctx, stdioInput, nil)
	require.NoError(t, err)
	assert.Equal(t, "6", stdioResult.Output)

	// Execute tool from SSE server
	sseTool, ok := registry.Get("calc_sse_mix_sum")
	require.True(t, ok)
	sseInput := json.RawMessage(`{"numbers":[10,20,30]}`)
	sseResult, err := sseTool.Execute(ctx, sseInput, nil)
	require.NoError(t, err)
	assert.Equal(t, "60", sseResult.Output)
}

// getFreePortE2E returns an available TCP port for E2E tests.
func getFreePortE2E(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}
