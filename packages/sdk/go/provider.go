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

// ProviderService contains methods and other services that help with interacting
// with the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewProviderService] method instead.
type ProviderService struct {
	Options []option.RequestOption
	OAuth   ProviderOAuthService
}

// NewProviderService generates a new service that applies the given options to
// each request. These options are applied after the parent client's options (if
// there is one), and before any request-specific options.
func NewProviderService(opts ...option.RequestOption) (r ProviderService) {
	r = ProviderService{}
	r.Options = opts
	r.OAuth = NewProviderOAuthService(opts...)
	return
}

// List all providers
func (r *ProviderService) List(ctx context.Context, query ProviderListParams, opts ...option.RequestOption) (res *ProviderListResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "provider"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

// Get provider authentication methods
func (r *ProviderService) GetAuthMethods(ctx context.Context, query ProviderGetAuthMethodsParams, opts ...option.RequestOption) (res *ProviderGetAuthMethodsResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "provider/auth"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

type ProviderListResponse struct {
	All       []Provider        `json:"all,required"`
	Connected []string          `json:"connected,required"`
	Default   map[string]string `json:"default,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		All         respjson.Field
		Connected   respjson.Field
		Default     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ProviderListResponse) RawJSON() string { return r.JSON.raw }
func (r *ProviderListResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ProviderGetAuthMethodsResponse map[string][]ProviderGetAuthMethodsResponseItem

type ProviderGetAuthMethodsResponseItem struct {
	Label string `json:"label,required"`
	// Any of "oauth", "api".
	Type ProviderGetAuthMethodsResponseItemType `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Label       respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ProviderGetAuthMethodsResponseItem) RawJSON() string { return r.JSON.raw }
func (r *ProviderGetAuthMethodsResponseItem) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ProviderGetAuthMethodsResponseItemType string

const (
	ProviderGetAuthMethodsResponseItemTypeOAuth ProviderGetAuthMethodsResponseItemType = "oauth"
	ProviderGetAuthMethodsResponseItemTypeAPI   ProviderGetAuthMethodsResponseItemType = "api"
)

type ProviderListParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [ProviderListParams]'s query parameters as `url.Values`.
func (r ProviderListParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type ProviderGetAuthMethodsParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [ProviderGetAuthMethodsParams]'s query parameters as
// `url.Values`.
func (r ProviderGetAuthMethodsParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}
