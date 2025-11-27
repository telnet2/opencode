// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package opencode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// ClientToolToolService contains methods and other services that help with
// interacting with the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewClientToolToolService] method instead.
type ClientToolToolService struct {
	Options []option.RequestOption
}

// NewClientToolToolService generates a new service that applies the given options
// to each request. These options are applied after the parent client's options (if
// there is one), and before any request-specific options.
func NewClientToolToolService(opts ...option.RequestOption) (r ClientToolToolService) {
	r = ClientToolToolService{}
	r.Options = opts
	return
}

// Get registered tools for a client
func (r *ClientToolToolService) Get(ctx context.Context, clientID string, query ClientToolToolGetParams, opts ...option.RequestOption) (res *[]ClientToolDefinition, err error) {
	opts = slices.Concat(r.Options, opts)
	if clientID == "" {
		err = errors.New("missing required clientID parameter")
		return
	}
	path := fmt.Sprintf("client-tools/tools/%s", clientID)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

// Get all registered client tools across all clients
func (r *ClientToolToolService) List(ctx context.Context, query ClientToolToolListParams, opts ...option.RequestOption) (res *ClientToolToolListResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "client-tools/tools"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

type ClientToolDefinition struct {
	ID          string         `json:"id,required"`
	Description string         `json:"description,required"`
	Parameters  map[string]any `json:"parameters,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Description respjson.Field
		Parameters  respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ClientToolDefinition) RawJSON() string { return r.JSON.raw }
func (r *ClientToolDefinition) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this ClientToolDefinition to a ClientToolDefinitionParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// ClientToolDefinitionParam.Overrides()
func (r ClientToolDefinition) ToParam() ClientToolDefinitionParam {
	return param.Override[ClientToolDefinitionParam](json.RawMessage(r.RawJSON()))
}

// The properties ID, Description, Parameters are required.
type ClientToolDefinitionParam struct {
	ID          string         `json:"id,required"`
	Description string         `json:"description,required"`
	Parameters  map[string]any `json:"parameters,omitzero,required"`
	paramObj
}

func (r ClientToolDefinitionParam) MarshalJSON() (data []byte, err error) {
	type shadow ClientToolDefinitionParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ClientToolDefinitionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ClientToolToolListResponse map[string]ClientToolDefinition

type ClientToolToolGetParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [ClientToolToolGetParams]'s query parameters as
// `url.Values`.
func (r ClientToolToolGetParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type ClientToolToolListParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [ClientToolToolListParams]'s query parameters as
// `url.Values`.
func (r ClientToolToolListParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}
