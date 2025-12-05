// Package agent provides multi-agent configuration and management for opencode.
//
// This package implements a flexible agent system that supports different operation
// modes, tool access controls, and permission management. Agents can operate as
// primary agents (user-facing) or subagents (invoked by other agents).
//
// # Agent Types
//
// The package provides four built-in agents:
//
//   - build: Primary agent for executing tasks, writing code, and making changes.
//     Has full tool access and permissive permissions.
//   - plan: Primary agent for analysis and exploration without making changes.
//     Restricted to read-only operations.
//   - general: Subagent for general-purpose searches and exploration.
//   - explore: Fast subagent specialized for codebase exploration.
//
// # Agent Modes
//
// Agents operate in one of three modes:
//
//   - ModePrimary: Can be selected as the main agent for a session
//   - ModeSubagent: Can only be invoked by other agents via the Task tool
//   - ModeAll: Can operate in both primary and subagent contexts
//
// # Tool Access Control
//
// Each agent has a Tools map that controls which tools are available. Tools can be
// enabled or disabled using exact names or wildcard patterns:
//
//	agent.Tools = map[string]bool{
//	    "*":     true,   // Enable all tools by default
//	    "bash":  false,  // Disable bash specifically
//	    "mcp_*": true,   // Enable all MCP tools
//	}
//
// The [Agent.ToolEnabled] method checks tool availability, supporting glob patterns
// including doublestar (**) for complex matching.
//
// # Permission System
//
// Agents define permissions for sensitive operations through [AgentPermission]:
//
//   - Edit: Controls file editing permissions
//   - Bash: Maps command patterns to permission actions
//   - WebFetch: Controls web fetching permissions
//   - ExternalDir: Controls access to directories outside the project
//   - DoomLoop: Controls handling of repeated failure patterns
//
// Permission actions are: allow, deny, or ask (prompt user).
//
// # Registry
//
// The [Registry] type manages agent configurations with thread-safe operations:
//
//	registry := agent.NewRegistry()  // Includes built-in agents
//	registry.Register(customAgent)   // Add custom agent
//	agent, err := registry.Get("build")
//	primaryAgents := registry.ListPrimary()
//	subagents := registry.ListSubagents()
//
// # Custom Configuration
//
// Custom agents can be loaded from configuration using [Registry.LoadFromConfig].
// Configurations can extend or override built-in agents:
//
//	config := map[string]agent.AgentConfig{
//	    "build": {
//	        Temperature: 0.7,
//	        Permission: &agent.AgentPermissionConfig{
//	            Edit: permission.ActionAsk,
//	        },
//	    },
//	    "custom": {
//	        Description: "Custom agent",
//	        Mode:        agent.ModePrimary,
//	        Tools:       map[string]bool{"read": true, "glob": true},
//	    },
//	}
//	registry.LoadFromConfig(config)
package agent
