package calculator

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculatorServer_Sum(t *testing.T) {
	// Create the calculator server
	server := NewServer()

	// Get the sum tool
	sumTool := server.GetTool("sum")
	require.NotNil(t, sumTool, "sum tool should exist")

	tests := []struct {
		name     string
		numbers  []float64
		expected float64
	}{
		{
			name:     "sum of positive numbers",
			numbers:  []float64{1, 2, 3, 4, 5},
			expected: 15,
		},
		{
			name:     "sum of negative numbers",
			numbers:  []float64{-1, -2, -3},
			expected: -6,
		},
		{
			name:     "sum of mixed numbers",
			numbers:  []float64{10, -5, 3.5, -2.5},
			expected: 6,
		},
		{
			name:     "sum of empty array",
			numbers:  []float64{},
			expected: 0,
		},
		{
			name:     "sum of single number",
			numbers:  []float64{42},
			expected: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the request with numbers argument
			request := mcp.CallToolRequest{}
			request.Params.Name = "sum"
			request.Params.Arguments = map[string]any{
				"numbers": tt.numbers,
			}

			// Call the handler
			ctx := context.Background()
			result, err := sumTool.Handler(ctx, request)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.False(t, result.IsError, "result should not be an error")

			// Extract the text result
			require.Len(t, result.Content, 1)
			textContent, ok := result.Content[0].(mcp.TextContent)
			require.True(t, ok, "content should be text")
			assert.Contains(t, textContent.Text, formatFloat(tt.expected))
		})
	}
}

func TestCalculatorServer_HasSumTool(t *testing.T) {
	server := NewServer()

	// Verify the sum tool exists
	sumTool := server.GetTool("sum")
	require.NotNil(t, sumTool, "sum tool should exist")
	assert.Equal(t, "sum", sumTool.Tool.Name)
	assert.Contains(t, sumTool.Tool.Description, "sum")
}
