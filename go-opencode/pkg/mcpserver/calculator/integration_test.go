package calculator

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCalculatorServer_MCPClient tests the calculator server using the
// modelcontextprotocol go-sdk client, verifying end-to-end MCP communication.
func TestCalculatorServer_MCPClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create the calculator MCP server
	mcpServer := NewServer()
	stdioServer := server.NewStdioServer(mcpServer)

	// Create pipes for bidirectional communication
	// serverReader <- clientWriter (client sends to server)
	// clientReader <- serverWriter (server sends to client)
	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	// Start the server in a goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- stdioServer.Listen(ctx, serverReader, serverWriter)
	}()

	// Create the MCP client
	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Use IOTransport with our pipes
	transport := &sdkmcp.IOTransport{
		Reader: clientReader,
		Writer: clientWriter,
	}

	// Connect the client to the server
	session, err := client.Connect(ctx, transport, nil)
	require.NoError(t, err, "failed to connect client to server")
	defer session.Close()

	// List tools and verify the sum tool exists
	listResult, err := session.ListTools(ctx, nil)
	require.NoError(t, err, "failed to list tools")
	require.NotEmpty(t, listResult.Tools, "expected at least one tool")

	var sumToolFound bool
	for _, tool := range listResult.Tools {
		if tool.Name == "sum" {
			sumToolFound = true
			assert.Contains(t, tool.Description, "sum", "tool description should mention sum")
			break
		}
	}
	require.True(t, sumToolFound, "sum tool should be registered")

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
			// Call the sum tool through the MCP client
			result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
				Name: "sum",
				Arguments: map[string]any{
					"numbers": tt.numbers,
				},
			})
			require.NoError(t, err, "failed to call sum tool")
			require.False(t, result.IsError, "tool call should not return an error")
			require.NotEmpty(t, result.Content, "result should have content")

			// Extract the text result
			textContent, ok := result.Content[0].(*sdkmcp.TextContent)
			require.True(t, ok, "content should be TextContent")
			assert.Equal(t, tt.expected, textContent.Text, "sum result mismatch")
		})
	}

	// Clean up
	cancel()
	clientWriter.Close()
	serverWriter.Close()
}

// TestCalculatorServer_SSE tests the calculator server using SSE transport
// with the modelcontextprotocol go-sdk client.
func TestCalculatorServer_SSE(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Find an available port
	port := getFreePort(t)
	addr := fmt.Sprintf("localhost:%d", port)
	sseURL := fmt.Sprintf("http://%s/sse", addr)

	// Create the calculator MCP server
	mcpServer := NewServer()

	// Create SSE server from mcp-go
	sseServer := server.NewSSEServer(mcpServer,
		server.WithBaseURL(fmt.Sprintf("http://%s", addr)),
	)

	// Start SSE server in background
	go func() {
		if err := sseServer.Start(addr); err != nil {
			t.Logf("SSE server error: %v", err)
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

	// Create the MCP client using SSEClientTransport
	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "test-client-sse",
		Version: "1.0.0",
	}, nil)

	transport := &sdkmcp.SSEClientTransport{
		Endpoint: sseURL,
	}

	// Connect the client to the server
	session, err := client.Connect(ctx, transport, nil)
	require.NoError(t, err, "failed to connect client to SSE server")
	defer session.Close()

	// List tools and verify the sum tool exists
	listResult, err := session.ListTools(ctx, nil)
	require.NoError(t, err, "failed to list tools")
	require.NotEmpty(t, listResult.Tools, "expected at least one tool")

	var sumToolFound bool
	for _, tool := range listResult.Tools {
		if tool.Name == "sum" {
			sumToolFound = true
			assert.Contains(t, tool.Description, "sum", "tool description should mention sum")
			break
		}
	}
	require.True(t, sumToolFound, "sum tool should be registered")

	// Test the sum tool
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
			result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
				Name: "sum",
				Arguments: map[string]any{
					"numbers": tt.numbers,
				},
			})
			require.NoError(t, err, "failed to call sum tool")
			require.False(t, result.IsError, "tool call should not return an error")
			require.NotEmpty(t, result.Content, "result should have content")

			textContent, ok := result.Content[0].(*sdkmcp.TextContent)
			require.True(t, ok, "content should be TextContent")
			assert.Equal(t, tt.expected, textContent.Text, "sum result mismatch")
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
