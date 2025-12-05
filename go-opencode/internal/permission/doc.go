// Package permission provides a comprehensive permission control system for tool execution
// in the OpenCode AI assistant. It manages user consent for potentially dangerous operations
// like file editing, web fetching, external directory access, and bash command execution.
//
// # Overview
//
// The permission system operates on a session-based model where each user interaction
// session can have different permission levels. It supports three main permission actions:
//   - Allow: Automatically approve the operation
//   - Deny: Automatically reject the operation
//   - Ask: Prompt the user for consent
//
// # Permission Types
//
// The system handles several types of operations:
//
//   - Bash: Command execution with pattern-based matching
//   - Edit: File modification operations
//   - WebFetch: External web resource access
//   - ExternalDir: Operations outside the working directory
//   - DoomLoop: Detection and prevention of infinite tool call loops
//
// # Core Components
//
// ## Checker
//
// The Checker is the central component that manages permission requests and approvals.
// It maintains session-based state for approved permissions and handles user prompts
// through an event system.
//
//	checker := NewChecker()
//	req := Request{
//		Type:      PermBash,
//		SessionID: "session-123",
//		Pattern:   []string{"git *"},
//		Title:     "Execute git command",
//	}
//	err := checker.Check(ctx, req, ActionAsk)
//
// ## Bash Command Parsing
//
// The system includes sophisticated bash command parsing that extracts command names,
// arguments, and subcommands for fine-grained permission control:
//
//	commands, err := ParseBashCommand("git commit -m 'fix bug'")
//	// Returns: BashCommand{Name: "git", Subcommand: "commit", Args: ["-m", "fix bug"]}
//
// ## Pattern Matching
//
// Bash permissions support wildcard patterns with hierarchical matching:
//   - "git commit *" - Matches git commit with any arguments
//   - "git *" - Matches any git subcommand
//   - "git" - Matches git command exactly
//   - "*" - Matches any command
//
// ## Doom Loop Detection
//
// The DoomLoopDetector prevents infinite loops by tracking tool call patterns:
//
//	detector := NewDoomLoopDetector()
//	isLoop := detector.Check(sessionID, "bash", commandInput)
//	if isLoop {
//		// Handle potential infinite loop
//	}
//
// # Permission Configuration
//
// AgentPermissions defines the permission policy for an agent:
//
//	permissions := AgentPermissions{
//		Edit:        ActionAsk,
//		WebFetch:    ActionAllow,
//		ExternalDir: ActionDeny,
//		DoomLoop:    ActionAsk,
//		Bash: map[string]PermissionAction{
//			"git *":        ActionAllow,
//			"rm *":         ActionAsk,
//			"sudo *":       ActionDeny,
//		},
//	}
//
// # Session Management
//
// The system maintains per-session state for approved permissions. When a user
// grants "always" permission, it's remembered for the duration of the session:
//
//	// Clear all approvals for a session
//	checker.ClearSession("session-123")
//	
//	// Check if permission is already approved
//	if checker.IsApproved("session-123", PermBash) {
//		// Skip asking user
//	}
//
// # Error Handling
//
// Permission denials are represented by RejectedError, which includes context
// about the denied operation:
//
//	if err != nil && IsRejectedError(err) {
//		rejErr := err.(*RejectedError)
//		log.Printf("Permission denied for %s: %s", rejErr.Type, rejErr.Message)
//	}
//
// # Event Integration
//
// The permission system integrates with the event system to notify UI components
// about permission requests and responses. This enables real-time user interaction
// through web interfaces or other UI systems.
//
// # Security Considerations
//
// The permission system is designed with security in mind:
//   - All bash commands are parsed and validated
//   - Pattern matching prevents bypass through command variations
//   - Doom loop detection prevents resource exhaustion
//   - Session isolation prevents permission escalation across sessions
//   - External directory access is explicitly controlled
//
// # Thread Safety
//
// All components in this package are thread-safe and can be used concurrently
// across multiple goroutines handling different user sessions.
package permission