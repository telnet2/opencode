package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/opencode-ai/opencode/pkg/types"
	"github.com/tidwall/jsonc"
)

// Load loads configuration from multiple sources (priority order):
// 1. Global config (~/.opencode/ - TypeScript compatible)
// 2. Global config (~/.config/opencode/ - XDG compatible)
// 3. Project config (.opencode/)
// 4. OPENCODE_CONFIG file
// 5. OPENCODE_CONFIG_CONTENT inline JSON
// 6. Environment variables
func Load(directory string) (*types.Config, error) {
	config := &types.Config{
		Provider: make(map[string]types.ProviderConfig),
		Agent:    make(map[string]types.AgentConfig),
	}

	// Track loaded files to avoid duplicates
	loaded := make(map[string]bool)

	loadOnce := func(path string, baseDir string) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return
		}
		if loaded[absPath] {
			return
		}
		if loadConfigFile(path, config, baseDir) == nil {
			loaded[absPath] = true
		}
	}

	// 1. TypeScript-compatible global config (~/.opencode/)
	home := os.Getenv("HOME")
	if home != "" {
		tsConfigDir := filepath.Join(home, ".opencode")
		loadOnce(filepath.Join(tsConfigDir, "config.json"), tsConfigDir)
		loadOnce(filepath.Join(tsConfigDir, "opencode.json"), tsConfigDir)
		loadOnce(filepath.Join(tsConfigDir, "opencode.jsonc"), tsConfigDir)
	}

	// 2. XDG-compatible global config (~/.config/opencode/)
	globalPath := GetPaths().Config
	loadOnce(filepath.Join(globalPath, "opencode.json"), globalPath)
	loadOnce(filepath.Join(globalPath, "opencode.jsonc"), globalPath)

	// 3. Project config
	if directory != "" {
		projectConfigDir := filepath.Join(directory, ".opencode")
		loadOnce(filepath.Join(directory, "opencode.json"), directory)
		loadOnce(filepath.Join(directory, "opencode.jsonc"), directory)
		loadOnce(filepath.Join(projectConfigDir, "opencode.json"), projectConfigDir)
		loadOnce(filepath.Join(projectConfigDir, "opencode.jsonc"), projectConfigDir)
	}

	// 4. OPENCODE_CONFIG file override
	if configPath := os.Getenv("OPENCODE_CONFIG"); configPath != "" {
		configDir := filepath.Dir(configPath)
		loadOnce(configPath, configDir)
	}

	// 5. OPENCODE_CONFIG_CONTENT inline JSON
	if configContent := os.Getenv("OPENCODE_CONFIG_CONTENT"); configContent != "" {
		var inlineConfig types.Config
		if err := json.Unmarshal([]byte(configContent), &inlineConfig); err == nil {
			mergeConfig(config, &inlineConfig)
		}
	}

	// 6. Environment variables (highest priority)
	applyEnvOverrides(config)

	// Normalize provider config (merge Options into direct fields)
	normalizeProviderConfig(config)

	return config, nil
}

// loadConfigFile loads a single config file with interpolation support.
func loadConfigFile(path string, config *types.Config, baseDir string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err // File doesn't exist, skip
	}

	// Strip JSONC comments using tidwall/jsonc
	data = jsonc.ToJSON(data)

	// Apply interpolation
	data = interpolate(data, baseDir)

	var fileConfig types.Config
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return err
	}

	mergeConfig(config, &fileConfig)
	return nil
}

// interpolate processes {env:VAR} and {file:path} placeholders.
func interpolate(data []byte, baseDir string) []byte {
	str := string(data)

	// Handle {env:VAR_NAME} placeholders
	envPattern := regexp.MustCompile(`\{env:([^}]+)\}`)
	str = envPattern.ReplaceAllStringFunc(str, func(match string) string {
		varName := envPattern.FindStringSubmatch(match)[1]
		return os.Getenv(varName)
	})

	// Handle {file:path} placeholders
	filePattern := regexp.MustCompile(`\{file:([^}]+)\}`)
	str = filePattern.ReplaceAllStringFunc(str, func(match string) string {
		filePath := filePattern.FindStringSubmatch(match)[1]

		// Resolve path
		if strings.HasPrefix(filePath, "~/") {
			home := os.Getenv("HOME")
			filePath = filepath.Join(home, filePath[2:])
		} else if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(baseDir, filePath)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return match // Keep original if file not found
		}

		// Escape for JSON string
		escaped := strings.ReplaceAll(string(content), "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		escaped = strings.ReplaceAll(escaped, "\n", "\\n")
		escaped = strings.ReplaceAll(escaped, "\r", "\\r")
		escaped = strings.ReplaceAll(escaped, "\t", "\\t")

		return escaped
	})

	return []byte(str)
}

// normalizeProviderConfig merges Options fields into direct fields for compatibility.
func normalizeProviderConfig(config *types.Config) {
	for name, provider := range config.Provider {
		if provider.Options != nil {
			// Options take precedence over direct fields
			if provider.Options.APIKey != "" {
				provider.APIKey = provider.Options.APIKey
			}
			if provider.Options.BaseURL != "" {
				provider.BaseURL = provider.Options.BaseURL
			}
		}
		config.Provider[name] = provider
	}
}

// mergeConfig merges source config into target.
func mergeConfig(target, source *types.Config) {
	if source.Schema != "" {
		target.Schema = source.Schema
	}
	if source.Username != "" {
		target.Username = source.Username
	}
	if source.Model != "" {
		target.Model = source.Model
	}
	if source.SmallModel != "" {
		target.SmallModel = source.SmallModel
	}
	if source.Theme != "" {
		target.Theme = source.Theme
	}
	if source.Share != "" {
		target.Share = source.Share
	}

	// Merge tools
	if source.Tools != nil {
		if target.Tools == nil {
			target.Tools = make(map[string]bool)
		}
		for k, v := range source.Tools {
			target.Tools[k] = v
		}
	}

	// Merge instructions
	if len(source.Instructions) > 0 {
		target.Instructions = append(target.Instructions, source.Instructions...)
	}

	// Merge prompt variables
	if source.PromptVariables != nil {
		if target.PromptVariables == nil {
			target.PromptVariables = make(map[string]string)
		}
		for k, v := range source.PromptVariables {
			target.PromptVariables[k] = v
		}
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

	// Merge commands
	if source.Command != nil {
		if target.Command == nil {
			target.Command = make(map[string]types.CommandConfig)
		}
		for k, v := range source.Command {
			target.Command[k] = v
		}
	}

	// Merge MCP
	if source.MCP != nil {
		if target.MCP == nil {
			target.MCP = make(map[string]types.MCPConfig)
		}
		for k, v := range source.MCP {
			target.MCP[k] = v
		}
	}

	// Merge formatter
	if source.Formatter != nil {
		if target.Formatter == nil {
			target.Formatter = make(map[string]types.FormatterConfig)
		}
		for k, v := range source.Formatter {
			target.Formatter[k] = v
		}
	}

	// Merge permission
	if source.Permission != nil {
		target.Permission = source.Permission
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

	// Permission override (JSON)
	if permJSON := os.Getenv("OPENCODE_PERMISSION"); permJSON != "" {
		var perm types.PermissionConfig
		if err := json.Unmarshal([]byte(permJSON), &perm); err == nil {
			config.Permission = &perm
		}
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

// GetConfigDir returns the config directory to use.
// Prefers OPENCODE_CONFIG_DIR, then ~/.opencode, then ~/.config/opencode.
func GetConfigDir() string {
	// Check environment variable first
	if dir := os.Getenv("OPENCODE_CONFIG_DIR"); dir != "" {
		return dir
	}

	// Check for TypeScript-compatible location
	home := os.Getenv("HOME")
	if home != "" {
		tsDir := filepath.Join(home, ".opencode")
		if _, err := os.Stat(tsDir); err == nil {
			return tsDir
		}
	}

	// Fall back to XDG location
	return GetPaths().Config
}
