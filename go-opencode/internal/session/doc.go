// Package session provides comprehensive session management functionality for the OpenCode AI assistant.
//
// This package implements the core session lifecycle, message processing, and agentic loop
// that powers OpenCode's AI-driven code assistance capabilities. It manages conversations
// between users and AI agents, handles tool execution, and maintains session state across
// multiple interactions.
//
// # Architecture Overview
//
// The session package is built around several key components:
//
//   - Service: High-level session management and CRUD operations
//   - Processor: Core agentic loop implementation with streaming LLM interactions
//   - Agent: Configurable AI agent profiles with different capabilities and permissions
//   - Tools: Integration with the tool registry for code manipulation and execution
//   - Storage: Persistent storage of sessions, messages, and conversation history
//
// # Core Components
//
// ## Service
//
// The Service struct provides the main API for session management:
//
//	service := session.NewService(storage)
//	
//	// Create a new session
//	sess, err := service.Create(ctx, "/path/to/project", "My Session")
//	
//	// Process user messages
//	msg, parts, err := service.ProcessMessage(ctx, sess, "Help me refactor this code", model, callback)
//
// ## Processor
//
// The Processor handles the agentic loop - the core AI reasoning cycle:
//
//	processor := session.NewProcessor(providerReg, toolReg, storage, permChecker, "anthropic", "claude-sonnet")
//	err := processor.Process(ctx, sessionID, agent, callback)
//
// The processor manages:
//   - LLM streaming and response processing
//   - Tool call execution with permission checking
//   - Context management and compaction
//   - Error handling and retries with exponential backoff
//   - Real-time event publishing for UI updates
//
// ## Agents
//
// Agents define AI behavior profiles with different capabilities:
//
//	// Default general-purpose agent
//	agent := session.DefaultAgent()
//	
//	// Code-focused agent with write permissions
//	codeAgent := session.CodeAgent()
//	
//	// Planning agent without file modification capabilities
//	planAgent := session.PlanAgent()
//
// Agent configuration includes:
//   - System prompts and personality
//   - Temperature and sampling parameters
//   - Tool access permissions
//   - Safety policies (doom loop detection, permission requirements)
//
// # Message Processing Flow
//
// The typical message processing flow follows these steps:
//
//  1. User creates a message with text/file parts
//  2. Service.ProcessMessage() initiates the agentic loop
//  3. Processor loads conversation history and builds LLM context
//  4. System prompt is constructed based on agent configuration
//  5. LLM generates streaming response with potential tool calls
//  6. Tools are executed with permission checking
//  7. Results are fed back to the LLM for continued reasoning
//  8. Process repeats until completion or step limit reached
//  9. Final response is saved and events published
//
// # Tool Integration
//
// The session package integrates tightly with the tool system:
//
//	// Tools are called by the LLM during processing
//	toolPart := &types.ToolPart{
//		Tool: "write_file",
//		State: types.ToolState{
//			Input: map[string]any{
//				"path": "main.go",
//				"content": "package main...",
//			},
//		},
//	}
//
// Tool execution includes:
//   - Permission validation based on agent policies
//   - Doom loop detection for repeated identical calls
//   - Real-time progress updates via callbacks
//   - Error handling and graceful degradation
//
// # Context Management
//
// The package implements intelligent context management:
//
//   - Automatic message compaction when context limits are approached
//   - Conversation summarization to preserve key information
//   - Token counting and optimization
//   - Configurable retention policies
//
// # Event System
//
// Real-time events are published throughout the processing lifecycle:
//
//	// Session status updates
//	event.SessionStatus{SessionID: "...", Status: "busy"}
//	
//	// Message creation and updates
//	event.MessageCreated{Info: message}
//	event.MessagePartUpdated{Part: part}
//	
//	// Session completion
//	event.SessionIdle{SessionID: "..."}
//
// # Permission System
//
// Fine-grained permission control is enforced:
//
//   - Tool-level permissions (allow/deny/ask)
//   - File system access controls
//   - Shell command execution policies
//   - Doom loop prevention
//
// # Storage and Persistence
//
// Sessions and messages are persisted using a hierarchical key-value structure:
//
//	session/{projectID}/{sessionID}     -> Session metadata
//	message/{sessionID}/{messageID}     -> Individual messages
//	part/{messageID}/{partID}          -> Message parts (text, files, tools)
//
// # Error Handling
//
// Robust error handling is implemented throughout:
//
//   - Exponential backoff for LLM API failures
//   - Graceful degradation when tools fail
//   - Context cancellation support
//   - Detailed error propagation and logging
//
// # Usage Examples
//
// ## Basic Session Creation
//
//	service := session.NewServiceWithProcessor(
//		storage, providerReg, toolReg, permChecker,
//		"anthropic", "claude-sonnet-4-20250514",
//	)
//	
//	sess, err := service.Create(ctx, "/home/user/project", "Code Review")
//	if err != nil {
//		log.Fatal(err)
//	}
//
// ## Processing User Input
//
//	callback := func(msg *types.Message, parts []types.Part) {
//		// Handle real-time updates
//		fmt.Printf("Response: %v\n", parts)
//	}
//	
//	model := &types.ModelRef{
//		ProviderID: "anthropic",
//		ModelID:    "claude-sonnet-4-20250514",
//	}
//	
//	msg, parts, err := service.ProcessMessage(ctx, sess, "Refactor this function", model, callback)
//
// ## Custom Agent Configuration
//
//	agent := &session.Agent{
//		Name:        "security-reviewer",
//		Temperature: 0.2,
//		MaxSteps:    20,
//		Prompt:      "You are a security-focused code reviewer...",
//		Tools:       []string{"read", "grep"},  // Read-only tools
//		Permission: session.AgentPermission{
//			Write: "deny",
//			Bash:  "deny",
//		},
//	}
//
// ## Session Management
//
//	// List sessions for a project
//	sessions, err := service.List(ctx, "/home/user/project")
//	
//	// Fork a session at a specific message
//	fork, err := service.Fork(ctx, sessionID, messageID)
//	
//	// Share a session
//	shareURL, err := service.Share(ctx, sessionID)
//	
//	// Abort active processing
//	err = service.Abort(ctx, sessionID)
//
// # Thread Safety
//
// The session package is designed for concurrent use:
//   - Service methods are thread-safe
//   - Processor handles concurrent session processing
//   - Proper synchronization prevents race conditions
//   - Context cancellation is respected throughout
//
// # Performance Considerations
//
//   - Streaming responses minimize latency
//   - Context compaction prevents memory bloat
//   - Efficient storage access patterns
//   - Configurable retry policies balance reliability and speed
//
// # Integration Points
//
// The session package integrates with several other OpenCode components:
//
//   - internal/provider: LLM provider abstraction
//   - internal/tool: Tool execution framework
//   - internal/storage: Persistent data storage
//   - internal/permission: Access control and security
//   - internal/event: Real-time event system
//   - pkg/types: Shared type definitions
//
// This package forms the core of OpenCode's conversational AI capabilities,
// providing a robust foundation for AI-assisted software development workflows.
package session