// Package mcp provides Model Context Protocol (MCP) client functionality
// using the official MCP Go SDK.
package mcp

import (
	"encoding/json"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Config defines MCP server configuration.
type Config struct {
	Enabled     bool              `json:"enabled"`
	Type        TransportType     `json:"type"`
	URL         string            `json:"url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Command     []string          `json:"command,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Timeout     int               `json:"timeout,omitempty"` // milliseconds
}

// TransportType represents the type of MCP transport.
type TransportType string

const (
	TransportTypeRemote TransportType = "remote"
	TransportTypeLocal  TransportType = "local"
	TransportTypeStdio  TransportType = "stdio"
)

// Tool represents an MCP tool - wrapping SDK type with JSON marshaling support.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// FromSDKTool converts an SDK tool to our Tool type.
func FromSDKTool(t *sdkmcp.Tool) Tool {
	var schema json.RawMessage
	if t.InputSchema != nil {
		schema, _ = json.Marshal(t.InputSchema)
	}
	return Tool{
		Name:        t.Name,
		Description: t.Description,
		InputSchema: schema,
	}
}

// Resource represents an MCP resource.
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// FromSDKResource converts an SDK resource to our Resource type.
func FromSDKResource(r *sdkmcp.Resource) Resource {
	return Resource{
		URI:         r.URI,
		Name:        r.Name,
		Description: r.Description,
		MimeType:    r.MIMEType,
	}
}

// Prompt represents an MCP prompt.
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument represents a prompt argument.
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// FromSDKPrompt converts an SDK prompt to our Prompt type.
func FromSDKPrompt(p *sdkmcp.Prompt) Prompt {
	args := make([]PromptArgument, len(p.Arguments))
	for i, a := range p.Arguments {
		args[i] = PromptArgument{
			Name:        a.Name,
			Description: a.Description,
			Required:    a.Required,
		}
	}
	return Prompt{
		Name:        p.Name,
		Description: p.Description,
		Arguments:   args,
	}
}

// ServerStatus represents the status of an MCP server.
type ServerStatus struct {
	Name      string  `json:"name"`
	Status    Status  `json:"status"`
	ToolCount int     `json:"toolCount"`
	Error     *string `json:"error,omitempty"`
}

// Status represents the connection status.
type Status string

const (
	StatusConnected    Status = "connected"
	StatusDisabled     Status = "disabled"
	StatusFailed       Status = "failed"
	StatusConnecting   Status = "connecting"
	StatusDisconnected Status = "disconnected"
)

// ServerInfo represents information about an MCP server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Content represents response content.
type Content struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data,omitempty"`
}

// ReadResourceResponse represents a resources/read response.
type ReadResourceResponse struct {
	Contents []ResourceContent `json:"contents"`
}

// ResourceContent represents resource content.
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

// ProtocolVersion is the MCP protocol version.
const ProtocolVersion = "2024-11-05"
