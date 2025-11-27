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

// FormatterService contains methods and other services that help with interacting
// with the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewFormatterService] method instead.
type FormatterService struct {
	Options []option.RequestOption
}

// NewFormatterService generates a new service that applies the given options to
// each request. These options are applied after the parent client's options (if
// there is one), and before any request-specific options.
func NewFormatterService(opts ...option.RequestOption) (r FormatterService) {
	r = FormatterService{}
	r.Options = opts
	return
}

// Get formatter status
func (r *FormatterService) Get(ctx context.Context, query FormatterGetParams, opts ...option.RequestOption) (res *[]FormatterGetResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "formatter"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

type FormatterGetResponse struct {
	Enabled    bool     `json:"enabled,required"`
	Extensions []string `json:"extensions,required"`
	Name       string   `json:"name,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Enabled     respjson.Field
		Extensions  respjson.Field
		Name        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r FormatterGetResponse) RawJSON() string { return r.JSON.raw }
func (r *FormatterGetResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type FormatterGetParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [FormatterGetParams]'s query parameters as `url.Values`.
func (r FormatterGetParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}
