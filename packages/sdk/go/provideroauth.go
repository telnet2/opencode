// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package opencode

import (
	"context"
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

// ProviderOAuthService contains methods and other services that help with
// interacting with the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewProviderOAuthService] method instead.
type ProviderOAuthService struct {
	Options []option.RequestOption
}

// NewProviderOAuthService generates a new service that applies the given options
// to each request. These options are applied after the parent client's options (if
// there is one), and before any request-specific options.
func NewProviderOAuthService(opts ...option.RequestOption) (r ProviderOAuthService) {
	r = ProviderOAuthService{}
	r.Options = opts
	return
}

// Authorize a provider using OAuth
func (r *ProviderOAuthService) Authorize(ctx context.Context, id string, params ProviderOAuthAuthorizeParams, opts ...option.RequestOption) (res *ProviderOAuthAuthorizeResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("provider/%s/oauth/authorize", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

// Handle OAuth callback for a provider
func (r *ProviderOAuthService) HandleCallback(ctx context.Context, id string, params ProviderOAuthHandleCallbackParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("provider/%s/oauth/callback", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

type ProviderOAuthAuthorizeResponse struct {
	Instructions string `json:"instructions,required"`
	// Any of "auto", "code".
	Method ProviderOAuthAuthorizeResponseMethod `json:"method,required"`
	URL    string                               `json:"url,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Instructions respjson.Field
		Method       respjson.Field
		URL          respjson.Field
		ExtraFields  map[string]respjson.Field
		raw          string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ProviderOAuthAuthorizeResponse) RawJSON() string { return r.JSON.raw }
func (r *ProviderOAuthAuthorizeResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ProviderOAuthAuthorizeResponseMethod string

const (
	ProviderOAuthAuthorizeResponseMethodAuto ProviderOAuthAuthorizeResponseMethod = "auto"
	ProviderOAuthAuthorizeResponseMethodCode ProviderOAuthAuthorizeResponseMethod = "code"
)

type ProviderOAuthAuthorizeParams struct {
	// Auth method index
	Method    float64           `json:"method,required"`
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

func (r ProviderOAuthAuthorizeParams) MarshalJSON() (data []byte, err error) {
	type shadow ProviderOAuthAuthorizeParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ProviderOAuthAuthorizeParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [ProviderOAuthAuthorizeParams]'s query parameters as
// `url.Values`.
func (r ProviderOAuthAuthorizeParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type ProviderOAuthHandleCallbackParams struct {
	// Auth method index
	Method    float64           `json:"method,required"`
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	// OAuth authorization code
	Code param.Opt[string] `json:"code,omitzero"`
	paramObj
}

func (r ProviderOAuthHandleCallbackParams) MarshalJSON() (data []byte, err error) {
	type shadow ProviderOAuthHandleCallbackParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ProviderOAuthHandleCallbackParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [ProviderOAuthHandleCallbackParams]'s query parameters as
// `url.Values`.
func (r ProviderOAuthHandleCallbackParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}
