// Package server provides the HTTP server implementation for the OpenCode API.
//
// The server package implements a comprehensive RESTful API server that serves as the
// backbone of the OpenCode application. It provides endpoints for managing AI-powered
// coding sessions, file operations, configuration management, and real-time event streaming.
//
// # Core Components
//
// The server is built around several key components:
//
//   - HTTP Server: Chi-based router with middleware for CORS, logging, and recovery
//   - Session Management: Handles AI conversation sessions with providers
//   - Event Streaming: Server-Sent Events (SSE) for real-time updates
//   - File Operations: File system operations and Git integration
//   - Provider Integration: Support for multiple AI providers (Anthropic, OpenAI, etc.)
//   - Tool Registry: Extensible tool system for AI capabilities
//   - MCP Integration: Model Context Protocol support for external tools
//   - LSP Integration: Language Server Protocol for code intelligence
//
// # API Endpoints
//
// The server exposes the following main endpoint categories:
//
//   - /session/*: Session lifecycle management and messaging
//   - /file/*: File system operations and Git status
//   - /config/*: Application configuration management
//   - /provider/*: AI provider management and authentication
//   - /event: Real-time event streaming via SSE
//   - /mcp/*: Model Context Protocol server management
//   - /tui/*: Terminal UI control endpoints
//   - /client-tools/*: External tool registration and execution
//
// # Session Management
//
// Sessions are the core abstraction for AI conversations. Each session:
//   - Maintains conversation history with an AI provider
//   - Has an associated working directory for file operations
//   - Can be forked to create branching conversations
//   - Supports real-time streaming of AI responses
//   - Integrates with tools for code analysis and modification
//
// # Event System
//
// The server implements a custom SSE-based event system for real-time updates:
//   - Session events (message updates, status changes)
//   - File system events (changes, Git status updates)
//   - Tool execution events
//   - Provider status updates
//
// # Tool Integration
//
// The server supports multiple tool systems:
//   - Built-in tools for file operations, shell commands, and code formatting
//   - MCP (Model Context Protocol) servers for external tool integration
//   - Client-registered tools for custom functionality
//   - LSP integration for code intelligence and symbol search
//
// # Configuration
//
// Server configuration is managed through:
//   - Static configuration file (types.Config)
//   - Runtime configuration updates via API
//   - Environment-based provider authentication
//   - Per-project settings and preferences
//
// # Usage Example
//
//	config := server.DefaultConfig()
//	config.Port = 8080
//	config.Directory = "/path/to/project"
//
//	srv := server.New(config, appConfig, storage, providerRegistry, toolRegistry)
//
//	// Initialize MCP servers
//	if err := srv.InitializeMCP(ctx); err != nil {
//		log.Fatal(err)
//	}
//	defer srv.CloseMCP()
//
//	// Start server
//	if err := srv.Start(); err != nil {
//		log.Fatal(err)
//	}
//
// # Architecture Notes
//
// The server uses a layered architecture:
//   - HTTP handlers for request/response processing
//   - Service layer for business logic (session, storage, etc.)
//   - Provider abstraction for AI model integration
//   - Event bus for decoupled component communication
//   - Storage layer for persistence
//
// The implementation favors composition over inheritance, with each major
// component (sessions, tools, providers) being independently testable
// and replaceable.
//
// # SSE Implementation
//
// The server includes a custom Server-Sent Events implementation optimized
// for the OpenCode use case. This provides real-time streaming of:
//   - AI response tokens as they're generated
//   - Tool execution progress and results
//   - File system change notifications
//   - Session status updates
//
// The SSE implementation includes heartbeat support, proper error handling,
// and session-based event filtering for efficient client updates.
package server