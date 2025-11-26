package agent

import (
	"testing"

	"github.com/opencode-ai/opencode/internal/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()

	// Should have built-in agents
	assert.True(t, r.Exists("build"))
	assert.True(t, r.Exists("plan"))
	assert.True(t, r.Exists("general"))
	assert.True(t, r.Exists("explore"))
	assert.Equal(t, 4, r.Count())
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()

	// Get existing agent
	agent, err := r.Get("build")
	require.NoError(t, err)
	assert.Equal(t, "build", agent.Name)

	// Get non-existing agent
	_, err = r.Get("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent not found")
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	customAgent := &Agent{
		Name:        "custom",
		Description: "Custom agent",
		Mode:        ModeSubagent,
	}

	r.Register(customAgent)

	// Verify it was added
	agent, err := r.Get("custom")
	require.NoError(t, err)
	assert.Equal(t, "custom", agent.Name)
	assert.Equal(t, "Custom agent", agent.Description)
	assert.Equal(t, 5, r.Count())
}

func TestRegistry_Unregister(t *testing.T) {
	r := NewRegistry()

	// Add and then remove an agent
	r.Register(&Agent{Name: "temp"})
	assert.True(t, r.Exists("temp"))

	r.Unregister("temp")
	assert.False(t, r.Exists("temp"))
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()

	agents := r.List()
	assert.Len(t, agents, 4)

	// Verify all built-in agents are in the list
	names := make(map[string]bool)
	for _, a := range agents {
		names[a.Name] = true
	}
	assert.True(t, names["build"])
	assert.True(t, names["plan"])
	assert.True(t, names["general"])
	assert.True(t, names["explore"])
}

func TestRegistry_ListPrimary(t *testing.T) {
	r := NewRegistry()

	primary := r.ListPrimary()

	// build and plan are primary
	assert.GreaterOrEqual(t, len(primary), 2)

	for _, a := range primary {
		assert.True(t, a.IsPrimary())
	}
}

func TestRegistry_ListSubagents(t *testing.T) {
	r := NewRegistry()

	subagents := r.ListSubagents()

	// general and explore are subagents
	assert.GreaterOrEqual(t, len(subagents), 2)

	for _, a := range subagents {
		assert.True(t, a.IsSubagent())
	}
}

func TestRegistry_Names(t *testing.T) {
	r := NewRegistry()

	names := r.Names()
	assert.Len(t, names, 4)
	assert.Contains(t, names, "build")
	assert.Contains(t, names, "plan")
	assert.Contains(t, names, "general")
	assert.Contains(t, names, "explore")
}

func TestRegistry_LoadFromConfig(t *testing.T) {
	r := NewRegistry()

	config := map[string]AgentConfig{
		// Modify existing agent
		"build": {
			Temperature: 0.5,
			Model: &ModelRef{
				ProviderID: "openai",
				ModelID:    "gpt-4",
			},
		},
		// Add new agent
		"custom-agent": {
			Description: "My custom agent",
			Mode:        ModeSubagent,
			Tools: map[string]bool{
				"read": true,
				"edit": false,
			},
			Permission: &AgentPermissionConfig{
				Edit: permission.ActionDeny,
				Bash: map[string]permission.PermissionAction{
					"ls*": permission.ActionAllow,
					"*":   permission.ActionDeny,
				},
			},
		},
	}

	r.LoadFromConfig(config)

	// Verify modified agent
	build, err := r.Get("build")
	require.NoError(t, err)
	assert.Equal(t, 0.5, build.Temperature)
	assert.NotNil(t, build.Model)
	assert.Equal(t, "openai", build.Model.ProviderID)
	assert.Equal(t, "gpt-4", build.Model.ModelID)
	assert.False(t, build.BuiltIn) // Mark as customized

	// Verify new agent
	custom, err := r.Get("custom-agent")
	require.NoError(t, err)
	assert.Equal(t, "My custom agent", custom.Description)
	assert.Equal(t, ModeSubagent, custom.Mode)
	assert.True(t, custom.Tools["read"])
	assert.False(t, custom.Tools["edit"])
	assert.Equal(t, permission.ActionDeny, custom.Permission.Edit)
	assert.Equal(t, permission.ActionAllow, custom.Permission.Bash["ls*"])
	assert.Equal(t, permission.ActionDeny, custom.Permission.Bash["*"])
}

func TestRegistry_LoadFromConfig_MergesPermissions(t *testing.T) {
	r := NewRegistry()

	// Get original plan agent permissions
	original, _ := r.Get("plan")
	originalBashCount := len(original.Permission.Bash)

	config := map[string]AgentConfig{
		"plan": {
			Permission: &AgentPermissionConfig{
				Bash: map[string]permission.PermissionAction{
					"npm*": permission.ActionAllow,
				},
			},
		},
	}

	r.LoadFromConfig(config)

	plan, _ := r.Get("plan")

	// Should have original permissions plus new one
	assert.GreaterOrEqual(t, len(plan.Permission.Bash), originalBashCount)
	assert.Equal(t, permission.ActionAllow, plan.Permission.Bash["npm*"])
}

func TestRegistry_Concurrency(t *testing.T) {
	r := NewRegistry()

	done := make(chan bool, 100)

	// Concurrent reads
	for i := 0; i < 50; i++ {
		go func() {
			_, _ = r.Get("build")
			r.List()
			r.Names()
			r.Count()
			done <- true
		}()
	}

	// Concurrent writes
	for i := 0; i < 50; i++ {
		go func(i int) {
			r.Register(&Agent{Name: "concurrent"})
			r.Unregister("concurrent")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
}
