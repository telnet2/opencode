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
	"github.com/sst/opencode-sdk-go/packages/ssestream"
	"github.com/sst/opencode-sdk-go/shared/constant"
)

// ClientToolService contains methods and other services that help with interacting
// with the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewClientToolService] method instead.
type ClientToolService struct {
	Options []option.RequestOption
	Tools   ClientToolToolService
}

// NewClientToolService generates a new service that applies the given options to
// each request. These options are applied after the parent client's options (if
// there is one), and before any request-specific options.
func NewClientToolService(opts ...option.RequestOption) (r ClientToolService) {
	r = ClientToolService{}
	r.Options = opts
	r.Tools = NewClientToolToolService(opts...)
	return
}

// Register client tools for a client
func (r *ClientToolService) Register(ctx context.Context, params ClientToolRegisterParams, opts ...option.RequestOption) (res *ClientToolRegisterResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "client-tools/register"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

// Stream pending tool execution requests to client
func (r *ClientToolService) StreamPendingRequestsStreaming(ctx context.Context, clientID string, query ClientToolStreamPendingRequestsParams, opts ...option.RequestOption) (stream *ssestream.Stream[ClientToolExecution]) {
	var (
		raw *http.Response
		err error
	)
	opts = slices.Concat(r.Options, opts)
	opts = append([]option.RequestOption{option.WithHeader("Accept", "text/event-stream")}, opts...)
	if clientID == "" {
		err = errors.New("missing required clientID parameter")
		return
	}
	path := fmt.Sprintf("client-tools/pending/%s", clientID)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &raw, opts...)
	return ssestream.NewStream[ClientToolExecution](ssestream.NewDecoder(raw), err)
}

// Submit tool execution result from client
func (r *ClientToolService) SubmitResult(ctx context.Context, params ClientToolSubmitResultParams, opts ...option.RequestOption) (res *ClientToolSubmitResultResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "client-tools/result"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

// Unregister client tools
func (r *ClientToolService) Unregister(ctx context.Context, params ClientToolUnregisterParams, opts ...option.RequestOption) (res *ClientToolUnregisterResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "client-tools/unregister"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodDelete, path, params, &res, opts...)
	return
}

type ClientToolExecution struct {
	CallID    string                     `json:"callID,required"`
	Input     map[string]any             `json:"input,required"`
	MessageID string                     `json:"messageID,required"`
	RequestID string                     `json:"requestID,required"`
	SessionID string                     `json:"sessionID,required"`
	Tool      string                     `json:"tool,required"`
	Type      constant.ClientToolRequest `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		CallID      respjson.Field
		Input       respjson.Field
		MessageID   respjson.Field
		RequestID   respjson.Field
		SessionID   respjson.Field
		Tool        respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ClientToolExecution) RawJSON() string { return r.JSON.raw }
func (r *ClientToolExecution) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ClientToolRegisterResponse struct {
	Registered []string `json:"registered,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Registered  respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ClientToolRegisterResponse) RawJSON() string { return r.JSON.raw }
func (r *ClientToolRegisterResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ClientToolSubmitResultResponse struct {
	Success bool `json:"success,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Success     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ClientToolSubmitResultResponse) RawJSON() string { return r.JSON.raw }
func (r *ClientToolSubmitResultResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ClientToolUnregisterResponse struct {
	Success      bool     `json:"success,required"`
	Unregistered []string `json:"unregistered,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Success      respjson.Field
		Unregistered respjson.Field
		ExtraFields  map[string]respjson.Field
		raw          string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ClientToolUnregisterResponse) RawJSON() string { return r.JSON.raw }
func (r *ClientToolUnregisterResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ClientToolRegisterParams struct {
	ClientID  string                      `json:"clientID,required"`
	Tools     []ClientToolDefinitionParam `json:"tools,omitzero,required"`
	Directory param.Opt[string]           `query:"directory,omitzero" json:"-"`
	paramObj
}

func (r ClientToolRegisterParams) MarshalJSON() (data []byte, err error) {
	type shadow ClientToolRegisterParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ClientToolRegisterParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [ClientToolRegisterParams]'s query parameters as
// `url.Values`.
func (r ClientToolRegisterParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type ClientToolStreamPendingRequestsParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [ClientToolStreamPendingRequestsParams]'s query parameters
// as `url.Values`.
func (r ClientToolStreamPendingRequestsParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type ClientToolSubmitResultParams struct {
	RequestID string                                  `json:"requestID,required"`
	Result    ClientToolSubmitResultParamsResultUnion `json:"result,omitzero,required"`
	Directory param.Opt[string]                       `query:"directory,omitzero" json:"-"`
	paramObj
}

func (r ClientToolSubmitResultParams) MarshalJSON() (data []byte, err error) {
	type shadow ClientToolSubmitResultParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ClientToolSubmitResultParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [ClientToolSubmitResultParams]'s query parameters as
// `url.Values`.
func (r ClientToolSubmitResultParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ClientToolSubmitResultParamsResultUnion struct {
	OfClientToolSubmitResultsResultClientToolResult *ClientToolSubmitResultParamsResultClientToolResult `json:",omitzero,inline"`
	OfClientToolSubmitResultsResultClientToolError  *ClientToolSubmitResultParamsResultClientToolError  `json:",omitzero,inline"`
	paramUnion
}

func (u ClientToolSubmitResultParamsResultUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfClientToolSubmitResultsResultClientToolResult, u.OfClientToolSubmitResultsResultClientToolError)
}
func (u *ClientToolSubmitResultParamsResultUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ClientToolSubmitResultParamsResultUnion) asAny() any {
	if !param.IsOmitted(u.OfClientToolSubmitResultsResultClientToolResult) {
		return u.OfClientToolSubmitResultsResultClientToolResult
	} else if !param.IsOmitted(u.OfClientToolSubmitResultsResultClientToolError) {
		return u.OfClientToolSubmitResultsResultClientToolError
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ClientToolSubmitResultParamsResultUnion) GetOutput() *string {
	if vt := u.OfClientToolSubmitResultsResultClientToolResult; vt != nil {
		return &vt.Output
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ClientToolSubmitResultParamsResultUnion) GetTitle() *string {
	if vt := u.OfClientToolSubmitResultsResultClientToolResult; vt != nil {
		return &vt.Title
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ClientToolSubmitResultParamsResultUnion) GetMetadata() map[string]any {
	if vt := u.OfClientToolSubmitResultsResultClientToolResult; vt != nil {
		return vt.Metadata
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ClientToolSubmitResultParamsResultUnion) GetError() *string {
	if vt := u.OfClientToolSubmitResultsResultClientToolError; vt != nil {
		return &vt.Error
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ClientToolSubmitResultParamsResultUnion) GetStatus() *string {
	if vt := u.OfClientToolSubmitResultsResultClientToolResult; vt != nil {
		return (*string)(&vt.Status)
	} else if vt := u.OfClientToolSubmitResultsResultClientToolError; vt != nil {
		return (*string)(&vt.Status)
	}
	return nil
}

// The properties Output, Status, Title are required.
type ClientToolSubmitResultParamsResultClientToolResult struct {
	Output   string         `json:"output,required"`
	Title    string         `json:"title,required"`
	Metadata map[string]any `json:"metadata,omitzero"`
	// This field can be elided, and will marshal its zero value as "success".
	Status constant.Success `json:"status,required"`
	paramObj
}

func (r ClientToolSubmitResultParamsResultClientToolResult) MarshalJSON() (data []byte, err error) {
	type shadow ClientToolSubmitResultParamsResultClientToolResult
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ClientToolSubmitResultParamsResultClientToolResult) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Error, Status are required.
type ClientToolSubmitResultParamsResultClientToolError struct {
	Error string `json:"error,required"`
	// This field can be elided, and will marshal its zero value as "error".
	Status constant.Error `json:"status,required"`
	paramObj
}

func (r ClientToolSubmitResultParamsResultClientToolError) MarshalJSON() (data []byte, err error) {
	type shadow ClientToolSubmitResultParamsResultClientToolError
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ClientToolSubmitResultParamsResultClientToolError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ClientToolUnregisterParams struct {
	ClientID  string            `json:"clientID,required"`
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	ToolIDs   []string          `json:"toolIDs,omitzero"`
	paramObj
}

func (r ClientToolUnregisterParams) MarshalJSON() (data []byte, err error) {
	type shadow ClientToolUnregisterParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ClientToolUnregisterParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [ClientToolUnregisterParams]'s query parameters as
// `url.Values`.
func (r ClientToolUnregisterParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}
