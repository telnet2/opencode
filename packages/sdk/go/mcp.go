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
	"github.com/sst/opencode-sdk-go/internal/requestconfig"
	"github.com/sst/opencode-sdk-go/option"
	"github.com/sst/opencode-sdk-go/packages/param"
	"github.com/sst/opencode-sdk-go/packages/respjson"
	"github.com/sst/opencode-sdk-go/shared"
	"github.com/sst/opencode-sdk-go/shared/constant"
)

// McpService contains methods and other services that help with interacting with
// the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewMcpService] method instead.
type McpService struct {
	Options []option.RequestOption
}

// NewMcpService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewMcpService(opts ...option.RequestOption) (r McpService) {
	r = McpService{}
	r.Options = opts
	return
}

// Add MCP server dynamically
func (r *McpService) AddServer(ctx context.Context, params McpAddServerParams, opts ...option.RequestOption) (res *McpAddServerResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "mcp"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

// Get MCP server status
func (r *McpService) GetStatus(ctx context.Context, query McpGetStatusParams, opts ...option.RequestOption) (res *McpGetStatusResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "mcp"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

type McpAddServerResponse map[string]McpAddServerResponseItemUnion

// McpAddServerResponseItemUnion contains all possible properties and values from
// [McpAddServerResponseItemMcpStatusConnected],
// [McpAddServerResponseItemMcpStatusDisabled],
// [McpAddServerResponseItemMcpStatusFailed].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type McpAddServerResponseItemUnion struct {
	Status string `json:"status"`
	// This field is from variant [McpAddServerResponseItemMcpStatusFailed].
	Error string `json:"error"`
	JSON  struct {
		Status respjson.Field
		Error  respjson.Field
		raw    string
	} `json:"-"`
}

func (u McpAddServerResponseItemUnion) AsMcpAddServerResponseItemMcpStatusConnected() (v McpAddServerResponseItemMcpStatusConnected) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u McpAddServerResponseItemUnion) AsMcpAddServerResponseItemMcpStatusDisabled() (v McpAddServerResponseItemMcpStatusDisabled) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u McpAddServerResponseItemUnion) AsMcpAddServerResponseItemMcpStatusFailed() (v McpAddServerResponseItemMcpStatusFailed) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u McpAddServerResponseItemUnion) RawJSON() string { return u.JSON.raw }

func (r *McpAddServerResponseItemUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type McpAddServerResponseItemMcpStatusConnected struct {
	Status constant.Connected `json:"status,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Status      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r McpAddServerResponseItemMcpStatusConnected) RawJSON() string { return r.JSON.raw }
func (r *McpAddServerResponseItemMcpStatusConnected) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type McpAddServerResponseItemMcpStatusDisabled struct {
	Status constant.Disabled `json:"status,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Status      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r McpAddServerResponseItemMcpStatusDisabled) RawJSON() string { return r.JSON.raw }
func (r *McpAddServerResponseItemMcpStatusDisabled) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type McpAddServerResponseItemMcpStatusFailed struct {
	Error  string          `json:"error,required"`
	Status constant.Failed `json:"status,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Error       respjson.Field
		Status      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r McpAddServerResponseItemMcpStatusFailed) RawJSON() string { return r.JSON.raw }
func (r *McpAddServerResponseItemMcpStatusFailed) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type McpGetStatusResponse map[string]McpGetStatusResponseItemUnion

// McpGetStatusResponseItemUnion contains all possible properties and values from
// [McpGetStatusResponseItemMcpStatusConnected],
// [McpGetStatusResponseItemMcpStatusDisabled],
// [McpGetStatusResponseItemMcpStatusFailed].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type McpGetStatusResponseItemUnion struct {
	Status string `json:"status"`
	// This field is from variant [McpGetStatusResponseItemMcpStatusFailed].
	Error string `json:"error"`
	JSON  struct {
		Status respjson.Field
		Error  respjson.Field
		raw    string
	} `json:"-"`
}

func (u McpGetStatusResponseItemUnion) AsMcpGetStatusResponseItemMcpStatusConnected() (v McpGetStatusResponseItemMcpStatusConnected) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u McpGetStatusResponseItemUnion) AsMcpGetStatusResponseItemMcpStatusDisabled() (v McpGetStatusResponseItemMcpStatusDisabled) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u McpGetStatusResponseItemUnion) AsMcpGetStatusResponseItemMcpStatusFailed() (v McpGetStatusResponseItemMcpStatusFailed) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u McpGetStatusResponseItemUnion) RawJSON() string { return u.JSON.raw }

func (r *McpGetStatusResponseItemUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type McpGetStatusResponseItemMcpStatusConnected struct {
	Status constant.Connected `json:"status,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Status      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r McpGetStatusResponseItemMcpStatusConnected) RawJSON() string { return r.JSON.raw }
func (r *McpGetStatusResponseItemMcpStatusConnected) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type McpGetStatusResponseItemMcpStatusDisabled struct {
	Status constant.Disabled `json:"status,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Status      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r McpGetStatusResponseItemMcpStatusDisabled) RawJSON() string { return r.JSON.raw }
func (r *McpGetStatusResponseItemMcpStatusDisabled) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type McpGetStatusResponseItemMcpStatusFailed struct {
	Error  string          `json:"error,required"`
	Status constant.Failed `json:"status,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Error       respjson.Field
		Status      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r McpGetStatusResponseItemMcpStatusFailed) RawJSON() string { return r.JSON.raw }
func (r *McpGetStatusResponseItemMcpStatusFailed) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type McpAddServerParams struct {
	Config    McpAddServerParamsConfigUnion `json:"config,omitzero,required"`
	Name      string                        `json:"name,required"`
	Directory param.Opt[string]             `query:"directory,omitzero" json:"-"`
	paramObj
}

func (r McpAddServerParams) MarshalJSON() (data []byte, err error) {
	type shadow McpAddServerParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *McpAddServerParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [McpAddServerParams]'s query parameters as `url.Values`.
func (r McpAddServerParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type McpAddServerParamsConfigUnion struct {
	OfMcpLocalConfig  *shared.McpLocalConfigParam  `json:",omitzero,inline"`
	OfMcpRemoteConfig *shared.McpRemoteConfigParam `json:",omitzero,inline"`
	paramUnion
}

func (u McpAddServerParamsConfigUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfMcpLocalConfig, u.OfMcpRemoteConfig)
}
func (u *McpAddServerParamsConfigUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *McpAddServerParamsConfigUnion) asAny() any {
	if !param.IsOmitted(u.OfMcpLocalConfig) {
		return u.OfMcpLocalConfig
	} else if !param.IsOmitted(u.OfMcpRemoteConfig) {
		return u.OfMcpRemoteConfig
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u McpAddServerParamsConfigUnion) GetCommand() []string {
	if vt := u.OfMcpLocalConfig; vt != nil {
		return vt.Command
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u McpAddServerParamsConfigUnion) GetEnvironment() map[string]string {
	if vt := u.OfMcpLocalConfig; vt != nil {
		return vt.Environment
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u McpAddServerParamsConfigUnion) GetURL() *string {
	if vt := u.OfMcpRemoteConfig; vt != nil {
		return &vt.URL
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u McpAddServerParamsConfigUnion) GetHeaders() map[string]string {
	if vt := u.OfMcpRemoteConfig; vt != nil {
		return vt.Headers
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u McpAddServerParamsConfigUnion) GetType() *string {
	if vt := u.OfMcpLocalConfig; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfMcpRemoteConfig; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u McpAddServerParamsConfigUnion) GetEnabled() *bool {
	if vt := u.OfMcpLocalConfig; vt != nil && vt.Enabled.Valid() {
		return &vt.Enabled.Value
	} else if vt := u.OfMcpRemoteConfig; vt != nil && vt.Enabled.Valid() {
		return &vt.Enabled.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u McpAddServerParamsConfigUnion) GetTimeout() *int64 {
	if vt := u.OfMcpLocalConfig; vt != nil && vt.Timeout.Valid() {
		return &vt.Timeout.Value
	} else if vt := u.OfMcpRemoteConfig; vt != nil && vt.Timeout.Valid() {
		return &vt.Timeout.Value
	}
	return nil
}

type McpGetStatusParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [McpGetStatusParams]'s query parameters as `url.Values`.
func (r McpGetStatusParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}
