package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/opencode-ai/opencode/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadTypeScriptConfig(t *testing.T) {
	// Create a temporary directory for test configs
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Isolate HOME to prevent loading other configs
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// TypeScript-style config with nested options
	tsConfig := `{
		"$schema": "https://opencode.ai/config.json",
		"model": "anthropic/claude-sonnet-4-20250514",
		"small_model": "anthropic/claude-3-5-haiku-20241022",
		"username": "testuser",
		"provider": {
			"anthropic": {
				"options": {
					"apiKey": "sk-ant-test123"
				}
			}
		},
		"agent": {
			"coder": {
				"temperature": 0.7,
				"top_p": 0.9,
				"tools": {
					"bash": true,
					"edit": true
				},
				"permission": {
					"edit": "allow",
					"bash": "ask"
				}
			}
		}
	}`

	// Write config to temp directory
	configPath := filepath.Join(tmpDir, ".opencode", "opencode.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(tsConfig), 0644))

	// Load config
	cfg, err := Load(tmpDir)
	require.NoError(t, err)

	// Verify TypeScript-style fields are parsed
	assert.Equal(t, "https://opencode.ai/config.json", cfg.Schema)
	assert.Equal(t, "anthropic/claude-sonnet-4-20250514", cfg.Model)
	assert.Equal(t, "anthropic/claude-3-5-haiku-20241022", cfg.SmallModel)
	assert.Equal(t, "testuser", cfg.Username)

	// Verify nested provider options are normalized
	anthropic := cfg.Provider["anthropic"]
	assert.Equal(t, "sk-ant-test123", anthropic.APIKey)

	// Verify agent config with top_p
	coder := cfg.Agent["coder"]
	assert.NotNil(t, coder.Temperature)
	assert.Equal(t, 0.7, *coder.Temperature)
	assert.NotNil(t, coder.TopP)
	assert.Equal(t, 0.9, *coder.TopP)
	assert.True(t, coder.Tools["bash"])
	assert.True(t, coder.Tools["edit"])
}

func TestLoadGoStyleConfig(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Isolate HOME
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Go-style config with direct fields
	goConfig := `{
		"model": "openai/gpt-4o",
		"provider": {
			"openai": {
				"apiKey": "sk-openai-test",
				"baseURL": "https://api.openai.com/v1"
			}
		}
	}`

	// Write config
	configPath := filepath.Join(tmpDir, ".opencode", "opencode.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(goConfig), 0644))

	// Load config
	cfg, err := Load(tmpDir)
	require.NoError(t, err)

	assert.Equal(t, "openai/gpt-4o", cfg.Model)
	assert.Equal(t, "sk-openai-test", cfg.Provider["openai"].APIKey)
	assert.Equal(t, "https://api.openai.com/v1", cfg.Provider["openai"].BaseURL)
}

func TestJSONCComments(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Isolate HOME
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// JSONC config with comments
	jsoncConfig := `{
		// This is a single-line comment
		"model": "anthropic/claude-sonnet-4-20250514",
		/* This is a
		   multi-line comment */
		"provider": {
			"anthropic": {
				"apiKey": "test-key" // inline comment
			}
		}
	}`

	// Write .jsonc file
	configPath := filepath.Join(tmpDir, ".opencode", "opencode.jsonc")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(jsoncConfig), 0644))

	// Load config
	cfg, err := Load(tmpDir)
	require.NoError(t, err)

	assert.Equal(t, "anthropic/claude-sonnet-4-20250514", cfg.Model)
	assert.Equal(t, "test-key", cfg.Provider["anthropic"].APIKey)
}

func TestEnvInterpolation(t *testing.T) {
	// Set test environment variable
	os.Setenv("TEST_API_KEY", "interpolated-key")
	defer os.Unsetenv("TEST_API_KEY")

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Isolate HOME
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Config with env interpolation
	config := `{
		"model": "anthropic/claude-sonnet-4",
		"provider": {
			"anthropic": {
				"apiKey": "{env:TEST_API_KEY}"
			}
		}
	}`

	configPath := filepath.Join(tmpDir, ".opencode", "opencode.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0644))

	// Load config
	cfg, err := Load(tmpDir)
	require.NoError(t, err)

	assert.Equal(t, "interpolated-key", cfg.Provider["anthropic"].APIKey)
}

func TestFileInterpolation(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Isolate HOME
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create a file to include
	instructionsFile := filepath.Join(tmpDir, "instructions.txt")
	require.NoError(t, os.WriteFile(instructionsFile, []byte("Custom instructions here"), 0644))

	// Config with file interpolation (relative path)
	config := `{
		"model": "anthropic/claude-sonnet-4",
		"instructions": ["{file:../instructions.txt}"]
	}`

	configDir := filepath.Join(tmpDir, ".opencode")
	configPath := filepath.Join(configDir, "opencode.json")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0644))

	// Load config
	cfg, err := Load(tmpDir)
	require.NoError(t, err)

	assert.Len(t, cfg.Instructions, 1)
	assert.Equal(t, "Custom instructions here", cfg.Instructions[0])
}

func TestConfigMerge(t *testing.T) {
	// Create temp directories for global and project configs
	tmpHome, err := os.MkdirTemp("", "opencode-home-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	tmpProject, err := os.MkdirTemp("", "opencode-project-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpProject)

	// Set HOME for test
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	// Global config
	globalConfig := `{
		"model": "anthropic/claude-sonnet-4",
		"provider": {
			"anthropic": {
				"apiKey": "global-key"
			}
		},
		"agent": {
			"coder": {
				"tools": {"bash": true}
			}
		}
	}`

	globalConfigDir := filepath.Join(tmpHome, ".opencode")
	require.NoError(t, os.MkdirAll(globalConfigDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(globalConfigDir, "opencode.json"), []byte(globalConfig), 0644))

	// Project config (should override)
	projectConfig := `{
		"model": "openai/gpt-4o",
		"agent": {
			"coder": {
				"tools": {"edit": true}
			}
		}
	}`

	projectConfigDir := filepath.Join(tmpProject, ".opencode")
	require.NoError(t, os.MkdirAll(projectConfigDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectConfigDir, "opencode.json"), []byte(projectConfig), 0644))

	// Load config
	cfg, err := Load(tmpProject)
	require.NoError(t, err)

	// Project model should override global
	assert.Equal(t, "openai/gpt-4o", cfg.Model)

	// Global provider should be preserved
	assert.Equal(t, "global-key", cfg.Provider["anthropic"].APIKey)

	// Agent tools should be merged (project overrides coder)
	assert.True(t, cfg.Agent["coder"].Tools["edit"])
}

func TestEnvVarOverride(t *testing.T) {
	// Set test environment variables
	os.Setenv("OPENCODE_MODEL", "env-model")
	os.Setenv("ANTHROPIC_API_KEY", "env-anthropic-key")
	defer func() {
		os.Unsetenv("OPENCODE_MODEL")
		os.Unsetenv("ANTHROPIC_API_KEY")
	}()

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Isolate HOME
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Config file
	config := `{
		"model": "file-model"
	}`

	configPath := filepath.Join(tmpDir, ".opencode", "opencode.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0644))

	// Load config
	cfg, err := Load(tmpDir)
	require.NoError(t, err)

	// Environment variable should override file config
	assert.Equal(t, "env-model", cfg.Model)

	// Provider key from env should be set
	assert.Equal(t, "env-anthropic-key", cfg.Provider["anthropic"].APIKey)
}

func TestOPENCODE_CONFIG(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Isolate HOME
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Custom config file
	customConfig := `{
		"model": "custom-config-model"
	}`

	customConfigPath := filepath.Join(tmpDir, "custom-config.json")
	require.NoError(t, os.WriteFile(customConfigPath, []byte(customConfig), 0644))

	// Set OPENCODE_CONFIG
	os.Setenv("OPENCODE_CONFIG", customConfigPath)
	defer os.Unsetenv("OPENCODE_CONFIG")

	// Load config (from a different directory)
	cfg, err := Load("/tmp")
	require.NoError(t, err)

	assert.Equal(t, "custom-config-model", cfg.Model)
}

func TestOPENCODE_CONFIG_CONTENT(t *testing.T) {
	// Create a temporary directory for HOME isolation
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Isolate HOME
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Set inline config
	inlineConfig := `{"model": "inline-model", "username": "inline-user"}`
	os.Setenv("OPENCODE_CONFIG_CONTENT", inlineConfig)
	defer os.Unsetenv("OPENCODE_CONFIG_CONTENT")

	// Load config
	cfg, err := Load("")
	require.NoError(t, err)

	assert.Equal(t, "inline-model", cfg.Model)
	assert.Equal(t, "inline-user", cfg.Username)
}

func TestMCPConfig(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Isolate HOME
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Config with MCP servers
	config := `{
		"model": "anthropic/claude-sonnet-4",
		"mcp": {
			"filesystem": {
				"type": "local",
				"command": ["npx", "-y", "@modelcontextprotocol/server-filesystem"],
				"environment": {
					"MCP_ROOT": "/home/user"
				},
				"enabled": true,
				"timeout": 5000
			},
			"remote-server": {
				"type": "remote",
				"url": "https://mcp.example.com",
				"headers": {
					"Authorization": "Bearer token"
				}
			}
		}
	}`

	configPath := filepath.Join(tmpDir, ".opencode", "opencode.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0644))

	// Load config
	cfg, err := Load(tmpDir)
	require.NoError(t, err)

	// Check local MCP
	fs := cfg.MCP["filesystem"]
	assert.Equal(t, "local", fs.Type)
	assert.Equal(t, []string{"npx", "-y", "@modelcontextprotocol/server-filesystem"}, fs.Command)
	assert.Equal(t, "/home/user", fs.Environment["MCP_ROOT"])
	assert.NotNil(t, fs.Enabled)
	assert.True(t, *fs.Enabled)
	assert.Equal(t, 5000, fs.Timeout)

	// Check remote MCP
	remote := cfg.MCP["remote-server"]
	assert.Equal(t, "remote", remote.Type)
	assert.Equal(t, "https://mcp.example.com", remote.URL)
	assert.Equal(t, "Bearer token", remote.Headers["Authorization"])
}

func TestCommandConfig(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Isolate HOME
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Config with custom commands
	config := `{
		"model": "anthropic/claude-sonnet-4",
		"command": {
			"review": {
				"template": "Review the code in this PR and provide feedback",
				"description": "Code review command",
				"agent": "coder"
			},
			"explain": {
				"template": "Explain this code: $FILE",
				"description": "Explain code",
				"model": "anthropic/claude-3-5-haiku-20241022"
			}
		}
	}`

	configPath := filepath.Join(tmpDir, ".opencode", "opencode.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0644))

	// Load config
	cfg, err := Load(tmpDir)
	require.NoError(t, err)

	review := cfg.Command["review"]
	assert.Equal(t, "Review the code in this PR and provide feedback", review.Template)
	assert.Equal(t, "Code review command", review.Description)
	assert.Equal(t, "coder", review.Agent)

	explain := cfg.Command["explain"]
	assert.Equal(t, "anthropic/claude-3-5-haiku-20241022", explain.Model)
}

func TestPermissionConfig(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Isolate HOME
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Config with permissions
	config := `{
		"model": "anthropic/claude-sonnet-4",
		"permission": {
			"edit": "allow",
			"bash": {
				"rm": "deny",
				"chmod": "ask",
				"git push": "deny"
			},
			"webfetch": "allow",
			"external_directory": "ask",
			"doom_loop": "ask"
		}
	}`

	configPath := filepath.Join(tmpDir, ".opencode", "opencode.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0644))

	// Load config
	cfg, err := Load(tmpDir)
	require.NoError(t, err)

	perm := cfg.Permission
	require.NotNil(t, perm)
	assert.Equal(t, "allow", perm.Edit)
	assert.Equal(t, "allow", perm.WebFetch)
	assert.Equal(t, "ask", perm.ExternalDir)
	assert.Equal(t, "ask", perm.DoomLoop)

	// Check bash permissions (can be map)
	bashPerm, ok := perm.Bash.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "deny", bashPerm["rm"])
	assert.Equal(t, "ask", bashPerm["chmod"])
}

func TestConfigSerialization(t *testing.T) {
	// Test that config can be serialized and deserialized correctly
	cfg := &types.Config{
		Schema:     "https://opencode.ai/config.json",
		Model:      "anthropic/claude-sonnet-4",
		SmallModel: "anthropic/claude-3-5-haiku",
		Username:   "testuser",
		Provider: map[string]types.ProviderConfig{
			"anthropic": {
				APIKey:  "test-key",
				BaseURL: "https://api.anthropic.com",
			},
		},
		Agent: map[string]types.AgentConfig{
			"coder": {
				Temperature: func() *float64 { v := 0.7; return &v }(),
				TopP:        func() *float64 { v := 0.9; return &v }(),
				Tools:       map[string]bool{"bash": true},
			},
		},
	}

	// Serialize
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)

	// Deserialize
	var loaded types.Config
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)

	assert.Equal(t, cfg.Schema, loaded.Schema)
	assert.Equal(t, cfg.Model, loaded.Model)
	assert.Equal(t, cfg.SmallModel, loaded.SmallModel)
	assert.Equal(t, cfg.Username, loaded.Username)
	assert.Equal(t, cfg.Provider["anthropic"].APIKey, loaded.Provider["anthropic"].APIKey)
	assert.Equal(t, *cfg.Agent["coder"].Temperature, *loaded.Agent["coder"].Temperature)
	assert.Equal(t, *cfg.Agent["coder"].TopP, *loaded.Agent["coder"].TopP)
}
