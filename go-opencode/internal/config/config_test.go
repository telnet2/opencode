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
				"npm": "@ai-sdk/anthropic",
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

	// Verify nested provider options
	anthropic := cfg.Provider["anthropic"]
	assert.Equal(t, "@ai-sdk/anthropic", anthropic.Npm)
	require.NotNil(t, anthropic.Options)
	assert.Equal(t, "sk-ant-test123", anthropic.Options.APIKey)

	// Verify agent config with top_p
	coder := cfg.Agent["coder"]
	assert.NotNil(t, coder.Temperature)
	assert.Equal(t, 0.7, *coder.Temperature)
	assert.NotNil(t, coder.TopP)
	assert.Equal(t, 0.9, *coder.TopP)
	assert.True(t, coder.Tools["bash"])
	assert.True(t, coder.Tools["edit"])
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
				"options": {
					"apiKey": "test-key" // inline comment
				}
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
	require.NotNil(t, cfg.Provider["anthropic"].Options)
	assert.Equal(t, "test-key", cfg.Provider["anthropic"].Options.APIKey)
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

	// Config with env interpolation using TypeScript-style options
	config := `{
		"model": "anthropic/claude-sonnet-4",
		"provider": {
			"anthropic": {
				"options": {
					"apiKey": "{env:TEST_API_KEY}"
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

	require.NotNil(t, cfg.Provider["anthropic"].Options)
	assert.Equal(t, "interpolated-key", cfg.Provider["anthropic"].Options.APIKey)
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

	// Global config with TypeScript-style options
	globalConfig := `{
		"model": "anthropic/claude-sonnet-4",
		"provider": {
			"anthropic": {
				"npm": "@ai-sdk/anthropic",
				"options": {
					"apiKey": "global-key"
				}
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
	require.NotNil(t, cfg.Provider["anthropic"].Options)
	assert.Equal(t, "global-key", cfg.Provider["anthropic"].Options.APIKey)

	// Agent tools should be merged (project overrides coder)
	assert.True(t, cfg.Agent["coder"].Tools["edit"])
}

func TestEnvVarOverride(t *testing.T) {
	// Set test environment variables
	os.Setenv("OPENCODE_MODEL", "env-model")
	defer os.Unsetenv("OPENCODE_MODEL")

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
				Npm: "@ai-sdk/anthropic",
				Options: &types.ProviderOptions{
					APIKey:  "test-key",
					BaseURL: "https://api.anthropic.com",
				},
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
	assert.Equal(t, "@ai-sdk/anthropic", loaded.Provider["anthropic"].Npm)
	require.NotNil(t, loaded.Provider["anthropic"].Options)
	assert.Equal(t, "test-key", loaded.Provider["anthropic"].Options.APIKey)
	assert.Equal(t, *cfg.Agent["coder"].Temperature, *loaded.Agent["coder"].Temperature)
	assert.Equal(t, *cfg.Agent["coder"].TopP, *loaded.Agent["coder"].TopP)
}

func TestOpenAICompatibleProvider(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Isolate HOME
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Config with OpenAI-compatible provider (like qwen)
	config := `{
		"model": "qwen/qwen-max",
		"provider": {
			"qwen": {
				"npm": "@ai-sdk/openai-compatible",
				"options": {
					"apiKey": "qwen-api-key",
					"baseURL": "https://dashscope.aliyuncs.com/compatible-mode/v1"
				},
				"models": {
					"qwen-max": {
						"id": "qwen-max",
						"reasoning": true,
						"tool_call": true
					}
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

	qwen := cfg.Provider["qwen"]
	assert.Equal(t, "@ai-sdk/openai-compatible", qwen.Npm)
	require.NotNil(t, qwen.Options)
	assert.Equal(t, "qwen-api-key", qwen.Options.APIKey)
	assert.Equal(t, "https://dashscope.aliyuncs.com/compatible-mode/v1", qwen.Options.BaseURL)

	// Check custom model config
	qwenMax := qwen.Models["qwen-max"]
	assert.Equal(t, "qwen-max", qwenMax.ID)
	assert.True(t, qwenMax.Reasoning)
	assert.True(t, qwenMax.ToolCall)
}

func TestProviderWithoutOptions(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Isolate HOME
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Config with provider but no options (should not panic)
	config := `{
		"model": "anthropic/claude-sonnet-4",
		"provider": {
			"anthropic": {
				"npm": "@ai-sdk/anthropic"
			}
		}
	}`

	configPath := filepath.Join(tmpDir, ".opencode", "opencode.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0644))

	// Load config - should not panic
	cfg, err := Load(tmpDir)
	require.NoError(t, err)

	anthropic := cfg.Provider["anthropic"]
	assert.Equal(t, "@ai-sdk/anthropic", anthropic.Npm)
	assert.Nil(t, anthropic.Options)
}

func TestInterpolateFunction(t *testing.T) {
	t.Run("interpolates env variables", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test-value")
		defer os.Unsetenv("TEST_VAR")

		input := []byte(`{"key": "{env:TEST_VAR}"}`)
		result := interpolate(input, "")

		assert.Equal(t, `{"key": "test-value"}`, string(result))
	})

	t.Run("handles missing env variables", func(t *testing.T) {
		os.Unsetenv("NONEXISTENT")

		input := []byte(`{"key": "{env:NONEXISTENT}"}`)
		result := interpolate(input, "")

		assert.Equal(t, `{"key": ""}`, string(result))
	})

	t.Run("interpolates multiple env variables", func(t *testing.T) {
		os.Setenv("VAR_A", "value-a")
		os.Setenv("VAR_B", "value-b")
		defer os.Unsetenv("VAR_A")
		defer os.Unsetenv("VAR_B")

		input := []byte(`{"a": "{env:VAR_A}", "b": "{env:VAR_B}"}`)
		result := interpolate(input, "")

		assert.Equal(t, `{"a": "value-a", "b": "value-b"}`, string(result))
	})

	t.Run("interpolates file contents", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretFile := filepath.Join(tmpDir, "secret.txt")
		err := os.WriteFile(secretFile, []byte("secret-content"), 0644)
		require.NoError(t, err)

		input := []byte(`{"key": "{file:secret.txt}"}`)
		result := interpolate(input, tmpDir)

		assert.Equal(t, `{"key": "secret-content"}`, string(result))
	})

	t.Run("handles missing file gracefully", func(t *testing.T) {
		input := []byte(`{"key": "{file:nonexistent.txt}"}`)
		result := interpolate(input, "/tmp")

		// Should keep original placeholder if file not found
		assert.Equal(t, `{"key": "{file:nonexistent.txt}"}`, string(result))
	})
}

func TestMergeConfigFunction(t *testing.T) {
	t.Run("merges providers", func(t *testing.T) {
		target := &types.Config{
			Provider: map[string]types.ProviderConfig{
				"anthropic": {Npm: "@ai-sdk/anthropic"},
			},
		}
		source := &types.Config{
			Provider: map[string]types.ProviderConfig{
				"openai": {Npm: "@ai-sdk/openai"},
			},
		}

		mergeConfig(target, source)

		assert.Len(t, target.Provider, 2)
		assert.Equal(t, "@ai-sdk/anthropic", target.Provider["anthropic"].Npm)
		assert.Equal(t, "@ai-sdk/openai", target.Provider["openai"].Npm)
	})

	t.Run("source overrides target for same key", func(t *testing.T) {
		target := &types.Config{
			Provider: map[string]types.ProviderConfig{
				"openai": {
					Npm: "@ai-sdk/openai",
					Options: &types.ProviderOptions{
						APIKey: "old-key",
					},
				},
			},
		}
		source := &types.Config{
			Provider: map[string]types.ProviderConfig{
				"openai": {
					Npm: "@ai-sdk/openai-compatible",
					Options: &types.ProviderOptions{
						APIKey:  "new-key",
						BaseURL: "https://custom.example.com",
					},
				},
			},
		}

		mergeConfig(target, source)

		openai := target.Provider["openai"]
		assert.Equal(t, "@ai-sdk/openai-compatible", openai.Npm)
		assert.Equal(t, "new-key", openai.Options.APIKey)
		assert.Equal(t, "https://custom.example.com", openai.Options.BaseURL)
	})

	t.Run("does not overwrite with empty model", func(t *testing.T) {
		target := &types.Config{
			Model: "anthropic/claude-sonnet-4",
		}
		source := &types.Config{
			SmallModel: "anthropic/claude-3-5-haiku",
		}

		mergeConfig(target, source)

		assert.Equal(t, "anthropic/claude-sonnet-4", target.Model)
		assert.Equal(t, "anthropic/claude-3-5-haiku", target.SmallModel)
	})
}

func TestApplyEnvOverridesFunction(t *testing.T) {
	t.Run("OPENCODE_MODEL overrides config", func(t *testing.T) {
		os.Setenv("OPENCODE_MODEL", "env-override-model")
		defer os.Unsetenv("OPENCODE_MODEL")

		config := &types.Config{
			Model:    "config-model",
			Provider: make(map[string]types.ProviderConfig),
		}

		applyEnvOverrides(config)

		assert.Equal(t, "env-override-model", config.Model)
	})

	t.Run("OPENCODE_SMALL_MODEL overrides config", func(t *testing.T) {
		os.Setenv("OPENCODE_SMALL_MODEL", "env-small-model")
		defer os.Unsetenv("OPENCODE_SMALL_MODEL")

		config := &types.Config{
			SmallModel: "config-small-model",
			Provider:   make(map[string]types.ProviderConfig),
		}

		applyEnvOverrides(config)

		assert.Equal(t, "env-small-model", config.SmallModel)
	})
}
