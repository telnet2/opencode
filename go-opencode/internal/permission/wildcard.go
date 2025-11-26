package permission

import (
	"strings"
)

// MatchBashPermission finds the matching permission action for a command.
func MatchBashPermission(cmd BashCommand, permissions map[string]PermissionAction) PermissionAction {
	// Build command string variations for matching
	cmdWithSubcommand := cmd.Name
	if cmd.Subcommand != "" {
		cmdWithSubcommand = cmd.Name + " " + cmd.Subcommand
	}

	// Try most specific match first: "git commit *"
	if cmd.Subcommand != "" {
		if action, ok := permissions[cmdWithSubcommand+" *"]; ok {
			return action
		}
	}

	// Try command + wildcard: "git *"
	if action, ok := permissions[cmd.Name+" *"]; ok {
		return action
	}

	// Try command alone: "git"
	if action, ok := permissions[cmd.Name]; ok {
		return action
	}

	// Try global wildcard: "*"
	if action, ok := permissions["*"]; ok {
		return action
	}

	// Default to ask
	return ActionAsk
}

// MatchPattern checks if a command matches a wildcard pattern.
// Pattern format: "command subcommand *" or "command *" or "*"
func MatchPattern(pattern string, cmd BashCommand) bool {
	parts := strings.Split(pattern, " ")
	if len(parts) == 0 {
		return false
	}

	// Global wildcard matches everything
	if parts[0] == "*" && len(parts) == 1 {
		return true
	}

	// Match command name
	if parts[0] != "*" && parts[0] != cmd.Name {
		return false
	}

	// If only command name, must match exactly
	if len(parts) == 1 {
		return cmd.Name == parts[0] && len(cmd.Args) == 0
	}

	// If pattern ends with *, match any subcommand/args
	if parts[len(parts)-1] == "*" {
		// Match intermediate parts (subcommands)
		for i := 1; i < len(parts)-1; i++ {
			argIndex := i - 1
			if argIndex >= len(cmd.Args) {
				return false
			}
			if parts[i] != "*" && parts[i] != cmd.Args[argIndex] {
				return false
			}
		}
		return true
	}

	// Exact match required
	if len(parts)-1 != len(cmd.Args) {
		return false
	}
	for i := 1; i < len(parts); i++ {
		if parts[i] != cmd.Args[i-1] {
			return false
		}
	}
	return true
}

// BuildPattern creates a permission pattern for a command.
// For "git commit -m msg", returns "git commit *"
// For "ls -la", returns "ls *"
func BuildPattern(cmd BashCommand) string {
	if cmd.Subcommand != "" {
		return cmd.Name + " " + cmd.Subcommand + " *"
	}
	return cmd.Name + " *"
}

// BuildPatterns creates permission patterns for multiple commands.
func BuildPatterns(commands []BashCommand) []string {
	seen := make(map[string]bool)
	var patterns []string

	for _, cmd := range commands {
		// Skip "cd" since we handle directory changes separately
		if cmd.Name == "cd" {
			continue
		}

		pattern := BuildPattern(cmd)
		if !seen[pattern] {
			seen[pattern] = true
			patterns = append(patterns, pattern)
		}
	}

	return patterns
}
