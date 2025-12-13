package types

// Config represents the OpenCode configuration.
// Compatible with TypeScript opencode configuration format.
type Config struct {
	// Schema reference (for editor support)
	Schema string `json:"$schema,omitempty"`

	// User identification
	Username string `json:"username,omitempty"`

	// Model selection
	Model      string `json:"model,omitempty"`      // "anthropic/claude-sonnet-4"
	SmallModel string `json:"smallModel,omitempty"` // For fast tasks (camelCase for TS compatibility)

	// Theme (TUI only, for compatibility)
	Theme string `json:"theme,omitempty"`

	// Keybinds (TUI shortcut configuration)
	Keybinds Keybinds `json:"keybinds"`

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
	// Npm package for the provider (TypeScript style)
	// Supported: @ai-sdk/openai, @ai-sdk/openai-compatible, @ai-sdk/anthropic
	Npm string `json:"npm,omitempty"`

	// Model/Endpoint ID (for providers like ARK that require endpoint specification)
	Model string `json:"model,omitempty"`

	// Nested options (TypeScript style)
	Options *ProviderOptions `json:"options,omitempty"`

	// Custom model definitions
	Models map[string]ModelConfig `json:"models,omitempty"`

	// Model filtering
	Whitelist []string `json:"whitelist,omitempty"`
	Blacklist []string `json:"blacklist,omitempty"`

	// Disable provider
	Disable bool `json:"disable,omitempty"`
}

// ModelConfig holds custom model configuration (TypeScript style).
type ModelConfig struct {
	ID        string `json:"id,omitempty"`
	Reasoning bool   `json:"reasoning,omitempty"`
	ToolCall  bool   `json:"toolcall,omitempty"` // No underscore - matches TS capabilities.toolcall
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
	TopP        *float64 `json:"topP,omitempty"` // camelCase for TS compatibility

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

// Keybinds defines TUI keyboard shortcuts. Keep field order and names aligned
// with the TypeScript config schema for compatibility.
type Keybinds struct {
	Leader                   string `json:"leader"`
	AppExit                  string `json:"app_exit"`
	EditorOpen               string `json:"editor_open"`
	ThemeList                string `json:"theme_list"`
	SidebarToggle            string `json:"sidebar_toggle"`
	UsernameToggle           string `json:"username_toggle"`
	StatusView               string `json:"status_view"`
	SessionExport            string `json:"session_export"`
	SessionNew               string `json:"session_new"`
	SessionList              string `json:"session_list"`
	SessionTimeline          string `json:"session_timeline"`
	SessionShare             string `json:"session_share"`
	SessionUnshare           string `json:"session_unshare"`
	SessionInterrupt         string `json:"session_interrupt"`
	SessionCompact           string `json:"session_compact"`
	MessagesPageUp           string `json:"messages_page_up"`
	MessagesPageDown         string `json:"messages_page_down"`
	MessagesHalfPageUp       string `json:"messages_half_page_up"`
	MessagesHalfPageDown     string `json:"messages_half_page_down"`
	MessagesFirst            string `json:"messages_first"`
	MessagesLast             string `json:"messages_last"`
	MessagesLastUser         string `json:"messages_last_user"`
	MessagesCopy             string `json:"messages_copy"`
	MessagesUndo             string `json:"messages_undo"`
	MessagesRedo             string `json:"messages_redo"`
	MessagesToggleConceal    string `json:"messages_toggle_conceal"`
	ToolDetails              string `json:"tool_details"`
	ModelList                string `json:"model_list"`
	ModelCycleRecent         string `json:"model_cycle_recent"`
	ModelCycleRecentReverse  string `json:"model_cycle_recent_reverse"`
	CommandList              string `json:"command_list"`
	AgentList                string `json:"agent_list"`
	AgentCycle               string `json:"agent_cycle"`
	AgentCycleReverse        string `json:"agent_cycle_reverse"`
	InputClear               string `json:"input_clear"`
	InputForwardDelete       string `json:"input_forward_delete"`
	InputPaste               string `json:"input_paste"`
	InputSubmit              string `json:"input_submit"`
	InputNewline             string `json:"input_newline"`
	HistoryPrevious          string `json:"history_previous"`
	HistoryNext              string `json:"history_next"`
	SessionChildCycle        string `json:"session_child_cycle"`
	SessionChildCycleReverse string `json:"session_child_cycle_reverse"`
	TerminalSuspend          string `json:"terminal_suspend"`
}

// DefaultKeybinds returns the default TUI keybindings, matching the TypeScript implementation.
func DefaultKeybinds() Keybinds {
	return Keybinds{
		Leader:                   "ctrl+x",
		AppExit:                  "ctrl+c,ctrl+d,<leader>q",
		EditorOpen:               "<leader>e",
		ThemeList:                "<leader>t",
		SidebarToggle:            "<leader>b",
		UsernameToggle:           "none",
		StatusView:               "<leader>s",
		SessionExport:            "<leader>x",
		SessionNew:               "<leader>n",
		SessionList:              "<leader>l",
		SessionTimeline:          "<leader>g",
		SessionShare:             "none",
		SessionUnshare:           "none",
		SessionInterrupt:         "escape",
		SessionCompact:           "<leader>c",
		MessagesPageUp:           "pageup",
		MessagesPageDown:         "pagedown",
		MessagesHalfPageUp:       "ctrl+alt+u",
		MessagesHalfPageDown:     "ctrl+alt+d",
		MessagesFirst:            "ctrl+g,home",
		MessagesLast:             "ctrl+alt+g,end",
		MessagesLastUser:         "none",
		MessagesCopy:             "<leader>y",
		MessagesUndo:             "<leader>u",
		MessagesRedo:             "<leader>r",
		MessagesToggleConceal:    "<leader>h",
		ToolDetails:              "none",
		ModelList:                "<leader>m",
		ModelCycleRecent:         "f2",
		ModelCycleRecentReverse:  "shift+f2",
		CommandList:              "ctrl+p",
		AgentList:                "<leader>a",
		AgentCycle:               "tab",
		AgentCycleReverse:        "shift+tab",
		InputClear:               "ctrl+c",
		InputForwardDelete:       "ctrl+d",
		InputPaste:               "ctrl+v",
		InputSubmit:              "return",
		InputNewline:             "shift+return,ctrl+j",
		HistoryPrevious:          "up",
		HistoryNext:              "down",
		SessionChildCycle:        "<leader>right",
		SessionChildCycleReverse: "<leader>left",
		TerminalSuspend:          "ctrl+z",
	}
}

// MergeKeybinds overlays overrides on top of base defaults, skipping empty values.
func MergeKeybinds(base, overrides Keybinds) Keybinds {
	if overrides.Leader != "" {
		base.Leader = overrides.Leader
	}
	if overrides.AppExit != "" {
		base.AppExit = overrides.AppExit
	}
	if overrides.EditorOpen != "" {
		base.EditorOpen = overrides.EditorOpen
	}
	if overrides.ThemeList != "" {
		base.ThemeList = overrides.ThemeList
	}
	if overrides.SidebarToggle != "" {
		base.SidebarToggle = overrides.SidebarToggle
	}
	if overrides.UsernameToggle != "" {
		base.UsernameToggle = overrides.UsernameToggle
	}
	if overrides.StatusView != "" {
		base.StatusView = overrides.StatusView
	}
	if overrides.SessionExport != "" {
		base.SessionExport = overrides.SessionExport
	}
	if overrides.SessionNew != "" {
		base.SessionNew = overrides.SessionNew
	}
	if overrides.SessionList != "" {
		base.SessionList = overrides.SessionList
	}
	if overrides.SessionTimeline != "" {
		base.SessionTimeline = overrides.SessionTimeline
	}
	if overrides.SessionShare != "" {
		base.SessionShare = overrides.SessionShare
	}
	if overrides.SessionUnshare != "" {
		base.SessionUnshare = overrides.SessionUnshare
	}
	if overrides.SessionInterrupt != "" {
		base.SessionInterrupt = overrides.SessionInterrupt
	}
	if overrides.SessionCompact != "" {
		base.SessionCompact = overrides.SessionCompact
	}
	if overrides.MessagesPageUp != "" {
		base.MessagesPageUp = overrides.MessagesPageUp
	}
	if overrides.MessagesPageDown != "" {
		base.MessagesPageDown = overrides.MessagesPageDown
	}
	if overrides.MessagesHalfPageUp != "" {
		base.MessagesHalfPageUp = overrides.MessagesHalfPageUp
	}
	if overrides.MessagesHalfPageDown != "" {
		base.MessagesHalfPageDown = overrides.MessagesHalfPageDown
	}
	if overrides.MessagesFirst != "" {
		base.MessagesFirst = overrides.MessagesFirst
	}
	if overrides.MessagesLast != "" {
		base.MessagesLast = overrides.MessagesLast
	}
	if overrides.MessagesLastUser != "" {
		base.MessagesLastUser = overrides.MessagesLastUser
	}
	if overrides.MessagesCopy != "" {
		base.MessagesCopy = overrides.MessagesCopy
	}
	if overrides.MessagesUndo != "" {
		base.MessagesUndo = overrides.MessagesUndo
	}
	if overrides.MessagesRedo != "" {
		base.MessagesRedo = overrides.MessagesRedo
	}
	if overrides.MessagesToggleConceal != "" {
		base.MessagesToggleConceal = overrides.MessagesToggleConceal
	}
	if overrides.ToolDetails != "" {
		base.ToolDetails = overrides.ToolDetails
	}
	if overrides.ModelList != "" {
		base.ModelList = overrides.ModelList
	}
	if overrides.ModelCycleRecent != "" {
		base.ModelCycleRecent = overrides.ModelCycleRecent
	}
	if overrides.ModelCycleRecentReverse != "" {
		base.ModelCycleRecentReverse = overrides.ModelCycleRecentReverse
	}
	if overrides.CommandList != "" {
		base.CommandList = overrides.CommandList
	}
	if overrides.AgentList != "" {
		base.AgentList = overrides.AgentList
	}
	if overrides.AgentCycle != "" {
		base.AgentCycle = overrides.AgentCycle
	}
	if overrides.AgentCycleReverse != "" {
		base.AgentCycleReverse = overrides.AgentCycleReverse
	}
	if overrides.InputClear != "" {
		base.InputClear = overrides.InputClear
	}
	if overrides.InputForwardDelete != "" {
		base.InputForwardDelete = overrides.InputForwardDelete
	}
	if overrides.InputPaste != "" {
		base.InputPaste = overrides.InputPaste
	}
	if overrides.InputSubmit != "" {
		base.InputSubmit = overrides.InputSubmit
	}
	if overrides.InputNewline != "" {
		base.InputNewline = overrides.InputNewline
	}
	if overrides.HistoryPrevious != "" {
		base.HistoryPrevious = overrides.HistoryPrevious
	}
	if overrides.HistoryNext != "" {
		base.HistoryNext = overrides.HistoryNext
	}
	if overrides.SessionChildCycle != "" {
		base.SessionChildCycle = overrides.SessionChildCycle
	}
	if overrides.SessionChildCycleReverse != "" {
		base.SessionChildCycleReverse = overrides.SessionChildCycleReverse
	}
	if overrides.TerminalSuspend != "" {
		base.TerminalSuspend = overrides.TerminalSuspend
	}

	return base
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
