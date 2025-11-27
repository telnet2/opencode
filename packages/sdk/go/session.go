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

// SessionService contains methods and other services that help with interacting
// with the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewSessionService] method instead.
type SessionService struct {
	Options []option.RequestOption
	Share   SessionShareService
	Message SessionMessageService
}

// NewSessionService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewSessionService(opts ...option.RequestOption) (r SessionService) {
	r = SessionService{}
	r.Options = opts
	r.Share = NewSessionShareService(opts...)
	r.Message = NewSessionMessageService(opts...)
	return
}

// Create a new session
func (r *SessionService) New(ctx context.Context, params SessionNewParams, opts ...option.RequestOption) (res *Session, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "session"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

// Get session
func (r *SessionService) Get(ctx context.Context, id string, query SessionGetParams, opts ...option.RequestOption) (res *Session, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("session/%s", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

// Update session properties
func (r *SessionService) Update(ctx context.Context, id string, params SessionUpdateParams, opts ...option.RequestOption) (res *Session, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("session/%s", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPatch, path, params, &res, opts...)
	return
}

// List all sessions
func (r *SessionService) List(ctx context.Context, query SessionListParams, opts ...option.RequestOption) (res *[]Session, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "session"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

// Delete a session and all its data
func (r *SessionService) Delete(ctx context.Context, id string, body SessionDeleteParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("session/%s", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodDelete, path, body, &res, opts...)
	return
}

// Abort a session
func (r *SessionService) Abort(ctx context.Context, id string, body SessionAbortParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("session/%s/abort", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

// Analyze the app and create an AGENTS.md file
func (r *SessionService) Analyze(ctx context.Context, id string, params SessionAnalyzeParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("session/%s/init", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

// Fork an existing session at a specific message
func (r *SessionService) Fork(ctx context.Context, id string, params SessionForkParams, opts ...option.RequestOption) (res *Session, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("session/%s/fork", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

// Get a session's children
func (r *SessionService) GetChildren(ctx context.Context, id string, query SessionGetChildrenParams, opts ...option.RequestOption) (res *[]Session, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("session/%s/children", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

// Get the diff for this session
func (r *SessionService) GetDiff(ctx context.Context, id string, query SessionGetDiffParams, opts ...option.RequestOption) (res *[]FileDiff, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("session/%s/diff", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

// Get session status
func (r *SessionService) GetStatus(ctx context.Context, query SessionGetStatusParams, opts ...option.RequestOption) (res *SessionGetStatusResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "session/status"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

// Get the todo list for a session
func (r *SessionService) GetTodo(ctx context.Context, id string, query SessionGetTodoParams, opts ...option.RequestOption) (res *[]Todo, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("session/%s/todo", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

// Respond to a permission request
func (r *SessionService) RespondToPermission(ctx context.Context, permissionID string, params SessionRespondToPermissionParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	if params.ID == "" {
		err = errors.New("missing required id parameter")
		return
	}
	if permissionID == "" {
		err = errors.New("missing required permissionID parameter")
		return
	}
	path := fmt.Sprintf("session/%s/permissions/%s", params.ID, permissionID)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

// Restore all reverted messages
func (r *SessionService) RestoreReverted(ctx context.Context, id string, body SessionRestoreRevertedParams, opts ...option.RequestOption) (res *Session, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("session/%s/unrevert", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

// Revert a message
func (r *SessionService) Revert(ctx context.Context, id string, params SessionRevertParams, opts ...option.RequestOption) (res *Session, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("session/%s/revert", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

// Run a shell command
func (r *SessionService) RunShell(ctx context.Context, id string, params SessionRunShellParams, opts ...option.RequestOption) (res *AssistantMessage, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("session/%s/shell", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

// Send a new command to a session
func (r *SessionService) SendCommand(ctx context.Context, id string, params SessionSendCommandParams, opts ...option.RequestOption) (res *SessionSendCommandResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("session/%s/command", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

// Summarize the session
func (r *SessionService) Summarize(ctx context.Context, id string, params SessionSummarizeParams, opts ...option.RequestOption) (res *bool, err error) {
	opts = slices.Concat(r.Options, opts)
	if id == "" {
		err = errors.New("missing required id parameter")
		return
	}
	path := fmt.Sprintf("session/%s/summarize", id)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

type APIError struct {
	Data APIErrorData      `json:"data,required"`
	Name constant.APIError `json:"name,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Data        respjson.Field
		Name        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r APIError) RawJSON() string { return r.JSON.raw }
func (r *APIError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type APIErrorData struct {
	IsRetryable     bool              `json:"isRetryable,required"`
	Message         string            `json:"message,required"`
	ResponseBody    string            `json:"responseBody"`
	ResponseHeaders map[string]string `json:"responseHeaders"`
	StatusCode      float64           `json:"statusCode"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		IsRetryable     respjson.Field
		Message         respjson.Field
		ResponseBody    respjson.Field
		ResponseHeaders respjson.Field
		StatusCode      respjson.Field
		ExtraFields     map[string]respjson.Field
		raw             string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r APIErrorData) RawJSON() string { return r.JSON.raw }
func (r *APIErrorData) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type AssistantMessage struct {
	ID         string                     `json:"id,required"`
	Cost       float64                    `json:"cost,required"`
	Mode       string                     `json:"mode,required"`
	ModelID    string                     `json:"modelID,required"`
	ParentID   string                     `json:"parentID,required"`
	Path       AssistantMessagePath       `json:"path,required"`
	ProviderID string                     `json:"providerID,required"`
	Role       constant.Assistant         `json:"role,required"`
	SessionID  string                     `json:"sessionID,required"`
	Time       AssistantMessageTime       `json:"time,required"`
	Tokens     AssistantMessageTokens     `json:"tokens,required"`
	Error      AssistantMessageErrorUnion `json:"error"`
	Finish     string                     `json:"finish"`
	Summary    bool                       `json:"summary"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Cost        respjson.Field
		Mode        respjson.Field
		ModelID     respjson.Field
		ParentID    respjson.Field
		Path        respjson.Field
		ProviderID  respjson.Field
		Role        respjson.Field
		SessionID   respjson.Field
		Time        respjson.Field
		Tokens      respjson.Field
		Error       respjson.Field
		Finish      respjson.Field
		Summary     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r AssistantMessage) RawJSON() string { return r.JSON.raw }
func (r *AssistantMessage) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type AssistantMessagePath struct {
	Cwd  string `json:"cwd,required"`
	Root string `json:"root,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Cwd         respjson.Field
		Root        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r AssistantMessagePath) RawJSON() string { return r.JSON.raw }
func (r *AssistantMessagePath) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type AssistantMessageTime struct {
	Created   float64 `json:"created,required"`
	Completed float64 `json:"completed"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Created     respjson.Field
		Completed   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r AssistantMessageTime) RawJSON() string { return r.JSON.raw }
func (r *AssistantMessageTime) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type AssistantMessageTokens struct {
	Cache     AssistantMessageTokensCache `json:"cache,required"`
	Input     float64                     `json:"input,required"`
	Output    float64                     `json:"output,required"`
	Reasoning float64                     `json:"reasoning,required"`
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
func (r AssistantMessageTokens) RawJSON() string { return r.JSON.raw }
func (r *AssistantMessageTokens) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type AssistantMessageTokensCache struct {
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
func (r AssistantMessageTokensCache) RawJSON() string { return r.JSON.raw }
func (r *AssistantMessageTokensCache) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// AssistantMessageErrorUnion contains all possible properties and values from
// [ProviderAuthError], [UnknownError], [MessageOutputLengthError],
// [MessageAbortedError], [APIError].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type AssistantMessageErrorUnion struct {
	// This field is a union of [ProviderAuthErrorData], [UnknownErrorData], [any],
	// [MessageAbortedErrorData], [APIErrorData]
	Data AssistantMessageErrorUnionData `json:"data"`
	Name string                         `json:"name"`
	JSON struct {
		Data respjson.Field
		Name respjson.Field
		raw  string
	} `json:"-"`
}

func (u AssistantMessageErrorUnion) AsProviderAuthError() (v ProviderAuthError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u AssistantMessageErrorUnion) AsUnknownError() (v UnknownError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u AssistantMessageErrorUnion) AsMessageOutputLengthError() (v MessageOutputLengthError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u AssistantMessageErrorUnion) AsMessageAbortedError() (v MessageAbortedError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u AssistantMessageErrorUnion) AsAPIError() (v APIError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u AssistantMessageErrorUnion) RawJSON() string { return u.JSON.raw }

func (r *AssistantMessageErrorUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// AssistantMessageErrorUnionData is an implicit subunion of
// [AssistantMessageErrorUnion]. AssistantMessageErrorUnionData provides convenient
// access to the sub-properties of the union.
//
// For type safety it is recommended to directly use a variant of the
// [AssistantMessageErrorUnion].
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfMessageOutputLengthErrorData]
type AssistantMessageErrorUnionData struct {
	// This field will be present if the value is a [any] instead of an object.
	OfMessageOutputLengthErrorData any    `json:",inline"`
	Message                        string `json:"message"`
	// This field is from variant [ProviderAuthErrorData].
	ProviderID string `json:"providerID"`
	// This field is from variant [APIErrorData].
	IsRetryable bool `json:"isRetryable"`
	// This field is from variant [APIErrorData].
	ResponseBody string `json:"responseBody"`
	// This field is from variant [APIErrorData].
	ResponseHeaders map[string]string `json:"responseHeaders"`
	// This field is from variant [APIErrorData].
	StatusCode float64 `json:"statusCode"`
	JSON       struct {
		OfMessageOutputLengthErrorData respjson.Field
		Message                        respjson.Field
		ProviderID                     respjson.Field
		IsRetryable                    respjson.Field
		ResponseBody                   respjson.Field
		ResponseHeaders                respjson.Field
		StatusCode                     respjson.Field
		raw                            string
	} `json:"-"`
}

func (r *AssistantMessageErrorUnionData) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type FileDiff struct {
	Additions float64 `json:"additions,required"`
	After     string  `json:"after,required"`
	Before    string  `json:"before,required"`
	Deletions float64 `json:"deletions,required"`
	File      string  `json:"file,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Additions   respjson.Field
		After       respjson.Field
		Before      respjson.Field
		Deletions   respjson.Field
		File        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r FileDiff) RawJSON() string { return r.JSON.raw }
func (r *FileDiff) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type MessageAbortedError struct {
	Data MessageAbortedErrorData      `json:"data,required"`
	Name constant.MessageAbortedError `json:"name,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Data        respjson.Field
		Name        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r MessageAbortedError) RawJSON() string { return r.JSON.raw }
func (r *MessageAbortedError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type MessageAbortedErrorData struct {
	Message string `json:"message,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Message     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r MessageAbortedErrorData) RawJSON() string { return r.JSON.raw }
func (r *MessageAbortedErrorData) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type MessageOutputLengthError struct {
	Data any                               `json:"data,required"`
	Name constant.MessageOutputLengthError `json:"name,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Data        respjson.Field
		Name        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r MessageOutputLengthError) RawJSON() string { return r.JSON.raw }
func (r *MessageOutputLengthError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ProviderAuthError struct {
	Data ProviderAuthErrorData      `json:"data,required"`
	Name constant.ProviderAuthError `json:"name,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Data        respjson.Field
		Name        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ProviderAuthError) RawJSON() string { return r.JSON.raw }
func (r *ProviderAuthError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ProviderAuthErrorData struct {
	Message    string `json:"message,required"`
	ProviderID string `json:"providerID,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Message     respjson.Field
		ProviderID  respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ProviderAuthErrorData) RawJSON() string { return r.JSON.raw }
func (r *ProviderAuthErrorData) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type Session struct {
	ID           string              `json:"id,required"`
	Directory    string              `json:"directory,required"`
	ProjectID    string              `json:"projectID,required"`
	Time         SessionTime         `json:"time,required"`
	Title        string              `json:"title,required"`
	Version      string              `json:"version,required"`
	CustomPrompt SessionCustomPrompt `json:"customPrompt"`
	ParentID     string              `json:"parentID"`
	Revert       SessionRevert       `json:"revert"`
	Share        SessionShare        `json:"share"`
	Summary      SessionSummary      `json:"summary"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID           respjson.Field
		Directory    respjson.Field
		ProjectID    respjson.Field
		Time         respjson.Field
		Title        respjson.Field
		Version      respjson.Field
		CustomPrompt respjson.Field
		ParentID     respjson.Field
		Revert       respjson.Field
		Share        respjson.Field
		Summary      respjson.Field
		ExtraFields  map[string]respjson.Field
		raw          string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r Session) RawJSON() string { return r.JSON.raw }
func (r *Session) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type SessionTime struct {
	Created    float64 `json:"created,required"`
	Updated    float64 `json:"updated,required"`
	Compacting float64 `json:"compacting"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Created     respjson.Field
		Updated     respjson.Field
		Compacting  respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r SessionTime) RawJSON() string { return r.JSON.raw }
func (r *SessionTime) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type SessionCustomPrompt struct {
	// Any of "file", "inline".
	Type      string            `json:"type,required"`
	Value     string            `json:"value,required"`
	LoadedAt  float64           `json:"loadedAt"`
	Variables map[string]string `json:"variables"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Type        respjson.Field
		Value       respjson.Field
		LoadedAt    respjson.Field
		Variables   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r SessionCustomPrompt) RawJSON() string { return r.JSON.raw }
func (r *SessionCustomPrompt) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type SessionRevert struct {
	MessageID string `json:"messageID,required"`
	Diff      string `json:"diff"`
	PartID    string `json:"partID"`
	Snapshot  string `json:"snapshot"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		MessageID   respjson.Field
		Diff        respjson.Field
		PartID      respjson.Field
		Snapshot    respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r SessionRevert) RawJSON() string { return r.JSON.raw }
func (r *SessionRevert) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type SessionShare struct {
	URL string `json:"url,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		URL         respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r SessionShare) RawJSON() string { return r.JSON.raw }
func (r *SessionShare) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type SessionSummary struct {
	Additions float64    `json:"additions,required"`
	Deletions float64    `json:"deletions,required"`
	Files     float64    `json:"files,required"`
	Diffs     []FileDiff `json:"diffs"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Additions   respjson.Field
		Deletions   respjson.Field
		Files       respjson.Field
		Diffs       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r SessionSummary) RawJSON() string { return r.JSON.raw }
func (r *SessionSummary) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type Todo struct {
	// Unique identifier for the todo item
	ID string `json:"id,required"`
	// Brief description of the task
	Content string `json:"content,required"`
	// Priority level of the task: high, medium, low
	Priority string `json:"priority,required"`
	// Current status of the task: pending, in_progress, completed, cancelled
	Status string `json:"status,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Content     respjson.Field
		Priority    respjson.Field
		Status      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r Todo) RawJSON() string { return r.JSON.raw }
func (r *Todo) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type UnknownError struct {
	Data UnknownErrorData      `json:"data,required"`
	Name constant.UnknownError `json:"name,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Data        respjson.Field
		Name        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r UnknownError) RawJSON() string { return r.JSON.raw }
func (r *UnknownError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type UnknownErrorData struct {
	Message string `json:"message,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Message     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r UnknownErrorData) RawJSON() string { return r.JSON.raw }
func (r *UnknownErrorData) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type SessionGetStatusResponse map[string]SessionGetStatusResponseItemUnion

// SessionGetStatusResponseItemUnion contains all possible properties and values
// from [SessionGetStatusResponseItemType], [SessionGetStatusResponseItemObject],
// [SessionGetStatusResponseItemType].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type SessionGetStatusResponseItemUnion struct {
	Type string `json:"type"`
	// This field is from variant [SessionGetStatusResponseItemObject].
	Attempt float64 `json:"attempt"`
	// This field is from variant [SessionGetStatusResponseItemObject].
	Message string `json:"message"`
	// This field is from variant [SessionGetStatusResponseItemObject].
	Next float64 `json:"next"`
	JSON struct {
		Type    respjson.Field
		Attempt respjson.Field
		Message respjson.Field
		Next    respjson.Field
		raw     string
	} `json:"-"`
}

func (u SessionGetStatusResponseItemUnion) AsSessionGetStatusResponseItemType() (v SessionGetStatusResponseItemType) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u SessionGetStatusResponseItemUnion) AsSessionGetStatusResponseItemObject() (v SessionGetStatusResponseItemObject) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u SessionGetStatusResponseItemUnion) AsVariant2() (v SessionGetStatusResponseItemType) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u SessionGetStatusResponseItemUnion) RawJSON() string { return u.JSON.raw }

func (r *SessionGetStatusResponseItemUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type SessionGetStatusResponseItemType struct {
	Type constant.Idle `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r SessionGetStatusResponseItemType) RawJSON() string { return r.JSON.raw }
func (r *SessionGetStatusResponseItemType) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type SessionGetStatusResponseItemObject struct {
	Attempt float64        `json:"attempt,required"`
	Message string         `json:"message,required"`
	Next    float64        `json:"next,required"`
	Type    constant.Retry `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Attempt     respjson.Field
		Message     respjson.Field
		Next        respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r SessionGetStatusResponseItemObject) RawJSON() string { return r.JSON.raw }
func (r *SessionGetStatusResponseItemObject) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type SessionSendCommandResponse struct {
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
func (r SessionSendCommandResponse) RawJSON() string { return r.JSON.raw }
func (r *SessionSendCommandResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type SessionNewParams struct {
	Directory    param.Opt[string]                 `query:"directory,omitzero" json:"-"`
	ParentID     param.Opt[string]                 `json:"parentID,omitzero"`
	Title        param.Opt[string]                 `json:"title,omitzero"`
	CustomPrompt SessionNewParamsCustomPromptUnion `json:"customPrompt,omitzero"`
	paramObj
}

func (r SessionNewParams) MarshalJSON() (data []byte, err error) {
	type shadow SessionNewParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionNewParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [SessionNewParams]'s query parameters as `url.Values`.
func (r SessionNewParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type SessionNewParamsCustomPromptUnion struct {
	OfString                        param.Opt[string]                   `json:",omitzero,inline"`
	OfSessionNewsCustomPromptObject *SessionNewParamsCustomPromptObject `json:",omitzero,inline"`
	paramUnion
}

func (u SessionNewParamsCustomPromptUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfString, u.OfSessionNewsCustomPromptObject)
}
func (u *SessionNewParamsCustomPromptUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *SessionNewParamsCustomPromptUnion) asAny() any {
	if !param.IsOmitted(u.OfString) {
		return &u.OfString.Value
	} else if !param.IsOmitted(u.OfSessionNewsCustomPromptObject) {
		return u.OfSessionNewsCustomPromptObject
	}
	return nil
}

// The properties Type, Value are required.
type SessionNewParamsCustomPromptObject struct {
	// Any of "file", "inline".
	Type      string            `json:"type,omitzero,required"`
	Value     string            `json:"value,required"`
	Variables map[string]string `json:"variables,omitzero"`
	paramObj
}

func (r SessionNewParamsCustomPromptObject) MarshalJSON() (data []byte, err error) {
	type shadow SessionNewParamsCustomPromptObject
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionNewParamsCustomPromptObject) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func init() {
	apijson.RegisterFieldValidator[SessionNewParamsCustomPromptObject](
		"type", "file", "inline",
	)
}

type SessionGetParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [SessionGetParams]'s query parameters as `url.Values`.
func (r SessionGetParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type SessionUpdateParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	Title     param.Opt[string] `json:"title,omitzero"`
	paramObj
}

func (r SessionUpdateParams) MarshalJSON() (data []byte, err error) {
	type shadow SessionUpdateParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionUpdateParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [SessionUpdateParams]'s query parameters as `url.Values`.
func (r SessionUpdateParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type SessionListParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [SessionListParams]'s query parameters as `url.Values`.
func (r SessionListParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type SessionDeleteParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [SessionDeleteParams]'s query parameters as `url.Values`.
func (r SessionDeleteParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type SessionAbortParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [SessionAbortParams]'s query parameters as `url.Values`.
func (r SessionAbortParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type SessionAnalyzeParams struct {
	MessageID  string            `json:"messageID,required"`
	ModelID    string            `json:"modelID,required"`
	ProviderID string            `json:"providerID,required"`
	Directory  param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

func (r SessionAnalyzeParams) MarshalJSON() (data []byte, err error) {
	type shadow SessionAnalyzeParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionAnalyzeParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [SessionAnalyzeParams]'s query parameters as `url.Values`.
func (r SessionAnalyzeParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type SessionForkParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	MessageID param.Opt[string] `json:"messageID,omitzero"`
	paramObj
}

func (r SessionForkParams) MarshalJSON() (data []byte, err error) {
	type shadow SessionForkParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionForkParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [SessionForkParams]'s query parameters as `url.Values`.
func (r SessionForkParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type SessionGetChildrenParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [SessionGetChildrenParams]'s query parameters as
// `url.Values`.
func (r SessionGetChildrenParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type SessionGetDiffParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	MessageID param.Opt[string] `query:"messageID,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [SessionGetDiffParams]'s query parameters as `url.Values`.
func (r SessionGetDiffParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type SessionGetStatusParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [SessionGetStatusParams]'s query parameters as `url.Values`.
func (r SessionGetStatusParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type SessionGetTodoParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [SessionGetTodoParams]'s query parameters as `url.Values`.
func (r SessionGetTodoParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type SessionRespondToPermissionParams struct {
	ID string `path:"id,required" json:"-"`
	// Any of "once", "always", "reject".
	Response  SessionRespondToPermissionParamsResponse `json:"response,omitzero,required"`
	Directory param.Opt[string]                        `query:"directory,omitzero" json:"-"`
	paramObj
}

func (r SessionRespondToPermissionParams) MarshalJSON() (data []byte, err error) {
	type shadow SessionRespondToPermissionParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionRespondToPermissionParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [SessionRespondToPermissionParams]'s query parameters as
// `url.Values`.
func (r SessionRespondToPermissionParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type SessionRespondToPermissionParamsResponse string

const (
	SessionRespondToPermissionParamsResponseOnce   SessionRespondToPermissionParamsResponse = "once"
	SessionRespondToPermissionParamsResponseAlways SessionRespondToPermissionParamsResponse = "always"
	SessionRespondToPermissionParamsResponseReject SessionRespondToPermissionParamsResponse = "reject"
)

type SessionRestoreRevertedParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [SessionRestoreRevertedParams]'s query parameters as
// `url.Values`.
func (r SessionRestoreRevertedParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type SessionRevertParams struct {
	MessageID string            `json:"messageID,required"`
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	PartID    param.Opt[string] `json:"partID,omitzero"`
	paramObj
}

func (r SessionRevertParams) MarshalJSON() (data []byte, err error) {
	type shadow SessionRevertParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionRevertParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [SessionRevertParams]'s query parameters as `url.Values`.
func (r SessionRevertParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type SessionRunShellParams struct {
	Agent     string                     `json:"agent,required"`
	Command   string                     `json:"command,required"`
	Directory param.Opt[string]          `query:"directory,omitzero" json:"-"`
	Model     SessionRunShellParamsModel `json:"model,omitzero"`
	paramObj
}

func (r SessionRunShellParams) MarshalJSON() (data []byte, err error) {
	type shadow SessionRunShellParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionRunShellParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [SessionRunShellParams]'s query parameters as `url.Values`.
func (r SessionRunShellParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

// The properties ModelID, ProviderID are required.
type SessionRunShellParamsModel struct {
	ModelID    string `json:"modelID,required"`
	ProviderID string `json:"providerID,required"`
	paramObj
}

func (r SessionRunShellParamsModel) MarshalJSON() (data []byte, err error) {
	type shadow SessionRunShellParamsModel
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionRunShellParamsModel) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type SessionSendCommandParams struct {
	Arguments string            `json:"arguments,required"`
	Command   string            `json:"command,required"`
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	Agent     param.Opt[string] `json:"agent,omitzero"`
	MessageID param.Opt[string] `json:"messageID,omitzero"`
	Model     param.Opt[string] `json:"model,omitzero"`
	paramObj
}

func (r SessionSendCommandParams) MarshalJSON() (data []byte, err error) {
	type shadow SessionSendCommandParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionSendCommandParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [SessionSendCommandParams]'s query parameters as
// `url.Values`.
func (r SessionSendCommandParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type SessionSummarizeParams struct {
	ModelID    string            `json:"modelID,required"`
	ProviderID string            `json:"providerID,required"`
	Directory  param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

func (r SessionSummarizeParams) MarshalJSON() (data []byte, err error) {
	type shadow SessionSummarizeParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SessionSummarizeParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [SessionSummarizeParams]'s query parameters as `url.Values`.
func (r SessionSummarizeParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}
