// Package command provides a flexible command execution system for OpenCode.
//
// This package implements a custom command system that allows users to define
// and execute templated commands with variable substitution. Commands can be
// defined in configuration files or as markdown files in the .opencode/command
// directory.
//
// # Command Sources
//
// Commands can be loaded from two sources:
//
//  1. Configuration files: Commands defined in the OpenCode configuration
//  2. Markdown files: Commands stored as .md files in .opencode/command/
//
// # Command Structure
//
// Each command consists of:
//   - Name: Unique identifier for the command
//   - Description: Human-readable description of what the command does
//   - Template: The template string that will be executed with variable substitution
//   - Agent: Optional agent to use for execution
//   - Model: Optional model to use for execution
//   - Subtask: Whether this command represents a subtask
//
// # Template System
//
// Commands use Go templates with additional support for simple variable substitution:
//
//   - ${variable} syntax for variable expansion
//   - $variable syntax for simple variable references
//   - $1, $2, ... for positional arguments
//   - $input for the full input string
//   - --name=value or --name value for named arguments
//
// # Template Context
//
// Templates have access to:
//   - args: Map of parsed arguments
//   - input: The raw input string
//   - vars: Configured prompt variables
//   - env: Environment variables
//   - workDir: Current working directory
//   - Custom template functions (env, default, trim, upper, lower, etc.)
//
// # Markdown Command Format
//
// Markdown commands can include YAML frontmatter:
//
//	---
//	description: Run tests
//	agent: test-agent
//	model: claude-3
//	subtask: true
//	---
//	Run tests for ${1} package
//
// # Built-in Commands
//
// The package provides several built-in commands:
//   - help: Show available commands and help information
//   - clear: Clear the current conversation
//   - compact: Compact the conversation to save context
//   - reset: Reset the session to its initial state
//   - undo: Undo the last message
//   - share: Share the current session
//   - export: Export the conversation
//
// # Example Usage
//
//	// Create executor
//	executor := NewExecutor("/path/to/work/dir", config)
//	
//	// Execute a command
//	result, err := executor.Execute(ctx, "greet", "World")
//	if err != nil {
//		log.Fatal(err)
//	}
//	
//	// Use the generated prompt
//	fmt.Println(result.Prompt) // "Hello, World!"
//
// # Dynamic Command Management
//
// Commands can be managed at runtime:
//
//	// Add a new command
//	executor.AddCommand(&Command{
//		Name:     "custom",
//		Template: "Custom command with $1",
//	})
//	
//	// Remove a command
//	executor.RemoveCommand("custom")
//	
//	// Reload all commands
//	executor.Reload()
//
// The command system is designed to be flexible and extensible, supporting
// both simple string substitution and complex Go template logic while
// maintaining ease of use for end users.
package command