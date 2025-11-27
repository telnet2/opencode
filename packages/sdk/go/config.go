// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package opencode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"

	"github.com/sst/opencode-sdk-go/internal/apijson"
	"github.com/sst/opencode-sdk-go/internal/apiquery"
	shimjson "github.com/sst/opencode-sdk-go/internal/encoding/json"
	"github.com/sst/opencode-sdk-go/internal/requestconfig"
	"github.com/sst/opencode-sdk-go/option"
	"github.com/sst/opencode-sdk-go/packages/param"
	"github.com/sst/opencode-sdk-go/packages/respjson"
	"github.com/sst/opencode-sdk-go/shared"
	"github.com/sst/opencode-sdk-go/shared/constant"
)

// ConfigService contains methods and other services that help with interacting
// with the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewConfigService] method instead.
type ConfigService struct {
	Options []option.RequestOption
}

// NewConfigService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewConfigService(opts ...option.RequestOption) (r ConfigService) {
	r = ConfigService{}
	r.Options = opts
	return
}

// Get config info
func (r *ConfigService) Get(ctx context.Context, query ConfigGetParams, opts ...option.RequestOption) (res *Configuration, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "config"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

// Update config
func (r *ConfigService) Update(ctx context.Context, params ConfigUpdateParams, opts ...option.RequestOption) (res *Configuration, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "config"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPatch, path, params, &res, opts...)
	return
}

// List all providers
func (r *ConfigService) ListProviders(ctx context.Context, query ConfigListProvidersParams, opts ...option.RequestOption) (res *ConfigListProvidersResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "config/providers"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

type AgentConfig struct {
	// Hex color code for the agent (e.g., #FF5733)
	Color string `json:"color"`
	// Description of when to use the agent
	Description string `json:"description"`
	Disable     bool   `json:"disable"`
	// Any of "subagent", "primary", "all".
	Mode        AgentConfigMode       `json:"mode"`
	Model       string                `json:"model"`
	Permission  AgentConfigPermission `json:"permission"`
	Prompt      string                `json:"prompt"`
	Temperature float64               `json:"temperature"`
	Tools       map[string]bool       `json:"tools"`
	TopP        float64               `json:"top_p"`
	ExtraFields map[string]any        `json:",extras"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Color       respjson.Field
		Description respjson.Field
		Disable     respjson.Field
		Mode        respjson.Field
		Model       respjson.Field
		Permission  respjson.Field
		Prompt      respjson.Field
		Temperature respjson.Field
		Tools       respjson.Field
		TopP        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r AgentConfig) RawJSON() string { return r.JSON.raw }
func (r *AgentConfig) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this AgentConfig to a AgentConfigParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// AgentConfigParam.Overrides()
func (r AgentConfig) ToParam() AgentConfigParam {
	return param.Override[AgentConfigParam](json.RawMessage(r.RawJSON()))
}

type AgentConfigMode string

const (
	AgentConfigModeSubagent AgentConfigMode = "subagent"
	AgentConfigModePrimary  AgentConfigMode = "primary"
	AgentConfigModeAll      AgentConfigMode = "all"
)

type AgentConfigPermission struct {
	Bash AgentConfigPermissionBashUnion `json:"bash"`
	// Any of "ask", "allow", "deny".
	DoomLoop string `json:"doom_loop"`
	// Any of "ask", "allow", "deny".
	Edit string `json:"edit"`
	// Any of "ask", "allow", "deny".
	ExternalDirectory string `json:"external_directory"`
	// Any of "ask", "allow", "deny".
	Webfetch string `json:"webfetch"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Bash              respjson.Field
		DoomLoop          respjson.Field
		Edit              respjson.Field
		ExternalDirectory respjson.Field
		Webfetch          respjson.Field
		ExtraFields       map[string]respjson.Field
		raw               string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r AgentConfigPermission) RawJSON() string { return r.JSON.raw }
func (r *AgentConfigPermission) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// AgentConfigPermissionBashUnion represents a union type that can be either a
// string ("ask", "allow", "deny") or a map[string]string for specific permissions.
type AgentConfigPermissionBashUnion struct {
	// This field will be present if the value is a string instead of a map.
	OfString string `json:",inline"`
	// This field will be present if the value is a map instead of a string.
	OfMap map[string]string `json:",inline"`
	JSON  struct {
		OfString respjson.Field
		OfMap    respjson.Field
		raw      string
	} `json:"-"`
}

func (u AgentConfigPermissionBashUnion) AsString() (v string) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u AgentConfigPermissionBashUnion) AsMap() (v map[string]string) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u AgentConfigPermissionBashUnion) RawJSON() string { return u.JSON.raw }

func (u *AgentConfigPermissionBashUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

type AgentConfigPermissionBashString string

const (
	AgentConfigPermissionBashStringAsk   AgentConfigPermissionBashString = "ask"
	AgentConfigPermissionBashStringAllow AgentConfigPermissionBashString = "allow"
	AgentConfigPermissionBashStringDeny  AgentConfigPermissionBashString = "deny"
)

type AgentConfigParam struct {
	// Hex color code for the agent (e.g., #FF5733)
	Color param.Opt[string] `json:"color,omitzero"`
	// Description of when to use the agent
	Description param.Opt[string]  `json:"description,omitzero"`
	Disable     param.Opt[bool]    `json:"disable,omitzero"`
	Model       param.Opt[string]  `json:"model,omitzero"`
	Prompt      param.Opt[string]  `json:"prompt,omitzero"`
	Temperature param.Opt[float64] `json:"temperature,omitzero"`
	TopP        param.Opt[float64] `json:"top_p,omitzero"`
	// Any of "subagent", "primary", "all".
	Mode        AgentConfigMode            `json:"mode,omitzero"`
	Permission  AgentConfigPermissionParam `json:"permission,omitzero"`
	Tools       map[string]bool            `json:"tools,omitzero"`
	ExtraFields map[string]any             `json:"-"`
	paramObj
}

func (r AgentConfigParam) MarshalJSON() (data []byte, err error) {
	type shadow AgentConfigParam
	return param.MarshalWithExtras(r, (*shadow)(&r), r.ExtraFields)
}
func (r *AgentConfigParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type AgentConfigPermissionParam struct {
	Bash AgentConfigPermissionBashUnionParam `json:"bash,omitzero"`
	// Any of "ask", "allow", "deny".
	DoomLoop string `json:"doom_loop,omitzero"`
	// Any of "ask", "allow", "deny".
	Edit string `json:"edit,omitzero"`
	// Any of "ask", "allow", "deny".
	ExternalDirectory string `json:"external_directory,omitzero"`
	// Any of "ask", "allow", "deny".
	Webfetch string `json:"webfetch,omitzero"`
	paramObj
}

func (r AgentConfigPermissionParam) MarshalJSON() (data []byte, err error) {
	type shadow AgentConfigPermissionParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *AgentConfigPermissionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func init() {
	apijson.RegisterFieldValidator[AgentConfigPermissionParam](
		"doom_loop", "ask", "allow", "deny",
	)
	apijson.RegisterFieldValidator[AgentConfigPermissionParam](
		"edit", "ask", "allow", "deny",
	)
	apijson.RegisterFieldValidator[AgentConfigPermissionParam](
		"external_directory", "ask", "allow", "deny",
	)
	apijson.RegisterFieldValidator[AgentConfigPermissionParam](
		"webfetch", "ask", "allow", "deny",
	)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type AgentConfigPermissionBashUnionParam struct {
	// Check if union is this variant with
	// !param.IsOmitted(union.OfAgentConfigPermissionBashString)
	OfAgentConfigPermissionBashString     param.Opt[string] `json:",omitzero,inline"`
	OfAgentConfigPermissionBashMapItemMap map[string]string `json:",omitzero,inline"`
	paramUnion
}

func (u AgentConfigPermissionBashUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfAgentConfigPermissionBashString, u.OfAgentConfigPermissionBashMapItemMap)
}
func (u *AgentConfigPermissionBashUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *AgentConfigPermissionBashUnionParam) asAny() any {
	if !param.IsOmitted(u.OfAgentConfigPermissionBashString) {
		return &u.OfAgentConfigPermissionBashString
	} else if !param.IsOmitted(u.OfAgentConfigPermissionBashMapItemMap) {
		return &u.OfAgentConfigPermissionBashMapItemMap
	}
	return nil
}

type Configuration struct {
	// JSON schema reference for configuration validation
	Schema string `json:"$schema"`
	// Agent configuration, see https://opencode.ai/docs/agent
	Agent ConfigurationAgent `json:"agent"`
	// @deprecated Use 'share' field instead. Share newly created sessions
	// automatically
	Autoshare bool `json:"autoshare"`
	// Automatically update to the latest version. Set to true to auto-update, false to
	// disable, or 'notify' to show update notifications
	Autoupdate ConfigurationAutoupdateUnion `json:"autoupdate"`
	// Command configuration, see https://opencode.ai/docs/commands
	Command map[string]ConfigurationCommand `json:"command"`
	// Disable providers that are loaded automatically
	DisabledProviders []string `json:"disabled_providers"`
	// When set, ONLY these providers will be enabled. All other providers will be
	// ignored
	EnabledProviders []string                    `json:"enabled_providers"`
	Enterprise       ConfigurationEnterprise     `json:"enterprise"`
	Experimental     ConfigurationExperimental   `json:"experimental"`
	Formatter        ConfigurationFormatterUnion `json:"formatter"`
	// Additional instruction files or patterns to include
	Instructions []string `json:"instructions"`
	// Custom keybind configurations
	Keybinds ConfigurationKeybinds `json:"keybinds"`
	// @deprecated Always uses stretch layout.
	//
	// Any of "auto", "stretch".
	Layout ConfigurationLayout   `json:"layout"`
	Lsp    ConfigurationLspUnion `json:"lsp"`
	// MCP (Model Context Protocol) server configurations
	Mcp map[string]ConfigurationMcpUnion `json:"mcp"`
	// @deprecated Use `agent` field instead.
	Mode ConfigurationMode `json:"mode"`
	// Model to use in the format of provider/model, eg anthropic/claude-2
	Model      string                  `json:"model"`
	Permission ConfigurationPermission `json:"permission"`
	Plugin     []string                `json:"plugin"`
	// Custom variables for prompt template interpolation (e.g., COMPANY_NAME,
	// TEAM_NAME)
	PromptVariables map[string]string `json:"promptVariables"`
	// Custom provider configurations and model overrides
	Provider map[string]ConfigurationProvider `json:"provider"`
	// Control sharing behavior:'manual' allows manual sharing via commands, 'auto'
	// enables automatic sharing, 'disabled' disables all sharing
	//
	// Any of "manual", "auto", "disabled".
	Share ConfigurationShare `json:"share"`
	// Small model to use for tasks like title generation in the format of
	// provider/model
	SmallModel string `json:"small_model"`
	Snapshot   bool   `json:"snapshot"`
	// Theme name to use for the interface
	Theme string          `json:"theme"`
	Tools map[string]bool `json:"tools"`
	// TUI specific settings
	Tui ConfigurationTui `json:"tui"`
	// Custom username to display in conversations instead of system username
	Username string               `json:"username"`
	Watcher  ConfigurationWatcher `json:"watcher"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Schema            respjson.Field
		Agent             respjson.Field
		Autoshare         respjson.Field
		Autoupdate        respjson.Field
		Command           respjson.Field
		DisabledProviders respjson.Field
		EnabledProviders  respjson.Field
		Enterprise        respjson.Field
		Experimental      respjson.Field
		Formatter         respjson.Field
		Instructions      respjson.Field
		Keybinds          respjson.Field
		Layout            respjson.Field
		Lsp               respjson.Field
		Mcp               respjson.Field
		Mode              respjson.Field
		Model             respjson.Field
		Permission        respjson.Field
		Plugin            respjson.Field
		PromptVariables   respjson.Field
		Provider          respjson.Field
		Share             respjson.Field
		SmallModel        respjson.Field
		Snapshot          respjson.Field
		Theme             respjson.Field
		Tools             respjson.Field
		Tui               respjson.Field
		Username          respjson.Field
		Watcher           respjson.Field
		ExtraFields       map[string]respjson.Field
		raw               string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r Configuration) RawJSON() string { return r.JSON.raw }
func (r *Configuration) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this Configuration to a ConfigurationParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// ConfigurationParam.Overrides()
func (r Configuration) ToParam() ConfigurationParam {
	return param.Override[ConfigurationParam](json.RawMessage(r.RawJSON()))
}

// Agent configuration, see https://opencode.ai/docs/agent
type ConfigurationAgent struct {
	Build       AgentConfig            `json:"build"`
	General     AgentConfig            `json:"general"`
	Plan        AgentConfig            `json:"plan"`
	ExtraFields map[string]AgentConfig `json:",extras"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Build       respjson.Field
		General     respjson.Field
		Plan        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationAgent) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationAgent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ConfigurationAutoupdateUnion contains all possible properties and values from
// [bool], [constant.Notify].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfBool OfNotify]
type ConfigurationAutoupdateUnion struct {
	// This field will be present if the value is a [bool] instead of an object.
	OfBool bool `json:",inline"`
	// This field will be present if the value is a [constant.Notify] instead of an
	// object.
	OfNotify constant.Notify `json:",inline"`
	JSON     struct {
		OfBool   respjson.Field
		OfNotify respjson.Field
		raw      string
	} `json:"-"`
}

func (u ConfigurationAutoupdateUnion) AsBool() (v bool) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConfigurationAutoupdateUnion) AsNotify() (v constant.Notify) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u ConfigurationAutoupdateUnion) RawJSON() string { return u.JSON.raw }

func (r *ConfigurationAutoupdateUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationCommand struct {
	Template    string `json:"template,required"`
	Agent       string `json:"agent"`
	Description string `json:"description"`
	Model       string `json:"model"`
	Subtask     bool   `json:"subtask"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Template    respjson.Field
		Agent       respjson.Field
		Description respjson.Field
		Model       respjson.Field
		Subtask     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationCommand) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationCommand) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationEnterprise struct {
	// Enterprise URL
	URL string `json:"url"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		URL         respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationEnterprise) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationEnterprise) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationExperimental struct {
	// Enable the batch tool
	BatchTool bool `json:"batch_tool"`
	// Number of retries for chat completions on failure
	ChatMaxRetries      float64                       `json:"chatMaxRetries"`
	DisablePasteSummary bool                          `json:"disable_paste_summary"`
	Hook                ConfigurationExperimentalHook `json:"hook"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		BatchTool           respjson.Field
		ChatMaxRetries      respjson.Field
		DisablePasteSummary respjson.Field
		Hook                respjson.Field
		ExtraFields         map[string]respjson.Field
		raw                 string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationExperimental) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationExperimental) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationExperimentalHook struct {
	FileEdited       map[string][]ConfigurationExperimentalHookFileEdited `json:"file_edited"`
	SessionCompleted []ConfigurationExperimentalHookSessionCompleted      `json:"session_completed"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		FileEdited       respjson.Field
		SessionCompleted respjson.Field
		ExtraFields      map[string]respjson.Field
		raw              string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationExperimentalHook) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationExperimentalHook) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationExperimentalHookFileEdited struct {
	Command     []string          `json:"command,required"`
	Environment map[string]string `json:"environment"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Command     respjson.Field
		Environment respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationExperimentalHookFileEdited) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationExperimentalHookFileEdited) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationExperimentalHookSessionCompleted struct {
	Command     []string          `json:"command,required"`
	Environment map[string]string `json:"environment"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Command     respjson.Field
		Environment respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationExperimentalHookSessionCompleted) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationExperimentalHookSessionCompleted) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ConfigurationFormatterUnion contains all possible properties and values from
// [bool], [map[string]ConfigurationFormatterMapItem].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfBool]
type ConfigurationFormatterUnion struct {
	// This field will be present if the value is a [bool] instead of an object.
	OfBool bool `json:",inline"`
	// This field is from variant [map[string]ConfigurationFormatterMapItem].
	Command []string `json:"command"`
	// This field is from variant [map[string]ConfigurationFormatterMapItem].
	Disabled bool `json:"disabled"`
	// This field is from variant [map[string]ConfigurationFormatterMapItem].
	Environment map[string]string `json:"environment"`
	// This field is from variant [map[string]ConfigurationFormatterMapItem].
	Extensions []string `json:"extensions"`
	JSON       struct {
		OfBool      respjson.Field
		Command     respjson.Field
		Disabled    respjson.Field
		Environment respjson.Field
		Extensions  respjson.Field
		raw         string
	} `json:"-"`
}

func (u ConfigurationFormatterUnion) AsBool() (v bool) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConfigurationFormatterUnion) AsConfigurationFormatterMapMap() (v map[string]ConfigurationFormatterMapItem) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u ConfigurationFormatterUnion) RawJSON() string { return u.JSON.raw }

func (r *ConfigurationFormatterUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationFormatterMapItem struct {
	Command     []string          `json:"command"`
	Disabled    bool              `json:"disabled"`
	Environment map[string]string `json:"environment"`
	Extensions  []string          `json:"extensions"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Command     respjson.Field
		Disabled    respjson.Field
		Environment respjson.Field
		Extensions  respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationFormatterMapItem) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationFormatterMapItem) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Custom keybind configurations
type ConfigurationKeybinds struct {
	// Next agent
	AgentCycle string `json:"agent_cycle"`
	// Previous agent
	AgentCycleReverse string `json:"agent_cycle_reverse"`
	// List agents
	AgentList string `json:"agent_list"`
	// Exit the application
	AppExit string `json:"app_exit"`
	// List available commands
	CommandList string `json:"command_list"`
	// Open external editor
	EditorOpen string `json:"editor_open"`
	// Next history item
	HistoryNext string `json:"history_next"`
	// Previous history item
	HistoryPrevious string `json:"history_previous"`
	// Clear input field
	InputClear string `json:"input_clear"`
	// Forward delete
	InputForwardDelete string `json:"input_forward_delete"`
	// Insert newline in input
	InputNewline string `json:"input_newline"`
	// Paste from clipboard
	InputPaste string `json:"input_paste"`
	// Submit input
	InputSubmit string `json:"input_submit"`
	// Leader key for keybind combinations
	Leader string `json:"leader"`
	// Copy message
	MessagesCopy string `json:"messages_copy"`
	// Navigate to first message
	MessagesFirst string `json:"messages_first"`
	// Scroll messages down by half page
	MessagesHalfPageDown string `json:"messages_half_page_down"`
	// Scroll messages up by half page
	MessagesHalfPageUp string `json:"messages_half_page_up"`
	// Navigate to last message
	MessagesLast string `json:"messages_last"`
	// Scroll messages down by one page
	MessagesPageDown string `json:"messages_page_down"`
	// Scroll messages up by one page
	MessagesPageUp string `json:"messages_page_up"`
	// Redo message
	MessagesRedo string `json:"messages_redo"`
	// Toggle code block concealment in messages
	MessagesToggleConceal string `json:"messages_toggle_conceal"`
	// Undo message
	MessagesUndo string `json:"messages_undo"`
	// Next recently used model
	ModelCycleRecent string `json:"model_cycle_recent"`
	// Previous recently used model
	ModelCycleRecentReverse string `json:"model_cycle_recent_reverse"`
	// List available models
	ModelList string `json:"model_list"`
	// Next child session
	SessionChildCycle string `json:"session_child_cycle"`
	// Previous child session
	SessionChildCycleReverse string `json:"session_child_cycle_reverse"`
	// Compact the session
	SessionCompact string `json:"session_compact"`
	// Export session to editor
	SessionExport string `json:"session_export"`
	// Interrupt current session
	SessionInterrupt string `json:"session_interrupt"`
	// List all sessions
	SessionList string `json:"session_list"`
	// Create a new session
	SessionNew string `json:"session_new"`
	// Share current session
	SessionShare string `json:"session_share"`
	// Show session timeline
	SessionTimeline string `json:"session_timeline"`
	// Unshare current session
	SessionUnshare string `json:"session_unshare"`
	// Toggle sidebar
	SidebarToggle string `json:"sidebar_toggle"`
	// View status
	StatusView string `json:"status_view"`
	// Suspend terminal
	TerminalSuspend string `json:"terminal_suspend"`
	// List available themes
	ThemeList string `json:"theme_list"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		AgentCycle               respjson.Field
		AgentCycleReverse        respjson.Field
		AgentList                respjson.Field
		AppExit                  respjson.Field
		CommandList              respjson.Field
		EditorOpen               respjson.Field
		HistoryNext              respjson.Field
		HistoryPrevious          respjson.Field
		InputClear               respjson.Field
		InputForwardDelete       respjson.Field
		InputNewline             respjson.Field
		InputPaste               respjson.Field
		InputSubmit              respjson.Field
		Leader                   respjson.Field
		MessagesCopy             respjson.Field
		MessagesFirst            respjson.Field
		MessagesHalfPageDown     respjson.Field
		MessagesHalfPageUp       respjson.Field
		MessagesLast             respjson.Field
		MessagesPageDown         respjson.Field
		MessagesPageUp           respjson.Field
		MessagesRedo             respjson.Field
		MessagesToggleConceal    respjson.Field
		MessagesUndo             respjson.Field
		ModelCycleRecent         respjson.Field
		ModelCycleRecentReverse  respjson.Field
		ModelList                respjson.Field
		SessionChildCycle        respjson.Field
		SessionChildCycleReverse respjson.Field
		SessionCompact           respjson.Field
		SessionExport            respjson.Field
		SessionInterrupt         respjson.Field
		SessionList              respjson.Field
		SessionNew               respjson.Field
		SessionShare             respjson.Field
		SessionTimeline          respjson.Field
		SessionUnshare           respjson.Field
		SidebarToggle            respjson.Field
		StatusView               respjson.Field
		TerminalSuspend          respjson.Field
		ThemeList                respjson.Field
		ExtraFields              map[string]respjson.Field
		raw                      string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationKeybinds) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationKeybinds) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// @deprecated Always uses stretch layout.
type ConfigurationLayout string

const (
	ConfigurationLayoutAuto    ConfigurationLayout = "auto"
	ConfigurationLayoutStretch ConfigurationLayout = "stretch"
)

// ConfigurationLspUnion contains all possible properties and values from [bool],
// [map[string]ConfigurationLspMapItemUnion].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfBool]
type ConfigurationLspUnion struct {
	// This field will be present if the value is a [bool] instead of an object.
	OfBool   bool `json:",inline"`
	Disabled bool `json:"disabled"`
	// This field is from variant [map[string]ConfigurationLspMapItemUnion].
	Command []string `json:"command"`
	// This field is from variant [map[string]ConfigurationLspMapItemUnion].
	Env map[string]string `json:"env"`
	// This field is from variant [map[string]ConfigurationLspMapItemUnion].
	Extensions []string `json:"extensions"`
	// This field is from variant [map[string]ConfigurationLspMapItemUnion].
	Initialization map[string]any `json:"initialization"`
	JSON           struct {
		OfBool         respjson.Field
		Disabled       respjson.Field
		Command        respjson.Field
		Env            respjson.Field
		Extensions     respjson.Field
		Initialization respjson.Field
		raw            string
	} `json:"-"`
}

func (u ConfigurationLspUnion) AsBool() (v bool) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConfigurationLspUnion) AsConfigurationLspMapMap() (v map[string]ConfigurationLspMapItemUnion) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u ConfigurationLspUnion) RawJSON() string { return u.JSON.raw }

func (r *ConfigurationLspUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ConfigurationLspMapItemUnion contains all possible properties and values from
// [ConfigurationLspMapItemDisabled], [ConfigurationLspMapItemObject].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type ConfigurationLspMapItemUnion struct {
	Disabled bool `json:"disabled"`
	// This field is from variant [ConfigurationLspMapItemObject].
	Command []string `json:"command"`
	// This field is from variant [ConfigurationLspMapItemObject].
	Env map[string]string `json:"env"`
	// This field is from variant [ConfigurationLspMapItemObject].
	Extensions []string `json:"extensions"`
	// This field is from variant [ConfigurationLspMapItemObject].
	Initialization map[string]any `json:"initialization"`
	JSON           struct {
		Disabled       respjson.Field
		Command        respjson.Field
		Env            respjson.Field
		Extensions     respjson.Field
		Initialization respjson.Field
		raw            string
	} `json:"-"`
}

func (u ConfigurationLspMapItemUnion) AsConfigurationLspMapItemDisabled() (v ConfigurationLspMapItemDisabled) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConfigurationLspMapItemUnion) AsConfigurationLspMapItemObject() (v ConfigurationLspMapItemObject) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u ConfigurationLspMapItemUnion) RawJSON() string { return u.JSON.raw }

func (r *ConfigurationLspMapItemUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationLspMapItemDisabled struct {
	Disabled bool `json:"disabled,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Disabled    respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationLspMapItemDisabled) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationLspMapItemDisabled) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationLspMapItemObject struct {
	Command        []string          `json:"command,required"`
	Disabled       bool              `json:"disabled"`
	Env            map[string]string `json:"env"`
	Extensions     []string          `json:"extensions"`
	Initialization map[string]any    `json:"initialization"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Command        respjson.Field
		Disabled       respjson.Field
		Env            respjson.Field
		Extensions     respjson.Field
		Initialization respjson.Field
		ExtraFields    map[string]respjson.Field
		raw            string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationLspMapItemObject) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationLspMapItemObject) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ConfigurationMcpUnion contains all possible properties and values from
// [shared.McpLocalConfig], [shared.McpRemoteConfig].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type ConfigurationMcpUnion struct {
	// This field is from variant [shared.McpLocalConfig].
	Command []string `json:"command"`
	Type    string   `json:"type"`
	Enabled bool     `json:"enabled"`
	// This field is from variant [shared.McpLocalConfig].
	Environment map[string]string `json:"environment"`
	Timeout     int64             `json:"timeout"`
	// This field is from variant [shared.McpRemoteConfig].
	URL string `json:"url"`
	// This field is from variant [shared.McpRemoteConfig].
	Headers map[string]string `json:"headers"`
	JSON    struct {
		Command     respjson.Field
		Type        respjson.Field
		Enabled     respjson.Field
		Environment respjson.Field
		Timeout     respjson.Field
		URL         respjson.Field
		Headers     respjson.Field
		raw         string
	} `json:"-"`
}

func (u ConfigurationMcpUnion) AsMcpLocalConfig() (v shared.McpLocalConfig) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConfigurationMcpUnion) AsMcpRemoteConfig() (v shared.McpRemoteConfig) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u ConfigurationMcpUnion) RawJSON() string { return u.JSON.raw }

func (r *ConfigurationMcpUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// @deprecated Use `agent` field instead.
type ConfigurationMode struct {
	Build       AgentConfig            `json:"build"`
	Plan        AgentConfig            `json:"plan"`
	ExtraFields map[string]AgentConfig `json:",extras"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Build       respjson.Field
		Plan        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationMode) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationMode) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationPermission struct {
	Bash ConfigurationPermissionBashUnion `json:"bash"`
	// Any of "ask", "allow", "deny".
	DoomLoop string `json:"doom_loop"`
	// Any of "ask", "allow", "deny".
	Edit string `json:"edit"`
	// Any of "ask", "allow", "deny".
	ExternalDirectory string `json:"external_directory"`
	// Any of "ask", "allow", "deny".
	Webfetch string `json:"webfetch"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Bash              respjson.Field
		DoomLoop          respjson.Field
		Edit              respjson.Field
		ExternalDirectory respjson.Field
		Webfetch          respjson.Field
		ExtraFields       map[string]respjson.Field
		raw               string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationPermission) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationPermission) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ConfigurationPermissionBashUnion represents a union type that can be either a
// string ("ask", "allow", "deny") or a map[string]string for specific permissions.
type ConfigurationPermissionBashUnion struct {
	// This field will be present if the value is a string instead of a map.
	OfString string `json:",inline"`
	// This field will be present if the value is a map instead of a string.
	OfMap map[string]string `json:",inline"`
	JSON  struct {
		OfString respjson.Field
		OfMap    respjson.Field
		raw      string
	} `json:"-"`
}

func (u ConfigurationPermissionBashUnion) AsString() (v string) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConfigurationPermissionBashUnion) AsMap() (v map[string]string) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u ConfigurationPermissionBashUnion) RawJSON() string { return u.JSON.raw }

func (u *ConfigurationPermissionBashUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

type ConfigurationPermissionBashString string

const (
	ConfigurationPermissionBashStringAsk   ConfigurationPermissionBashString = "ask"
	ConfigurationPermissionBashStringAllow ConfigurationPermissionBashString = "allow"
	ConfigurationPermissionBashStringDeny  ConfigurationPermissionBashString = "deny"
)

type ConfigurationProvider struct {
	ID        string                                `json:"id"`
	API       string                                `json:"api"`
	Blacklist []string                              `json:"blacklist"`
	Env       []string                              `json:"env"`
	Models    map[string]ConfigurationProviderModel `json:"models"`
	Name      string                                `json:"name"`
	Npm       string                                `json:"npm"`
	Options   ConfigurationProviderOptions          `json:"options"`
	Whitelist []string                              `json:"whitelist"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		API         respjson.Field
		Blacklist   respjson.Field
		Env         respjson.Field
		Models      respjson.Field
		Name        respjson.Field
		Npm         respjson.Field
		Options     respjson.Field
		Whitelist   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationProvider) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationProvider) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationProviderModel struct {
	ID           string                               `json:"id"`
	Attachment   bool                                 `json:"attachment"`
	Cost         ConfigurationProviderModelCost       `json:"cost"`
	Experimental bool                                 `json:"experimental"`
	Headers      map[string]string                    `json:"headers"`
	Limit        ConfigurationProviderModelLimit      `json:"limit"`
	Modalities   ConfigurationProviderModelModalities `json:"modalities"`
	Name         string                               `json:"name"`
	Options      map[string]any                       `json:"options"`
	Provider     ConfigurationProviderModelProvider   `json:"provider"`
	Reasoning    bool                                 `json:"reasoning"`
	ReleaseDate  string                               `json:"release_date"`
	// Any of "alpha", "beta", "deprecated".
	Status      string `json:"status"`
	Temperature bool   `json:"temperature"`
	ToolCall    bool   `json:"tool_call"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID           respjson.Field
		Attachment   respjson.Field
		Cost         respjson.Field
		Experimental respjson.Field
		Headers      respjson.Field
		Limit        respjson.Field
		Modalities   respjson.Field
		Name         respjson.Field
		Options      respjson.Field
		Provider     respjson.Field
		Reasoning    respjson.Field
		ReleaseDate  respjson.Field
		Status       respjson.Field
		Temperature  respjson.Field
		ToolCall     respjson.Field
		ExtraFields  map[string]respjson.Field
		raw          string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationProviderModel) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationProviderModel) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationProviderModelCost struct {
	Input           float64                                       `json:"input,required"`
	Output          float64                                       `json:"output,required"`
	CacheRead       float64                                       `json:"cache_read"`
	CacheWrite      float64                                       `json:"cache_write"`
	ContextOver200k ConfigurationProviderModelCostContextOver200k `json:"context_over_200k"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Input           respjson.Field
		Output          respjson.Field
		CacheRead       respjson.Field
		CacheWrite      respjson.Field
		ContextOver200k respjson.Field
		ExtraFields     map[string]respjson.Field
		raw             string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationProviderModelCost) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationProviderModelCost) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationProviderModelCostContextOver200k struct {
	Input      float64 `json:"input,required"`
	Output     float64 `json:"output,required"`
	CacheRead  float64 `json:"cache_read"`
	CacheWrite float64 `json:"cache_write"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Input       respjson.Field
		Output      respjson.Field
		CacheRead   respjson.Field
		CacheWrite  respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationProviderModelCostContextOver200k) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationProviderModelCostContextOver200k) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationProviderModelLimit struct {
	Context float64 `json:"context,required"`
	Output  float64 `json:"output,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Context     respjson.Field
		Output      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationProviderModelLimit) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationProviderModelLimit) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationProviderModelModalities struct {
	// Any of "text", "audio", "image", "video", "pdf".
	Input []string `json:"input,required"`
	// Any of "text", "audio", "image", "video", "pdf".
	Output []string `json:"output,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Input       respjson.Field
		Output      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationProviderModelModalities) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationProviderModelModalities) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationProviderModelProvider struct {
	Npm string `json:"npm,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Npm         respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationProviderModelProvider) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationProviderModelProvider) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationProviderOptions struct {
	APIKey  string `json:"apiKey"`
	BaseURL string `json:"baseURL"`
	// GitHub Enterprise URL for copilot authentication
	EnterpriseURL string `json:"enterpriseUrl"`
	// Timeout in milliseconds for requests to this provider. Default is 300000 (5
	// minutes). Set to false to disable timeout.
	Timeout     ConfigurationProviderOptionsTimeoutUnion `json:"timeout"`
	ExtraFields map[string]any                           `json:",extras"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		APIKey        respjson.Field
		BaseURL       respjson.Field
		EnterpriseURL respjson.Field
		Timeout       respjson.Field
		ExtraFields   map[string]respjson.Field
		raw           string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationProviderOptions) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationProviderOptions) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ConfigurationProviderOptionsTimeoutUnion contains all possible properties and
// values from [int64], [bool].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfInt OfBool]
type ConfigurationProviderOptionsTimeoutUnion struct {
	// This field will be present if the value is a [int64] instead of an object.
	OfInt int64 `json:",inline"`
	// This field will be present if the value is a [bool] instead of an object.
	OfBool bool `json:",inline"`
	JSON   struct {
		OfInt  respjson.Field
		OfBool respjson.Field
		raw    string
	} `json:"-"`
}

func (u ConfigurationProviderOptionsTimeoutUnion) AsInt() (v int64) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConfigurationProviderOptionsTimeoutUnion) AsBool() (v bool) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u ConfigurationProviderOptionsTimeoutUnion) RawJSON() string { return u.JSON.raw }

func (r *ConfigurationProviderOptionsTimeoutUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Control sharing behavior:'manual' allows manual sharing via commands, 'auto'
// enables automatic sharing, 'disabled' disables all sharing
type ConfigurationShare string

const (
	ConfigurationShareManual   ConfigurationShare = "manual"
	ConfigurationShareAuto     ConfigurationShare = "auto"
	ConfigurationShareDisabled ConfigurationShare = "disabled"
)

// TUI specific settings
type ConfigurationTui struct {
	// Scroll acceleration settings
	ScrollAcceleration ConfigurationTuiScrollAcceleration `json:"scroll_acceleration"`
	// TUI scroll speed
	ScrollSpeed float64 `json:"scroll_speed"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ScrollAcceleration respjson.Field
		ScrollSpeed        respjson.Field
		ExtraFields        map[string]respjson.Field
		raw                string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationTui) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationTui) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Scroll acceleration settings
type ConfigurationTuiScrollAcceleration struct {
	// Enable scroll acceleration
	Enabled bool `json:"enabled,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Enabled     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationTuiScrollAcceleration) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationTuiScrollAcceleration) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationWatcher struct {
	Ignore []string `json:"ignore"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Ignore      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigurationWatcher) RawJSON() string { return r.JSON.raw }
func (r *ConfigurationWatcher) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationParam struct {
	// JSON schema reference for configuration validation
	Schema param.Opt[string] `json:"$schema,omitzero"`
	// @deprecated Use 'share' field instead. Share newly created sessions
	// automatically
	Autoshare param.Opt[bool] `json:"autoshare,omitzero"`
	// Model to use in the format of provider/model, eg anthropic/claude-2
	Model param.Opt[string] `json:"model,omitzero"`
	// Small model to use for tasks like title generation in the format of
	// provider/model
	SmallModel param.Opt[string] `json:"small_model,omitzero"`
	Snapshot   param.Opt[bool]   `json:"snapshot,omitzero"`
	// Theme name to use for the interface
	Theme param.Opt[string] `json:"theme,omitzero"`
	// Custom username to display in conversations instead of system username
	Username param.Opt[string] `json:"username,omitzero"`
	// Agent configuration, see https://opencode.ai/docs/agent
	Agent ConfigurationAgentParam `json:"agent,omitzero"`
	// Automatically update to the latest version. Set to true to auto-update, false to
	// disable, or 'notify' to show update notifications
	Autoupdate ConfigurationAutoupdateUnionParam `json:"autoupdate,omitzero"`
	// Command configuration, see https://opencode.ai/docs/commands
	Command map[string]ConfigurationCommandParam `json:"command,omitzero"`
	// Disable providers that are loaded automatically
	DisabledProviders []string `json:"disabled_providers,omitzero"`
	// When set, ONLY these providers will be enabled. All other providers will be
	// ignored
	EnabledProviders []string                         `json:"enabled_providers,omitzero"`
	Enterprise       ConfigurationEnterpriseParam     `json:"enterprise,omitzero"`
	Experimental     ConfigurationExperimentalParam   `json:"experimental,omitzero"`
	Formatter        ConfigurationFormatterUnionParam `json:"formatter,omitzero"`
	// Additional instruction files or patterns to include
	Instructions []string `json:"instructions,omitzero"`
	// Custom keybind configurations
	Keybinds ConfigurationKeybindsParam `json:"keybinds,omitzero"`
	// @deprecated Always uses stretch layout.
	//
	// Any of "auto", "stretch".
	Layout ConfigurationLayout        `json:"layout,omitzero"`
	Lsp    ConfigurationLspUnionParam `json:"lsp,omitzero"`
	// MCP (Model Context Protocol) server configurations
	Mcp map[string]ConfigurationMcpUnionParam `json:"mcp,omitzero"`
	// @deprecated Use `agent` field instead.
	Mode       ConfigurationModeParam       `json:"mode,omitzero"`
	Permission ConfigurationPermissionParam `json:"permission,omitzero"`
	Plugin     []string                     `json:"plugin,omitzero"`
	// Custom variables for prompt template interpolation (e.g., COMPANY_NAME,
	// TEAM_NAME)
	PromptVariables map[string]string `json:"promptVariables,omitzero"`
	// Custom provider configurations and model overrides
	Provider map[string]ConfigurationProviderParam `json:"provider,omitzero"`
	// Control sharing behavior:'manual' allows manual sharing via commands, 'auto'
	// enables automatic sharing, 'disabled' disables all sharing
	//
	// Any of "manual", "auto", "disabled".
	Share ConfigurationShare `json:"share,omitzero"`
	Tools map[string]bool    `json:"tools,omitzero"`
	// TUI specific settings
	Tui     ConfigurationTuiParam     `json:"tui,omitzero"`
	Watcher ConfigurationWatcherParam `json:"watcher,omitzero"`
	paramObj
}

func (r ConfigurationParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Agent configuration, see https://opencode.ai/docs/agent
type ConfigurationAgentParam struct {
	Build       AgentConfigParam            `json:"build,omitzero"`
	General     AgentConfigParam            `json:"general,omitzero"`
	Plan        AgentConfigParam            `json:"plan,omitzero"`
	ExtraFields map[string]AgentConfigParam `json:"-"`
	paramObj
}

func (r ConfigurationAgentParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationAgentParam
	return param.MarshalWithExtras(r, (*shadow)(&r), r.ExtraFields)
}
func (r *ConfigurationAgentParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ConfigurationAutoupdateUnionParam struct {
	OfBool param.Opt[bool] `json:",omitzero,inline"`
	// Construct this variant with constant.ValueOf[constant.Notify]()
	OfNotify constant.Notify `json:",omitzero,inline"`
	paramUnion
}

func (u ConfigurationAutoupdateUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfBool, u.OfNotify)
}
func (u *ConfigurationAutoupdateUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ConfigurationAutoupdateUnionParam) asAny() any {
	if !param.IsOmitted(u.OfBool) {
		return &u.OfBool.Value
	} else if !param.IsOmitted(u.OfNotify) {
		return &u.OfNotify
	}
	return nil
}

// The property Template is required.
type ConfigurationCommandParam struct {
	Template    string            `json:"template,required"`
	Agent       param.Opt[string] `json:"agent,omitzero"`
	Description param.Opt[string] `json:"description,omitzero"`
	Model       param.Opt[string] `json:"model,omitzero"`
	Subtask     param.Opt[bool]   `json:"subtask,omitzero"`
	paramObj
}

func (r ConfigurationCommandParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationCommandParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationCommandParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationEnterpriseParam struct {
	// Enterprise URL
	URL param.Opt[string] `json:"url,omitzero"`
	paramObj
}

func (r ConfigurationEnterpriseParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationEnterpriseParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationEnterpriseParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationExperimentalParam struct {
	// Enable the batch tool
	BatchTool param.Opt[bool] `json:"batch_tool,omitzero"`
	// Number of retries for chat completions on failure
	ChatMaxRetries      param.Opt[float64]                 `json:"chatMaxRetries,omitzero"`
	DisablePasteSummary param.Opt[bool]                    `json:"disable_paste_summary,omitzero"`
	Hook                ConfigurationExperimentalHookParam `json:"hook,omitzero"`
	paramObj
}

func (r ConfigurationExperimentalParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationExperimentalParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationExperimentalParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationExperimentalHookParam struct {
	FileEdited       map[string][]ConfigurationExperimentalHookFileEditedParam `json:"file_edited,omitzero"`
	SessionCompleted []ConfigurationExperimentalHookSessionCompletedParam      `json:"session_completed,omitzero"`
	paramObj
}

func (r ConfigurationExperimentalHookParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationExperimentalHookParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationExperimentalHookParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The property Command is required.
type ConfigurationExperimentalHookFileEditedParam struct {
	Command     []string          `json:"command,omitzero,required"`
	Environment map[string]string `json:"environment,omitzero"`
	paramObj
}

func (r ConfigurationExperimentalHookFileEditedParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationExperimentalHookFileEditedParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationExperimentalHookFileEditedParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The property Command is required.
type ConfigurationExperimentalHookSessionCompletedParam struct {
	Command     []string          `json:"command,omitzero,required"`
	Environment map[string]string `json:"environment,omitzero"`
	paramObj
}

func (r ConfigurationExperimentalHookSessionCompletedParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationExperimentalHookSessionCompletedParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationExperimentalHookSessionCompletedParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ConfigurationFormatterUnionParam struct {
	OfBool                         param.Opt[bool]                               `json:",omitzero,inline"`
	OfConfigurationFormatterMapMap map[string]ConfigurationFormatterMapItemParam `json:",omitzero,inline"`
	paramUnion
}

func (u ConfigurationFormatterUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfBool, u.OfConfigurationFormatterMapMap)
}
func (u *ConfigurationFormatterUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ConfigurationFormatterUnionParam) asAny() any {
	if !param.IsOmitted(u.OfBool) {
		return &u.OfBool.Value
	} else if !param.IsOmitted(u.OfConfigurationFormatterMapMap) {
		return &u.OfConfigurationFormatterMapMap
	}
	return nil
}

type ConfigurationFormatterMapItemParam struct {
	Disabled    param.Opt[bool]   `json:"disabled,omitzero"`
	Command     []string          `json:"command,omitzero"`
	Environment map[string]string `json:"environment,omitzero"`
	Extensions  []string          `json:"extensions,omitzero"`
	paramObj
}

func (r ConfigurationFormatterMapItemParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationFormatterMapItemParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationFormatterMapItemParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Custom keybind configurations
type ConfigurationKeybindsParam struct {
	// Next agent
	AgentCycle param.Opt[string] `json:"agent_cycle,omitzero"`
	// Previous agent
	AgentCycleReverse param.Opt[string] `json:"agent_cycle_reverse,omitzero"`
	// List agents
	AgentList param.Opt[string] `json:"agent_list,omitzero"`
	// Exit the application
	AppExit param.Opt[string] `json:"app_exit,omitzero"`
	// List available commands
	CommandList param.Opt[string] `json:"command_list,omitzero"`
	// Open external editor
	EditorOpen param.Opt[string] `json:"editor_open,omitzero"`
	// Next history item
	HistoryNext param.Opt[string] `json:"history_next,omitzero"`
	// Previous history item
	HistoryPrevious param.Opt[string] `json:"history_previous,omitzero"`
	// Clear input field
	InputClear param.Opt[string] `json:"input_clear,omitzero"`
	// Forward delete
	InputForwardDelete param.Opt[string] `json:"input_forward_delete,omitzero"`
	// Insert newline in input
	InputNewline param.Opt[string] `json:"input_newline,omitzero"`
	// Paste from clipboard
	InputPaste param.Opt[string] `json:"input_paste,omitzero"`
	// Submit input
	InputSubmit param.Opt[string] `json:"input_submit,omitzero"`
	// Leader key for keybind combinations
	Leader param.Opt[string] `json:"leader,omitzero"`
	// Copy message
	MessagesCopy param.Opt[string] `json:"messages_copy,omitzero"`
	// Navigate to first message
	MessagesFirst param.Opt[string] `json:"messages_first,omitzero"`
	// Scroll messages down by half page
	MessagesHalfPageDown param.Opt[string] `json:"messages_half_page_down,omitzero"`
	// Scroll messages up by half page
	MessagesHalfPageUp param.Opt[string] `json:"messages_half_page_up,omitzero"`
	// Navigate to last message
	MessagesLast param.Opt[string] `json:"messages_last,omitzero"`
	// Scroll messages down by one page
	MessagesPageDown param.Opt[string] `json:"messages_page_down,omitzero"`
	// Scroll messages up by one page
	MessagesPageUp param.Opt[string] `json:"messages_page_up,omitzero"`
	// Redo message
	MessagesRedo param.Opt[string] `json:"messages_redo,omitzero"`
	// Toggle code block concealment in messages
	MessagesToggleConceal param.Opt[string] `json:"messages_toggle_conceal,omitzero"`
	// Undo message
	MessagesUndo param.Opt[string] `json:"messages_undo,omitzero"`
	// Next recently used model
	ModelCycleRecent param.Opt[string] `json:"model_cycle_recent,omitzero"`
	// Previous recently used model
	ModelCycleRecentReverse param.Opt[string] `json:"model_cycle_recent_reverse,omitzero"`
	// List available models
	ModelList param.Opt[string] `json:"model_list,omitzero"`
	// Next child session
	SessionChildCycle param.Opt[string] `json:"session_child_cycle,omitzero"`
	// Previous child session
	SessionChildCycleReverse param.Opt[string] `json:"session_child_cycle_reverse,omitzero"`
	// Compact the session
	SessionCompact param.Opt[string] `json:"session_compact,omitzero"`
	// Export session to editor
	SessionExport param.Opt[string] `json:"session_export,omitzero"`
	// Interrupt current session
	SessionInterrupt param.Opt[string] `json:"session_interrupt,omitzero"`
	// List all sessions
	SessionList param.Opt[string] `json:"session_list,omitzero"`
	// Create a new session
	SessionNew param.Opt[string] `json:"session_new,omitzero"`
	// Share current session
	SessionShare param.Opt[string] `json:"session_share,omitzero"`
	// Show session timeline
	SessionTimeline param.Opt[string] `json:"session_timeline,omitzero"`
	// Unshare current session
	SessionUnshare param.Opt[string] `json:"session_unshare,omitzero"`
	// Toggle sidebar
	SidebarToggle param.Opt[string] `json:"sidebar_toggle,omitzero"`
	// View status
	StatusView param.Opt[string] `json:"status_view,omitzero"`
	// Suspend terminal
	TerminalSuspend param.Opt[string] `json:"terminal_suspend,omitzero"`
	// List available themes
	ThemeList param.Opt[string] `json:"theme_list,omitzero"`
	paramObj
}

func (r ConfigurationKeybindsParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationKeybindsParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationKeybindsParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ConfigurationLspUnionParam struct {
	OfBool                   param.Opt[bool]                              `json:",omitzero,inline"`
	OfConfigurationLspMapMap map[string]ConfigurationLspMapItemUnionParam `json:",omitzero,inline"`
	paramUnion
}

func (u ConfigurationLspUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfBool, u.OfConfigurationLspMapMap)
}
func (u *ConfigurationLspUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ConfigurationLspUnionParam) asAny() any {
	if !param.IsOmitted(u.OfBool) {
		return &u.OfBool.Value
	} else if !param.IsOmitted(u.OfConfigurationLspMapMap) {
		return &u.OfConfigurationLspMapMap
	}
	return nil
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ConfigurationLspMapItemUnionParam struct {
	OfConfigurationLspMapItemDisabled *ConfigurationLspMapItemDisabledParam `json:",omitzero,inline"`
	OfConfigurationLspMapItemObject   *ConfigurationLspMapItemObjectParam   `json:",omitzero,inline"`
	paramUnion
}

func (u ConfigurationLspMapItemUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfConfigurationLspMapItemDisabled, u.OfConfigurationLspMapItemObject)
}
func (u *ConfigurationLspMapItemUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ConfigurationLspMapItemUnionParam) asAny() any {
	if !param.IsOmitted(u.OfConfigurationLspMapItemDisabled) {
		return u.OfConfigurationLspMapItemDisabled
	} else if !param.IsOmitted(u.OfConfigurationLspMapItemObject) {
		return u.OfConfigurationLspMapItemObject
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ConfigurationLspMapItemUnionParam) GetCommand() []string {
	if vt := u.OfConfigurationLspMapItemObject; vt != nil {
		return vt.Command
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ConfigurationLspMapItemUnionParam) GetEnv() map[string]string {
	if vt := u.OfConfigurationLspMapItemObject; vt != nil {
		return vt.Env
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ConfigurationLspMapItemUnionParam) GetExtensions() []string {
	if vt := u.OfConfigurationLspMapItemObject; vt != nil {
		return vt.Extensions
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ConfigurationLspMapItemUnionParam) GetInitialization() map[string]any {
	if vt := u.OfConfigurationLspMapItemObject; vt != nil {
		return vt.Initialization
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ConfigurationLspMapItemUnionParam) GetDisabled() *bool {
	if vt := u.OfConfigurationLspMapItemDisabled; vt != nil {
		return (*bool)(&vt.Disabled)
	} else if vt := u.OfConfigurationLspMapItemObject; vt != nil && vt.Disabled.Valid() {
		return &vt.Disabled.Value
	}
	return nil
}

func NewConfigurationLspMapItemDisabledParam() ConfigurationLspMapItemDisabledParam {
	return ConfigurationLspMapItemDisabledParam{
		Disabled: true,
	}
}

// This struct has a constant value, construct it with
// [NewConfigurationLspMapItemDisabledParam].
type ConfigurationLspMapItemDisabledParam struct {
	Disabled bool `json:"disabled,omitzero,required"`
	paramObj
}

func (r ConfigurationLspMapItemDisabledParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationLspMapItemDisabledParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationLspMapItemDisabledParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func init() {
	apijson.RegisterFieldValidator[ConfigurationLspMapItemDisabledParam](
		"disabled", true,
	)
}

// The property Command is required.
type ConfigurationLspMapItemObjectParam struct {
	Command        []string          `json:"command,omitzero,required"`
	Disabled       param.Opt[bool]   `json:"disabled,omitzero"`
	Env            map[string]string `json:"env,omitzero"`
	Extensions     []string          `json:"extensions,omitzero"`
	Initialization map[string]any    `json:"initialization,omitzero"`
	paramObj
}

func (r ConfigurationLspMapItemObjectParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationLspMapItemObjectParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationLspMapItemObjectParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ConfigurationMcpUnionParam struct {
	OfMcpLocalConfig  *shared.McpLocalConfigParam  `json:",omitzero,inline"`
	OfMcpRemoteConfig *shared.McpRemoteConfigParam `json:",omitzero,inline"`
	paramUnion
}

func (u ConfigurationMcpUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfMcpLocalConfig, u.OfMcpRemoteConfig)
}
func (u *ConfigurationMcpUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ConfigurationMcpUnionParam) asAny() any {
	if !param.IsOmitted(u.OfMcpLocalConfig) {
		return u.OfMcpLocalConfig
	} else if !param.IsOmitted(u.OfMcpRemoteConfig) {
		return u.OfMcpRemoteConfig
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ConfigurationMcpUnionParam) GetCommand() []string {
	if vt := u.OfMcpLocalConfig; vt != nil {
		return vt.Command
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ConfigurationMcpUnionParam) GetEnvironment() map[string]string {
	if vt := u.OfMcpLocalConfig; vt != nil {
		return vt.Environment
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ConfigurationMcpUnionParam) GetURL() *string {
	if vt := u.OfMcpRemoteConfig; vt != nil {
		return &vt.URL
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ConfigurationMcpUnionParam) GetHeaders() map[string]string {
	if vt := u.OfMcpRemoteConfig; vt != nil {
		return vt.Headers
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ConfigurationMcpUnionParam) GetType() *string {
	if vt := u.OfMcpLocalConfig; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfMcpRemoteConfig; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ConfigurationMcpUnionParam) GetEnabled() *bool {
	if vt := u.OfMcpLocalConfig; vt != nil && vt.Enabled.Valid() {
		return &vt.Enabled.Value
	} else if vt := u.OfMcpRemoteConfig; vt != nil && vt.Enabled.Valid() {
		return &vt.Enabled.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ConfigurationMcpUnionParam) GetTimeout() *int64 {
	if vt := u.OfMcpLocalConfig; vt != nil && vt.Timeout.Valid() {
		return &vt.Timeout.Value
	} else if vt := u.OfMcpRemoteConfig; vt != nil && vt.Timeout.Valid() {
		return &vt.Timeout.Value
	}
	return nil
}

// @deprecated Use `agent` field instead.
type ConfigurationModeParam struct {
	Build       AgentConfigParam            `json:"build,omitzero"`
	Plan        AgentConfigParam            `json:"plan,omitzero"`
	ExtraFields map[string]AgentConfigParam `json:"-"`
	paramObj
}

func (r ConfigurationModeParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationModeParam
	return param.MarshalWithExtras(r, (*shadow)(&r), r.ExtraFields)
}
func (r *ConfigurationModeParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationPermissionParam struct {
	Bash ConfigurationPermissionBashUnionParam `json:"bash,omitzero"`
	// Any of "ask", "allow", "deny".
	DoomLoop string `json:"doom_loop,omitzero"`
	// Any of "ask", "allow", "deny".
	Edit string `json:"edit,omitzero"`
	// Any of "ask", "allow", "deny".
	ExternalDirectory string `json:"external_directory,omitzero"`
	// Any of "ask", "allow", "deny".
	Webfetch string `json:"webfetch,omitzero"`
	paramObj
}

func (r ConfigurationPermissionParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationPermissionParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationPermissionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func init() {
	apijson.RegisterFieldValidator[ConfigurationPermissionParam](
		"doom_loop", "ask", "allow", "deny",
	)
	apijson.RegisterFieldValidator[ConfigurationPermissionParam](
		"edit", "ask", "allow", "deny",
	)
	apijson.RegisterFieldValidator[ConfigurationPermissionParam](
		"external_directory", "ask", "allow", "deny",
	)
	apijson.RegisterFieldValidator[ConfigurationPermissionParam](
		"webfetch", "ask", "allow", "deny",
	)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ConfigurationPermissionBashUnionParam struct {
	// Check if union is this variant with
	// !param.IsOmitted(union.OfConfigurationPermissionBashString)
	OfConfigurationPermissionBashString     param.Opt[string] `json:",omitzero,inline"`
	OfConfigurationPermissionBashMapItemMap map[string]string `json:",omitzero,inline"`
	paramUnion
}

func (u ConfigurationPermissionBashUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfConfigurationPermissionBashString, u.OfConfigurationPermissionBashMapItemMap)
}
func (u *ConfigurationPermissionBashUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ConfigurationPermissionBashUnionParam) asAny() any {
	if !param.IsOmitted(u.OfConfigurationPermissionBashString) {
		return &u.OfConfigurationPermissionBashString
	} else if !param.IsOmitted(u.OfConfigurationPermissionBashMapItemMap) {
		return &u.OfConfigurationPermissionBashMapItemMap
	}
	return nil
}

type ConfigurationProviderParam struct {
	ID        param.Opt[string]                          `json:"id,omitzero"`
	API       param.Opt[string]                          `json:"api,omitzero"`
	Name      param.Opt[string]                          `json:"name,omitzero"`
	Npm       param.Opt[string]                          `json:"npm,omitzero"`
	Blacklist []string                                   `json:"blacklist,omitzero"`
	Env       []string                                   `json:"env,omitzero"`
	Models    map[string]ConfigurationProviderModelParam `json:"models,omitzero"`
	Options   ConfigurationProviderOptionsParam          `json:"options,omitzero"`
	Whitelist []string                                   `json:"whitelist,omitzero"`
	paramObj
}

func (r ConfigurationProviderParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationProviderParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationProviderParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationProviderModelParam struct {
	ID           param.Opt[string]                         `json:"id,omitzero"`
	Attachment   param.Opt[bool]                           `json:"attachment,omitzero"`
	Experimental param.Opt[bool]                           `json:"experimental,omitzero"`
	Name         param.Opt[string]                         `json:"name,omitzero"`
	Reasoning    param.Opt[bool]                           `json:"reasoning,omitzero"`
	ReleaseDate  param.Opt[string]                         `json:"release_date,omitzero"`
	Temperature  param.Opt[bool]                           `json:"temperature,omitzero"`
	ToolCall     param.Opt[bool]                           `json:"tool_call,omitzero"`
	Cost         ConfigurationProviderModelCostParam       `json:"cost,omitzero"`
	Headers      map[string]string                         `json:"headers,omitzero"`
	Limit        ConfigurationProviderModelLimitParam      `json:"limit,omitzero"`
	Modalities   ConfigurationProviderModelModalitiesParam `json:"modalities,omitzero"`
	Options      map[string]any                            `json:"options,omitzero"`
	Provider     ConfigurationProviderModelProviderParam   `json:"provider,omitzero"`
	// Any of "alpha", "beta", "deprecated".
	Status string `json:"status,omitzero"`
	paramObj
}

func (r ConfigurationProviderModelParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationProviderModelParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationProviderModelParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func init() {
	apijson.RegisterFieldValidator[ConfigurationProviderModelParam](
		"status", "alpha", "beta", "deprecated",
	)
}

// The properties Input, Output are required.
type ConfigurationProviderModelCostParam struct {
	Input           float64                                            `json:"input,required"`
	Output          float64                                            `json:"output,required"`
	CacheRead       param.Opt[float64]                                 `json:"cache_read,omitzero"`
	CacheWrite      param.Opt[float64]                                 `json:"cache_write,omitzero"`
	ContextOver200k ConfigurationProviderModelCostContextOver200kParam `json:"context_over_200k,omitzero"`
	paramObj
}

func (r ConfigurationProviderModelCostParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationProviderModelCostParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationProviderModelCostParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Input, Output are required.
type ConfigurationProviderModelCostContextOver200kParam struct {
	Input      float64            `json:"input,required"`
	Output     float64            `json:"output,required"`
	CacheRead  param.Opt[float64] `json:"cache_read,omitzero"`
	CacheWrite param.Opt[float64] `json:"cache_write,omitzero"`
	paramObj
}

func (r ConfigurationProviderModelCostContextOver200kParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationProviderModelCostContextOver200kParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationProviderModelCostContextOver200kParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Context, Output are required.
type ConfigurationProviderModelLimitParam struct {
	Context float64 `json:"context,required"`
	Output  float64 `json:"output,required"`
	paramObj
}

func (r ConfigurationProviderModelLimitParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationProviderModelLimitParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationProviderModelLimitParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Input, Output are required.
type ConfigurationProviderModelModalitiesParam struct {
	// Any of "text", "audio", "image", "video", "pdf".
	Input []string `json:"input,omitzero,required"`
	// Any of "text", "audio", "image", "video", "pdf".
	Output []string `json:"output,omitzero,required"`
	paramObj
}

func (r ConfigurationProviderModelModalitiesParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationProviderModelModalitiesParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationProviderModelModalitiesParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The property Npm is required.
type ConfigurationProviderModelProviderParam struct {
	Npm string `json:"npm,required"`
	paramObj
}

func (r ConfigurationProviderModelProviderParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationProviderModelProviderParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationProviderModelProviderParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationProviderOptionsParam struct {
	APIKey  param.Opt[string] `json:"apiKey,omitzero"`
	BaseURL param.Opt[string] `json:"baseURL,omitzero"`
	// GitHub Enterprise URL for copilot authentication
	EnterpriseURL param.Opt[string] `json:"enterpriseUrl,omitzero"`
	// Timeout in milliseconds for requests to this provider. Default is 300000 (5
	// minutes). Set to false to disable timeout.
	Timeout     ConfigurationProviderOptionsTimeoutUnionParam `json:"timeout,omitzero"`
	ExtraFields map[string]any                                `json:"-"`
	paramObj
}

func (r ConfigurationProviderOptionsParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationProviderOptionsParam
	return param.MarshalWithExtras(r, (*shadow)(&r), r.ExtraFields)
}
func (r *ConfigurationProviderOptionsParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ConfigurationProviderOptionsTimeoutUnionParam struct {
	OfInt  param.Opt[int64] `json:",omitzero,inline"`
	OfBool param.Opt[bool]  `json:",omitzero,inline"`
	paramUnion
}

func (u ConfigurationProviderOptionsTimeoutUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfInt, u.OfBool)
}
func (u *ConfigurationProviderOptionsTimeoutUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ConfigurationProviderOptionsTimeoutUnionParam) asAny() any {
	if !param.IsOmitted(u.OfInt) {
		return &u.OfInt.Value
	} else if !param.IsOmitted(u.OfBool) {
		return &u.OfBool.Value
	}
	return nil
}

// TUI specific settings
type ConfigurationTuiParam struct {
	// TUI scroll speed
	ScrollSpeed param.Opt[float64] `json:"scroll_speed,omitzero"`
	// Scroll acceleration settings
	ScrollAcceleration ConfigurationTuiScrollAccelerationParam `json:"scroll_acceleration,omitzero"`
	paramObj
}

func (r ConfigurationTuiParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationTuiParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationTuiParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Scroll acceleration settings
//
// The property Enabled is required.
type ConfigurationTuiScrollAccelerationParam struct {
	// Enable scroll acceleration
	Enabled bool `json:"enabled,required"`
	paramObj
}

func (r ConfigurationTuiScrollAccelerationParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationTuiScrollAccelerationParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationTuiScrollAccelerationParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigurationWatcherParam struct {
	Ignore []string `json:"ignore,omitzero"`
	paramObj
}

func (r ConfigurationWatcherParam) MarshalJSON() (data []byte, err error) {
	type shadow ConfigurationWatcherParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConfigurationWatcherParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type Provider struct {
	ID     string                   `json:"id,required"`
	Env    []string                 `json:"env,required"`
	Models map[string]ProviderModel `json:"models,required"`
	Name   string                   `json:"name,required"`
	API    string                   `json:"api"`
	Npm    string                   `json:"npm"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Env         respjson.Field
		Models      respjson.Field
		Name        respjson.Field
		API         respjson.Field
		Npm         respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r Provider) RawJSON() string { return r.JSON.raw }
func (r *Provider) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ProviderModel struct {
	ID           string                  `json:"id,required"`
	Attachment   bool                    `json:"attachment,required"`
	Cost         ProviderModelCost       `json:"cost,required"`
	Limit        ProviderModelLimit      `json:"limit,required"`
	Name         string                  `json:"name,required"`
	Options      map[string]any          `json:"options,required"`
	Reasoning    bool                    `json:"reasoning,required"`
	ReleaseDate  string                  `json:"release_date,required"`
	Temperature  bool                    `json:"temperature,required"`
	ToolCall     bool                    `json:"tool_call,required"`
	Experimental bool                    `json:"experimental"`
	Headers      map[string]string       `json:"headers"`
	Modalities   ProviderModelModalities `json:"modalities"`
	Provider     ProviderModelProvider   `json:"provider"`
	// Any of "alpha", "beta", "deprecated".
	Status string `json:"status"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID           respjson.Field
		Attachment   respjson.Field
		Cost         respjson.Field
		Limit        respjson.Field
		Name         respjson.Field
		Options      respjson.Field
		Reasoning    respjson.Field
		ReleaseDate  respjson.Field
		Temperature  respjson.Field
		ToolCall     respjson.Field
		Experimental respjson.Field
		Headers      respjson.Field
		Modalities   respjson.Field
		Provider     respjson.Field
		Status       respjson.Field
		ExtraFields  map[string]respjson.Field
		raw          string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ProviderModel) RawJSON() string { return r.JSON.raw }
func (r *ProviderModel) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ProviderModelCost struct {
	Input           float64                          `json:"input,required"`
	Output          float64                          `json:"output,required"`
	CacheRead       float64                          `json:"cache_read"`
	CacheWrite      float64                          `json:"cache_write"`
	ContextOver200k ProviderModelCostContextOver200k `json:"context_over_200k"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Input           respjson.Field
		Output          respjson.Field
		CacheRead       respjson.Field
		CacheWrite      respjson.Field
		ContextOver200k respjson.Field
		ExtraFields     map[string]respjson.Field
		raw             string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ProviderModelCost) RawJSON() string { return r.JSON.raw }
func (r *ProviderModelCost) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ProviderModelCostContextOver200k struct {
	Input      float64 `json:"input,required"`
	Output     float64 `json:"output,required"`
	CacheRead  float64 `json:"cache_read"`
	CacheWrite float64 `json:"cache_write"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Input       respjson.Field
		Output      respjson.Field
		CacheRead   respjson.Field
		CacheWrite  respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ProviderModelCostContextOver200k) RawJSON() string { return r.JSON.raw }
func (r *ProviderModelCostContextOver200k) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ProviderModelLimit struct {
	Context float64 `json:"context,required"`
	Output  float64 `json:"output,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Context     respjson.Field
		Output      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ProviderModelLimit) RawJSON() string { return r.JSON.raw }
func (r *ProviderModelLimit) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ProviderModelModalities struct {
	// Any of "text", "audio", "image", "video", "pdf".
	Input []string `json:"input,required"`
	// Any of "text", "audio", "image", "video", "pdf".
	Output []string `json:"output,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Input       respjson.Field
		Output      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ProviderModelModalities) RawJSON() string { return r.JSON.raw }
func (r *ProviderModelModalities) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ProviderModelProvider struct {
	Npm string `json:"npm,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Npm         respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ProviderModelProvider) RawJSON() string { return r.JSON.raw }
func (r *ProviderModelProvider) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigListProvidersResponse struct {
	Default   map[string]string `json:"default,required"`
	Providers []Provider        `json:"providers,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Default     respjson.Field
		Providers   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConfigListProvidersResponse) RawJSON() string { return r.JSON.raw }
func (r *ConfigListProvidersResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConfigGetParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [ConfigGetParams]'s query parameters as `url.Values`.
func (r ConfigGetParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type ConfigUpdateParams struct {
	Directory     param.Opt[string] `query:"directory,omitzero" json:"-"`
	Configuration ConfigurationParam
	paramObj
}

func (r ConfigUpdateParams) MarshalJSON() (data []byte, err error) {
	return shimjson.Marshal(r.Configuration)
}
func (r *ConfigUpdateParams) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &r.Configuration)
}

// URLQuery serializes [ConfigUpdateParams]'s query parameters as `url.Values`.
func (r ConfigUpdateParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type ConfigListProvidersParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [ConfigListProvidersParams]'s query parameters as
// `url.Values`.
func (r ConfigListProvidersParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}
