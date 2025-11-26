package types

// Config represents the OpenCode configuration.
// Compatible with TypeScript opencode configuration format.
type Config struct {
	// Schema reference (for editor support)
	Schema string `json:"$schema,omitempty"`

	// User identification
	Username string `json:"username,omitempty"`

	// Model selection
	Model      string `json:"model,omitempty"`       // "anthropic/claude-sonnet-4"
	SmallModel string `json:"small_model,omitempty"` // For fast tasks

	// Theme (TUI only, for compatibility)
	Theme string `json:"theme,omitempty"`

	// Sharing behavior
	Share string `json:"share,omitempty"` // "manual"|"auto"|"disabled"

	// Global tools enable/disable
	Tools map[string]bool `json:"tools,omitempty"`

	// Additional instruction files
	Instructions []string `json:"instructions,omitempty"`

	// Custom prompt variables
	PromptVariables map[string]string `json:"promptVariables,omitempty"`

	// Provider configs
	Provider map[string]ProviderConfig `json:"provider,omitempty"`

	// Agent configs
	Agent map[string]AgentConfig `json:"agent,omitempty"`

	// Command configs (custom slash commands)
	Command map[string]CommandConfig `json:"command,omitempty"`

	// Global permission settings
	Permission *PermissionConfig `json:"permission,omitempty"`

	// MCP server configs
	MCP map[string]MCPConfig `json:"mcp,omitempty"`

	// LSP
	LSP *LSPConfig `json:"lsp,omitempty"`

	// Formatter settings
	Formatter map[string]FormatterConfig `json:"formatter,omitempty"`

	// File watcher
	Watcher *WatcherConfig `json:"watcher,omitempty"`

	// Experimental features
	Experimental *ExperimentalConfig `json:"experimental,omitempty"`
}

// ProviderConfig holds configuration for a specific provider.
// Compatible with TypeScript opencode provider configuration.
type ProviderConfig struct {
	// Direct API key (Go style)
	APIKey  string `json:"apiKey,omitempty"`
	BaseURL string `json:"baseURL,omitempty"` // Changed to match TS (was baseUrl)

	// Model/Endpoint ID (for providers like ARK that require endpoint specification)
	Model string `json:"model,omitempty"`

	// Nested options (TypeScript style)
	Options *ProviderOptions `json:"options,omitempty"`

	// Model filtering
	Whitelist []string `json:"whitelist,omitempty"`
	Blacklist []string `json:"blacklist,omitempty"`

	// Disable provider
	Disable bool `json:"disable,omitempty"`
}

// ProviderOptions holds nested provider options (TypeScript style).
type ProviderOptions struct {
	APIKey        string `json:"apiKey,omitempty"`
	BaseURL       string `json:"baseURL,omitempty"`
	EnterpriseURL string `json:"enterpriseUrl,omitempty"`
	Timeout       *int   `json:"timeout,omitempty"` // ms, nil = default, 0 = disabled
}

// AgentConfig holds configuration for an agent.
// Compatible with TypeScript opencode agent configuration.
type AgentConfig struct {
	// Model override for this agent
	Model string `json:"model,omitempty"`

	// Generation parameters
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"` // Changed to match TS (was topP)

	// Custom system prompt
	Prompt string `json:"prompt,omitempty"`

	// Tool configuration
	Tools map[string]bool `json:"tools,omitempty"`

	// Permission settings
	Permission *PermissionConfig `json:"permission,omitempty"`

	// Agent metadata
	Description string `json:"description,omitempty"`
	Mode        string `json:"mode,omitempty"`  // "subagent"|"primary"|"all"
	Color       string `json:"color,omitempty"` // Hex color

	// Disable this agent
	Disable bool `json:"disable,omitempty"`
}

// PermissionConfig holds permission settings.
// Compatible with TypeScript opencode permission configuration.
type PermissionConfig struct {
	Edit        string      `json:"edit,omitempty"`               // "allow"|"deny"|"ask"
	Bash        interface{} `json:"bash,omitempty"`               // string or map[string]string
	WebFetch    string      `json:"webfetch,omitempty"`           // "allow"|"deny"|"ask"
	ExternalDir string      `json:"external_directory,omitempty"` // "allow"|"deny"|"ask"
	DoomLoop    string      `json:"doom_loop,omitempty"`          // "allow"|"deny"|"ask"
}

// Deprecated: Use PermissionConfig instead
type AgentPermissionConfig = PermissionConfig

// CommandConfig holds custom command configuration.
type CommandConfig struct {
	Template    string `json:"template"`
	Description string `json:"description,omitempty"`
	Agent       string `json:"agent,omitempty"`
	Model       string `json:"model,omitempty"`
	Subtask     bool   `json:"subtask,omitempty"`
}

// MCPConfig holds MCP server configuration.
type MCPConfig struct {
	Type        string            `json:"type,omitempty"` // "local"|"remote"
	Command     []string          `json:"command,omitempty"`
	URL         string            `json:"url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty"`
	Timeout     int               `json:"timeout,omitempty"`
}

// FormatterConfig holds code formatter configuration.
type FormatterConfig struct {
	Disabled    bool              `json:"disabled,omitempty"`
	Command     []string          `json:"command,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Extensions  []string          `json:"extensions,omitempty"`
}

// LSPConfig holds LSP server configuration.
type LSPConfig struct {
	Disabled bool              `json:"disabled,omitempty"`
	Servers  map[string]string `json:"servers,omitempty"` // language -> command
}

// WatcherConfig holds file watcher configuration.
type WatcherConfig struct {
	Ignore []string `json:"ignore,omitempty"`
}

// ExperimentalConfig holds experimental feature flags.
type ExperimentalConfig struct {
	BatchTool bool `json:"batch_tool,omitempty"`
}

// Model represents an LLM model available from a provider.
type Model struct {
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	ProviderID        string       `json:"providerID"`
	ContextLength     int          `json:"contextLength"`
	MaxOutputTokens   int          `json:"maxOutputTokens,omitempty"`
	SupportsTools     bool         `json:"supportsTools"`
	SupportsVision    bool         `json:"supportsVision"`
	SupportsReasoning bool         `json:"supportsReasoning,omitempty"`
	InputPrice        float64      `json:"inputPrice,omitempty"`  // per 1M tokens
	OutputPrice       float64      `json:"outputPrice,omitempty"` // per 1M tokens
	Options           ModelOptions `json:"options,omitempty"`
}

// ModelOptions contains model-specific options.
type ModelOptions struct {
	Temperature    *float64 `json:"temperature,omitempty"`
	TopP           *float64 `json:"topP,omitempty"`
	PromptCaching  bool     `json:"promptCaching,omitempty"`
	ExtendedOutput bool     `json:"extendedOutput,omitempty"`
}
