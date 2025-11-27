// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package opencode

import (
	"context"
	"net/http"
	"net/url"
	"slices"

	"github.com/sst/opencode-sdk-go/internal/apijson"
	"github.com/sst/opencode-sdk-go/internal/apiquery"
	"github.com/sst/opencode-sdk-go/internal/requestconfig"
	"github.com/sst/opencode-sdk-go/option"
	"github.com/sst/opencode-sdk-go/packages/param"
	"github.com/sst/opencode-sdk-go/packages/respjson"
)

// AgentService contains methods and other services that help with interacting with
// the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewAgentService] method instead.
type AgentService struct {
	Options []option.RequestOption
}

// NewAgentService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewAgentService(opts ...option.RequestOption) (r AgentService) {
	r = AgentService{}
	r.Options = opts
	return
}

// List all agents
func (r *AgentService) List(ctx context.Context, query AgentListParams, opts ...option.RequestOption) (res *[]AgentListResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "agent"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

type AgentListResponse struct {
	BuiltIn bool `json:"builtIn,required"`
	// Any of "subagent", "primary", "all".
	Mode        AgentListResponseMode       `json:"mode,required"`
	Name        string                      `json:"name,required"`
	Options     map[string]any              `json:"options,required"`
	Permission  AgentListResponsePermission `json:"permission,required"`
	Tools       map[string]bool             `json:"tools,required"`
	Color       string                      `json:"color"`
	Description string                      `json:"description"`
	Model       AgentListResponseModel      `json:"model"`
	Prompt      string                      `json:"prompt"`
	Temperature float64                     `json:"temperature"`
	TopP        float64                     `json:"topP"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		BuiltIn     respjson.Field
		Mode        respjson.Field
		Name        respjson.Field
		Options     respjson.Field
		Permission  respjson.Field
		Tools       respjson.Field
		Color       respjson.Field
		Description respjson.Field
		Model       respjson.Field
		Prompt      respjson.Field
		Temperature respjson.Field
		TopP        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r AgentListResponse) RawJSON() string { return r.JSON.raw }
func (r *AgentListResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type AgentListResponseMode string

const (
	AgentListResponseModeSubagent AgentListResponseMode = "subagent"
	AgentListResponseModePrimary  AgentListResponseMode = "primary"
	AgentListResponseModeAll      AgentListResponseMode = "all"
)

type AgentListResponsePermission struct {
	// Any of "ask", "allow", "deny".
	Bash map[string]string `json:"bash,required"`
	// Any of "ask", "allow", "deny".
	Edit string `json:"edit,required"`
	// Any of "ask", "allow", "deny".
	DoomLoop string `json:"doom_loop"`
	// Any of "ask", "allow", "deny".
	ExternalDirectory string `json:"external_directory"`
	// Any of "ask", "allow", "deny".
	Webfetch string `json:"webfetch"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Bash              respjson.Field
		Edit              respjson.Field
		DoomLoop          respjson.Field
		ExternalDirectory respjson.Field
		Webfetch          respjson.Field
		ExtraFields       map[string]respjson.Field
		raw               string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r AgentListResponsePermission) RawJSON() string { return r.JSON.raw }
func (r *AgentListResponsePermission) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type AgentListResponseModel struct {
	ModelID    string `json:"modelID,required"`
	ProviderID string `json:"providerID,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ModelID     respjson.Field
		ProviderID  respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r AgentListResponseModel) RawJSON() string { return r.JSON.raw }
func (r *AgentListResponseModel) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type AgentListParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [AgentListParams]'s query parameters as `url.Values`.
func (r AgentListParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}
