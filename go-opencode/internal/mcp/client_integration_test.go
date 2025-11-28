package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/opencode-ai/opencode/pkg/mcpserver/calculator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClient_CalculatorMCP tests the MCP client by connecting to the calculator
// MCP server via stdio transport.
func TestClient_CalculatorMCP(t *testing.T) {
	// Build the calculator-mcp binary
	binaryPath := buildCalculatorMCP(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create MCP client
	client := NewClient()
	defer client.Close()

	// Add the calculator server using stdio transport
	config := &Config{
		Enabled: true,
		Type:    TransportTypeStdio,
		Command: []string{binaryPath},
		Timeout: 10000, // 10 seconds
	}

	err := client.AddServer(ctx, "calculator", config)
	require.NoError(t, err, "failed to add calculator server")

	// Verify server is connected
	status, err := client.GetServer("calculator")
	require.NoError(t, err)
	assert.Equal(t, StatusConnected, status.Status, "server should be connected")

	// List tools and verify the sum tool exists
	tools := client.Tools()
	require.NotEmpty(t, tools, "expected at least one tool")

	var sumToolFound bool
	var sumToolName string
	for _, tool := range tools {
		// Tool name is prefixed with server name: calculator_sum
		if tool.Name == "calculator_sum" {
			sumToolFound = true
			sumToolName = tool.Name
			assert.Contains(t, tool.Description, "sum", "tool description should mention sum")
			break
		}
	}
	require.True(t, sumToolFound, "sum tool should be registered, got tools: %v", toolNames(tools))

	// Test cases for the sum tool
	tests := []struct {
		name     string
		numbers  []float64
		expected string
	}{
		{
			name:     "sum of positive numbers",
			numbers:  []float64{1, 2, 3, 4, 5},
			expected: "15",
		},
		{
			name:     "sum of negative numbers",
			numbers:  []float64{-1, -2, -3},
			expected: "-6",
		},
		{
			name:     "sum of mixed numbers",
			numbers:  []float64{10, -5, 3.5, -2.5},
			expected: "6",
		},
		{
			name:     "sum of empty array",
			numbers:  []float64{},
			expected: "0",
		},
		{
			name:     "sum of single number",
			numbers:  []float64{42},
			expected: "42",
		},
		{
			name:     "sum with decimals",
			numbers:  []float64{1.1, 2.2, 3.3},
			expected: "6.6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build arguments JSON
			args, err := json.Marshal(map[string]any{
				"numbers": tt.numbers,
			})
			require.NoError(t, err)

			// Execute the tool
			result, err := client.ExecuteTool(ctx, sumToolName, args)
			require.NoError(t, err, "failed to execute sum tool")
			assert.Equal(t, tt.expected, result, "sum result mismatch")
		})
	}
}

// buildCalculatorMCP builds the calculator-mcp binary and returns its path.
func buildCalculatorMCP(t *testing.T) string {
	t.Helper()

	// Create temp directory for binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "calculator-mcp")

	// Build the binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/calculator-mcp")
	cmd.Dir = getProjectRoot(t)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	require.NoError(t, err, "failed to build calculator-mcp binary")

	return binaryPath
}

// getProjectRoot returns the project root directory.
func getProjectRoot(t *testing.T) string {
	t.Helper()

	// Start from current directory and walk up to find go.mod
	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}

// toolNames returns the names of all tools for debugging.
func toolNames(tools []Tool) []string {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name
	}
	return names
}

// TestClient_CalculatorMCP_SSE tests the MCP client by connecting to the calculator
// MCP server via SSE transport.
func TestClient_CalculatorMCP_SSE(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Find an available port
	port := getFreePort(t)
	addr := fmt.Sprintf("localhost:%d", port)
	sseURL := fmt.Sprintf("http://%s/sse", addr)

	// Create the calculator MCP server
	mcpServer := calculator.NewServer()

	// Create SSE server
	sseServer := server.NewSSEServer(mcpServer,
		server.WithBaseURL(fmt.Sprintf("http://%s", addr)),
	)

	// Start SSE server in background
	go func() {
		if err := sseServer.Start(addr); err != nil {
			t.Logf("SSE server stopped: %v", err)
		}
	}()

	// Wait for server to be ready
	waitForServer(t, addr, 5*time.Second)

	// Ensure server is shut down at the end
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		sseServer.Shutdown(shutdownCtx)
	}()

	// Create MCP client
	client := NewClient()
	defer client.Close()

	// Add the calculator server using SSE transport
	config := &Config{
		Enabled: true,
		Type:    TransportTypeRemote,
		URL:     sseURL,
		Timeout: 10000, // 10 seconds
	}

	err := client.AddServer(ctx, "calculator-sse", config)
	require.NoError(t, err, "failed to add calculator SSE server")

	// Verify server is connected
	status, err := client.GetServer("calculator-sse")
	require.NoError(t, err)
	assert.Equal(t, StatusConnected, status.Status, "server should be connected")

	// List tools and verify the sum tool exists
	tools := client.Tools()
	require.NotEmpty(t, tools, "expected at least one tool")

	var sumToolFound bool
	var sumToolName string
	for _, tool := range tools {
		// Tool name is prefixed with server name: calculator_sse_sum
		if tool.Name == "calculator_sse_sum" {
			sumToolFound = true
			sumToolName = tool.Name
			assert.Contains(t, tool.Description, "sum", "tool description should mention sum")
			break
		}
	}
	require.True(t, sumToolFound, "sum tool should be registered, got tools: %v", toolNames(tools))

	// Test cases for the sum tool
	tests := []struct {
		name     string
		numbers  []float64
		expected string
	}{
		{
			name:     "sum of positive numbers",
			numbers:  []float64{1, 2, 3, 4, 5},
			expected: "15",
		},
		{
			name:     "sum of negative numbers",
			numbers:  []float64{-1, -2, -3},
			expected: "-6",
		},
		{
			name:     "sum of mixed numbers",
			numbers:  []float64{10, -5, 3.5, -2.5},
			expected: "6",
		},
		{
			name:     "sum of empty array",
			numbers:  []float64{},
			expected: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build arguments JSON
			args, err := json.Marshal(map[string]any{
				"numbers": tt.numbers,
			})
			require.NoError(t, err)

			// Execute the tool
			result, err := client.ExecuteTool(ctx, sumToolName, args)
			require.NoError(t, err, "failed to execute sum tool")
			assert.Equal(t, tt.expected, result, "sum result mismatch")
		})
	}
}

// getFreePort returns an available TCP port.
func getFreePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

// waitForServer waits until the server is accepting connections.
func waitForServer(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("server did not start within %v", timeout)
}
