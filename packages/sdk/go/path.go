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

// PathService contains methods and other services that help with interacting with
// the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewPathService] method instead.
type PathService struct {
	Options []option.RequestOption
}

// NewPathService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewPathService(opts ...option.RequestOption) (r PathService) {
	r = PathService{}
	r.Options = opts
	return
}

// Get the current path
func (r *PathService) Get(ctx context.Context, query PathGetParams, opts ...option.RequestOption) (res *PathGetResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "path"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

type PathGetResponse struct {
	Config    string `json:"config,required"`
	Directory string `json:"directory,required"`
	State     string `json:"state,required"`
	Worktree  string `json:"worktree,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Config      respjson.Field
		Directory   respjson.Field
		State       respjson.Field
		Worktree    respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PathGetResponse) RawJSON() string { return r.JSON.raw }
func (r *PathGetResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PathGetParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [PathGetParams]'s query parameters as `url.Values`.
func (r PathGetParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}
