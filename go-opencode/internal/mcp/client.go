package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Client manages MCP server connections using the official MCP SDK.
type Client struct {
	mu        sync.RWMutex
	servers   map[string]*mcpServer
	sdkClient *sdkmcp.Client
}

// mcpServer represents a connected MCP server.
type mcpServer struct {
	name       string
	config     *Config
	session    *sdkmcp.ClientSession
	tools      []Tool
	resources  []Resource
	prompts    []Prompt
	status     Status
	error      string
	serverInfo *ServerInfo
}

// NewClient creates a new MCP client.
func NewClient() *Client {
	sdkClient := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "opencode",
		Version: "1.0.0",
	}, nil)

	return &Client{
		servers:   make(map[string]*mcpServer),
		sdkClient: sdkClient,
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

// connectServer establishes connection to an MCP server using the SDK.
func (c *Client) connectServer(ctx context.Context, name string, config *Config) (*mcpServer, error) {
	timeout := time.Duration(config.Timeout) * time.Millisecond
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var transport sdkmcp.Transport

	switch config.Type {
	case TransportTypeRemote:
		// Use SSE transport for remote HTTP servers
		httpClient := &http.Client{Timeout: timeout}
		transport = &sdkmcp.SSEClientTransport{
			Endpoint:   config.URL,
			HTTPClient: httpClient,
		}

	case TransportTypeLocal, TransportTypeStdio:
		if len(config.Command) == 0 {
			return nil, fmt.Errorf("empty command")
		}

		cmd := exec.Command(config.Command[0], config.Command[1:]...)

		// Set environment
		cmd.Env = os.Environ()
		for k, v := range config.Environment {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}

		transport = &sdkmcp.CommandTransport{Command: cmd}

	default:
		return nil, fmt.Errorf("unknown transport type: %s", config.Type)
	}

	server := &mcpServer{
		name:   name,
		config: config,
		status: StatusConnecting,
	}

	// Connect using the SDK client
	session, err := c.sdkClient.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	server.session = session

	// Get server info from initialization result
	initResult := session.InitializeResult()
	if initResult != nil {
		server.serverInfo = &ServerInfo{
			Name:    initResult.ServerInfo.Name,
			Version: initResult.ServerInfo.Version,
		}
	}

	// List tools
	if err := server.listTools(ctx); err != nil {
		// Non-fatal, tools might not be supported
		server.tools = []Tool{}
	}

	server.status = StatusConnected
	return server, nil
}

// listTools lists available tools from the server using the SDK.
func (s *mcpServer) listTools(ctx context.Context) error {
	if s.session == nil {
		return fmt.Errorf("not connected")
	}

	result, err := s.session.ListTools(ctx, nil)
	if err != nil {
		return err
	}

	s.tools = make([]Tool, len(result.Tools))
	for i, t := range result.Tools {
		s.tools[i] = FromSDKTool(t)
	}

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

	if targetServer.session == nil {
		return "", fmt.Errorf("server not connected: %s", targetServer.name)
	}

	// Parse arguments into a map
	var argsMap map[string]any
	if len(args) > 0 {
		if err := json.Unmarshal(args, &argsMap); err != nil {
			return "", fmt.Errorf("failed to parse arguments: %w", err)
		}
	}

	// Execute tool using SDK
	params := &sdkmcp.CallToolParams{
		Name:      originalToolName,
		Arguments: argsMap,
	}

	result, err := targetServer.session.CallTool(ctx, params)
	if err != nil {
		return "", err
	}

	if result.IsError {
		// Extract error message from content
		for _, content := range result.Content {
			if textContent, ok := content.(*sdkmcp.TextContent); ok {
				return "", fmt.Errorf("tool error: %s", textContent.Text)
			}
		}
		return "", fmt.Errorf("tool execution failed")
	}

	// Extract text content
	var output strings.Builder
	for _, content := range result.Content {
		if textContent, ok := content.(*sdkmcp.TextContent); ok {
			output.WriteString(textContent.Text)
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
		if server.status != StatusConnected || server.session == nil {
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
	if s.session == nil {
		return nil, fmt.Errorf("not connected")
	}

	result, err := s.session.ListResources(ctx, nil)
	if err != nil {
		return nil, err
	}

	resources := make([]Resource, len(result.Resources))
	for i, r := range result.Resources {
		resources[i] = FromSDKResource(r)
	}

	return resources, nil
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
	if s.session == nil {
		return nil, fmt.Errorf("not connected")
	}

	params := &sdkmcp.ReadResourceParams{URI: uri}
	result, err := s.session.ReadResource(ctx, params)
	if err != nil {
		return nil, err
	}

	resp := &ReadResourceResponse{
		Contents: make([]ResourceContent, len(result.Contents)),
	}

	for i, c := range result.Contents {
		content := ResourceContent{
			URI:      c.URI,
			MimeType: c.MIMEType,
			Text:     c.Text,
		}

		// Handle blob content
		if len(c.Blob) > 0 {
			content.Blob = string(c.Blob)
		}

		resp.Contents[i] = content
	}

	return resp, nil
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

	if server.session != nil {
		server.session.Close()
	}

	delete(c.servers, name)
	return nil
}

// Close disconnects all servers.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, server := range c.servers {
		if server.session != nil {
			server.session.Close()
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
