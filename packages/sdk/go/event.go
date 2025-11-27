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
	"github.com/sst/opencode-sdk-go/packages/ssestream"
	"github.com/sst/opencode-sdk-go/shared/constant"
)

// EventService contains methods and other services that help with interacting with
// the opencode API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewEventService] method instead.
type EventService struct {
	Options []option.RequestOption
}

// NewEventService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewEventService(opts ...option.RequestOption) (r EventService) {
	r = EventService{}
	r.Options = opts
	return
}

// Get events
func (r *EventService) ListStreaming(ctx context.Context, query EventListParams, opts ...option.RequestOption) (stream *ssestream.Stream[EventUnion]) {
	var (
		raw *http.Response
		err error
	)
	opts = slices.Concat(r.Options, opts)
	opts = append([]option.RequestOption{option.WithHeader("Accept", "text/event-stream")}, opts...)
	path := "event"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &raw, opts...)
	return ssestream.NewStream[EventUnion](ssestream.NewDecoder(raw), err)
}

// EventUnion contains all possible properties and values from
// [EventEventInstallationUpdated], [EventEventInstallationUpdateAvailable],
// [EventEventLspClientDiagnostics], [EventEventLspUpdated],
// [EventEventMessageUpdated], [EventEventMessageRemoved],
// [EventEventMessagePartUpdated], [EventEventMessagePartRemoved],
// [EventEventPermissionUpdated], [EventEventPermissionReplied],
// [EventEventSessionStatus], [EventEventSessionIdle],
// [EventEventSessionCompacted], [EventEventFileEdited], [EventEventTodoUpdated],
// [EventEventWorkflowStarted], [EventEventWorkflowStepStarted],
// [EventEventWorkflowStepCompleted], [EventEventWorkflowStepFailed],
// [EventEventWorkflowPaused], [EventEventWorkflowResumed],
// [EventEventWorkflowCompleted], [EventEventWorkflowFailed],
// [EventEventWorkflowCancelled], [EventEventCommandExecuted],
// [EventEventSessionCreated], [EventEventSessionUpdated],
// [EventEventSessionDeleted], [EventEventSessionDiff], [EventEventSessionError],
// [EventEventClientToolRequest], [EventEventClientToolRegistered],
// [EventEventClientToolUnregistered], [EventEventClientToolExecuting],
// [EventEventClientToolCompleted], [EventEventClientToolFailed],
// [EventPromptAppend], [EventCommandExecute], [EventToastShow],
// [EventEventServerConnected], [EventEventFileWatcherUpdated].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type EventUnion struct {
	// This field is a union of [EventEventInstallationUpdatedProperties],
	// [EventEventInstallationUpdateAvailableProperties],
	// [EventEventLspClientDiagnosticsProperties], [any],
	// [EventEventMessageUpdatedProperties], [EventEventMessageRemovedProperties],
	// [EventEventMessagePartUpdatedProperties],
	// [EventEventMessagePartRemovedProperties],
	// [EventEventPermissionUpdatedProperties],
	// [EventEventPermissionRepliedProperties], [EventEventSessionStatusProperties],
	// [EventEventSessionIdleProperties], [EventEventSessionCompactedProperties],
	// [EventEventFileEditedProperties], [EventEventTodoUpdatedProperties],
	// [EventEventWorkflowStartedProperties],
	// [EventEventWorkflowStepStartedProperties],
	// [EventEventWorkflowStepCompletedProperties],
	// [EventEventWorkflowStepFailedProperties], [EventEventWorkflowPausedProperties],
	// [EventEventWorkflowResumedProperties], [EventEventWorkflowCompletedProperties],
	// [EventEventWorkflowFailedProperties], [EventEventWorkflowCancelledProperties],
	// [EventEventCommandExecutedProperties], [EventEventSessionCreatedProperties],
	// [EventEventSessionUpdatedProperties], [EventEventSessionDeletedProperties],
	// [EventEventSessionDiffProperties], [EventEventSessionErrorProperties],
	// [EventEventClientToolRequestProperties],
	// [EventEventClientToolRegisteredProperties],
	// [EventEventClientToolUnregisteredProperties],
	// [EventEventClientToolExecutingProperties],
	// [EventEventClientToolCompletedProperties],
	// [EventEventClientToolFailedProperties], [EventPromptAppendProperties],
	// [EventCommandExecuteProperties], [EventToastShowProperties], [any],
	// [EventEventFileWatcherUpdatedProperties]
	Properties EventUnionProperties `json:"properties"`
	Type       string               `json:"type"`
	JSON       struct {
		Properties respjson.Field
		Type       respjson.Field
		raw        string
	} `json:"-"`
}

func (u EventUnion) AsEventEventInstallationUpdated() (v EventEventInstallationUpdated) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventInstallationUpdateAvailable() (v EventEventInstallationUpdateAvailable) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventLspClientDiagnostics() (v EventEventLspClientDiagnostics) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventLspUpdated() (v EventEventLspUpdated) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventMessageUpdated() (v EventEventMessageUpdated) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventMessageRemoved() (v EventEventMessageRemoved) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventMessagePartUpdated() (v EventEventMessagePartUpdated) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventMessagePartRemoved() (v EventEventMessagePartRemoved) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventPermissionUpdated() (v EventEventPermissionUpdated) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventPermissionReplied() (v EventEventPermissionReplied) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventSessionStatus() (v EventEventSessionStatus) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventSessionIdle() (v EventEventSessionIdle) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventSessionCompacted() (v EventEventSessionCompacted) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventFileEdited() (v EventEventFileEdited) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventTodoUpdated() (v EventEventTodoUpdated) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventWorkflowStarted() (v EventEventWorkflowStarted) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventWorkflowStepStarted() (v EventEventWorkflowStepStarted) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventWorkflowStepCompleted() (v EventEventWorkflowStepCompleted) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventWorkflowStepFailed() (v EventEventWorkflowStepFailed) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventWorkflowPaused() (v EventEventWorkflowPaused) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventWorkflowResumed() (v EventEventWorkflowResumed) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventWorkflowCompleted() (v EventEventWorkflowCompleted) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventWorkflowFailed() (v EventEventWorkflowFailed) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventWorkflowCancelled() (v EventEventWorkflowCancelled) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventCommandExecuted() (v EventEventCommandExecuted) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventSessionCreated() (v EventEventSessionCreated) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventSessionUpdated() (v EventEventSessionUpdated) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventSessionDeleted() (v EventEventSessionDeleted) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventSessionDiff() (v EventEventSessionDiff) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventSessionError() (v EventEventSessionError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventClientToolRequest() (v EventEventClientToolRequest) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventClientToolRegistered() (v EventEventClientToolRegistered) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventClientToolUnregistered() (v EventEventClientToolUnregistered) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventClientToolExecuting() (v EventEventClientToolExecuting) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventClientToolCompleted() (v EventEventClientToolCompleted) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventClientToolFailed() (v EventEventClientToolFailed) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventPromptAppend() (v EventPromptAppend) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventCommandExecute() (v EventCommandExecute) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventToastShow() (v EventToastShow) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventServerConnected() (v EventEventServerConnected) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventUnion) AsEventEventFileWatcherUpdated() (v EventEventFileWatcherUpdated) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u EventUnion) RawJSON() string { return u.JSON.raw }

func (r *EventUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// EventUnionProperties is an implicit subunion of [EventUnion].
// EventUnionProperties provides convenient access to the sub-properties of the
// union.
//
// For type safety it is recommended to directly use a variant of the [EventUnion].
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfEventEventServerConnectedProperties]
type EventUnionProperties struct {
	// This field will be present if the value is a [any] instead of an object.
	OfEventEventServerConnectedProperties any    `json:",inline"`
	Version                               string `json:"version"`
	// This field is from variant [EventEventLspClientDiagnosticsProperties].
	Path string `json:"path"`
	// This field is from variant [EventEventLspClientDiagnosticsProperties].
	ServerID string `json:"serverID"`
	// This field is a union of [MessageUnion], [Session]
	Info      EventUnionPropertiesInfo `json:"info"`
	MessageID string                   `json:"messageID"`
	SessionID string                   `json:"sessionID"`
	// This field is from variant [EventEventMessagePartUpdatedProperties].
	Part PartUnion `json:"part"`
	// This field is from variant [EventEventMessagePartUpdatedProperties].
	Delta string `json:"delta"`
	// This field is from variant [EventEventMessagePartRemovedProperties].
	PartID string `json:"partID"`
	// This field is from variant [EventEventPermissionUpdatedProperties].
	ID string `json:"id"`
	// This field is from variant [EventEventPermissionUpdatedProperties].
	Metadata map[string]any `json:"metadata"`
	// This field is from variant [EventEventPermissionUpdatedProperties].
	Time  EventEventPermissionUpdatedPropertiesTime `json:"time"`
	Title string                                    `json:"title"`
	// This field is from variant [EventEventPermissionUpdatedProperties].
	Type   string `json:"type"`
	CallID string `json:"callID"`
	// This field is from variant [EventEventPermissionUpdatedProperties].
	Pattern EventEventPermissionUpdatedPropertiesPatternUnion `json:"pattern"`
	// This field is from variant [EventEventPermissionRepliedProperties].
	PermissionID string `json:"permissionID"`
	// This field is from variant [EventEventPermissionRepliedProperties].
	Response string `json:"response"`
	// This field is from variant [EventEventSessionStatusProperties].
	Status EventEventSessionStatusPropertiesStatusUnion `json:"status"`
	File   string                                       `json:"file"`
	// This field is from variant [EventEventTodoUpdatedProperties].
	Todos []Todo `json:"todos"`
	// This field is from variant [EventEventWorkflowStartedProperties].
	Inputs     map[string]any `json:"inputs"`
	InstanceID string         `json:"instanceId"`
	WorkflowID string         `json:"workflowId"`
	StepID     string         `json:"stepId"`
	// This field is from variant [EventEventWorkflowStepStartedProperties].
	StepType string  `json:"stepType"`
	Duration float64 `json:"duration"`
	// This field is from variant [EventEventWorkflowStepCompletedProperties].
	Output any `json:"output"`
	// This field is a union of [string], [string],
	// [EventEventSessionErrorPropertiesErrorUnion], [string]
	Error EventUnionPropertiesError `json:"error"`
	// This field is from variant [EventEventWorkflowStepFailedProperties].
	RetryCount float64 `json:"retryCount"`
	Message    string  `json:"message"`
	// This field is from variant [EventEventWorkflowPausedProperties].
	Options any `json:"options"`
	// This field is from variant [EventEventWorkflowResumedProperties].
	Approved bool `json:"approved"`
	// This field is from variant [EventEventWorkflowResumedProperties].
	Feedback string `json:"feedback"`
	// This field is from variant [EventEventWorkflowCompletedProperties].
	Outputs map[string]any `json:"outputs"`
	// This field is from variant [EventEventWorkflowCancelledProperties].
	Reason string `json:"reason"`
	// This field is from variant [EventEventCommandExecutedProperties].
	Arguments string `json:"arguments"`
	// This field is from variant [EventEventCommandExecutedProperties].
	Name string `json:"name"`
	// This field is from variant [EventEventSessionDiffProperties].
	Diff     []FileDiff `json:"diff"`
	ClientID string     `json:"clientID"`
	// This field is from variant [EventEventClientToolRequestProperties].
	Request ClientToolExecution `json:"request"`
	ToolIDs []string            `json:"toolIDs"`
	Tool    string              `json:"tool"`
	// This field is from variant [EventEventClientToolCompletedProperties].
	Success bool `json:"success"`
	// This field is from variant [EventPromptAppendProperties].
	Text string `json:"text"`
	// This field is from variant [EventCommandExecuteProperties].
	Command string `json:"command"`
	// This field is from variant [EventToastShowProperties].
	Variant string `json:"variant"`
	// This field is from variant [EventEventFileWatcherUpdatedProperties].
	Event EventEventFileWatcherUpdatedPropertiesEvent `json:"event"`
	JSON  struct {
		OfEventEventServerConnectedProperties respjson.Field
		Version                               respjson.Field
		Path                                  respjson.Field
		ServerID                              respjson.Field
		Info                                  respjson.Field
		MessageID                             respjson.Field
		SessionID                             respjson.Field
		Part                                  respjson.Field
		Delta                                 respjson.Field
		PartID                                respjson.Field
		ID                                    respjson.Field
		Metadata                              respjson.Field
		Time                                  respjson.Field
		Title                                 respjson.Field
		Type                                  respjson.Field
		CallID                                respjson.Field
		Pattern                               respjson.Field
		PermissionID                          respjson.Field
		Response                              respjson.Field
		Status                                respjson.Field
		File                                  respjson.Field
		Todos                                 respjson.Field
		Inputs                                respjson.Field
		InstanceID                            respjson.Field
		WorkflowID                            respjson.Field
		StepID                                respjson.Field
		StepType                              respjson.Field
		Duration                              respjson.Field
		Output                                respjson.Field
		Error                                 respjson.Field
		RetryCount                            respjson.Field
		Message                               respjson.Field
		Options                               respjson.Field
		Approved                              respjson.Field
		Feedback                              respjson.Field
		Outputs                               respjson.Field
		Reason                                respjson.Field
		Arguments                             respjson.Field
		Name                                  respjson.Field
		Diff                                  respjson.Field
		ClientID                              respjson.Field
		Request                               respjson.Field
		ToolIDs                               respjson.Field
		Tool                                  respjson.Field
		Success                               respjson.Field
		Text                                  respjson.Field
		Command                               respjson.Field
		Variant                               respjson.Field
		Event                                 respjson.Field
		raw                                   string
	} `json:"-"`
}

func (r *EventUnionProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// EventUnionPropertiesInfo is an implicit subunion of [EventUnion].
// EventUnionPropertiesInfo provides convenient access to the sub-properties of the
// union.
//
// For type safety it is recommended to directly use a variant of the [EventUnion].
type EventUnionPropertiesInfo struct {
	ID string `json:"id"`
	// This field is from variant [MessageUnion].
	Agent string `json:"agent"`
	// This field is from variant [MessageUnion].
	Model     MessageUserMessageModel `json:"model"`
	Role      string                  `json:"role"`
	SessionID string                  `json:"sessionID"`
	// This field is a union of [MessageUserMessageTime], [AssistantMessageTime],
	// [SessionTime]
	Time EventUnionPropertiesInfoTime `json:"time"`
	// This field is a union of [MessageUserMessageSummary], [bool], [SessionSummary]
	Summary EventUnionPropertiesInfoSummary `json:"summary"`
	// This field is from variant [MessageUnion].
	System string `json:"system"`
	// This field is from variant [MessageUnion].
	Tools map[string]bool `json:"tools"`
	// This field is from variant [MessageUnion].
	Cost float64 `json:"cost"`
	// This field is from variant [MessageUnion].
	Mode string `json:"mode"`
	// This field is from variant [MessageUnion].
	ModelID  string `json:"modelID"`
	ParentID string `json:"parentID"`
	// This field is from variant [MessageUnion].
	Path AssistantMessagePath `json:"path"`
	// This field is from variant [MessageUnion].
	ProviderID string `json:"providerID"`
	// This field is from variant [MessageUnion].
	Tokens AssistantMessageTokens `json:"tokens"`
	// This field is from variant [MessageUnion].
	Error AssistantMessageErrorUnion `json:"error"`
	// This field is from variant [MessageUnion].
	Finish string `json:"finish"`
	// This field is from variant [Session].
	Directory string `json:"directory"`
	// This field is from variant [Session].
	ProjectID string `json:"projectID"`
	// This field is from variant [Session].
	Title string `json:"title"`
	// This field is from variant [Session].
	Version string `json:"version"`
	// This field is from variant [Session].
	CustomPrompt SessionCustomPrompt `json:"customPrompt"`
	// This field is from variant [Session].
	Revert SessionRevert `json:"revert"`
	// This field is from variant [Session].
	Share SessionShare `json:"share"`
	JSON  struct {
		ID           respjson.Field
		Agent        respjson.Field
		Model        respjson.Field
		Role         respjson.Field
		SessionID    respjson.Field
		Time         respjson.Field
		Summary      respjson.Field
		System       respjson.Field
		Tools        respjson.Field
		Cost         respjson.Field
		Mode         respjson.Field
		ModelID      respjson.Field
		ParentID     respjson.Field
		Path         respjson.Field
		ProviderID   respjson.Field
		Tokens       respjson.Field
		Error        respjson.Field
		Finish       respjson.Field
		Directory    respjson.Field
		ProjectID    respjson.Field
		Title        respjson.Field
		Version      respjson.Field
		CustomPrompt respjson.Field
		Revert       respjson.Field
		Share        respjson.Field
		raw          string
	} `json:"-"`
}

func (r *EventUnionPropertiesInfo) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// EventUnionPropertiesInfoTime is an implicit subunion of [EventUnion].
// EventUnionPropertiesInfoTime provides convenient access to the sub-properties of
// the union.
//
// For type safety it is recommended to directly use a variant of the [EventUnion].
type EventUnionPropertiesInfoTime struct {
	Created float64 `json:"created"`
	// This field is from variant [AssistantMessageTime].
	Completed float64 `json:"completed"`
	// This field is from variant [SessionTime].
	Updated float64 `json:"updated"`
	// This field is from variant [SessionTime].
	Compacting float64 `json:"compacting"`
	JSON       struct {
		Created    respjson.Field
		Completed  respjson.Field
		Updated    respjson.Field
		Compacting respjson.Field
		raw        string
	} `json:"-"`
}

func (r *EventUnionPropertiesInfoTime) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// EventUnionPropertiesInfoSummary is an implicit subunion of [EventUnion].
// EventUnionPropertiesInfoSummary provides convenient access to the sub-properties
// of the union.
//
// For type safety it is recommended to directly use a variant of the [EventUnion].
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfBool]
type EventUnionPropertiesInfoSummary struct {
	// This field will be present if the value is a [bool] instead of an object.
	OfBool bool       `json:",inline"`
	Diffs  []FileDiff `json:"diffs"`
	// This field is from variant [MessageUserMessageSummary].
	Body string `json:"body"`
	// This field is from variant [MessageUserMessageSummary].
	Title string `json:"title"`
	// This field is from variant [SessionSummary].
	Additions float64 `json:"additions"`
	// This field is from variant [SessionSummary].
	Deletions float64 `json:"deletions"`
	// This field is from variant [SessionSummary].
	Files float64 `json:"files"`
	JSON  struct {
		OfBool    respjson.Field
		Diffs     respjson.Field
		Body      respjson.Field
		Title     respjson.Field
		Additions respjson.Field
		Deletions respjson.Field
		Files     respjson.Field
		raw       string
	} `json:"-"`
}

func (r *EventUnionPropertiesInfoSummary) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// EventUnionPropertiesError is an implicit subunion of [EventUnion].
// EventUnionPropertiesError provides convenient access to the sub-properties of
// the union.
//
// For type safety it is recommended to directly use a variant of the [EventUnion].
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfString]
type EventUnionPropertiesError struct {
	// This field will be present if the value is a [string] instead of an object.
	OfString string `json:",inline"`
	// This field is a union of [ProviderAuthErrorData], [UnknownErrorData], [any],
	// [MessageAbortedErrorData], [APIErrorData]
	Data EventUnionPropertiesErrorData `json:"data"`
	Name string                        `json:"name"`
	JSON struct {
		OfString respjson.Field
		Data     respjson.Field
		Name     respjson.Field
		raw      string
	} `json:"-"`
}

func (r *EventUnionPropertiesError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// EventUnionPropertiesErrorData is an implicit subunion of [EventUnion].
// EventUnionPropertiesErrorData provides convenient access to the sub-properties
// of the union.
//
// For type safety it is recommended to directly use a variant of the [EventUnion].
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfMessageOutputLengthErrorData]
type EventUnionPropertiesErrorData struct {
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

func (r *EventUnionPropertiesErrorData) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventInstallationUpdated struct {
	Properties EventEventInstallationUpdatedProperties `json:"properties,required"`
	Type       constant.InstallationUpdated            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventInstallationUpdated) RawJSON() string { return r.JSON.raw }
func (r *EventEventInstallationUpdated) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventInstallationUpdatedProperties struct {
	Version string `json:"version,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Version     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventInstallationUpdatedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventInstallationUpdatedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventInstallationUpdateAvailable struct {
	Properties EventEventInstallationUpdateAvailableProperties `json:"properties,required"`
	Type       constant.InstallationUpdateAvailable            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventInstallationUpdateAvailable) RawJSON() string { return r.JSON.raw }
func (r *EventEventInstallationUpdateAvailable) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventInstallationUpdateAvailableProperties struct {
	Version string `json:"version,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Version     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventInstallationUpdateAvailableProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventInstallationUpdateAvailableProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventLspClientDiagnostics struct {
	Properties EventEventLspClientDiagnosticsProperties `json:"properties,required"`
	Type       constant.LspClientDiagnostics            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventLspClientDiagnostics) RawJSON() string { return r.JSON.raw }
func (r *EventEventLspClientDiagnostics) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventLspClientDiagnosticsProperties struct {
	Path     string `json:"path,required"`
	ServerID string `json:"serverID,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Path        respjson.Field
		ServerID    respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventLspClientDiagnosticsProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventLspClientDiagnosticsProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventLspUpdated struct {
	Properties any                 `json:"properties,required"`
	Type       constant.LspUpdated `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventLspUpdated) RawJSON() string { return r.JSON.raw }
func (r *EventEventLspUpdated) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventMessageUpdated struct {
	Properties EventEventMessageUpdatedProperties `json:"properties,required"`
	Type       constant.MessageUpdated            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventMessageUpdated) RawJSON() string { return r.JSON.raw }
func (r *EventEventMessageUpdated) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventMessageUpdatedProperties struct {
	Info MessageUnion `json:"info,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Info        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventMessageUpdatedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventMessageUpdatedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventMessageRemoved struct {
	Properties EventEventMessageRemovedProperties `json:"properties,required"`
	Type       constant.MessageRemoved            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventMessageRemoved) RawJSON() string { return r.JSON.raw }
func (r *EventEventMessageRemoved) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventMessageRemovedProperties struct {
	MessageID string `json:"messageID,required"`
	SessionID string `json:"sessionID,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		MessageID   respjson.Field
		SessionID   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventMessageRemovedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventMessageRemovedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventMessagePartUpdated struct {
	Properties EventEventMessagePartUpdatedProperties `json:"properties,required"`
	Type       constant.MessagePartUpdated            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventMessagePartUpdated) RawJSON() string { return r.JSON.raw }
func (r *EventEventMessagePartUpdated) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventMessagePartUpdatedProperties struct {
	Part  PartUnion `json:"part,required"`
	Delta string    `json:"delta"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Part        respjson.Field
		Delta       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventMessagePartUpdatedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventMessagePartUpdatedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventMessagePartRemoved struct {
	Properties EventEventMessagePartRemovedProperties `json:"properties,required"`
	Type       constant.MessagePartRemoved            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventMessagePartRemoved) RawJSON() string { return r.JSON.raw }
func (r *EventEventMessagePartRemoved) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventMessagePartRemovedProperties struct {
	MessageID string `json:"messageID,required"`
	PartID    string `json:"partID,required"`
	SessionID string `json:"sessionID,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		MessageID   respjson.Field
		PartID      respjson.Field
		SessionID   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventMessagePartRemovedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventMessagePartRemovedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventPermissionUpdated struct {
	Properties EventEventPermissionUpdatedProperties `json:"properties,required"`
	Type       constant.PermissionUpdated            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventPermissionUpdated) RawJSON() string { return r.JSON.raw }
func (r *EventEventPermissionUpdated) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventPermissionUpdatedProperties struct {
	ID        string                                            `json:"id,required"`
	MessageID string                                            `json:"messageID,required"`
	Metadata  map[string]any                                    `json:"metadata,required"`
	SessionID string                                            `json:"sessionID,required"`
	Time      EventEventPermissionUpdatedPropertiesTime         `json:"time,required"`
	Title     string                                            `json:"title,required"`
	Type      string                                            `json:"type,required"`
	CallID    string                                            `json:"callID"`
	Pattern   EventEventPermissionUpdatedPropertiesPatternUnion `json:"pattern"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		MessageID   respjson.Field
		Metadata    respjson.Field
		SessionID   respjson.Field
		Time        respjson.Field
		Title       respjson.Field
		Type        respjson.Field
		CallID      respjson.Field
		Pattern     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventPermissionUpdatedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventPermissionUpdatedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventPermissionUpdatedPropertiesTime struct {
	Created float64 `json:"created,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Created     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventPermissionUpdatedPropertiesTime) RawJSON() string { return r.JSON.raw }
func (r *EventEventPermissionUpdatedPropertiesTime) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// EventEventPermissionUpdatedPropertiesPatternUnion contains all possible
// properties and values from [string], [[]string].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfString OfStringArray]
type EventEventPermissionUpdatedPropertiesPatternUnion struct {
	// This field will be present if the value is a [string] instead of an object.
	OfString string `json:",inline"`
	// This field will be present if the value is a [[]string] instead of an object.
	OfStringArray []string `json:",inline"`
	JSON          struct {
		OfString      respjson.Field
		OfStringArray respjson.Field
		raw           string
	} `json:"-"`
}

func (u EventEventPermissionUpdatedPropertiesPatternUnion) AsString() (v string) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventEventPermissionUpdatedPropertiesPatternUnion) AsStringArray() (v []string) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u EventEventPermissionUpdatedPropertiesPatternUnion) RawJSON() string { return u.JSON.raw }

func (r *EventEventPermissionUpdatedPropertiesPatternUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventPermissionReplied struct {
	Properties EventEventPermissionRepliedProperties `json:"properties,required"`
	Type       constant.PermissionReplied            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventPermissionReplied) RawJSON() string { return r.JSON.raw }
func (r *EventEventPermissionReplied) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventPermissionRepliedProperties struct {
	PermissionID string `json:"permissionID,required"`
	Response     string `json:"response,required"`
	SessionID    string `json:"sessionID,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		PermissionID respjson.Field
		Response     respjson.Field
		SessionID    respjson.Field
		ExtraFields  map[string]respjson.Field
		raw          string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventPermissionRepliedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventPermissionRepliedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventSessionStatus struct {
	Properties EventEventSessionStatusProperties `json:"properties,required"`
	Type       constant.SessionStatus            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventSessionStatus) RawJSON() string { return r.JSON.raw }
func (r *EventEventSessionStatus) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventSessionStatusProperties struct {
	SessionID string                                       `json:"sessionID,required"`
	Status    EventEventSessionStatusPropertiesStatusUnion `json:"status,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		SessionID   respjson.Field
		Status      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventSessionStatusProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventSessionStatusProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// EventEventSessionStatusPropertiesStatusUnion contains all possible properties
// and values from [EventEventSessionStatusPropertiesStatusType],
// [EventEventSessionStatusPropertiesStatusObject],
// [EventEventSessionStatusPropertiesStatusType].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type EventEventSessionStatusPropertiesStatusUnion struct {
	Type string `json:"type"`
	// This field is from variant [EventEventSessionStatusPropertiesStatusObject].
	Attempt float64 `json:"attempt"`
	// This field is from variant [EventEventSessionStatusPropertiesStatusObject].
	Message string `json:"message"`
	// This field is from variant [EventEventSessionStatusPropertiesStatusObject].
	Next float64 `json:"next"`
	JSON struct {
		Type    respjson.Field
		Attempt respjson.Field
		Message respjson.Field
		Next    respjson.Field
		raw     string
	} `json:"-"`
}

func (u EventEventSessionStatusPropertiesStatusUnion) AsEventEventSessionStatusPropertiesStatusType() (v EventEventSessionStatusPropertiesStatusType) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventEventSessionStatusPropertiesStatusUnion) AsEventEventSessionStatusPropertiesStatusObject() (v EventEventSessionStatusPropertiesStatusObject) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventEventSessionStatusPropertiesStatusUnion) AsVariant2() (v EventEventSessionStatusPropertiesStatusType) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u EventEventSessionStatusPropertiesStatusUnion) RawJSON() string { return u.JSON.raw }

func (r *EventEventSessionStatusPropertiesStatusUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventSessionStatusPropertiesStatusType struct {
	Type constant.Idle `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventSessionStatusPropertiesStatusType) RawJSON() string { return r.JSON.raw }
func (r *EventEventSessionStatusPropertiesStatusType) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventSessionStatusPropertiesStatusObject struct {
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
func (r EventEventSessionStatusPropertiesStatusObject) RawJSON() string { return r.JSON.raw }
func (r *EventEventSessionStatusPropertiesStatusObject) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventSessionIdle struct {
	Properties EventEventSessionIdleProperties `json:"properties,required"`
	Type       constant.SessionIdle            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventSessionIdle) RawJSON() string { return r.JSON.raw }
func (r *EventEventSessionIdle) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventSessionIdleProperties struct {
	SessionID string `json:"sessionID,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		SessionID   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventSessionIdleProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventSessionIdleProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventSessionCompacted struct {
	Properties EventEventSessionCompactedProperties `json:"properties,required"`
	Type       constant.SessionCompacted            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventSessionCompacted) RawJSON() string { return r.JSON.raw }
func (r *EventEventSessionCompacted) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventSessionCompactedProperties struct {
	SessionID string `json:"sessionID,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		SessionID   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventSessionCompactedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventSessionCompactedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventFileEdited struct {
	Properties EventEventFileEditedProperties `json:"properties,required"`
	Type       constant.FileEdited            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventFileEdited) RawJSON() string { return r.JSON.raw }
func (r *EventEventFileEdited) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventFileEditedProperties struct {
	File string `json:"file,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		File        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventFileEditedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventFileEditedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventTodoUpdated struct {
	Properties EventEventTodoUpdatedProperties `json:"properties,required"`
	Type       constant.TodoUpdated            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventTodoUpdated) RawJSON() string { return r.JSON.raw }
func (r *EventEventTodoUpdated) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventTodoUpdatedProperties struct {
	SessionID string `json:"sessionID,required"`
	Todos     []Todo `json:"todos,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		SessionID   respjson.Field
		Todos       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventTodoUpdatedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventTodoUpdatedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventWorkflowStarted struct {
	Properties EventEventWorkflowStartedProperties `json:"properties,required"`
	Type       constant.WorkflowStarted            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventWorkflowStarted) RawJSON() string { return r.JSON.raw }
func (r *EventEventWorkflowStarted) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventWorkflowStartedProperties struct {
	Inputs     map[string]any `json:"inputs,required"`
	InstanceID string         `json:"instanceId,required"`
	WorkflowID string         `json:"workflowId,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Inputs      respjson.Field
		InstanceID  respjson.Field
		WorkflowID  respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventWorkflowStartedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventWorkflowStartedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventWorkflowStepStarted struct {
	Properties EventEventWorkflowStepStartedProperties `json:"properties,required"`
	Type       constant.WorkflowStepStarted            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventWorkflowStepStarted) RawJSON() string { return r.JSON.raw }
func (r *EventEventWorkflowStepStarted) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventWorkflowStepStartedProperties struct {
	InstanceID string `json:"instanceId,required"`
	StepID     string `json:"stepId,required"`
	StepType   string `json:"stepType,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		InstanceID  respjson.Field
		StepID      respjson.Field
		StepType    respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventWorkflowStepStartedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventWorkflowStepStartedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventWorkflowStepCompleted struct {
	Properties EventEventWorkflowStepCompletedProperties `json:"properties,required"`
	Type       constant.WorkflowStepCompleted            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventWorkflowStepCompleted) RawJSON() string { return r.JSON.raw }
func (r *EventEventWorkflowStepCompleted) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventWorkflowStepCompletedProperties struct {
	Duration   float64 `json:"duration,required"`
	InstanceID string  `json:"instanceId,required"`
	StepID     string  `json:"stepId,required"`
	Output     any     `json:"output"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Duration    respjson.Field
		InstanceID  respjson.Field
		StepID      respjson.Field
		Output      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventWorkflowStepCompletedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventWorkflowStepCompletedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventWorkflowStepFailed struct {
	Properties EventEventWorkflowStepFailedProperties `json:"properties,required"`
	Type       constant.WorkflowStepFailed            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventWorkflowStepFailed) RawJSON() string { return r.JSON.raw }
func (r *EventEventWorkflowStepFailed) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventWorkflowStepFailedProperties struct {
	Error      string  `json:"error,required"`
	InstanceID string  `json:"instanceId,required"`
	RetryCount float64 `json:"retryCount,required"`
	StepID     string  `json:"stepId,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Error       respjson.Field
		InstanceID  respjson.Field
		RetryCount  respjson.Field
		StepID      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventWorkflowStepFailedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventWorkflowStepFailedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventWorkflowPaused struct {
	Properties EventEventWorkflowPausedProperties `json:"properties,required"`
	Type       constant.WorkflowPaused            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventWorkflowPaused) RawJSON() string { return r.JSON.raw }
func (r *EventEventWorkflowPaused) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventWorkflowPausedProperties struct {
	InstanceID string `json:"instanceId,required"`
	Message    string `json:"message,required"`
	StepID     string `json:"stepId,required"`
	Options    any    `json:"options"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		InstanceID  respjson.Field
		Message     respjson.Field
		StepID      respjson.Field
		Options     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventWorkflowPausedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventWorkflowPausedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventWorkflowResumed struct {
	Properties EventEventWorkflowResumedProperties `json:"properties,required"`
	Type       constant.WorkflowResumed            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventWorkflowResumed) RawJSON() string { return r.JSON.raw }
func (r *EventEventWorkflowResumed) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventWorkflowResumedProperties struct {
	Approved   bool   `json:"approved,required"`
	InstanceID string `json:"instanceId,required"`
	StepID     string `json:"stepId,required"`
	Feedback   string `json:"feedback"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Approved    respjson.Field
		InstanceID  respjson.Field
		StepID      respjson.Field
		Feedback    respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventWorkflowResumedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventWorkflowResumedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventWorkflowCompleted struct {
	Properties EventEventWorkflowCompletedProperties `json:"properties,required"`
	Type       constant.WorkflowCompleted            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventWorkflowCompleted) RawJSON() string { return r.JSON.raw }
func (r *EventEventWorkflowCompleted) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventWorkflowCompletedProperties struct {
	Duration   float64        `json:"duration,required"`
	InstanceID string         `json:"instanceId,required"`
	Outputs    map[string]any `json:"outputs,required"`
	WorkflowID string         `json:"workflowId,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Duration    respjson.Field
		InstanceID  respjson.Field
		Outputs     respjson.Field
		WorkflowID  respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventWorkflowCompletedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventWorkflowCompletedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventWorkflowFailed struct {
	Properties EventEventWorkflowFailedProperties `json:"properties,required"`
	Type       constant.WorkflowFailed            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventWorkflowFailed) RawJSON() string { return r.JSON.raw }
func (r *EventEventWorkflowFailed) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventWorkflowFailedProperties struct {
	Error      string `json:"error,required"`
	InstanceID string `json:"instanceId,required"`
	WorkflowID string `json:"workflowId,required"`
	StepID     string `json:"stepId"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Error       respjson.Field
		InstanceID  respjson.Field
		WorkflowID  respjson.Field
		StepID      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventWorkflowFailedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventWorkflowFailedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventWorkflowCancelled struct {
	Properties EventEventWorkflowCancelledProperties `json:"properties,required"`
	Type       constant.WorkflowCancelled            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventWorkflowCancelled) RawJSON() string { return r.JSON.raw }
func (r *EventEventWorkflowCancelled) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventWorkflowCancelledProperties struct {
	InstanceID string `json:"instanceId,required"`
	WorkflowID string `json:"workflowId,required"`
	Reason     string `json:"reason"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		InstanceID  respjson.Field
		WorkflowID  respjson.Field
		Reason      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventWorkflowCancelledProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventWorkflowCancelledProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventCommandExecuted struct {
	Properties EventEventCommandExecutedProperties `json:"properties,required"`
	Type       constant.CommandExecuted            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventCommandExecuted) RawJSON() string { return r.JSON.raw }
func (r *EventEventCommandExecuted) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventCommandExecutedProperties struct {
	Arguments string `json:"arguments,required"`
	MessageID string `json:"messageID,required"`
	Name      string `json:"name,required"`
	SessionID string `json:"sessionID,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Arguments   respjson.Field
		MessageID   respjson.Field
		Name        respjson.Field
		SessionID   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventCommandExecutedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventCommandExecutedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventSessionCreated struct {
	Properties EventEventSessionCreatedProperties `json:"properties,required"`
	Type       constant.SessionCreated            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventSessionCreated) RawJSON() string { return r.JSON.raw }
func (r *EventEventSessionCreated) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventSessionCreatedProperties struct {
	Info Session `json:"info,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Info        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventSessionCreatedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventSessionCreatedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventSessionUpdated struct {
	Properties EventEventSessionUpdatedProperties `json:"properties,required"`
	Type       constant.SessionUpdated            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventSessionUpdated) RawJSON() string { return r.JSON.raw }
func (r *EventEventSessionUpdated) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventSessionUpdatedProperties struct {
	Info Session `json:"info,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Info        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventSessionUpdatedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventSessionUpdatedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventSessionDeleted struct {
	Properties EventEventSessionDeletedProperties `json:"properties,required"`
	Type       constant.SessionDeleted            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventSessionDeleted) RawJSON() string { return r.JSON.raw }
func (r *EventEventSessionDeleted) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventSessionDeletedProperties struct {
	Info Session `json:"info,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Info        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventSessionDeletedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventSessionDeletedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventSessionDiff struct {
	Properties EventEventSessionDiffProperties `json:"properties,required"`
	Type       constant.SessionDiff            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventSessionDiff) RawJSON() string { return r.JSON.raw }
func (r *EventEventSessionDiff) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventSessionDiffProperties struct {
	Diff      []FileDiff `json:"diff,required"`
	SessionID string     `json:"sessionID,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Diff        respjson.Field
		SessionID   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventSessionDiffProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventSessionDiffProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventSessionError struct {
	Properties EventEventSessionErrorProperties `json:"properties,required"`
	Type       constant.SessionError            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventSessionError) RawJSON() string { return r.JSON.raw }
func (r *EventEventSessionError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventSessionErrorProperties struct {
	Error     EventEventSessionErrorPropertiesErrorUnion `json:"error"`
	SessionID string                                     `json:"sessionID"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Error       respjson.Field
		SessionID   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventSessionErrorProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventSessionErrorProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// EventEventSessionErrorPropertiesErrorUnion contains all possible properties and
// values from [ProviderAuthError], [UnknownError], [MessageOutputLengthError],
// [MessageAbortedError], [APIError].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type EventEventSessionErrorPropertiesErrorUnion struct {
	// This field is a union of [ProviderAuthErrorData], [UnknownErrorData], [any],
	// [MessageAbortedErrorData], [APIErrorData]
	Data EventEventSessionErrorPropertiesErrorUnionData `json:"data"`
	Name string                                         `json:"name"`
	JSON struct {
		Data respjson.Field
		Name respjson.Field
		raw  string
	} `json:"-"`
}

func (u EventEventSessionErrorPropertiesErrorUnion) AsProviderAuthError() (v ProviderAuthError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventEventSessionErrorPropertiesErrorUnion) AsUnknownError() (v UnknownError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventEventSessionErrorPropertiesErrorUnion) AsMessageOutputLengthError() (v MessageOutputLengthError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventEventSessionErrorPropertiesErrorUnion) AsMessageAbortedError() (v MessageAbortedError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u EventEventSessionErrorPropertiesErrorUnion) AsAPIError() (v APIError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u EventEventSessionErrorPropertiesErrorUnion) RawJSON() string { return u.JSON.raw }

func (r *EventEventSessionErrorPropertiesErrorUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// EventEventSessionErrorPropertiesErrorUnionData is an implicit subunion of
// [EventEventSessionErrorPropertiesErrorUnion].
// EventEventSessionErrorPropertiesErrorUnionData provides convenient access to the
// sub-properties of the union.
//
// For type safety it is recommended to directly use a variant of the
// [EventEventSessionErrorPropertiesErrorUnion].
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfMessageOutputLengthErrorData]
type EventEventSessionErrorPropertiesErrorUnionData struct {
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

func (r *EventEventSessionErrorPropertiesErrorUnionData) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventClientToolRequest struct {
	Properties EventEventClientToolRequestProperties `json:"properties,required"`
	Type       constant.ClientToolRequest            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventClientToolRequest) RawJSON() string { return r.JSON.raw }
func (r *EventEventClientToolRequest) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventClientToolRequestProperties struct {
	ClientID string              `json:"clientID,required"`
	Request  ClientToolExecution `json:"request,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ClientID    respjson.Field
		Request     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventClientToolRequestProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventClientToolRequestProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventClientToolRegistered struct {
	Properties EventEventClientToolRegisteredProperties `json:"properties,required"`
	Type       constant.ClientToolRegistered            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventClientToolRegistered) RawJSON() string { return r.JSON.raw }
func (r *EventEventClientToolRegistered) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventClientToolRegisteredProperties struct {
	ClientID string   `json:"clientID,required"`
	ToolIDs  []string `json:"toolIDs,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ClientID    respjson.Field
		ToolIDs     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventClientToolRegisteredProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventClientToolRegisteredProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventClientToolUnregistered struct {
	Properties EventEventClientToolUnregisteredProperties `json:"properties,required"`
	Type       constant.ClientToolUnregistered            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventClientToolUnregistered) RawJSON() string { return r.JSON.raw }
func (r *EventEventClientToolUnregistered) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventClientToolUnregisteredProperties struct {
	ClientID string   `json:"clientID,required"`
	ToolIDs  []string `json:"toolIDs,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ClientID    respjson.Field
		ToolIDs     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventClientToolUnregisteredProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventClientToolUnregisteredProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventClientToolExecuting struct {
	Properties EventEventClientToolExecutingProperties `json:"properties,required"`
	Type       constant.ClientToolExecuting            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventClientToolExecuting) RawJSON() string { return r.JSON.raw }
func (r *EventEventClientToolExecuting) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventClientToolExecutingProperties struct {
	CallID    string `json:"callID,required"`
	ClientID  string `json:"clientID,required"`
	MessageID string `json:"messageID,required"`
	SessionID string `json:"sessionID,required"`
	Tool      string `json:"tool,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		CallID      respjson.Field
		ClientID    respjson.Field
		MessageID   respjson.Field
		SessionID   respjson.Field
		Tool        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventClientToolExecutingProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventClientToolExecutingProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventClientToolCompleted struct {
	Properties EventEventClientToolCompletedProperties `json:"properties,required"`
	Type       constant.ClientToolCompleted            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventClientToolCompleted) RawJSON() string { return r.JSON.raw }
func (r *EventEventClientToolCompleted) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventClientToolCompletedProperties struct {
	CallID    string `json:"callID,required"`
	ClientID  string `json:"clientID,required"`
	MessageID string `json:"messageID,required"`
	SessionID string `json:"sessionID,required"`
	Success   bool   `json:"success,required"`
	Tool      string `json:"tool,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		CallID      respjson.Field
		ClientID    respjson.Field
		MessageID   respjson.Field
		SessionID   respjson.Field
		Success     respjson.Field
		Tool        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventClientToolCompletedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventClientToolCompletedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventClientToolFailed struct {
	Properties EventEventClientToolFailedProperties `json:"properties,required"`
	Type       constant.ClientToolFailed            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventClientToolFailed) RawJSON() string { return r.JSON.raw }
func (r *EventEventClientToolFailed) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventClientToolFailedProperties struct {
	CallID    string `json:"callID,required"`
	ClientID  string `json:"clientID,required"`
	Error     string `json:"error,required"`
	MessageID string `json:"messageID,required"`
	SessionID string `json:"sessionID,required"`
	Tool      string `json:"tool,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		CallID      respjson.Field
		ClientID    respjson.Field
		Error       respjson.Field
		MessageID   respjson.Field
		SessionID   respjson.Field
		Tool        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventClientToolFailedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventClientToolFailedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventServerConnected struct {
	Properties any                      `json:"properties,required"`
	Type       constant.ServerConnected `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventServerConnected) RawJSON() string { return r.JSON.raw }
func (r *EventEventServerConnected) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventFileWatcherUpdated struct {
	Properties EventEventFileWatcherUpdatedProperties `json:"properties,required"`
	Type       constant.FileWatcherUpdated            `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Properties  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventFileWatcherUpdated) RawJSON() string { return r.JSON.raw }
func (r *EventEventFileWatcherUpdated) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventFileWatcherUpdatedProperties struct {
	// Any of "add", "change", "unlink".
	Event EventEventFileWatcherUpdatedPropertiesEvent `json:"event,required"`
	File  string                                      `json:"file,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Event       respjson.Field
		File        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EventEventFileWatcherUpdatedProperties) RawJSON() string { return r.JSON.raw }
func (r *EventEventFileWatcherUpdatedProperties) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type EventEventFileWatcherUpdatedPropertiesEvent string

const (
	EventEventFileWatcherUpdatedPropertiesEventAdd    EventEventFileWatcherUpdatedPropertiesEvent = "add"
	EventEventFileWatcherUpdatedPropertiesEventChange EventEventFileWatcherUpdatedPropertiesEvent = "change"
	EventEventFileWatcherUpdatedPropertiesEventUnlink EventEventFileWatcherUpdatedPropertiesEvent = "unlink"
)

type EventListParams struct {
	Directory param.Opt[string] `query:"directory,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [EventListParams]'s query parameters as `url.Values`.
func (r EventListParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}
