package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Client manages MCP server connections.
type Client struct {
	mu      sync.RWMutex
	servers map[string]*mcpServer
}

// mcpServer represents a connected MCP server.
type mcpServer struct {
	name       string
	config     *Config
	transport  Transport
	tools      []Tool
	resources  []Resource
	prompts    []Prompt
	status     Status
	error      string
	serverInfo *ServerInfo
}

// NewClient creates a new MCP client.
func NewClient() *Client {
	return &Client{
		servers: make(map[string]*mcpServer),
	}
}

// AddServer adds and connects to an MCP server.
func (c *Client) AddServer(ctx context.Context, name string, config *Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already exists
	if _, ok := c.servers[name]; ok {
		return fmt.Errorf("server already exists: %s", name)
	}

	if !config.Enabled {
		c.servers[name] = &mcpServer{
			name:   name,
			config: config,
			status: StatusDisabled,
		}
		return nil
	}

	server, err := c.connectServer(ctx, name, config)
	if err != nil {
		c.servers[name] = &mcpServer{
			name:   name,
			config: config,
			status: StatusFailed,
			error:  err.Error(),
		}
		return err
	}

	c.servers[name] = server
	return nil
}

// connectServer establishes connection to an MCP server.
func (c *Client) connectServer(ctx context.Context, name string, config *Config) (*mcpServer, error) {
	var transport Transport
	var err error

	timeout := time.Duration(config.Timeout) * time.Millisecond
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch config.Type {
	case TransportTypeRemote:
		transport, err = NewHTTPTransport(config.URL, config.Headers)
	case TransportTypeLocal, TransportTypeStdio:
		transport, err = NewStdioTransport(ctx, config.Command, config.Environment)
	default:
		return nil, fmt.Errorf("unknown transport type: %s", config.Type)
	}

	if err != nil {
		return nil, err
	}

	server := &mcpServer{
		name:      name,
		config:    config,
		transport: transport,
		status:    StatusConnecting,
	}

	// Initialize and get capabilities
	if err := server.initialize(ctx); err != nil {
		transport.Close()
		return nil, err
	}

	server.status = StatusConnected
	return server, nil
}

// initialize sends the initialize request and lists tools.
func (s *mcpServer) initialize(ctx context.Context) error {
	// Initialize
	initReq := InitializeRequest{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ClientCapabilities{
			Roots: &RootsCapability{ListChanged: false},
		},
		ClientInfo: ClientInfo{
			Name:    "opencode",
			Version: "1.0.0",
		},
	}

	result, err := s.transport.Send(ctx, "initialize", initReq)
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	var initResp InitializeResponse
	if err := json.Unmarshal(result, &initResp); err != nil {
		return fmt.Errorf("failed to parse initialize response: %w", err)
	}

	s.serverInfo = &initResp.ServerInfo

	// Send initialized notification
	if err := s.transport.Notify(ctx, "notifications/initialized", nil); err != nil {
		return fmt.Errorf("initialized notification failed: %w", err)
	}

	// List tools
	if err := s.listTools(ctx); err != nil {
		// Non-fatal, tools might not be supported
		s.tools = []Tool{}
	}

	return nil
}

// listTools lists available tools from the server.
func (s *mcpServer) listTools(ctx context.Context) error {
	result, err := s.transport.Send(ctx, "tools/list", nil)
	if err != nil {
		return err
	}

	var toolsResp ListToolsResponse
	if err := json.Unmarshal(result, &toolsResp); err != nil {
		return err
	}

	s.tools = toolsResp.Tools
	return nil
}

// Tools returns all tools from all connected servers.
func (c *Client) Tools() []Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var allTools []Tool
	for name, server := range c.servers {
		if server.status != StatusConnected {
			continue
		}

		for _, tool := range server.tools {
			// Prefix tool name with server name
			prefixedTool := Tool{
				Name:        sanitizeToolName(name) + "_" + sanitizeToolName(tool.Name),
				Description: tool.Description,
				InputSchema: tool.InputSchema,
			}
			allTools = append(allTools, prefixedTool)
		}
	}

	return allTools
}

// ExecuteTool executes a tool on the appropriate server.
func (c *Client) ExecuteTool(ctx context.Context, toolName string, args json.RawMessage) (string, error) {
	c.mu.RLock()

	// Find server and tool
	var targetServer *mcpServer
	var originalToolName string

	for name, server := range c.servers {
		if server.status != StatusConnected {
			continue
		}

		prefix := sanitizeToolName(name) + "_"
		if strings.HasPrefix(toolName, prefix) {
			targetServer = server
			originalToolName = strings.TrimPrefix(toolName, prefix)
			// Need to unsanitize the tool name
			for _, t := range server.tools {
				if sanitizeToolName(t.Name) == originalToolName {
					originalToolName = t.Name
					break
				}
			}
			break
		}
	}
	c.mu.RUnlock()

	if targetServer == nil {
		return "", fmt.Errorf("no server found for tool: %s", toolName)
	}

	// Execute tool
	callReq := CallToolRequest{
		Name:      originalToolName,
		Arguments: args,
	}

	result, err := targetServer.transport.Send(ctx, "tools/call", callReq)
	if err != nil {
		return "", err
	}

	var callResp CallToolResponse
	if err := json.Unmarshal(result, &callResp); err != nil {
		return string(result), nil
	}

	if callResp.IsError {
		// Extract error message from content
		for _, c := range callResp.Content {
			if c.Type == "text" {
				return "", fmt.Errorf("tool error: %s", c.Text)
			}
		}
		return "", fmt.Errorf("tool execution failed")
	}

	// Extract text content
	var output strings.Builder
	for _, c := range callResp.Content {
		if c.Type == "text" {
			output.WriteString(c.Text)
		}
	}

	return output.String(), nil
}

// ListResources lists all resources from all connected servers.
func (c *Client) ListResources(ctx context.Context) ([]Resource, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var allResources []Resource

	for name, server := range c.servers {
		if server.status != StatusConnected {
			continue
		}

		resources, err := server.listResources(ctx)
		if err != nil {
			continue // Skip servers that fail
		}

		// Prefix resource URIs with server name
		for _, r := range resources {
			prefixed := Resource{
				URI:         fmt.Sprintf("mcp://%s/%s", name, r.URI),
				Name:        r.Name,
				Description: r.Description,
				MimeType:    r.MimeType,
			}
			allResources = append(allResources, prefixed)
		}
	}

	return allResources, nil
}

func (s *mcpServer) listResources(ctx context.Context) ([]Resource, error) {
	result, err := s.transport.Send(ctx, "resources/list", nil)
	if err != nil {
		return nil, err
	}

	var resp ListResourcesResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, err
	}

	return resp.Resources, nil
}

// ReadResource reads a resource from a server.
func (c *Client) ReadResource(ctx context.Context, uri string) (*ReadResourceResponse, error) {
	// Parse the URI to find the server
	if !strings.HasPrefix(uri, "mcp://") {
		return nil, fmt.Errorf("invalid MCP URI: %s", uri)
	}

	parts := strings.SplitN(strings.TrimPrefix(uri, "mcp://"), "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid MCP URI format: %s", uri)
	}

	serverName := parts[0]
	resourceURI := parts[1]

	c.mu.RLock()
	server, ok := c.servers[serverName]
	c.mu.RUnlock()

	if !ok || server.status != StatusConnected {
		return nil, fmt.Errorf("server not connected: %s", serverName)
	}

	return server.readResource(ctx, resourceURI)
}

func (s *mcpServer) readResource(ctx context.Context, uri string) (*ReadResourceResponse, error) {
	req := ReadResourceRequest{URI: uri}

	result, err := s.transport.Send(ctx, "resources/read", req)
	if err != nil {
		return nil, err
	}

	var resp ReadResourceResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Status returns status of all MCP servers.
func (c *Client) Status() []ServerStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var status []ServerStatus
	for name, server := range c.servers {
		s := ServerStatus{
			Name:      name,
			Status:    server.status,
			ToolCount: len(server.tools),
		}
		if server.error != "" {
			s.Error = &server.error
		}
		status = append(status, s)
	}
	return status
}

// GetServer returns information about a specific server.
func (c *Client) GetServer(name string) (*ServerStatus, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	server, ok := c.servers[name]
	if !ok {
		return nil, fmt.Errorf("server not found: %s", name)
	}

	s := &ServerStatus{
		Name:      name,
		Status:    server.status,
		ToolCount: len(server.tools),
	}
	if server.error != "" {
		s.Error = &server.error
	}

	return s, nil
}

// RemoveServer removes and disconnects a server.
func (c *Client) RemoveServer(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	server, ok := c.servers[name]
	if !ok {
		return fmt.Errorf("server not found: %s", name)
	}

	if server.transport != nil {
		server.transport.Close()
	}

	delete(c.servers, name)
	return nil
}

// Close disconnects all servers.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, server := range c.servers {
		if server.transport != nil {
			server.transport.Close()
		}
	}

	c.servers = make(map[string]*mcpServer)
	return nil
}

// ServerCount returns the number of configured servers.
func (c *Client) ServerCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.servers)
}

// ConnectedCount returns the number of connected servers.
func (c *Client) ConnectedCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := 0
	for _, server := range c.servers {
		if server.status == StatusConnected {
			count++
		}
	}
	return count
}

// sanitizeToolName replaces non-alphanumeric chars with underscore.
func sanitizeToolName(name string) string {
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		} else {
			result.WriteRune('_')
		}
	}
	return result.String()
}
