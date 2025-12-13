// Package calculator provides an MCP server with a calculator tool.
package calculator

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates a new MCP server with calculator tools.
func NewServer() *server.MCPServer {
	s := server.NewMCPServer(
		"calculator",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Define the sum tool that accepts an array of numbers
	sumTool := mcp.NewTool("sum",
		mcp.WithDescription("Calculates the sum of an array of numbers"),
		mcp.WithArray("numbers",
			mcp.Required(),
			mcp.Description("Array of numbers to sum"),
			mcp.Items(map[string]any{
				"type": "number",
			}),
		),
	)

	// Add the tool with its handler
	s.AddTool(sumTool, sumHandler)

	return s
}

// sumHandler handles the sum tool call.
func sumHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract the numbers argument
	args := request.GetArguments()
	numbersArg, ok := args["numbers"]
	if !ok {
		return mcp.NewToolResultError("numbers argument is required"), nil
	}

	// Convert to slice of float64
	numbers, err := toFloat64Slice(numbersArg)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid numbers: %v", err)), nil
	}

	// Calculate sum
	var sum float64
	for _, n := range numbers {
		sum += n
	}

	return mcp.NewToolResultText(formatFloat(sum)), nil
}

// toFloat64Slice converts an interface{} to []float64.
func toFloat64Slice(v any) ([]float64, error) {
	switch arr := v.(type) {
	case []any:
		result := make([]float64, len(arr))
		for i, elem := range arr {
			switch n := elem.(type) {
			case float64:
				result[i] = n
			case int:
				result[i] = float64(n)
			case int64:
				result[i] = float64(n)
			default:
				return nil, fmt.Errorf("element %d is not a number: %T", i, elem)
			}
		}
		return result, nil
	case []float64:
		return arr, nil
	case []int:
		result := make([]float64, len(arr))
		for i, n := range arr {
			result[i] = float64(n)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("expected array, got %T", v)
	}
}

// formatFloat formats a float64 as a string, removing trailing zeros.
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
