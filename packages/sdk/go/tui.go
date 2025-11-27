// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package opencode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"

	"github.com/sst/opencode-sdk-go/internal/apijson"
	"github.com/sst/opencode-sdk-go/internal/apiquery"
	"github.com/sst/opencode-sdk-go/internal/requestconfig"
	"github.com/sst/opencode-sdk-go/option"
	"github.com/sst/opencode-sdk-go/packages/param"
	"github.com/sst/opencode-sdk-go/packages/respjson"
	"github.com/sst/opencode-sdk-go/shared/constant"
)

// TuiService contains methods and other services that help with interacting with
// the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewTuiService] method instead.
type TuiService struct {
	Options []option.RequestOption
	Control TuiControlService
}

// NewTuiService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewTuiService(opts ...option.RequestOption) (r TuiService) {
	r = TuiService{}
	r.Options = opts
	r.Control = NewTuiControlService(opts...)
	return
}

// Append prompt to the TUI
func (r *TuiService) AppendPrompt(ctx context.Context, params TuiAppendPromptParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "tui/append-prompt"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

// Clear the prompt
func (r *TuiService) ClearPrompt(ctx context.Context, body TuiClearPromptParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "tui/clear-prompt"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

// Execute a TUI command (e.g. agent_cycle)
func (r *TuiService) ExecuteCommand(ctx context.Context, params TuiExecuteCommandParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "tui/execute-command"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

// Open the help dialog
func (r *TuiService) OpenHelp(ctx context.Context, body TuiOpenHelpParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "tui/open-help"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

// Open the model dialog
func (r *TuiService) OpenModels(ctx context.Context, body TuiOpenModelsParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "tui/open-models"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

// Open the session dialog
func (r *TuiService) OpenSessions(ctx context.Context, body TuiOpenSessionsParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "tui/open-sessions"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

// Open the theme dialog
func (r *TuiService) OpenThemes(ctx context.Context, body TuiOpenThemesParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "tui/open-themes"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

// Publish a TUI event
func (r *TuiService) Publish(ctx context.Context, params TuiPublishParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "tui/publish"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

// Show a toast notification in the TUI
func (r *TuiService) ShowToast(ctx context.Context, params TuiShowToastParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "tui/show-toast"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

// Submit the prompt
func (r *TuiService) SubmitPrompt(ctx context.Context, body TuiSubmitPromptParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "tui/submit-prompt"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

type EventCommandExecute struct {
	Properties EventCommandExecuteProperties `json:"properties,required"`
	Type       constant.TuiCommandExecute    `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventCommandExecute) RawJSON() string { return r.JSON.raw }
func (r *EventCommandExecute) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this EventCommandExecute to a EventCommandExecuteParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// EventCommandExecuteParam.Overrides()
func (r EventCommandExecute) ToParam() EventCommandExecuteParam {
	return param.Override[EventCommandExecuteParam](json.RawMessage(r.RawJSON()))
}

type EventCommandExecuteProperties struct {
	Command string `json:"command,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Command     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventCommandExecuteProperties) RawJSON() string { return r.JSON.raw }
func (r *EventCommandExecuteProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Properties, Type are required.
type EventCommandExecuteParam struct {
	Properties EventCommandExecutePropertiesParam `json:"properties,omitzero,required"`
	// This field can be elided, and will marshal its zero value as
	// "tui.command.execute".
	Type constant.TuiCommandExecute `json:"type,required"`
	paramObj
}

func (r EventCommandExecuteParam) MarshalJSON() (data []byte, err error) {
	type shadow EventCommandExecuteParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *EventCommandExecuteParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The property Command is required.
type EventCommandExecutePropertiesParam struct {
	Command string `json:"command,omitzero,required"`
	paramObj
}

func (r EventCommandExecutePropertiesParam) MarshalJSON() (data []byte, err error) {
	type shadow EventCommandExecutePropertiesParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *EventCommandExecutePropertiesParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventPromptAppend struct {
	Properties EventPromptAppendProperties `json:"properties,required"`
	Type       constant.TuiPromptAppend    `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventPromptAppend) RawJSON() string { return r.JSON.raw }
func (r *EventPromptAppend) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this EventPromptAppend to a EventPromptAppendParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// EventPromptAppendParam.Overrides()
func (r EventPromptAppend) ToParam() EventPromptAppendParam {
	return param.Override[EventPromptAppendParam](json.RawMessage(r.RawJSON()))
}

type EventPromptAppendProperties struct {
	Text string `json:"text,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Text        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventPromptAppendProperties) RawJSON() string { return r.JSON.raw }
func (r *EventPromptAppendProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Properties, Type are required.
type EventPromptAppendParam struct {
	Properties EventPromptAppendPropertiesParam `json:"properties,omitzero,required"`
	// This field can be elided, and will marshal its zero value as
	// "tui.prompt.append".
	Type constant.TuiPromptAppend `json:"type,required"`
	paramObj
}

func (r EventPromptAppendParam) MarshalJSON() (data []byte, err error) {
	type shadow EventPromptAppendParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *EventPromptAppendParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The property Text is required.
type EventPromptAppendPropertiesParam struct {
	Text string `json:"text,required"`
	paramObj
}

func (r EventPromptAppendPropertiesParam) MarshalJSON() (data []byte, err error) {
	type shadow EventPromptAppendPropertiesParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *EventPromptAppendPropertiesParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventToastShow struct {
	Properties EventToastShowProperties `json:"properties,required"`
	Type       constant.TuiToastShow    `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventToastShow) RawJSON() string { return r.JSON.raw }
func (r *EventToastShow) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this EventToastShow to a EventToastShowParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// EventToastShowParam.Overrides()
func (r EventToastShow) ToParam() EventToastShowParam {
	return param.Override[EventToastShowParam](json.RawMessage(r.RawJSON()))
}

type EventToastShowProperties struct {
	Message string `json:"message,required"`
	// Any of "info", "success", "warning", "error".
	Variant string `json:"variant,required"`
	// Duration in milliseconds
	Duration float64 `json:"duration"`
	Title    string  `json:"title"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Message     respjson.Field
		Variant     respjson.Field
		Duration    respjson.Field
		Title       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventToastShowProperties) RawJSON() string { return r.JSON.raw }
func (r *EventToastShowProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Properties, Type are required.
type EventToastShowParam struct {
	Properties EventToastShowPropertiesParam `json:"properties,omitzero,required"`
	// This field can be elided, and will marshal its zero value as "tui.toast.show".
	Type constant.TuiToastShow `json:"type,required"`
	paramObj
}

func (r EventToastShowParam) MarshalJSON() (data []byte, err error) {
	type shadow EventToastShowParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *EventToastShowParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Message, Variant are required.
type EventToastShowPropertiesParam struct {
	Message string `json:"message,required"`
	// Any of "info", "success", "warning", "error".
	Variant string `json:"variant,omitzero,required"`
	// Duration in milliseconds
	Duration param.Opt[float64] `json:"duration,omitzero"`
	Title    param.Opt[string]  `json:"title,omitzero"`
	paramObj
}

func (r EventToastShowPropertiesParam) MarshalJSON() (data []byte, err error) {
	type shadow EventToastShowPropertiesParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *EventToastShowPropertiesParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func init() {
	apijson.RegisterFieldValidator[EventToastShowPropertiesParam](
		"variant", "info", "success", "warning", "error",
	)
}

type TuiAppendPromptParams struct {
	Text      string            `json:"text,required"`
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

func (r TuiAppendPromptParams) MarshalJSON() (data []byte, err error) {
	type shadow TuiAppendPromptParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *TuiAppendPromptParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [TuiAppendPromptParams]'s query parameters as `url.Values`.
func (r TuiAppendPromptParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type TuiClearPromptParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [TuiClearPromptParams]'s query parameters as `url.Values`.
func (r TuiClearPromptParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type TuiExecuteCommandParams struct {
	Command   string            `json:"command,required"`
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

func (r TuiExecuteCommandParams) MarshalJSON() (data []byte, err error) {
	type shadow TuiExecuteCommandParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *TuiExecuteCommandParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [TuiExecuteCommandParams]'s query parameters as
// `url.Values`.
func (r TuiExecuteCommandParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type TuiOpenHelpParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [TuiOpenHelpParams]'s query parameters as `url.Values`.
func (r TuiOpenHelpParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type TuiOpenModelsParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [TuiOpenModelsParams]'s query parameters as `url.Values`.
func (r TuiOpenModelsParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type TuiOpenSessionsParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [TuiOpenSessionsParams]'s query parameters as `url.Values`.
func (r TuiOpenSessionsParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type TuiOpenThemesParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [TuiOpenThemesParams]'s query parameters as `url.Values`.
func (r TuiOpenThemesParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type TuiPublishParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`

	//
	// Request body variants
	//

	// This field is a request body variant, only one variant field can be set.
	OfEventPromptAppend *EventPromptAppendParam `json:",inline"`
	// This field is a request body variant, only one variant field can be set.
	OfEventCommandExecute *EventCommandExecuteParam `json:",inline"`
	// This field is a request body variant, only one variant field can be set.
	OfEventToastShow *EventToastShowParam `json:",inline"`

	paramObj
}

func (u TuiPublishParams) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfEventPromptAppend, u.OfEventCommandExecute, u.OfEventToastShow)
}
func (r *TuiPublishParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [TuiPublishParams]'s query parameters as `url.Values`.
func (r TuiPublishParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type TuiShowToastParams struct {
	Message string `json:"message,required"`
	// Any of "info", "success", "warning", "error".
	Variant   TuiShowToastParamsVariant `json:"variant,omitzero,required"`
	Directory param.Opt[string]         `query:"directory,omitzero" json:"-"`
	// Duration in milliseconds
	Duration param.Opt[float64] `json:"duration,omitzero"`
	Title    param.Opt[string]  `json:"title,omitzero"`
	paramObj
}

func (r TuiShowToastParams) MarshalJSON() (data []byte, err error) {
	type shadow TuiShowToastParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *TuiShowToastParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [TuiShowToastParams]'s query parameters as `url.Values`.
func (r TuiShowToastParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type TuiShowToastParamsVariant string

const (
	TuiShowToastParamsVariantInfo    TuiShowToastParamsVariant = "info"
	TuiShowToastParamsVariantSuccess TuiShowToastParamsVariant = "success"
	TuiShowToastParamsVariantWarning TuiShowToastParamsVariant = "warning"
	TuiShowToastParamsVariantError   TuiShowToastParamsVariant = "error"
)

type TuiSubmitPromptParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [TuiSubmitPromptParams]'s query parameters as `url.Values`.
func (r TuiSubmitPromptParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}
