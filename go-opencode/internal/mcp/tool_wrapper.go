// Package mcp provides Model Context Protocol (MCP) client functionality.
package mcp

import (
	"context"
	"encoding/json"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/opencode-ai/opencode/internal/tool"
)

// MCPToolWrapper wraps an MCP tool to implement the tool.Tool interface.
// This allows MCP tools to be registered in the standard tool registry
// and used seamlessly in the agentic loop.
type MCPToolWrapper struct {
	mcpTool Tool    // The MCP tool metadata (already has prefixed name from client.Tools())
	client  *Client // Reference to MCP client for execution
}

// NewMCPToolWrapper creates a wrapper for an MCP tool.
func NewMCPToolWrapper(mcpTool Tool, client *Client) *MCPToolWrapper {
	return &MCPToolWrapper{
		mcpTool: mcpTool,
		client:  client,
	}
}

// ID returns the prefixed tool name (e.g., "serverName_toolName").
func (w *MCPToolWrapper) ID() string {
	return w.mcpTool.Name
}

// Description returns the tool description.
func (w *MCPToolWrapper) Description() string {
	return w.mcpTool.Description
}

// Parameters returns the JSON Schema for tool parameters.
func (w *MCPToolWrapper) Parameters() json.RawMessage {
	return w.mcpTool.InputSchema
}

// Execute executes the tool via MCP client.
func (w *MCPToolWrapper) Execute(ctx context.Context, input json.RawMessage, toolCtx *tool.Context) (*tool.Result, error) {
	// Execute tool through MCP client
	output, err := w.client.ExecuteTool(ctx, w.mcpTool.Name, input)
	if err != nil {
		return nil, err
	}

	// Update metadata if context is available
	if toolCtx != nil {
		toolCtx.SetMetadata(w.mcpTool.Name, map[string]any{
			"type":   "mcp",
			"tool":   w.mcpTool.Name,
			"output": output,
		})
	}

	return &tool.Result{
		Title:  w.mcpTool.Name,
		Output: output,
	}, nil
}

// EinoTool returns an Eino-compatible tool implementation.
func (w *MCPToolWrapper) EinoTool() einotool.InvokableTool {
	return &mcpEinoWrapper{wrapper: w}
}

// mcpEinoWrapper implements Eino's InvokableTool interface for MCP tools.
type mcpEinoWrapper struct {
	wrapper *MCPToolWrapper
}

// Info returns the tool information.
func (e *mcpEinoWrapper) Info(ctx context.Context) (*schema.ToolInfo, error) {
	params := parseInputSchemaToParams(e.wrapper.mcpTool.InputSchema)
	return &schema.ToolInfo{
		Name:        e.wrapper.ID(),
		Desc:        e.wrapper.Description(),
		ParamsOneOf: schema.NewParamsOneOfByParams(params),
	}, nil
}

// InvokableRun executes the tool.
func (e *mcpEinoWrapper) InvokableRun(ctx context.Context, argsJSON string, opts ...einotool.Option) (string, error) {
	result, err := e.wrapper.Execute(ctx, json.RawMessage(argsJSON), nil)
	if err != nil {
		return "", err
	}
	return result.Output, nil
}

// parseInputSchemaToParams converts JSON Schema to Eino ParameterInfo.
func parseInputSchemaToParams(schemaJSON json.RawMessage) map[string]*schema.ParameterInfo {
	var jsonSchema struct {
		Properties map[string]struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"properties"`
		Required []string `json:"required"`
	}

	if err := json.Unmarshal(schemaJSON, &jsonSchema); err != nil {
		return nil
	}

	requiredSet := make(map[string]bool)
	for _, r := range jsonSchema.Required {
		requiredSet[r] = true
	}

	params := make(map[string]*schema.ParameterInfo)
	for name, prop := range jsonSchema.Properties {
		paramType := schema.String
		switch prop.Type {
		case "integer":
			paramType = schema.Integer
		case "number":
			paramType = schema.Number
		case "boolean":
			paramType = schema.Boolean
		case "array":
			paramType = schema.Array
		case "object":
			paramType = schema.Object
		}

		params[name] = &schema.ParameterInfo{
			Type:     paramType,
			Desc:     prop.Description,
			Required: requiredSet[name],
		}
	}

	return params
}

// RegisterMCPTools registers all MCP tools from the client to a tool registry.
// This function fetches all available tools from connected MCP servers
// and wraps them to implement the tool.Tool interface.
func RegisterMCPTools(client *Client, registry *tool.Registry) {
	if client == nil || registry == nil {
		return
	}

	tools := client.Tools()
	for _, mcpTool := range tools {
		wrapper := NewMCPToolWrapper(mcpTool, client)
		registry.Register(wrapper)
	}
}
