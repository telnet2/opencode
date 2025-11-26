// Package tool provides the tool framework for LLM tool execution.
package tool

import (
	"context"
	"encoding/json"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// Tool defines the interface for all tools.
type Tool interface {
	// ID returns the tool identifier.
	ID() string

	// Description returns the tool description.
	Description() string

	// Parameters returns the JSON Schema for tool parameters.
	Parameters() json.RawMessage

	// Execute executes the tool with the given input.
	Execute(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error)

	// EinoTool returns an Eino-compatible tool implementation.
	EinoTool() einotool.InvokableTool
}

// Context provides execution context to tools.
type Context struct {
	SessionID string
	MessageID string
	CallID    string
	Agent     string
	WorkDir   string
	AbortCh   <-chan struct{}
	Extra     map[string]any

	// Metadata callback for real-time updates
	OnMetadata func(title string, meta map[string]any)
}

// SetMetadata updates tool execution metadata.
func (c *Context) SetMetadata(title string, meta map[string]any) {
	if c.OnMetadata != nil {
		c.OnMetadata(title, meta)
	}
}

// IsAborted checks if the tool execution has been aborted.
func (c *Context) IsAborted() bool {
	select {
	case <-c.AbortCh:
		return true
	default:
		return false
	}
}

// Result represents the output of a tool execution.
type Result struct {
	Title       string            `json:"title"`
	Output      string            `json:"output"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
	Attachments []Attachment      `json:"attachments,omitempty"`
	Error       error             `json:"-"`
}

// Attachment represents a file attachment.
type Attachment struct {
	Filename  string `json:"filename"`
	MediaType string `json:"mediaType"`
	URL       string `json:"url"` // data: URL or file path
}

// BaseTool provides a base implementation for tools.
type BaseTool struct {
	id          string
	description string
	parameters  json.RawMessage
	execute     func(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error)
}

// NewBaseTool creates a new base tool.
func NewBaseTool(id, description string, params json.RawMessage, execute func(context.Context, json.RawMessage, *Context) (*Result, error)) *BaseTool {
	return &BaseTool{
		id:          id,
		description: description,
		parameters:  params,
		execute:     execute,
	}
}

func (t *BaseTool) ID() string                   { return t.id }
func (t *BaseTool) Description() string          { return t.description }
func (t *BaseTool) Parameters() json.RawMessage  { return t.parameters }

func (t *BaseTool) Execute(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error) {
	return t.execute(ctx, input, toolCtx)
}

// EinoTool returns an Eino-compatible tool implementation.
func (t *BaseTool) EinoTool() einotool.InvokableTool {
	return &einoToolWrapper{tool: t}
}

// einoToolWrapper wraps a Tool to implement Eino's InvokableTool interface.
type einoToolWrapper struct {
	tool Tool
}

// Info returns the tool information.
func (w *einoToolWrapper) Info(ctx context.Context) (*schema.ToolInfo, error) {
	params := parseJSONSchemaToParams(w.tool.Parameters())
	return &schema.ToolInfo{
		Name:        w.tool.ID(),
		Desc:        w.tool.Description(),
		ParamsOneOf: schema.NewParamsOneOfByParams(params),
	}, nil
}

// InvokableRun executes the tool.
func (w *einoToolWrapper) InvokableRun(ctx context.Context, argsJSON string, opts ...einotool.Option) (string, error) {
	toolCtx := &Context{
		WorkDir: "",
	}

	result, err := w.tool.Execute(ctx, json.RawMessage(argsJSON), toolCtx)
	if err != nil {
		return "", err
	}

	return result.Output, nil
}

// parseJSONSchemaToParams converts JSON Schema to Eino ParameterInfo.
func parseJSONSchemaToParams(schemaJSON json.RawMessage) map[string]*schema.ParameterInfo {
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
