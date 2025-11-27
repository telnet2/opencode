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

// LspService contains methods and other services that help with interacting with
// the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewLspService] method instead.
type LspService struct {
	Options []option.RequestOption
}

// NewLspService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewLspService(opts ...option.RequestOption) (r LspService) {
	r = LspService{}
	r.Options = opts
	return
}

// Get LSP server status
func (r *LspService) Get(ctx context.Context, query LspGetParams, opts ...option.RequestOption) (res *[]LspGetResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "lsp"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

type LspGetResponse struct {
	ID   string `json:"id,required"`
	Name string `json:"name,required"`
	Root string `json:"root,required"`
	// Any of "connected", "error".
	Status LspGetResponseStatus `json:"status,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Name        respjson.Field
		Root        respjson.Field
		Status      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r LspGetResponse) RawJSON() string { return r.JSON.raw }
func (r *LspGetResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type LspGetResponseStatus string

const (
	LspGetResponseStatusConnected LspGetResponseStatus = "connected"
	LspGetResponseStatusError     LspGetResponseStatus = "error"
)

type LspGetParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [LspGetParams]'s query parameters as `url.Values`.
func (r LspGetParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}
