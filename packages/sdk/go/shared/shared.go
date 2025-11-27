// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package shared

import (
	"encoding/json"

	"github.com/sst/opencode-sdk-go/internal/apijson"
	"github.com/sst/opencode-sdk-go/packages/param"
	"github.com/sst/opencode-sdk-go/packages/respjson"
	"github.com/sst/opencode-sdk-go/shared/constant"
)

// aliased to make [param.APIUnion] private when embedding
type paramUnion = param.APIUnion

// aliased to make [param.APIObject] private when embedding
type paramObj = param.APIObject

type McpLocalConfig struct {
	// Command and arguments to run the MCP server
	Command []string `json:"command,required"`
	// Type of MCP server connection
	Type constant.Local `json:"type,required"`
	// Enable or disable the MCP server on startup
	Enabled bool `json:"enabled"`
	// Environment variables to set when running the MCP server
	Environment map[string]string `json:"environment"`
	// Timeout in ms for fetching tools from the MCP server. Defaults to 5000 (5
	// seconds) if not specified.
	Timeout int64 `json:"timeout"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Command     respjson.Field
		Type        respjson.Field
		Enabled     respjson.Field
		Environment respjson.Field
		Timeout     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r McpLocalConfig) RawJSON() string { return r.JSON.raw }
func (r *McpLocalConfig) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this McpLocalConfig to a McpLocalConfigParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// McpLocalConfigParam.Overrides()
func (r McpLocalConfig) ToParam() McpLocalConfigParam {
	return param.Override[McpLocalConfigParam](json.RawMessage(r.RawJSON()))
}

// The properties Command, Type are required.
type McpLocalConfigParam struct {
	// Command and arguments to run the MCP server
	Command []string `json:"command,omitzero,required"`
	// Enable or disable the MCP server on startup
	Enabled param.Opt[bool] `json:"enabled,omitzero"`
	// Timeout in ms for fetching tools from the MCP server. Defaults to 5000 (5
	// seconds) if not specified.
	Timeout param.Opt[int64] `json:"timeout,omitzero"`
	// Environment variables to set when running the MCP server
	Environment map[string]string `json:"environment,omitzero"`
	// Type of MCP server connection
	//
	// This field can be elided, and will marshal its zero value as "local".
	Type constant.Local `json:"type,required"`
	paramObj
}

func (r McpLocalConfigParam) MarshalJSON() (data []byte, err error) {
	type shadow McpLocalConfigParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *McpLocalConfigParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type McpRemoteConfig struct {
	// Type of MCP server connection
	Type constant.Remote `json:"type,required"`
	// URL of the remote MCP server
	URL string `json:"url,required"`
	// Enable or disable the MCP server on startup
	Enabled bool `json:"enabled"`
	// Headers to send with the request
	Headers map[string]string `json:"headers"`
	// Timeout in ms for fetching tools from the MCP server. Defaults to 5000 (5
	// seconds) if not specified.
	Timeout int64 `json:"timeout"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Type        respjson.Field
		URL         respjson.Field
		Enabled     respjson.Field
		Headers     respjson.Field
		Timeout     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r McpRemoteConfig) RawJSON() string { return r.JSON.raw }
func (r *McpRemoteConfig) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this McpRemoteConfig to a McpRemoteConfigParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// McpRemoteConfigParam.Overrides()
func (r McpRemoteConfig) ToParam() McpRemoteConfigParam {
	return param.Override[McpRemoteConfigParam](json.RawMessage(r.RawJSON()))
}

// The properties Type, URL are required.
type McpRemoteConfigParam struct {
	// URL of the remote MCP server
	URL string `json:"url,required"`
	// Enable or disable the MCP server on startup
	Enabled param.Opt[bool] `json:"enabled,omitzero"`
	// Timeout in ms for fetching tools from the MCP server. Defaults to 5000 (5
	// seconds) if not specified.
	Timeout param.Opt[int64] `json:"timeout,omitzero"`
	// Headers to send with the request
	Headers map[string]string `json:"headers,omitzero"`
	// Type of MCP server connection
	//
	// This field can be elided, and will marshal its zero value as "remote".
	Type constant.Remote `json:"type,required"`
	paramObj
}

func (r McpRemoteConfigParam) MarshalJSON() (data []byte, err error) {
	type shadow McpRemoteConfigParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *McpRemoteConfigParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}
