package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"

	"github.com/opencode-ai/opencode/pkg/types"
)

// Load loads configuration from multiple sources (priority order):
// 1. Global config (~/.config/opencode/)
// 2. Project config (.opencode/)
// 3. Environment variables
func Load(directory string) (*types.Config, error) {
	config := &types.Config{
		Provider: make(map[string]types.ProviderConfig),
		Agent:    make(map[string]types.AgentConfig),
	}

	// 1. Global config
	globalPath := GetPaths().Config
	loadConfigFile(filepath.Join(globalPath, "opencode.json"), config)
	loadConfigFile(filepath.Join(globalPath, "opencode.jsonc"), config)

	// 2. Project config
	if directory != "" {
		loadConfigFile(filepath.Join(directory, ".opencode", "opencode.json"), config)
		loadConfigFile(filepath.Join(directory, ".opencode", "opencode.jsonc"), config)
	}

	// 3. Environment variables
	applyEnvOverrides(config)

	return config, nil
}

// loadConfigFile loads a single config file.
func loadConfigFile(path string, config *types.Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err // File doesn't exist, skip
	}

	// Strip JSONC comments if needed
	data = stripJSONComments(data)

	var fileConfig types.Config
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return err
	}

	mergeConfig(config, &fileConfig)
	return nil
}

// stripJSONComments removes // and /* */ comments from JSONC.
func stripJSONComments(data []byte) []byte {
	// Remove single-line comments
	singleLine := regexp.MustCompile(`//.*$`)
	lines := bytes.Split(data, []byte("\n"))
	for i, line := range lines {
		lines[i] = singleLine.ReplaceAll(line, nil)
	}
	data = bytes.Join(lines, []byte("\n"))

	// Remove multi-line comments
	multiLine := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	data = multiLine.ReplaceAll(data, nil)

	return data
}

// mergeConfig merges source config into target.
func mergeConfig(target, source *types.Config) {
	if source.Model != "" {
		target.Model = source.Model
	}
	if source.SmallModel != "" {
		target.SmallModel = source.SmallModel
	}

	// Merge providers
	if source.Provider != nil {
		if target.Provider == nil {
			target.Provider = make(map[string]types.ProviderConfig)
		}
		for k, v := range source.Provider {
			target.Provider[k] = v
		}
	}

	// Merge agents
	if source.Agent != nil {
		if target.Agent == nil {
			target.Agent = make(map[string]types.AgentConfig)
		}
		for k, v := range source.Agent {
			target.Agent[k] = v
		}
	}

	// Merge LSP config
	if source.LSP != nil {
		target.LSP = source.LSP
	}

	// Merge watcher config
	if source.Watcher != nil {
		target.Watcher = source.Watcher
	}

	// Merge experimental config
	if source.Experimental != nil {
		target.Experimental = source.Experimental
	}
}

// applyEnvOverrides applies environment variable overrides.
func applyEnvOverrides(config *types.Config) {
	// Provider API keys
	providerEnvMap := map[string]string{
		"anthropic": "ANTHROPIC_API_KEY",
		"openai":    "OPENAI_API_KEY",
		"google":    "GOOGLE_API_KEY",
		"bedrock":   "AWS_ACCESS_KEY_ID",
	}

	for provider, envVar := range providerEnvMap {
		if apiKey := os.Getenv(envVar); apiKey != "" {
			if config.Provider == nil {
				config.Provider = make(map[string]types.ProviderConfig)
			}
			p := config.Provider[provider]
			if p.APIKey == "" {
				p.APIKey = apiKey
				config.Provider[provider] = p
			}
		}
	}

	// Model override
	if model := os.Getenv("OPENCODE_MODEL"); model != "" {
		config.Model = model
	}

	// Small model override
	if smallModel := os.Getenv("OPENCODE_SMALL_MODEL"); smallModel != "" {
		config.SmallModel = smallModel
	}
}

// Save saves the configuration to a file.
func Save(config *types.Config, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
