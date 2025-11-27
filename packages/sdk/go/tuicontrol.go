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
)

// TuiControlService contains methods and other services that help with interacting
// with the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewTuiControlService] method instead.
type TuiControlService struct {
	Options []option.RequestOption
}

// NewTuiControlService generates a new service that applies the given options to
// each request. These options are applied after the parent client's options (if
// there is one), and before any request-specific options.
func NewTuiControlService(opts ...option.RequestOption) (r TuiControlService) {
	r = TuiControlService{}
	r.Options = opts
	return
}

// Get the next TUI request from the queue
func (r *TuiControlService) GetNextRequest(ctx context.Context, query TuiControlGetNextRequestParams, opts ...option.RequestOption) (res *TuiControlGetNextRequestResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "tui/control/next"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

// Submit a response to the TUI request queue
func (r *TuiControlService) SubmitResponse(ctx context.Context, params TuiControlSubmitResponseParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "tui/control/response"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

type TuiControlGetNextRequestResponse struct {
	Body any    `json:"body,required"`
	Path string `json:"path,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Body        respjson.Field
		Path        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r TuiControlGetNextRequestResponse) RawJSON() string { return r.JSON.raw }
func (r *TuiControlGetNextRequestResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type TuiControlGetNextRequestParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [TuiControlGetNextRequestParams]'s query parameters as
// `url.Values`.
func (r TuiControlGetNextRequestParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type TuiControlSubmitResponseParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	Body      any
	paramObj
}

func (r TuiControlSubmitResponseParams) MarshalJSON() (data []byte, err error) {
	return shimjson.Marshal(r.Body)
}
func (r *TuiControlSubmitResponseParams) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &r.Body)
}

// URLQuery serializes [TuiControlSubmitResponseParams]'s query parameters as
// `url.Values`.
func (r TuiControlSubmitResponseParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}
