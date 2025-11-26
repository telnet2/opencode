// Package session provides session processing and the agentic loop.
package session

// Agent represents an agent configuration for processing.
type Agent struct {
	// Name is the agent identifier.
	Name string `json:"name"`

	// Prompt is the base system prompt for this agent.
	Prompt string `json:"prompt"`

	// Temperature for LLM sampling.
	Temperature float64 `json:"temperature,omitempty"`

	// TopP for nucleus sampling.
	TopP float64 `json:"topP,omitempty"`

	// MaxSteps is the maximum number of agentic loop iterations.
	MaxSteps int `json:"maxSteps,omitempty"`

	// Tools is the list of enabled tool IDs.
	Tools []string `json:"tools,omitempty"`

	// DisabledTools is the list of disabled tool IDs.
	DisabledTools []string `json:"disabledTools,omitempty"`

	// Permission contains permission policy for this agent.
	Permission AgentPermission `json:"permission,omitempty"`
}

// AgentPermission defines permission policies for an agent.
type AgentPermission struct {
	// DoomLoop defines how to handle repeated identical tool calls.
	// Values: "allow", "deny", "ask" (default)
	DoomLoop string `json:"doomLoop,omitempty"`

	// Bash defines the permission policy for bash commands.
	// Values: "allow", "deny", "ask" (default)
	Bash string `json:"bash,omitempty"`

	// Write defines the permission policy for file writes.
	// Values: "allow", "deny", "ask" (default)
	Write string `json:"write,omitempty"`
}

// ToolEnabled returns whether a tool is enabled for this agent.
func (a *Agent) ToolEnabled(toolID string) bool {
	// Check if explicitly disabled
	for _, disabled := range a.DisabledTools {
		if disabled == toolID {
			return false
		}
	}

	// If Tools is empty, all tools are enabled
	if len(a.Tools) == 0 {
		return true
	}

	// Check if explicitly enabled
	for _, enabled := range a.Tools {
		if enabled == toolID {
			return true
		}
	}

	return false
}

// DefaultAgent returns the default agent configuration.
func DefaultAgent() *Agent {
	return &Agent{
		Name:        "default",
		Temperature: 0.7,
		TopP:        1.0,
		MaxSteps:    50,
		Permission: AgentPermission{
			DoomLoop: "ask",
			Bash:     "ask",
			Write:    "ask",
		},
	}
}

// CodeAgent returns an agent optimized for coding tasks.
func CodeAgent() *Agent {
	return &Agent{
		Name:        "code",
		Temperature: 0.3,
		TopP:        0.95,
		MaxSteps:    100,
		Prompt: `You are an expert software engineer helping with coding tasks.
Focus on writing clean, maintainable code. Follow best practices and existing conventions in the codebase.
When making changes, prefer minimal modifications and explain your reasoning.`,
		Permission: AgentPermission{
			DoomLoop: "ask",
			Bash:     "ask",
			Write:    "allow",
		},
	}
}

// PlanAgent returns an agent optimized for planning tasks.
func PlanAgent() *Agent {
	return &Agent{
		Name:        "plan",
		Temperature: 0.5,
		TopP:        1.0,
		MaxSteps:    20,
		Prompt: `You are a helpful assistant focused on planning and analysis.
Break down complex tasks into manageable steps and provide clear explanations.
Focus on understanding the problem before suggesting solutions.`,
		DisabledTools: []string{"Write", "Edit", "Bash"},
		Permission: AgentPermission{
			DoomLoop: "deny",
			Bash:     "deny",
			Write:    "deny",
		},
	}
}
