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

// ExperimentalToolService contains methods and other services that help with
// interacting with the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewExperimentalToolService] method instead.
type ExperimentalToolService struct {
	Options []option.RequestOption
}

// NewExperimentalToolService generates a new service that applies the given
// options to each request. These options are applied after the parent client's
// options (if there is one), and before any request-specific options.
func NewExperimentalToolService(opts ...option.RequestOption) (r ExperimentalToolService) {
	r = ExperimentalToolService{}
	r.Options = opts
	return
}

// List tools with JSON schema parameters for a provider/model
func (r *ExperimentalToolService) List(ctx context.Context, query ExperimentalToolListParams, opts ...option.RequestOption) (res *[]ExperimentalToolListResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "experimental/tool"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

// List all tool IDs (including built-in and dynamically registered)
func (r *ExperimentalToolService) ListIDs(ctx context.Context, query ExperimentalToolListIDsParams, opts ...option.RequestOption) (res *[]string, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "experimental/tool/ids"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

type ExperimentalToolListResponse struct {
	ID          string `json:"id,required"`
	Description string `json:"description,required"`
	Parameters  any    `json:"parameters,required"`
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
func (r ExperimentalToolListResponse) RawJSON() string { return r.JSON.raw }
func (r *ExperimentalToolListResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ExperimentalToolListParams struct {
	Model     string            `query:"model,required" json:"-"`
	Provider  string            `query:"provider,required" json:"-"`
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [ExperimentalToolListParams]'s query parameters as
// `url.Values`.
func (r ExperimentalToolListParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type ExperimentalToolListIDsParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [ExperimentalToolListIDsParams]'s query parameters as
// `url.Values`.
func (r ExperimentalToolListIDsParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}
