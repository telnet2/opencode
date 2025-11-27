// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package opencode

import (
	"context"
	"net/http"
	"net/url"
	"slices"

	"github.com/sst/opencode-sdk-go/internal/apiquery"
	"github.com/sst/opencode-sdk-go/internal/requestconfig"
	"github.com/sst/opencode-sdk-go/option"
	"github.com/sst/opencode-sdk-go/packages/param"
)

// InstanceService contains methods and other services that help with interacting
// with the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewInstanceService] method instead.
type InstanceService struct {
	Options []option.RequestOption
}

// NewInstanceService generates a new service that applies the given options to
// each request. These options are applied after the parent client's options (if
// there is one), and before any request-specific options.
func NewInstanceService(opts ...option.RequestOption) (r InstanceService) {
	r = InstanceService{}
	r.Options = opts
	return
}

// Dispose the current instance
func (r *InstanceService) Dispose(ctx context.Context, body InstanceDisposeParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "instance/dispose"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

type InstanceDisposeParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [InstanceDisposeParams]'s query parameters as `url.Values`.
func (r InstanceDisposeParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}
