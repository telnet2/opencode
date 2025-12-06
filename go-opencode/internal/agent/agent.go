// Package agent provides multi-agent configuration and management.
package agent

import (
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/opencode-ai/opencode/internal/permission"
)

// Agent represents an agent configuration.
type Agent struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Mode        Mode              `json:"mode"`
	BuiltIn     bool              `json:"builtIn"`
	Permission  AgentPermission   `json:"permission"`
	Tools       map[string]bool   `json:"tools"`
	Options     map[string]any    `json:"options,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	TopP        float64           `json:"topP,omitempty"`
	Model       *ModelRef         `json:"model,omitempty"`
	Prompt      string            `json:"prompt,omitempty"`
	Color       string            `json:"color,omitempty"`
}

// Mode represents the agent operation mode.
type Mode string

const (
	ModePrimary  Mode = "primary"
	ModeSubagent Mode = "subagent"
	ModeAll      Mode = "all"
)

// ModelRef references a specific model.
type ModelRef struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
}

// AgentPermission defines agent-specific permissions.
type AgentPermission struct {
	Edit        permission.PermissionAction            `json:"edit,omitempty"`
	Bash        map[string]permission.PermissionAction `json:"bash,omitempty"`
	WebFetch    permission.PermissionAction            `json:"webfetch,omitempty"`
	ExternalDir permission.PermissionAction            `json:"external_directory,omitempty"`
	DoomLoop    permission.PermissionAction            `json:"doom_loop,omitempty"`
}

// ToolEnabled checks if a tool is enabled for this agent.
func (a *Agent) ToolEnabled(toolID string) bool {
	// Check exact match
	if enabled, ok := a.Tools[toolID]; ok {
		return enabled
	}

	// Check wildcard patterns
	for pattern, enabled := range a.Tools {
		if matchWildcard(pattern, toolID) {
			return enabled
		}
	}

	// Default: enabled
	return true
}

// CheckBashPermission checks bash command permission for this agent.
func (a *Agent) CheckBashPermission(command string) permission.PermissionAction {
	// Check each pattern (more specific patterns first would be ideal)
	for pattern, action := range a.Permission.Bash {
		if matchWildcard(pattern, command) {
			return action
		}
	}

	// Default: ask
	return permission.ActionAsk
}

// GetPermission returns the permission action for a given permission type.
func (a *Agent) GetPermission(permType permission.PermissionType) permission.PermissionAction {
	switch permType {
	case permission.PermEdit:
		if a.Permission.Edit != "" {
			return a.Permission.Edit
		}
	case permission.PermWebFetch:
		if a.Permission.WebFetch != "" {
			return a.Permission.WebFetch
		}
	case permission.PermExternalDir:
		if a.Permission.ExternalDir != "" {
			return a.Permission.ExternalDir
		}
	case permission.PermDoomLoop:
		if a.Permission.DoomLoop != "" {
			return a.Permission.DoomLoop
		}
	}
	return permission.ActionAsk
}

// IsPrimary returns true if the agent can be used as a primary agent.
func (a *Agent) IsPrimary() bool {
	return a.Mode == ModePrimary || a.Mode == ModeAll
}

// IsSubagent returns true if the agent can be used as a subagent.
func (a *Agent) IsSubagent() bool {
	return a.Mode == ModeSubagent || a.Mode == ModeAll
}

// Clone creates a deep copy of the agent.
func (a *Agent) Clone() *Agent {
	clone := &Agent{
		Name:        a.Name,
		Description: a.Description,
		Mode:        a.Mode,
		BuiltIn:     a.BuiltIn,
		Temperature: a.Temperature,
		TopP:        a.TopP,
		Prompt:      a.Prompt,
		Color:       a.Color,
	}

	// Copy permission
	clone.Permission = AgentPermission{
		Edit:        a.Permission.Edit,
		WebFetch:    a.Permission.WebFetch,
		ExternalDir: a.Permission.ExternalDir,
		DoomLoop:    a.Permission.DoomLoop,
	}
	if a.Permission.Bash != nil {
		clone.Permission.Bash = make(map[string]permission.PermissionAction)
		for k, v := range a.Permission.Bash {
			clone.Permission.Bash[k] = v
		}
	}

	// Copy tools
	if a.Tools != nil {
		clone.Tools = make(map[string]bool)
		for k, v := range a.Tools {
			clone.Tools[k] = v
		}
	}

	// Copy options
	if a.Options != nil {
		clone.Options = make(map[string]any)
		for k, v := range a.Options {
			clone.Options[k] = v
		}
	}

	// Copy model ref
	if a.Model != nil {
		clone.Model = &ModelRef{
			ProviderID: a.Model.ProviderID,
			ModelID:    a.Model.ModelID,
		}
	}

	return clone
}

// matchWildcard checks if a string matches a wildcard pattern.
// For simple patterns (* at start/end), uses string matching.
// For complex patterns (containing **), uses doublestar.
func matchWildcard(pattern, s string) bool {
	// Global wildcard matches everything
	if pattern == "*" {
		return true
	}

	// For patterns with **, use doublestar
	if strings.Contains(pattern, "**") {
		matched, _ := doublestar.Match(pattern, s)
		return matched
	}

	// Simple suffix wildcard (prefix*)
	if strings.HasSuffix(pattern, "*") && !strings.HasPrefix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(s, prefix)
	}

	// Simple prefix wildcard (*suffix)
	if strings.HasPrefix(pattern, "*") && !strings.HasSuffix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(s, suffix)
	}

	// For patterns with * in the middle or multiple *, use doublestar
	if strings.Contains(pattern, "*") {
		matched, _ := doublestar.Match(pattern, s)
		return matched
	}

	// Exact match
	return pattern == s
}

// ExploreAgentPrompt is the system prompt for the explore agent.
const ExploreAgentPrompt = `You are a file search specialist. You excel at thoroughly navigating and exploring codebases.

Your strengths:
- Rapidly finding files using glob patterns
- Searching code and text with powerful regex patterns
- Reading and analyzing file contents

Guidelines:
- Use Glob for broad file pattern matching
- Use Grep for searching file contents with regex
- Use Read when you know the specific file path you need to read
- Use Bash for file operations like copying, moving, or listing directory contents
- Adapt your search approach based on the thoroughness level specified by the caller
- Return file paths as absolute paths in your final response
- For clear communication, avoid using emojis
- Do not create any files, or run bash commands that modify the user's system state in any way

Complete the user's search request efficiently and report your findings clearly.`

// BuiltInAgents returns the default agent configurations.
func BuiltInAgents() map[string]*Agent {
	return map[string]*Agent{
		"build": {
			Name:        "build",
			Description: "Primary agent for executing tasks, writing code, and making changes",
			Mode:        ModePrimary,
			BuiltIn:     true,
			Permission: AgentPermission{
				Edit:        permission.ActionAllow,
				Bash:        map[string]permission.PermissionAction{"*": permission.ActionAllow},
				WebFetch:    permission.ActionAllow,
				ExternalDir: permission.ActionAsk,
				DoomLoop:    permission.ActionAsk,
			},
			Tools: map[string]bool{
				"*": true,
			},
		},
		"plan": {
			Name:        "plan",
			Description: "Planning agent for analysis and exploration without making changes",
			Mode:        ModePrimary,
			BuiltIn:     true,
			Permission: AgentPermission{
				Edit: permission.ActionDeny,
				Bash: map[string]permission.PermissionAction{
					"cut*":           permission.ActionAllow,
					"diff*":          permission.ActionAllow,
					"du*":            permission.ActionAllow,
					"file *":         permission.ActionAllow,
					"find * -delete*":  permission.ActionAsk,
					"find * -exec*":    permission.ActionAsk,
					"find * -fprint*":  permission.ActionAsk,
					"find * -fls*":     permission.ActionAsk,
					"find * -fprintf*": permission.ActionAsk,
					"find * -ok*":      permission.ActionAsk,
					"find *":         permission.ActionAllow,
					"git diff*":      permission.ActionAllow,
					"git log*":       permission.ActionAllow,
					"git show*":      permission.ActionAllow,
					"git status*":    permission.ActionAllow,
					"git branch":     permission.ActionAllow,
					"git branch -v":  permission.ActionAllow,
					"grep*":          permission.ActionAllow,
					"head*":          permission.ActionAllow,
					"less*":          permission.ActionAllow,
					"ls*":            permission.ActionAllow,
					"more*":          permission.ActionAllow,
					"pwd*":           permission.ActionAllow,
					"rg*":            permission.ActionAllow,
					"sort --output=*": permission.ActionAsk,
					"sort -o *":      permission.ActionAsk,
					"sort*":          permission.ActionAllow,
					"stat*":          permission.ActionAllow,
					"tail*":          permission.ActionAllow,
					"tree -o *":      permission.ActionAsk,
					"tree*":          permission.ActionAllow,
					"uniq*":          permission.ActionAllow,
					"wc*":            permission.ActionAllow,
					"whereis*":       permission.ActionAllow,
					"which*":         permission.ActionAllow,
					"*":              permission.ActionAsk,
				},
				WebFetch:    permission.ActionAllow,
				ExternalDir: permission.ActionDeny,
				DoomLoop:    permission.ActionDeny,
			},
			Tools: map[string]bool{
				"*":         true,
				"edit":      false,
				"write":     false,
				"todoread":  false,
				"todowrite": false,
			},
		},
		"general": {
			Name:        "general",
			Description: "General-purpose agent for researching complex questions and executing multi-step tasks. Use this agent to execute multiple units of work in parallel.",
			Mode:        ModeSubagent,
			BuiltIn:     true,
			Permission: AgentPermission{
				Edit:        permission.ActionAllow,
				Bash:        map[string]permission.PermissionAction{"*": permission.ActionAllow},
				WebFetch:    permission.ActionAllow,
				ExternalDir: permission.ActionAsk,
				DoomLoop:    permission.ActionAsk,
			},
			Tools: map[string]bool{
				"*":         true,
				"todoread":  false,
				"todowrite": false,
				"task":      false, // Prevent recursive task calls
			},
		},
		"explore": {
			Name:        "explore",
			Description: `Fast agent specialized for exploring codebases. Use this when you need to quickly find files by patterns (eg. "src/components/**/*.tsx"), search code for keywords (eg. "API endpoints"), or answer questions about the codebase (eg. "how do API endpoints work?"). When calling this agent, specify the desired thoroughness level: "quick" for basic searches, "medium" for moderate exploration, or "very thorough" for comprehensive analysis across multiple locations and naming conventions.`,
			Mode:        ModeSubagent,
			BuiltIn:     true,
			Prompt:      ExploreAgentPrompt,
			Permission: AgentPermission{
				Edit:        permission.ActionAllow,
				Bash:        map[string]permission.PermissionAction{"*": permission.ActionAllow},
				WebFetch:    permission.ActionAllow,
				ExternalDir: permission.ActionAsk,
				DoomLoop:    permission.ActionAsk,
			},
			Tools: map[string]bool{
				"*":         true,
				"todoread":  false,
				"todowrite": false,
				"edit":      false,
				"write":     false,
				"task":      false, // Prevent recursive task calls
			},
		},
	}
}
