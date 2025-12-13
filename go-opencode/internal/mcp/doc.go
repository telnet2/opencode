// Package mcp provides Model Context Protocol (MCP) client functionality for
// integrating with MCP servers using the official MCP Go SDK.
//
// The Model Context Protocol (MCP) is an open standard that enables secure
// connections between host applications (like IDEs, chat interfaces, or
// other tools) and external data sources and tools. This package implements
// a client that can connect to MCP servers and expose their tools, resources,
// and prompts to the OpenCode system.
//
// # Key Features
//
// • Multiple transport types: stdio, local command execution, and remote HTTP
// • Tool execution with automatic registration in the tool registry
// • Resource access for reading files and data from MCP servers
// • Prompt management for interacting with server-provided prompts
// • Connection management with status monitoring and error handling
// • Thread-safe operations with proper synchronization
//
// # Transport Types
//
// The package supports three transport mechanisms:
//
//	TransportTypeStdio  - Communication via stdin/stdout with a subprocess
//	TransportTypeLocal  - Direct execution of local commands
//	TransportTypeRemote - HTTP-based communication with remote servers
//
// # Basic Usage
//
//	// Create a new MCP client
//	client := mcp.NewClient()
//	
//	// Configure a server connection
//	config := &mcp.Config{
//		Enabled: true,
//		Type:    mcp.TransportTypeStdio,
//		Command: []string{"python", "-m", "my_mcp_server"},
//		Timeout: 5000, // 5 seconds
//	}
//	
//	// Add and connect to the server
//	err := client.AddServer(ctx, "my-server", config)
//	if err != nil {
//		log.Fatal(err)
//	}
//	
//	// List available tools
//	tools := client.Tools()
//	for _, tool := range tools {
//		fmt.Printf("Tool: %s - %s\n", tool.Name, tool.Description)
//	}
//	
//	// Execute a tool
//	args := json.RawMessage(`{"query": "example"}`)
//	result, err := client.ExecuteTool(ctx, "my-server_search", args)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println("Result:", result)
//
// # Tool Integration
//
// MCP tools are automatically wrapped and can be registered in the standard
// tool registry using MCPToolWrapper. This allows them to be used seamlessly
// in the agentic execution loop:
//
//	// Wrap an MCP tool for use in the tool registry
//	wrapper := mcp.NewMCPToolWrapper(mcpTool, client)
//	
//	// Register in the tool registry (typically done automatically)
//	registry.RegisterTool(wrapper.ID(), wrapper)
//
// # Configuration
//
// Server configurations support various options:
//
//	config := &mcp.Config{
//		Enabled:     true,
//		Type:        mcp.TransportTypeRemote,
//		URL:         "http://localhost:8080/mcp",
//		Headers:     map[string]string{"Authorization": "Bearer token"},
//		Environment: map[string]string{"API_KEY": "secret"},
//		Timeout:     10000, // 10 seconds
//	}
//
// # Error Handling
//
// The package provides comprehensive error handling and status monitoring:
//
//	// Check server status
//	status := client.Status()
//	for _, server := range status {
//		if server.Status == mcp.StatusFailed {
//			fmt.Printf("Server %s failed: %s\n", server.Name, *server.Error)
//		}
//	}
//	
//	// Get specific server status
//	serverStatus, err := client.GetServer("my-server")
//	if err != nil {
//		log.Printf("Server not found: %v", err)
//	}
//
// # Resource Access
//
// MCP servers can expose resources (files, data sources, etc.) that can be
// accessed through the client:
//
//	// List available resources
//	resources, err := client.ListResources(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//	
//	// Read a specific resource
//	response, err := client.ReadResource(ctx, "file:///path/to/file.txt")
//	if err != nil {
//		log.Fatal(err)
//	}
//	
//	for _, content := range response.Contents {
//		fmt.Printf("Content: %s\n", content.Text)
//	}
//
// # Connection Management
//
// The client manages multiple server connections concurrently:
//
//	// Get connection statistics
//	total := client.ServerCount()
//	connected := client.ConnectedCount()
//	fmt.Printf("Servers: %d total, %d connected\n", total, connected)
//	
//	// Remove a server
//	err := client.RemoveServer("my-server")
//	if err != nil {
//		log.Printf("Failed to remove server: %v", err)
//	}
//	
//	// Close all connections
//	err = client.Close()
//	if err != nil {
//		log.Printf("Error closing client: %v", err)
//	}
//
// # Thread Safety
//
// All client operations are thread-safe and can be called concurrently from
// multiple goroutines. The client uses appropriate synchronization mechanisms
// to ensure data consistency and prevent race conditions.
//
// # Protocol Version
//
// This package implements MCP protocol version 2024-11-05 using the official
// MCP Go SDK. It provides compatibility with standard MCP servers and follows
// the protocol specifications for tool execution, resource access, and
// communication patterns.
package mcp