// Package mcp provides Model Context Protocol (MCP) client functionality.
package mcp

import "encoding/json"

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

// Tool represents an MCP tool.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// Resource represents an MCP resource.
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// Prompt represents an MCP prompt.
type Prompt struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Arguments   []PromptArgument  `json:"arguments,omitempty"`
}

// PromptArgument represents a prompt argument.
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
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

// ServerCapabilities represents server capabilities.
type ServerCapabilities struct {
	Tools     *ToolCapability     `json:"tools,omitempty"`
	Resources *ResourceCapability `json:"resources,omitempty"`
	Prompts   *PromptCapability   `json:"prompts,omitempty"`
}

// ToolCapability represents tool capabilities.
type ToolCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourceCapability represents resource capabilities.
type ResourceCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptCapability represents prompt capabilities.
type PromptCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ClientInfo represents client information.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientCapabilities represents client capabilities.
type ClientCapabilities struct {
	Roots   *RootsCapability   `json:"roots,omitempty"`
	Sampling *SamplingCapability `json:"sampling,omitempty"`
}

// RootsCapability represents roots capabilities.
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingCapability represents sampling capabilities.
type SamplingCapability struct{}

// InitializeRequest represents an initialize request.
type InitializeRequest struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
}

// InitializeResponse represents an initialize response.
type InitializeResponse struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

// ListToolsResponse represents a tools/list response.
type ListToolsResponse struct {
	Tools []Tool `json:"tools"`
}

// ListResourcesResponse represents a resources/list response.
type ListResourcesResponse struct {
	Resources []Resource `json:"resources"`
}

// ListPromptsResponse represents a prompts/list response.
type ListPromptsResponse struct {
	Prompts []Prompt `json:"prompts"`
}

// CallToolRequest represents a tools/call request.
type CallToolRequest struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// CallToolResponse represents a tools/call response.
type CallToolResponse struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents response content.
type Content struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data,omitempty"`
}

// ReadResourceRequest represents a resources/read request.
type ReadResourceRequest struct {
	URI string `json:"uri"`
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

// GetPromptRequest represents a prompts/get request.
type GetPromptRequest struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

// GetPromptResponse represents a prompts/get response.
type GetPromptResponse struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// PromptMessage represents a prompt message.
type PromptMessage struct {
	Role    string  `json:"role"`
	Content Content `json:"content"`
}

// JSONRPCRequest represents a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ProtocolVersion is the MCP protocol version.
const ProtocolVersion = "2024-11-05"
