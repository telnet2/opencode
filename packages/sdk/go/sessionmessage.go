// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package opencode

import (
	"context"
	"encoding/json"
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
	"github.com/sst/opencode-sdk-go/shared/constant"
)

// SessionMessageService contains methods and other services that help with
// interacting with the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewSessionMessageService] method instead.
type SessionMessageService struct {
	Options []option.RequestOption
}

// NewSessionMessageService generates a new service that applies the given options
// to each request. These options are applied after the parent client's options (if
// there is one), and before any request-specific options.
func NewSessionMessageService(opts ...option.RequestOption) (r SessionMessageService) {
	r = SessionMessageService{}
	r.Options = opts
	return
}

// Create and send a new message to a session
func (r *SessionMessageService) New(ctx context.Context, id string, params SessionMessageNewParams, opts ...option.RequestOption) (res *SessionMessageNewResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("session/%s/message", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

// Get a message from a session
func (r *SessionMessageService) Get(ctx context.Context, messageID string, params SessionMessageGetParams, opts ...option.RequestOption) (res *SessionMessageGetResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	if params.ID == "" {
		err = errors.New("missing required id parameter")
		return
	}
	if messageID == "" {
		err = errors.New("missing required messageID parameter")
		return
	}
	path := fmt.Sprintf("session/%s/message/%s", params.ID, messageID)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, params, &res, opts...)
	return
}

// List messages for a session
func (r *SessionMessageService) List(ctx context.Context, id string, query SessionMessageListParams, opts ...option.RequestOption) (res *[]SessionMessageListResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("session/%s/message", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

type FilePart struct {
	ID        string              `json:"id,required"`
	MessageID string              `json:"messageID,required"`
	Mime      string              `json:"mime,required"`
	SessionID string              `json:"sessionID,required"`
	Type      constant.File       `json:"type,required"`
	URL       string              `json:"url,required"`
	Filename  string              `json:"filename"`
	Source    FilePartSourceUnion `json:"source"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		MessageID   respjson.Field
		Mime        respjson.Field
		SessionID   respjson.Field
		Type        respjson.Field
		URL         respjson.Field
		Filename    respjson.Field
		Source      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r FilePart) RawJSON() string { return r.JSON.raw }
func (r *FilePart) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// FilePartSourceUnion contains all possible properties and values from
// [FilePartSourceFileSource], [FilePartSourceSymbolSource].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type FilePartSourceUnion struct {
	Path string `json:"path"`
	// This field is from variant [FilePartSourceFileSource].
	Text FilePartSourceText `json:"text"`
	Type string             `json:"type"`
	// This field is from variant [FilePartSourceSymbolSource].
	Kind int64 `json:"kind"`
	// This field is from variant [FilePartSourceSymbolSource].
	Name string `json:"name"`
	// This field is from variant [FilePartSourceSymbolSource].
	Range Range `json:"range"`
	JSON  struct {
		Path  respjson.Field
		Text  respjson.Field
		Type  respjson.Field
		Kind  respjson.Field
		Name  respjson.Field
		Range respjson.Field
		raw   string
	} `json:"-"`
}

func (u FilePartSourceUnion) AsFilePartSourceFileSource() (v FilePartSourceFileSource) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u FilePartSourceUnion) AsFilePartSourceSymbolSource() (v FilePartSourceSymbolSource) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u FilePartSourceUnion) RawJSON() string { return u.JSON.raw }

func (r *FilePartSourceUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this FilePartSourceUnion to a FilePartSourceUnionParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// FilePartSourceUnionParam.Overrides()
func (r FilePartSourceUnion) ToParam() FilePartSourceUnionParam {
	return param.Override[FilePartSourceUnionParam](json.RawMessage(r.RawJSON()))
}

type FilePartSourceFileSource struct {
	Path string             `json:"path,required"`
	Text FilePartSourceText `json:"text,required"`
	Type constant.File      `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Path        respjson.Field
		Text        respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r FilePartSourceFileSource) RawJSON() string { return r.JSON.raw }
func (r *FilePartSourceFileSource) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type FilePartSourceSymbolSource struct {
	Kind  int64              `json:"kind,required"`
	Name  string             `json:"name,required"`
	Path  string             `json:"path,required"`
	Range Range              `json:"range,required"`
	Text  FilePartSourceText `json:"text,required"`
	Type  constant.Symbol    `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Kind        respjson.Field
		Name        respjson.Field
		Path        respjson.Field
		Range       respjson.Field
		Text        respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r FilePartSourceSymbolSource) RawJSON() string { return r.JSON.raw }
func (r *FilePartSourceSymbolSource) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func FilePartSourceParamOfFilePartSourceFileSource(path string, text FilePartSourceTextParam) FilePartSourceUnionParam {
	var variant FilePartSourceFileSourceParam
	variant.Path = path
	variant.Text = text
	return FilePartSourceUnionParam{OfFilePartSourceFileSource: &variant}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type FilePartSourceUnionParam struct {
	OfFilePartSourceFileSource   *FilePartSourceFileSourceParam   `json:",omitzero,inline"`
	OfFilePartSourceSymbolSource *FilePartSourceSymbolSourceParam `json:",omitzero,inline"`
	paramUnion
}

func (u FilePartSourceUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfFilePartSourceFileSource, u.OfFilePartSourceSymbolSource)
}
func (u *FilePartSourceUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *FilePartSourceUnionParam) asAny() any {
	if !param.IsOmitted(u.OfFilePartSourceFileSource) {
		return u.OfFilePartSourceFileSource
	} else if !param.IsOmitted(u.OfFilePartSourceSymbolSource) {
		return u.OfFilePartSourceSymbolSource
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u FilePartSourceUnionParam) GetKind() *int64 {
	if vt := u.OfFilePartSourceSymbolSource; vt != nil {
		return &vt.Kind
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u FilePartSourceUnionParam) GetName() *string {
	if vt := u.OfFilePartSourceSymbolSource; vt != nil {
		return &vt.Name
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u FilePartSourceUnionParam) GetRange() *RangeParam {
	if vt := u.OfFilePartSourceSymbolSource; vt != nil {
		return &vt.Range
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u FilePartSourceUnionParam) GetPath() *string {
	if vt := u.OfFilePartSourceFileSource; vt != nil {
		return (*string)(&vt.Path)
	} else if vt := u.OfFilePartSourceSymbolSource; vt != nil {
		return (*string)(&vt.Path)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u FilePartSourceUnionParam) GetType() *string {
	if vt := u.OfFilePartSourceFileSource; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfFilePartSourceSymbolSource; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's Text property, if present.
func (u FilePartSourceUnionParam) GetText() *FilePartSourceTextParam {
	if vt := u.OfFilePartSourceFileSource; vt != nil {
		return &vt.Text
	} else if vt := u.OfFilePartSourceSymbolSource; vt != nil {
		return &vt.Text
	}
	return nil
}

// The properties Path, Text, Type are required.
type FilePartSourceFileSourceParam struct {
	Path string                  `json:"path,required"`
	Text FilePartSourceTextParam `json:"text,omitzero,required"`
	// This field can be elided, and will marshal its zero value as "file".
	Type constant.File `json:"type,required"`
	paramObj
}

func (r FilePartSourceFileSourceParam) MarshalJSON() (data []byte, err error) {
	type shadow FilePartSourceFileSourceParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *FilePartSourceFileSourceParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Kind, Name, Path, Range, Text, Type are required.
type FilePartSourceSymbolSourceParam struct {
	Kind  int64                   `json:"kind,required"`
	Name  string                  `json:"name,required"`
	Path  string                  `json:"path,required"`
	Range RangeParam              `json:"range,omitzero,required"`
	Text  FilePartSourceTextParam `json:"text,omitzero,required"`
	// This field can be elided, and will marshal its zero value as "symbol".
	Type constant.Symbol `json:"type,required"`
	paramObj
}

func (r FilePartSourceSymbolSourceParam) MarshalJSON() (data []byte, err error) {
	type shadow FilePartSourceSymbolSourceParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *FilePartSourceSymbolSourceParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type FilePartSourceText struct {
	End   int64  `json:"end,required"`
	Start int64  `json:"start,required"`
	Value string `json:"value,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		End         respjson.Field
		Start       respjson.Field
		Value       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r FilePartSourceText) RawJSON() string { return r.JSON.raw }
func (r *FilePartSourceText) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this FilePartSourceText to a FilePartSourceTextParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// FilePartSourceTextParam.Overrides()
func (r FilePartSourceText) ToParam() FilePartSourceTextParam {
	return param.Override[FilePartSourceTextParam](json.RawMessage(r.RawJSON()))
}

// The properties End, Start, Value are required.
type FilePartSourceTextParam struct {
	End   int64  `json:"end,required"`
	Start int64  `json:"start,required"`
	Value string `json:"value,required"`
	paramObj
}

func (r FilePartSourceTextParam) MarshalJSON() (data []byte, err error) {
	type shadow FilePartSourceTextParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *FilePartSourceTextParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// MessageUnion contains all possible properties and values from
// [MessageUserMessage], [AssistantMessage].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type MessageUnion struct {
	ID string `json:"id"`
	// This field is from variant [MessageUserMessage].
	Agent string `json:"agent"`
	// This field is from variant [MessageUserMessage].
	Model     MessageUserMessageModel `json:"model"`
	Role      string                  `json:"role"`
	SessionID string                  `json:"sessionID"`
	// This field is a union of [MessageUserMessageTime], [AssistantMessageTime]
	Time MessageUnionTime `json:"time"`
	// This field is a union of [MessageUserMessageSummary], [bool]
	Summary MessageUnionSummary `json:"summary"`
	// This field is from variant [MessageUserMessage].
	System string `json:"system"`
	// This field is from variant [MessageUserMessage].
	Tools map[string]bool `json:"tools"`
	// This field is from variant [AssistantMessage].
	Cost float64 `json:"cost"`
	// This field is from variant [AssistantMessage].
	Mode string `json:"mode"`
	// This field is from variant [AssistantMessage].
	ModelID string `json:"modelID"`
	// This field is from variant [AssistantMessage].
	ParentID string `json:"parentID"`
	// This field is from variant [AssistantMessage].
	Path AssistantMessagePath `json:"path"`
	// This field is from variant [AssistantMessage].
	ProviderID string `json:"providerID"`
	// This field is from variant [AssistantMessage].
	Tokens AssistantMessageTokens `json:"tokens"`
	// This field is from variant [AssistantMessage].
	Error AssistantMessageErrorUnion `json:"error"`
	// This field is from variant [AssistantMessage].
	Finish string `json:"finish"`
	JSON   struct {
		ID         respjson.Field
		Agent      respjson.Field
		Model      respjson.Field
		Role       respjson.Field
		SessionID  respjson.Field
		Time       respjson.Field
		Summary    respjson.Field
		System     respjson.Field
		Tools      respjson.Field
		Cost       respjson.Field
		Mode       respjson.Field
		ModelID    respjson.Field
		ParentID   respjson.Field
		Path       respjson.Field
		ProviderID respjson.Field
		Tokens     respjson.Field
		Error      respjson.Field
		Finish     respjson.Field
		raw        string
	} `json:"-"`
}

func (u MessageUnion) AsMessageUserMessage() (v MessageUserMessage) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u MessageUnion) AsAssistantMessage() (v AssistantMessage) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u MessageUnion) RawJSON() string { return u.JSON.raw }

func (r *MessageUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// MessageUnionTime is an implicit subunion of [MessageUnion]. MessageUnionTime
// provides convenient access to the sub-properties of the union.
//
// For type safety it is recommended to directly use a variant of the
// [MessageUnion].
type MessageUnionTime struct {
	Created float64 `json:"created"`
	// This field is from variant [AssistantMessageTime].
	Completed float64 `json:"completed"`
	JSON      struct {
		Created   respjson.Field
		Completed respjson.Field
		raw       string
	} `json:"-"`
}

func (r *MessageUnionTime) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// MessageUnionSummary is an implicit subunion of [MessageUnion].
// MessageUnionSummary provides convenient access to the sub-properties of the
// union.
//
// For type safety it is recommended to directly use a variant of the
// [MessageUnion].
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfBool]
type MessageUnionSummary struct {
	// This field will be present if the value is a [bool] instead of an object.
	OfBool bool `json:",inline"`
	// This field is from variant [MessageUserMessageSummary].
	Diffs []FileDiff `json:"diffs"`
	// This field is from variant [MessageUserMessageSummary].
	Body string `json:"body"`
	// This field is from variant [MessageUserMessageSummary].
	Title string `json:"title"`
	JSON  struct {
		OfBool respjson.Field
		Diffs  respjson.Field
		Body   respjson.Field
		Title  respjson.Field
		raw    string
	} `json:"-"`
}

func (r *MessageUnionSummary) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type MessageUserMessage struct {
	ID        string                    `json:"id,required"`
	Agent     string                    `json:"agent,required"`
	Model     MessageUserMessageModel   `json:"model,required"`
	Role      constant.User             `json:"role,required"`
	SessionID string                    `json:"sessionID,required"`
	Time      MessageUserMessageTime    `json:"time,required"`
	Summary   MessageUserMessageSummary `json:"summary"`
	System    string                    `json:"system"`
	Tools     map[string]bool           `json:"tools"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Agent       respjson.Field
		Model       respjson.Field
		Role        respjson.Field
		SessionID   respjson.Field
		Time        respjson.Field
		Summary     respjson.Field
		System      respjson.Field
		Tools       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r MessageUserMessage) RawJSON() string { return r.JSON.raw }
func (r *MessageUserMessage) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type MessageUserMessageModel struct {
	ModelID    string `json:"modelID,required"`
	ProviderID string `json:"providerID,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ModelID     respjson.Field
		ProviderID  respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r MessageUserMessageModel) RawJSON() string { return r.JSON.raw }
func (r *MessageUserMessageModel) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type MessageUserMessageTime struct {
	Created float64 `json:"created,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Created     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r MessageUserMessageTime) RawJSON() string { return r.JSON.raw }
func (r *MessageUserMessageTime) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type MessageUserMessageSummary struct {
	Diffs []FileDiff `json:"diffs,required"`
	Body  string     `json:"body"`
	Title string     `json:"title"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Diffs       respjson.Field
		Body        respjson.Field
		Title       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r MessageUserMessageSummary) RawJSON() string { return r.JSON.raw }
func (r *MessageUserMessageSummary) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// PartUnion contains all possible properties and values from [PartTextPart],
// [PartObject], [PartReasoningPart], [FilePart], [PartToolPart],
// [PartStepStartPart], [PartStepFinishPart], [PartSnapshotPart], [PartPatchPart],
// [PartAgentPart], [PartRetryPart], [PartCompactionPart].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type PartUnion struct {
	ID        string `json:"id"`
	MessageID string `json:"messageID"`
	SessionID string `json:"sessionID"`
	Text      string `json:"text"`
	Type      string `json:"type"`
	// This field is from variant [PartTextPart].
	Ignored  bool `json:"ignored"`
	Metadata any  `json:"metadata"`
	// This field is from variant [PartTextPart].
	Synthetic bool `json:"synthetic"`
	// This field is a union of [PartTextPartTime], [PartReasoningPartTime],
	// [PartRetryPartTime]
	Time PartUnionTime `json:"time"`
	// This field is from variant [PartObject].
	Agent string `json:"agent"`
	// This field is from variant [PartObject].
	Description string `json:"description"`
	// This field is from variant [PartObject].
	Prompt string `json:"prompt"`
	// This field is from variant [FilePart].
	Mime string `json:"mime"`
	// This field is from variant [FilePart].
	URL string `json:"url"`
	// This field is from variant [FilePart].
	Filename string `json:"filename"`
	// This field is a union of [FilePartSourceUnion], [PartAgentPartSource]
	Source PartUnionSource `json:"source"`
	// This field is from variant [PartToolPart].
	CallID string `json:"callID"`
	// This field is from variant [PartToolPart].
	State PartToolPartStateUnion `json:"state"`
	// This field is from variant [PartToolPart].
	Tool     string `json:"tool"`
	Snapshot string `json:"snapshot"`
	// This field is from variant [PartStepFinishPart].
	Cost float64 `json:"cost"`
	// This field is from variant [PartStepFinishPart].
	Reason string `json:"reason"`
	// This field is from variant [PartStepFinishPart].
	Tokens PartStepFinishPartTokens `json:"tokens"`
	// This field is from variant [PartPatchPart].
	Files []string `json:"files"`
	// This field is from variant [PartPatchPart].
	Hash string `json:"hash"`
	// This field is from variant [PartAgentPart].
	Name string `json:"name"`
	// This field is from variant [PartRetryPart].
	Attempt float64 `json:"attempt"`
	// This field is from variant [PartRetryPart].
	Error APIError `json:"error"`
	JSON  struct {
		ID          respjson.Field
		MessageID   respjson.Field
		SessionID   respjson.Field
		Text        respjson.Field
		Type        respjson.Field
		Ignored     respjson.Field
		Metadata    respjson.Field
		Synthetic   respjson.Field
		Time        respjson.Field
		Agent       respjson.Field
		Description respjson.Field
		Prompt      respjson.Field
		Mime        respjson.Field
		URL         respjson.Field
		Filename    respjson.Field
		Source      respjson.Field
		CallID      respjson.Field
		State       respjson.Field
		Tool        respjson.Field
		Snapshot    respjson.Field
		Cost        respjson.Field
		Reason      respjson.Field
		Tokens      respjson.Field
		Files       respjson.Field
		Hash        respjson.Field
		Name        respjson.Field
		Attempt     respjson.Field
		Error       respjson.Field
		raw         string
	} `json:"-"`
}

func (u PartUnion) AsPartTextPart() (v PartTextPart) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u PartUnion) AsPartObject() (v PartObject) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u PartUnion) AsPartReasoningPart() (v PartReasoningPart) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u PartUnion) AsFilePart() (v FilePart) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u PartUnion) AsPartToolPart() (v PartToolPart) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u PartUnion) AsPartStepStartPart() (v PartStepStartPart) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u PartUnion) AsPartStepFinishPart() (v PartStepFinishPart) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u PartUnion) AsPartSnapshotPart() (v PartSnapshotPart) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u PartUnion) AsPartPatchPart() (v PartPatchPart) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u PartUnion) AsPartAgentPart() (v PartAgentPart) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u PartUnion) AsPartRetryPart() (v PartRetryPart) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u PartUnion) AsPartCompactionPart() (v PartCompactionPart) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u PartUnion) RawJSON() string { return u.JSON.raw }

func (r *PartUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// PartUnionTime is an implicit subunion of [PartUnion]. PartUnionTime provides
// convenient access to the sub-properties of the union.
//
// For type safety it is recommended to directly use a variant of the [PartUnion].
type PartUnionTime struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	// This field is from variant [PartRetryPartTime].
	Created float64 `json:"created"`
	JSON    struct {
		Start   respjson.Field
		End     respjson.Field
		Created respjson.Field
		raw     string
	} `json:"-"`
}

func (r *PartUnionTime) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// PartUnionSource is an implicit subunion of [PartUnion]. PartUnionSource provides
// convenient access to the sub-properties of the union.
//
// For type safety it is recommended to directly use a variant of the [PartUnion].
type PartUnionSource struct {
	Path string `json:"path"`
	// This field is from variant [FilePartSourceUnion].
	Text FilePartSourceText `json:"text"`
	Type string             `json:"type"`
	// This field is from variant [FilePartSourceUnion].
	Kind int64 `json:"kind"`
	// This field is from variant [FilePartSourceUnion].
	Name string `json:"name"`
	// This field is from variant [FilePartSourceUnion].
	Range Range `json:"range"`
	// This field is from variant [PartAgentPartSource].
	End int64 `json:"end"`
	// This field is from variant [PartAgentPartSource].
	Start int64 `json:"start"`
	// This field is from variant [PartAgentPartSource].
	Value string `json:"value"`
	JSON  struct {
		Path  respjson.Field
		Text  respjson.Field
		Type  respjson.Field
		Kind  respjson.Field
		Name  respjson.Field
		Range respjson.Field
		End   respjson.Field
		Start respjson.Field
		Value respjson.Field
		raw   string
	} `json:"-"`
}

func (r *PartUnionSource) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartTextPart struct {
	ID        string           `json:"id,required"`
	MessageID string           `json:"messageID,required"`
	SessionID string           `json:"sessionID,required"`
	Text      string           `json:"text,required"`
	Type      constant.Text    `json:"type,required"`
	Ignored   bool             `json:"ignored"`
	Metadata  map[string]any   `json:"metadata"`
	Synthetic bool             `json:"synthetic"`
	Time      PartTextPartTime `json:"time"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		MessageID   respjson.Field
		SessionID   respjson.Field
		Text        respjson.Field
		Type        respjson.Field
		Ignored     respjson.Field
		Metadata    respjson.Field
		Synthetic   respjson.Field
		Time        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartTextPart) RawJSON() string { return r.JSON.raw }
func (r *PartTextPart) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartTextPartTime struct {
	Start float64 `json:"start,required"`
	End   float64 `json:"end"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Start       respjson.Field
		End         respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartTextPartTime) RawJSON() string { return r.JSON.raw }
func (r *PartTextPartTime) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartObject struct {
	ID          string           `json:"id,required"`
	Agent       string           `json:"agent,required"`
	Description string           `json:"description,required"`
	MessageID   string           `json:"messageID,required"`
	Prompt      string           `json:"prompt,required"`
	SessionID   string           `json:"sessionID,required"`
	Type        constant.Subtask `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Agent       respjson.Field
		Description respjson.Field
		MessageID   respjson.Field
		Prompt      respjson.Field
		SessionID   respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartObject) RawJSON() string { return r.JSON.raw }
func (r *PartObject) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartReasoningPart struct {
	ID        string                `json:"id,required"`
	MessageID string                `json:"messageID,required"`
	SessionID string                `json:"sessionID,required"`
	Text      string                `json:"text,required"`
	Time      PartReasoningPartTime `json:"time,required"`
	Type      constant.Reasoning    `json:"type,required"`
	Metadata  map[string]any        `json:"metadata"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		MessageID   respjson.Field
		SessionID   respjson.Field
		Text        respjson.Field
		Time        respjson.Field
		Type        respjson.Field
		Metadata    respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartReasoningPart) RawJSON() string { return r.JSON.raw }
func (r *PartReasoningPart) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartReasoningPartTime struct {
	Start float64 `json:"start,required"`
	End   float64 `json:"end"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Start       respjson.Field
		End         respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartReasoningPartTime) RawJSON() string { return r.JSON.raw }
func (r *PartReasoningPartTime) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartToolPart struct {
	ID        string                 `json:"id,required"`
	CallID    string                 `json:"callID,required"`
	MessageID string                 `json:"messageID,required"`
	SessionID string                 `json:"sessionID,required"`
	State     PartToolPartStateUnion `json:"state,required"`
	Tool      string                 `json:"tool,required"`
	Type      constant.Tool          `json:"type,required"`
	Metadata  map[string]any         `json:"metadata"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		CallID      respjson.Field
		MessageID   respjson.Field
		SessionID   respjson.Field
		State       respjson.Field
		Tool        respjson.Field
		Type        respjson.Field
		Metadata    respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartToolPart) RawJSON() string { return r.JSON.raw }
func (r *PartToolPart) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// PartToolPartStateUnion contains all possible properties and values from
// [PartToolPartStateToolStatePending], [PartToolPartStateToolStateRunning],
// [PartToolPartStateToolStateCompleted], [PartToolPartStateToolStateError].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type PartToolPartStateUnion struct {
	Input any `json:"input"`
	// This field is from variant [PartToolPartStateToolStatePending].
	Raw    string `json:"raw"`
	Status string `json:"status"`
	// This field is a union of [PartToolPartStateToolStateRunningTime],
	// [PartToolPartStateToolStateCompletedTime], [PartToolPartStateToolStateErrorTime]
	Time     PartToolPartStateUnionTime `json:"time"`
	Metadata any                        `json:"metadata"`
	Title    string                     `json:"title"`
	// This field is from variant [PartToolPartStateToolStateCompleted].
	Output string `json:"output"`
	// This field is from variant [PartToolPartStateToolStateCompleted].
	Attachments []FilePart `json:"attachments"`
	// This field is from variant [PartToolPartStateToolStateError].
	Error string `json:"error"`
	JSON  struct {
		Input       respjson.Field
		Raw         respjson.Field
		Status      respjson.Field
		Time        respjson.Field
		Metadata    respjson.Field
		Title       respjson.Field
		Output      respjson.Field
		Attachments respjson.Field
		Error       respjson.Field
		raw         string
	} `json:"-"`
}

func (u PartToolPartStateUnion) AsPartToolPartStateToolStatePending() (v PartToolPartStateToolStatePending) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u PartToolPartStateUnion) AsPartToolPartStateToolStateRunning() (v PartToolPartStateToolStateRunning) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u PartToolPartStateUnion) AsPartToolPartStateToolStateCompleted() (v PartToolPartStateToolStateCompleted) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u PartToolPartStateUnion) AsPartToolPartStateToolStateError() (v PartToolPartStateToolStateError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u PartToolPartStateUnion) RawJSON() string { return u.JSON.raw }

func (r *PartToolPartStateUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// PartToolPartStateUnionTime is an implicit subunion of [PartToolPartStateUnion].
// PartToolPartStateUnionTime provides convenient access to the sub-properties of
// the union.
//
// For type safety it is recommended to directly use a variant of the
// [PartToolPartStateUnion].
type PartToolPartStateUnionTime struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	// This field is from variant [PartToolPartStateToolStateCompletedTime].
	Compacted float64 `json:"compacted"`
	JSON      struct {
		Start     respjson.Field
		End       respjson.Field
		Compacted respjson.Field
		raw       string
	} `json:"-"`
}

func (r *PartToolPartStateUnionTime) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartToolPartStateToolStatePending struct {
	Input  map[string]any   `json:"input,required"`
	Raw    string           `json:"raw,required"`
	Status constant.Pending `json:"status,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Input       respjson.Field
		Raw         respjson.Field
		Status      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartToolPartStateToolStatePending) RawJSON() string { return r.JSON.raw }
func (r *PartToolPartStateToolStatePending) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartToolPartStateToolStateRunning struct {
	Input    map[string]any                        `json:"input,required"`
	Status   constant.Running                      `json:"status,required"`
	Time     PartToolPartStateToolStateRunningTime `json:"time,required"`
	Metadata map[string]any                        `json:"metadata"`
	Title    string                                `json:"title"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Input       respjson.Field
		Status      respjson.Field
		Time        respjson.Field
		Metadata    respjson.Field
		Title       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartToolPartStateToolStateRunning) RawJSON() string { return r.JSON.raw }
func (r *PartToolPartStateToolStateRunning) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartToolPartStateToolStateRunningTime struct {
	Start float64 `json:"start,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Start       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartToolPartStateToolStateRunningTime) RawJSON() string { return r.JSON.raw }
func (r *PartToolPartStateToolStateRunningTime) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartToolPartStateToolStateCompleted struct {
	Input       map[string]any                          `json:"input,required"`
	Metadata    map[string]any                          `json:"metadata,required"`
	Output      string                                  `json:"output,required"`
	Status      constant.Completed                      `json:"status,required"`
	Time        PartToolPartStateToolStateCompletedTime `json:"time,required"`
	Title       string                                  `json:"title,required"`
	Attachments []FilePart                              `json:"attachments"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Input       respjson.Field
		Metadata    respjson.Field
		Output      respjson.Field
		Status      respjson.Field
		Time        respjson.Field
		Title       respjson.Field
		Attachments respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartToolPartStateToolStateCompleted) RawJSON() string { return r.JSON.raw }
func (r *PartToolPartStateToolStateCompleted) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartToolPartStateToolStateCompletedTime struct {
	End       float64 `json:"end,required"`
	Start     float64 `json:"start,required"`
	Compacted float64 `json:"compacted"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		End         respjson.Field
		Start       respjson.Field
		Compacted   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartToolPartStateToolStateCompletedTime) RawJSON() string { return r.JSON.raw }
func (r *PartToolPartStateToolStateCompletedTime) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartToolPartStateToolStateError struct {
	Error    string                              `json:"error,required"`
	Input    map[string]any                      `json:"input,required"`
	Status   constant.Error                      `json:"status,required"`
	Time     PartToolPartStateToolStateErrorTime `json:"time,required"`
	Metadata map[string]any                      `json:"metadata"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Error       respjson.Field
		Input       respjson.Field
		Status      respjson.Field
		Time        respjson.Field
		Metadata    respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartToolPartStateToolStateError) RawJSON() string { return r.JSON.raw }
func (r *PartToolPartStateToolStateError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartToolPartStateToolStateErrorTime struct {
	End   float64 `json:"end,required"`
	Start float64 `json:"start,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		End         respjson.Field
		Start       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartToolPartStateToolStateErrorTime) RawJSON() string { return r.JSON.raw }
func (r *PartToolPartStateToolStateErrorTime) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartStepStartPart struct {
	ID        string             `json:"id,required"`
	MessageID string             `json:"messageID,required"`
	SessionID string             `json:"sessionID,required"`
	Type      constant.StepStart `json:"type,required"`
	Snapshot  string             `json:"snapshot"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		MessageID   respjson.Field
		SessionID   respjson.Field
		Type        respjson.Field
		Snapshot    respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartStepStartPart) RawJSON() string { return r.JSON.raw }
func (r *PartStepStartPart) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartStepFinishPart struct {
	ID        string                   `json:"id,required"`
	Cost      float64                  `json:"cost,required"`
	MessageID string                   `json:"messageID,required"`
	Reason    string                   `json:"reason,required"`
	SessionID string                   `json:"sessionID,required"`
	Tokens    PartStepFinishPartTokens `json:"tokens,required"`
	Type      constant.StepFinish      `json:"type,required"`
	Snapshot  string                   `json:"snapshot"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Cost        respjson.Field
		MessageID   respjson.Field
		Reason      respjson.Field
		SessionID   respjson.Field
		Tokens      respjson.Field
		Type        respjson.Field
		Snapshot    respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartStepFinishPart) RawJSON() string { return r.JSON.raw }
func (r *PartStepFinishPart) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartStepFinishPartTokens struct {
	Cache     PartStepFinishPartTokensCache `json:"cache,required"`
	Input     float64                       `json:"input,required"`
	Output    float64                       `json:"output,required"`
	Reasoning float64                       `json:"reasoning,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Cache       respjson.Field
		Input       respjson.Field
		Output      respjson.Field
		Reasoning   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartStepFinishPartTokens) RawJSON() string { return r.JSON.raw }
func (r *PartStepFinishPartTokens) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartStepFinishPartTokensCache struct {
	Read  float64 `json:"read,required"`
	Write float64 `json:"write,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Read        respjson.Field
		Write       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartStepFinishPartTokensCache) RawJSON() string { return r.JSON.raw }
func (r *PartStepFinishPartTokensCache) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartSnapshotPart struct {
	ID        string            `json:"id,required"`
	MessageID string            `json:"messageID,required"`
	SessionID string            `json:"sessionID,required"`
	Snapshot  string            `json:"snapshot,required"`
	Type      constant.Snapshot `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		MessageID   respjson.Field
		SessionID   respjson.Field
		Snapshot    respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartSnapshotPart) RawJSON() string { return r.JSON.raw }
func (r *PartSnapshotPart) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartPatchPart struct {
	ID        string         `json:"id,required"`
	Files     []string       `json:"files,required"`
	Hash      string         `json:"hash,required"`
	MessageID string         `json:"messageID,required"`
	SessionID string         `json:"sessionID,required"`
	Type      constant.Patch `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Files       respjson.Field
		Hash        respjson.Field
		MessageID   respjson.Field
		SessionID   respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartPatchPart) RawJSON() string { return r.JSON.raw }
func (r *PartPatchPart) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartAgentPart struct {
	ID        string              `json:"id,required"`
	MessageID string              `json:"messageID,required"`
	Name      string              `json:"name,required"`
	SessionID string              `json:"sessionID,required"`
	Type      constant.Agent      `json:"type,required"`
	Source    PartAgentPartSource `json:"source"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		MessageID   respjson.Field
		Name        respjson.Field
		SessionID   respjson.Field
		Type        respjson.Field
		Source      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartAgentPart) RawJSON() string { return r.JSON.raw }
func (r *PartAgentPart) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartAgentPartSource struct {
	End   int64  `json:"end,required"`
	Start int64  `json:"start,required"`
	Value string `json:"value,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		End         respjson.Field
		Start       respjson.Field
		Value       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartAgentPartSource) RawJSON() string { return r.JSON.raw }
func (r *PartAgentPartSource) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartRetryPart struct {
	ID        string            `json:"id,required"`
	Attempt   float64           `json:"attempt,required"`
	Error     APIError          `json:"error,required"`
	MessageID string            `json:"messageID,required"`
	SessionID string            `json:"sessionID,required"`
	Time      PartRetryPartTime `json:"time,required"`
	Type      constant.Retry    `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Attempt     respjson.Field
		Error       respjson.Field
		MessageID   respjson.Field
		SessionID   respjson.Field
		Time        respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartRetryPart) RawJSON() string { return r.JSON.raw }
func (r *PartRetryPart) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartRetryPartTime struct {
	Created float64 `json:"created,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Created     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartRetryPartTime) RawJSON() string { return r.JSON.raw }
func (r *PartRetryPartTime) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type PartCompactionPart struct {
	ID        string              `json:"id,required"`
	MessageID string              `json:"messageID,required"`
	SessionID string              `json:"sessionID,required"`
	Type      constant.Compaction `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		MessageID   respjson.Field
		SessionID   respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PartCompactionPart) RawJSON() string { return r.JSON.raw }
func (r *PartCompactionPart) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type SessionMessageNewResponse struct {
	Info  AssistantMessage `json:"info,required"`
	Parts []PartUnion      `json:"parts,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Info        respjson.Field
		Parts       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r SessionMessageNewResponse) RawJSON() string { return r.JSON.raw }
func (r *SessionMessageNewResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type SessionMessageGetResponse struct {
	Info  MessageUnion `json:"info,required"`
	Parts []PartUnion  `json:"parts,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Info        respjson.Field
		Parts       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r SessionMessageGetResponse) RawJSON() string { return r.JSON.raw }
func (r *SessionMessageGetResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type SessionMessageListResponse struct {
	Info  MessageUnion `json:"info,required"`
	Parts []PartUnion  `json:"parts,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Info        respjson.Field
		Parts       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r SessionMessageListResponse) RawJSON() string { return r.JSON.raw }
func (r *SessionMessageListResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type SessionMessageNewParams struct {
	Parts     []SessionMessageNewParamsPartUnion `json:"parts,omitzero,required"`
	Directory param.Opt[string]                  `query:"directory,omitzero" json:"-"`
	Agent     param.Opt[string]                  `json:"agent,omitzero"`
	MessageID param.Opt[string]                  `json:"messageID,omitzero"`
	NoReply   param.Opt[bool]                    `json:"noReply,omitzero"`
	System    param.Opt[string]                  `json:"system,omitzero"`
	Model     SessionMessageNewParamsModel       `json:"model,omitzero"`
	Tools     map[string]bool                    `json:"tools,omitzero"`
	paramObj
}

func (r SessionMessageNewParams) MarshalJSON() (data []byte, err error) {
	type shadow SessionMessageNewParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionMessageNewParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [SessionMessageNewParams]'s query parameters as
// `url.Values`.
func (r SessionMessageNewParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type SessionMessageNewParamsPartUnion struct {
	OfSessionMessageNewsPartTextPartInput    *SessionMessageNewParamsPartTextPartInput    `json:",omitzero,inline"`
	OfSessionMessageNewsPartFilePartInput    *SessionMessageNewParamsPartFilePartInput    `json:",omitzero,inline"`
	OfSessionMessageNewsPartAgentPartInput   *SessionMessageNewParamsPartAgentPartInput   `json:",omitzero,inline"`
	OfSessionMessageNewsPartSubtaskPartInput *SessionMessageNewParamsPartSubtaskPartInput `json:",omitzero,inline"`
	paramUnion
}

func (u SessionMessageNewParamsPartUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfSessionMessageNewsPartTextPartInput, u.OfSessionMessageNewsPartFilePartInput, u.OfSessionMessageNewsPartAgentPartInput, u.OfSessionMessageNewsPartSubtaskPartInput)
}
func (u *SessionMessageNewParamsPartUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *SessionMessageNewParamsPartUnion) asAny() any {
	if !param.IsOmitted(u.OfSessionMessageNewsPartTextPartInput) {
		return u.OfSessionMessageNewsPartTextPartInput
	} else if !param.IsOmitted(u.OfSessionMessageNewsPartFilePartInput) {
		return u.OfSessionMessageNewsPartFilePartInput
	} else if !param.IsOmitted(u.OfSessionMessageNewsPartAgentPartInput) {
		return u.OfSessionMessageNewsPartAgentPartInput
	} else if !param.IsOmitted(u.OfSessionMessageNewsPartSubtaskPartInput) {
		return u.OfSessionMessageNewsPartSubtaskPartInput
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u SessionMessageNewParamsPartUnion) GetText() *string {
	if vt := u.OfSessionMessageNewsPartTextPartInput; vt != nil {
		return &vt.Text
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u SessionMessageNewParamsPartUnion) GetIgnored() *bool {
	if vt := u.OfSessionMessageNewsPartTextPartInput; vt != nil && vt.Ignored.Valid() {
		return &vt.Ignored.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u SessionMessageNewParamsPartUnion) GetMetadata() map[string]any {
	if vt := u.OfSessionMessageNewsPartTextPartInput; vt != nil {
		return vt.Metadata
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u SessionMessageNewParamsPartUnion) GetSynthetic() *bool {
	if vt := u.OfSessionMessageNewsPartTextPartInput; vt != nil && vt.Synthetic.Valid() {
		return &vt.Synthetic.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u SessionMessageNewParamsPartUnion) GetTime() *SessionMessageNewParamsPartTextPartInputTime {
	if vt := u.OfSessionMessageNewsPartTextPartInput; vt != nil {
		return &vt.Time
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u SessionMessageNewParamsPartUnion) GetMime() *string {
	if vt := u.OfSessionMessageNewsPartFilePartInput; vt != nil {
		return &vt.Mime
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u SessionMessageNewParamsPartUnion) GetURL() *string {
	if vt := u.OfSessionMessageNewsPartFilePartInput; vt != nil {
		return &vt.URL
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u SessionMessageNewParamsPartUnion) GetFilename() *string {
	if vt := u.OfSessionMessageNewsPartFilePartInput; vt != nil && vt.Filename.Valid() {
		return &vt.Filename.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u SessionMessageNewParamsPartUnion) GetName() *string {
	if vt := u.OfSessionMessageNewsPartAgentPartInput; vt != nil {
		return &vt.Name
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u SessionMessageNewParamsPartUnion) GetAgent() *string {
	if vt := u.OfSessionMessageNewsPartSubtaskPartInput; vt != nil {
		return &vt.Agent
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u SessionMessageNewParamsPartUnion) GetDescription() *string {
	if vt := u.OfSessionMessageNewsPartSubtaskPartInput; vt != nil {
		return &vt.Description
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u SessionMessageNewParamsPartUnion) GetPrompt() *string {
	if vt := u.OfSessionMessageNewsPartSubtaskPartInput; vt != nil {
		return &vt.Prompt
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u SessionMessageNewParamsPartUnion) GetType() *string {
	if vt := u.OfSessionMessageNewsPartTextPartInput; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfSessionMessageNewsPartFilePartInput; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfSessionMessageNewsPartAgentPartInput; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfSessionMessageNewsPartSubtaskPartInput; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u SessionMessageNewParamsPartUnion) GetID() *string {
	if vt := u.OfSessionMessageNewsPartTextPartInput; vt != nil && vt.ID.Valid() {
		return &vt.ID.Value
	} else if vt := u.OfSessionMessageNewsPartFilePartInput; vt != nil && vt.ID.Valid() {
		return &vt.ID.Value
	} else if vt := u.OfSessionMessageNewsPartAgentPartInput; vt != nil && vt.ID.Valid() {
		return &vt.ID.Value
	} else if vt := u.OfSessionMessageNewsPartSubtaskPartInput; vt != nil && vt.ID.Valid() {
		return &vt.ID.Value
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u SessionMessageNewParamsPartUnion) GetSource() (res sessionMessageNewParamsPartUnionSource) {
	if vt := u.OfSessionMessageNewsPartFilePartInput; vt != nil {
		res.any = vt.Source.asAny()
	} else if vt := u.OfSessionMessageNewsPartAgentPartInput; vt != nil {
		res.any = &vt.Source
	}
	return
}

// Can have the runtime types [*FilePartSourceFileSourceParam],
// [*FilePartSourceSymbolSourceParam],
// [*SessionMessageNewParamsPartAgentPartInputSource]
type sessionMessageNewParamsPartUnionSource struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *opencode.FilePartSourceFileSourceParam:
//	case *opencode.FilePartSourceSymbolSourceParam:
//	case *opencode.SessionMessageNewParamsPartAgentPartInputSource:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u sessionMessageNewParamsPartUnionSource) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's property, if present.
func (u sessionMessageNewParamsPartUnionSource) GetKind() *int64 {
	switch vt := u.any.(type) {
	case *FilePartSourceUnionParam:
		return vt.GetKind()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u sessionMessageNewParamsPartUnionSource) GetName() *string {
	switch vt := u.any.(type) {
	case *FilePartSourceUnionParam:
		return vt.GetName()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u sessionMessageNewParamsPartUnionSource) GetRange() *RangeParam {
	switch vt := u.any.(type) {
	case *FilePartSourceUnionParam:
		return vt.GetRange()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u sessionMessageNewParamsPartUnionSource) GetEnd() *int64 {
	switch vt := u.any.(type) {
	case *SessionMessageNewParamsPartAgentPartInputSource:
		return &vt.End
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u sessionMessageNewParamsPartUnionSource) GetStart() *int64 {
	switch vt := u.any.(type) {
	case *SessionMessageNewParamsPartAgentPartInputSource:
		return &vt.Start
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u sessionMessageNewParamsPartUnionSource) GetValue() *string {
	switch vt := u.any.(type) {
	case *SessionMessageNewParamsPartAgentPartInputSource:
		return &vt.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u sessionMessageNewParamsPartUnionSource) GetPath() *string {
	switch vt := u.any.(type) {
	case *FilePartSourceUnionParam:
		return vt.GetPath()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u sessionMessageNewParamsPartUnionSource) GetType() *string {
	switch vt := u.any.(type) {
	case *FilePartSourceUnionParam:
		return vt.GetType()
	}
	return nil
}

// Returns a pointer to the underlying variant's Text property, if present.
func (u sessionMessageNewParamsPartUnionSource) GetText() *FilePartSourceTextParam {
	switch vt := u.any.(type) {
	case *FilePartSourceUnionParam:
		return vt.GetText()
	}
	return nil
}

// The properties Text, Type are required.
type SessionMessageNewParamsPartTextPartInput struct {
	Text      string                                       `json:"text,required"`
	ID        param.Opt[string]                            `json:"id,omitzero"`
	Ignored   param.Opt[bool]                              `json:"ignored,omitzero"`
	Synthetic param.Opt[bool]                              `json:"synthetic,omitzero"`
	Metadata  map[string]any                               `json:"metadata,omitzero"`
	Time      SessionMessageNewParamsPartTextPartInputTime `json:"time,omitzero"`
	// This field can be elided, and will marshal its zero value as "text".
	Type constant.Text `json:"type,required"`
	paramObj
}

func (r SessionMessageNewParamsPartTextPartInput) MarshalJSON() (data []byte, err error) {
	type shadow SessionMessageNewParamsPartTextPartInput
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionMessageNewParamsPartTextPartInput) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The property Start is required.
type SessionMessageNewParamsPartTextPartInputTime struct {
	Start float64            `json:"start,required"`
	End   param.Opt[float64] `json:"end,omitzero"`
	paramObj
}

func (r SessionMessageNewParamsPartTextPartInputTime) MarshalJSON() (data []byte, err error) {
	type shadow SessionMessageNewParamsPartTextPartInputTime
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionMessageNewParamsPartTextPartInputTime) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Mime, Type, URL are required.
type SessionMessageNewParamsPartFilePartInput struct {
	Mime     string                   `json:"mime,required"`
	URL      string                   `json:"url,required"`
	ID       param.Opt[string]        `json:"id,omitzero"`
	Filename param.Opt[string]        `json:"filename,omitzero"`
	Source   FilePartSourceUnionParam `json:"source,omitzero"`
	// This field can be elided, and will marshal its zero value as "file".
	Type constant.File `json:"type,required"`
	paramObj
}

func (r SessionMessageNewParamsPartFilePartInput) MarshalJSON() (data []byte, err error) {
	type shadow SessionMessageNewParamsPartFilePartInput
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionMessageNewParamsPartFilePartInput) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type SessionMessageNewParamsPartAgentPartInput struct {
	Name   string                                          `json:"name,required"`
	ID     param.Opt[string]                               `json:"id,omitzero"`
	Source SessionMessageNewParamsPartAgentPartInputSource `json:"source,omitzero"`
	// This field can be elided, and will marshal its zero value as "agent".
	Type constant.Agent `json:"type,required"`
	paramObj
}

func (r SessionMessageNewParamsPartAgentPartInput) MarshalJSON() (data []byte, err error) {
	type shadow SessionMessageNewParamsPartAgentPartInput
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionMessageNewParamsPartAgentPartInput) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties End, Start, Value are required.
type SessionMessageNewParamsPartAgentPartInputSource struct {
	End   int64  `json:"end,required"`
	Start int64  `json:"start,required"`
	Value string `json:"value,required"`
	paramObj
}

func (r SessionMessageNewParamsPartAgentPartInputSource) MarshalJSON() (data []byte, err error) {
	type shadow SessionMessageNewParamsPartAgentPartInputSource
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionMessageNewParamsPartAgentPartInputSource) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Agent, Description, Prompt, Type are required.
type SessionMessageNewParamsPartSubtaskPartInput struct {
	Agent       string            `json:"agent,required"`
	Description string            `json:"description,required"`
	Prompt      string            `json:"prompt,required"`
	ID          param.Opt[string] `json:"id,omitzero"`
	// This field can be elided, and will marshal its zero value as "subtask".
	Type constant.Subtask `json:"type,required"`
	paramObj
}

func (r SessionMessageNewParamsPartSubtaskPartInput) MarshalJSON() (data []byte, err error) {
	type shadow SessionMessageNewParamsPartSubtaskPartInput
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionMessageNewParamsPartSubtaskPartInput) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties ModelID, ProviderID are required.
type SessionMessageNewParamsModel struct {
	ModelID    string `json:"modelID,required"`
	ProviderID string `json:"providerID,required"`
	paramObj
}

func (r SessionMessageNewParamsModel) MarshalJSON() (data []byte, err error) {
	type shadow SessionMessageNewParamsModel
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionMessageNewParamsModel) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type SessionMessageGetParams struct {
	ID        string            `path:"id,required" json:"-"`
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [SessionMessageGetParams]'s query parameters as
// `url.Values`.
func (r SessionMessageGetParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type SessionMessageListParams struct {
	Directory param.Opt[string]  `query:"directory,omitzero" json:"-"`
	Limit     param.Opt[float64] `query:"limit,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [SessionMessageListParams]'s query parameters as
// `url.Values`.
func (r SessionMessageListParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}
