package agent

import (
	"fmt"
	"sync"

	"github.com/opencode-ai/opencode/internal/permission"
)

// Registry manages agent configurations.
type Registry struct {
	mu     sync.RWMutex
	agents map[string]*Agent
}

// NewRegistry creates a new agent registry.
func NewRegistry() *Registry {
	r := &Registry{
		agents: make(map[string]*Agent),
	}

	// Register built-in agents
	for name, agent := range BuiltInAgents() {
		r.agents[name] = agent
	}

	return r
}

// Get retrieves an agent by name.
func (r *Registry) Get(name string) (*Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, ok := r.agents[name]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", name)
	}

	return agent, nil
}

// Register adds or updates an agent.
func (r *Registry) Register(agent *Agent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[agent.Name] = agent
}

// Unregister removes an agent by name.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, name)
}

// List returns all registered agents.
func (r *Registry) List() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agents := make([]*Agent, 0, len(r.agents))
	for _, agent := range r.agents {
		agents = append(agents, agent)
	}
	return agents
}

// ListPrimary returns agents with primary mode.
func (r *Registry) ListPrimary() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var agents []*Agent
	for _, agent := range r.agents {
		if agent.IsPrimary() {
			agents = append(agents, agent)
		}
	}
	return agents
}

// ListSubagents returns agents with subagent mode.
func (r *Registry) ListSubagents() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var agents []*Agent
	for _, agent := range r.agents {
		if agent.IsSubagent() {
			agents = append(agents, agent)
		}
	}
	return agents
}

// Names returns all agent names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}
	return names
}

// Exists checks if an agent exists.
func (r *Registry) Exists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.agents[name]
	return ok
}

// Count returns the number of registered agents.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.agents)
}

// LoadFromConfig loads custom agents from configuration.
func (r *Registry) LoadFromConfig(config map[string]AgentConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for name, cfg := range config {
		// Start with existing or create new
		agent, exists := r.agents[name]
		if !exists {
			agent = &Agent{
				Name:    name,
				Mode:    ModePrimary,
				BuiltIn: false,
				Tools:   make(map[string]bool),
			}
		} else {
			// Clone existing to avoid modifying built-in directly
			agent = agent.Clone()
			agent.BuiltIn = false // Mark as customized
		}

		// Apply config overrides
		if cfg.Description != "" {
			agent.Description = cfg.Description
		}
		if cfg.Mode != "" {
			agent.Mode = cfg.Mode
		}
		if cfg.Model != nil {
			agent.Model = cfg.Model
		}
		if cfg.Prompt != "" {
			agent.Prompt = cfg.Prompt
		}
		if cfg.Temperature > 0 {
			agent.Temperature = cfg.Temperature
		}
		if cfg.TopP > 0 {
			agent.TopP = cfg.TopP
		}
		if cfg.Color != "" {
			agent.Color = cfg.Color
		}
		if cfg.Tools != nil {
			if agent.Tools == nil {
				agent.Tools = make(map[string]bool)
			}
			for k, v := range cfg.Tools {
				agent.Tools[k] = v
			}
		}
		if cfg.Permission != nil {
			// Merge permissions
			if cfg.Permission.Edit != "" {
				agent.Permission.Edit = cfg.Permission.Edit
			}
			if cfg.Permission.WebFetch != "" {
				agent.Permission.WebFetch = cfg.Permission.WebFetch
			}
			if cfg.Permission.ExternalDir != "" {
				agent.Permission.ExternalDir = cfg.Permission.ExternalDir
			}
			if cfg.Permission.DoomLoop != "" {
				agent.Permission.DoomLoop = cfg.Permission.DoomLoop
			}
			if cfg.Permission.Bash != nil {
				if agent.Permission.Bash == nil {
					agent.Permission.Bash = make(map[string]permission.PermissionAction)
				}
				for k, v := range cfg.Permission.Bash {
					agent.Permission.Bash[k] = v
				}
			}
		}
		if cfg.Options != nil {
			if agent.Options == nil {
				agent.Options = make(map[string]any)
			}
			for k, v := range cfg.Options {
				agent.Options[k] = v
			}
		}

		r.agents[name] = agent
	}
}

// AgentConfig represents user configuration for an agent.
type AgentConfig struct {
	Description string                 `json:"description,omitempty"`
	Mode        Mode                   `json:"mode,omitempty"`
	Model       *ModelRef              `json:"model,omitempty"`
	Prompt      string                 `json:"prompt,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	TopP        float64                `json:"topP,omitempty"`
	Color       string                 `json:"color,omitempty"`
	Tools       map[string]bool        `json:"tools,omitempty"`
	Permission  *AgentPermissionConfig `json:"permission,omitempty"`
	Options     map[string]any         `json:"options,omitempty"`
}

// AgentPermissionConfig represents permission configuration.
type AgentPermissionConfig struct {
	Edit        permission.PermissionAction            `json:"edit,omitempty"`
	Bash        map[string]permission.PermissionAction `json:"bash,omitempty"`
	WebFetch    permission.PermissionAction            `json:"webfetch,omitempty"`
	ExternalDir permission.PermissionAction            `json:"external_directory,omitempty"`
	DoomLoop    permission.PermissionAction            `json:"doom_loop,omitempty"`
}
