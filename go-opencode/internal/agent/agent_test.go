package agent

import (
	"testing"

	"github.com/opencode-ai/opencode/internal/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgent_ToolEnabled(t *testing.T) {
	tests := []struct {
		name     string
		agent    *Agent
		toolID   string
		expected bool
	}{
		{
			name: "exact match enabled",
			agent: &Agent{
				Tools: map[string]bool{"read": true},
			},
			toolID:   "read",
			expected: true,
		},
		{
			name: "exact match disabled",
			agent: &Agent{
				Tools: map[string]bool{"write": false},
			},
			toolID:   "write",
			expected: false,
		},
		{
			name: "wildcard all enabled",
			agent: &Agent{
				Tools: map[string]bool{"*": true},
			},
			toolID:   "anytool",
			expected: true,
		},
		{
			name: "prefix wildcard",
			agent: &Agent{
				Tools: map[string]bool{"mcp_*": true},
			},
			toolID:   "mcp_server_tool",
			expected: true,
		},
		{
			name: "suffix wildcard",
			agent: &Agent{
				Tools: map[string]bool{"*_read": false},
			},
			toolID:   "file_read",
			expected: false,
		},
		{
			name: "default enabled when not specified",
			agent: &Agent{
				Tools: map[string]bool{"other": true},
			},
			toolID:   "unknown",
			expected: true,
		},
		{
			name: "nil tools map defaults to enabled",
			agent: &Agent{
				Tools: nil,
			},
			toolID:   "anything",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.agent.ToolEnabled(tt.toolID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgent_CheckBashPermission(t *testing.T) {
	tests := []struct {
		name     string
		agent    *Agent
		command  string
		expected permission.PermissionAction
	}{
		{
			name: "exact match",
			agent: &Agent{
				Permission: AgentPermission{
					Bash: map[string]permission.PermissionAction{
						"git status": permission.ActionAllow,
					},
				},
			},
			command:  "git status",
			expected: permission.ActionAllow,
		},
		{
			name: "prefix wildcard match",
			agent: &Agent{
				Permission: AgentPermission{
					Bash: map[string]permission.PermissionAction{
						"git diff*": permission.ActionAllow,
					},
				},
			},
			command:  "git diff --cached",
			expected: permission.ActionAllow,
		},
		{
			name: "wildcard all",
			agent: &Agent{
				Permission: AgentPermission{
					Bash: map[string]permission.PermissionAction{
						"*": permission.ActionDeny,
					},
				},
			},
			command:  "rm -rf /",
			expected: permission.ActionDeny,
		},
		{
			name: "default to ask",
			agent: &Agent{
				Permission: AgentPermission{
					Bash: map[string]permission.PermissionAction{},
				},
			},
			command:  "unknown command",
			expected: permission.ActionAsk,
		},
		{
			name: "nil bash map defaults to ask",
			agent: &Agent{
				Permission: AgentPermission{
					Bash: nil,
				},
			},
			command:  "any",
			expected: permission.ActionAsk,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.agent.CheckBashPermission(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgent_GetPermission(t *testing.T) {
	agent := &Agent{
		Permission: AgentPermission{
			Edit:        permission.ActionAllow,
			WebFetch:    permission.ActionDeny,
			ExternalDir: permission.ActionAsk,
			DoomLoop:    permission.ActionDeny,
		},
	}

	tests := []struct {
		permType permission.PermissionType
		expected permission.PermissionAction
	}{
		{permission.PermEdit, permission.ActionAllow},
		{permission.PermWebFetch, permission.ActionDeny},
		{permission.PermExternalDir, permission.ActionAsk},
		{permission.PermDoomLoop, permission.ActionDeny},
		{permission.PermBash, permission.ActionAsk}, // bash uses CheckBashPermission
	}

	for _, tt := range tests {
		t.Run(string(tt.permType), func(t *testing.T) {
			result := agent.GetPermission(tt.permType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgent_IsPrimaryAndIsSubagent(t *testing.T) {
	tests := []struct {
		mode      Mode
		isPrimary bool
		isSubagent bool
	}{
		{ModePrimary, true, false},
		{ModeSubagent, false, true},
		{ModeAll, true, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			agent := &Agent{Mode: tt.mode}
			assert.Equal(t, tt.isPrimary, agent.IsPrimary())
			assert.Equal(t, tt.isSubagent, agent.IsSubagent())
		})
	}
}

func TestAgent_Clone(t *testing.T) {
	original := &Agent{
		Name:        "test",
		Description: "Test agent",
		Mode:        ModePrimary,
		BuiltIn:     true,
		Temperature: 0.7,
		TopP:        0.9,
		Prompt:      "You are a test agent",
		Color:       "#FF0000",
		Permission: AgentPermission{
			Edit:        permission.ActionAllow,
			Bash:        map[string]permission.PermissionAction{"*": permission.ActionDeny},
			WebFetch:    permission.ActionAsk,
			ExternalDir: permission.ActionDeny,
			DoomLoop:    permission.ActionDeny,
		},
		Tools: map[string]bool{
			"read":  true,
			"write": false,
		},
		Options: map[string]any{
			"key": "value",
		},
		Model: &ModelRef{
			ProviderID: "anthropic",
			ModelID:    "claude-3-sonnet",
		},
	}

	clone := original.Clone()

	// Verify values are equal
	assert.Equal(t, original.Name, clone.Name)
	assert.Equal(t, original.Description, clone.Description)
	assert.Equal(t, original.Mode, clone.Mode)
	assert.Equal(t, original.BuiltIn, clone.BuiltIn)
	assert.Equal(t, original.Temperature, clone.Temperature)
	assert.Equal(t, original.TopP, clone.TopP)
	assert.Equal(t, original.Prompt, clone.Prompt)
	assert.Equal(t, original.Color, clone.Color)
	assert.Equal(t, original.Permission.Edit, clone.Permission.Edit)
	assert.Equal(t, original.Model.ProviderID, clone.Model.ProviderID)
	assert.Equal(t, original.Model.ModelID, clone.Model.ModelID)

	// Verify maps are independent
	clone.Tools["read"] = false
	assert.True(t, original.Tools["read"], "modifying clone should not affect original")

	clone.Permission.Bash["new"] = permission.ActionAllow
	_, exists := original.Permission.Bash["new"]
	assert.False(t, exists, "modifying clone should not affect original")

	clone.Options["new"] = "value"
	_, exists = original.Options["new"]
	assert.False(t, exists, "modifying clone should not affect original")
}

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		pattern  string
		s        string
		expected bool
	}{
		{"*", "anything", true},
		{"*", "", true},
		{"prefix*", "prefix-hello", true},
		{"prefix*", "prefixworld", true},
		{"prefix*", "other", false},
		{"*suffix", "hello-suffix", true},
		{"*suffix", "worldsuffix", true},
		{"*suffix", "other", false},
		{"exact", "exact", true},
		{"exact", "different", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.s, func(t *testing.T) {
			result := matchWildcard(tt.pattern, tt.s)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuiltInAgents(t *testing.T) {
	agents := BuiltInAgents()

	// Verify expected agents exist
	expectedAgents := []string{"build", "plan", "general", "explore"}
	for _, name := range expectedAgents {
		agent, ok := agents[name]
		require.True(t, ok, "expected agent %s to exist", name)
		assert.True(t, agent.BuiltIn, "built-in agent should have BuiltIn=true")
	}

	// Verify build agent
	build := agents["build"]
	assert.Equal(t, ModePrimary, build.Mode)
	assert.Equal(t, permission.ActionAllow, build.Permission.Edit)

	// Verify plan agent
	plan := agents["plan"]
	assert.Equal(t, ModePrimary, plan.Mode)
	assert.Equal(t, permission.ActionDeny, plan.Permission.Edit)
	assert.False(t, plan.Tools["edit"])
	assert.False(t, plan.Tools["write"])

	// Verify general agent
	general := agents["general"]
	assert.Equal(t, ModeSubagent, general.Mode)
	assert.Equal(t, permission.ActionDeny, general.Permission.Edit)

	// Verify explore agent
	explore := agents["explore"]
	assert.Equal(t, ModeSubagent, explore.Mode)
	assert.True(t, explore.Tools["read"])
	assert.True(t, explore.Tools["glob"])
}
