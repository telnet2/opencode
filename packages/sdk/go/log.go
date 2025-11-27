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
)

// LogService contains methods and other services that help with interacting with
// the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewLogService] method instead.
type LogService struct {
	Options []option.RequestOption
}

// NewLogService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewLogService(opts ...option.RequestOption) (r LogService) {
	r = LogService{}
	r.Options = opts
	return
}

// Write a log entry to the server logs
func (r *LogService) New(ctx context.Context, params LogNewParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "log"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

type LogNewParams struct {
	// Log level
	//
	// Any of "debug", "info", "error", "warn".
	Level LogNewParamsLevel `json:"level,omitzero,required"`
	// Log message
	Message string `json:"message,required"`
	// Service name for the log entry
	Service   string            `json:"service,required"`
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	// Additional metadata for the log entry
	Extra map[string]any `json:"extra,omitzero"`
	paramObj
}

func (r LogNewParams) MarshalJSON() (data []byte, err error) {
	type shadow LogNewParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *LogNewParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [LogNewParams]'s query parameters as `url.Values`.
func (r LogNewParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

// Log level
type LogNewParamsLevel string

const (
	LogNewParamsLevelDebug LogNewParamsLevel = "debug"
	LogNewParamsLevelInfo  LogNewParamsLevel = "info"
	LogNewParamsLevelError LogNewParamsLevel = "error"
	LogNewParamsLevelWarn  LogNewParamsLevel = "warn"
)
