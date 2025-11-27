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
	"github.com/sst/opencode-sdk-go/shared/constant"
)

// AuthService contains methods and other services that help with interacting with
// the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewAuthService] method instead.
type AuthService struct {
	Options []option.RequestOption
}

// NewAuthService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewAuthService(opts ...option.RequestOption) (r AuthService) {
	r = AuthService{}
	r.Options = opts
	return
}

// Set authentication credentials
func (r *AuthService) UpdateCredentials(ctx context.Context, id string, params AuthUpdateCredentialsParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("auth/%s", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPut, path, params, &res, opts...)
	return
}

type AuthUpdateCredentialsParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`

	//
	// Request body variants
	//

	// This field is a request body variant, only one variant field can be set.
	OfOAuth *AuthUpdateCredentialsParamsBodyOAuth `json:",inline"`
	// This field is a request body variant, only one variant field can be set.
	OfAPIAuth *AuthUpdateCredentialsParamsBodyAPIAuth `json:",inline"`
	// This field is a request body variant, only one variant field can be set.
	OfWellKnownAuth *AuthUpdateCredentialsParamsBodyWellKnownAuth `json:",inline"`

	paramObj
}

func (u AuthUpdateCredentialsParams) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfOAuth, u.OfAPIAuth, u.OfWellKnownAuth)
}
func (r *AuthUpdateCredentialsParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [AuthUpdateCredentialsParams]'s query parameters as
// `url.Values`.
func (r AuthUpdateCredentialsParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

// The properties Access, Expires, Refresh, Type are required.
type AuthUpdateCredentialsParamsBodyOAuth struct {
	Access        string            `json:"access,required"`
	Expires       float64           `json:"expires,required"`
	Refresh       string            `json:"refresh,required"`
	EnterpriseURL param.Opt[string] `json:"enterpriseUrl,omitzero"`
	// This field can be elided, and will marshal its zero value as "oauth".
	Type constant.OAuth `json:"type,required"`
	paramObj
}

func (r AuthUpdateCredentialsParamsBodyOAuth) MarshalJSON() (data []byte, err error) {
	type shadow AuthUpdateCredentialsParamsBodyOAuth
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *AuthUpdateCredentialsParamsBodyOAuth) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Key, Type are required.
type AuthUpdateCredentialsParamsBodyAPIAuth struct {
	Key string `json:"key,required"`
	// This field can be elided, and will marshal its zero value as "api".
	Type constant.API `json:"type,required"`
	paramObj
}

func (r AuthUpdateCredentialsParamsBodyAPIAuth) MarshalJSON() (data []byte, err error) {
	type shadow AuthUpdateCredentialsParamsBodyAPIAuth
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *AuthUpdateCredentialsParamsBodyAPIAuth) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Token, Key, Type are required.
type AuthUpdateCredentialsParamsBodyWellKnownAuth struct {
	Token string `json:"token,required"`
	Key   string `json:"key,required"`
	// This field can be elided, and will marshal its zero value as "wellknown".
	Type constant.Wellknown `json:"type,required"`
	paramObj
}

func (r AuthUpdateCredentialsParamsBodyWellKnownAuth) MarshalJSON() (data []byte, err error) {
	type shadow AuthUpdateCredentialsParamsBodyWellKnownAuth
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *AuthUpdateCredentialsParamsBodyWellKnownAuth) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}
