package types

// Config represents the OpenCode configuration.
type Config struct {
	// Model selection
	Model      string `json:"model,omitempty"`       // "anthropic/claude-sonnet-4"
	SmallModel string `json:"small_model,omitempty"` // For fast tasks

	// Provider configs
	Provider map[string]ProviderConfig `json:"provider,omitempty"`

	// Agent configs
	Agent map[string]AgentConfig `json:"agent,omitempty"`

	// LSP
	LSP *LSPConfig `json:"lsp,omitempty"`

	// File watcher
	Watcher *WatcherConfig `json:"watcher,omitempty"`

	// Experimental features
	Experimental *ExperimentalConfig `json:"experimental,omitempty"`
}

// ProviderConfig holds configuration for a specific provider.
type ProviderConfig struct {
	APIKey  string `json:"apiKey,omitempty"`
	BaseURL string `json:"baseUrl,omitempty"`
	Disable bool   `json:"disable,omitempty"`
}

// AgentConfig holds configuration for an agent.
type AgentConfig struct {
	Tools      map[string]bool       `json:"tools,omitempty"`
	Permission AgentPermissionConfig `json:"permission,omitempty"`
}

// AgentPermissionConfig holds permission settings for an agent.
type AgentPermissionConfig struct {
	Edit        string            `json:"edit,omitempty"`    // "allow"|"deny"|"ask"
	Bash        map[string]string `json:"bash,omitempty"`    // pattern -> action
	WebFetch    string            `json:"webfetch,omitempty"`
	ExternalDir string            `json:"external_directory,omitempty"`
	DoomLoop    string            `json:"doom_loop,omitempty"`
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
