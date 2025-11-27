// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package constant

import (
	shimjson "github.com/sst/opencode-sdk-go/internal/encoding/json"
)

type Constant[T any] interface {
	Default() T
}

// ValueOf gives the default value of a constant from its type. It's helpful when
// constructing constants as variants in a one-of. Note that empty structs are
// marshalled by default. Usage: constant.ValueOf[constant.Foo]()
func ValueOf[T Constant[T]]() T {
	var t T
	return t.Default()
}

type Agent string                       // Always "agent"
type API string                         // Always "api"
type APIError string                    // Always "APIError"
type Assistant string                   // Always "assistant"
type Busy string                        // Always "busy"
type ClientToolRequest string           // Always "client-tool-request"
type ClientToolCompleted string         // Always "client-tool.completed"
type ClientToolExecuting string         // Always "client-tool.executing"
type ClientToolFailed string            // Always "client-tool.failed"
type ClientToolRegistered string        // Always "client-tool.registered"
type ClientToolRequested string         // Always "client-tool.request"
type ClientToolUnregistered string      // Always "client-tool.unregistered"
type CommandExecuted string             // Always "command.executed"
type Compaction string                  // Always "compaction"
type Completed string                   // Always "completed"
type Connected string                   // Always "connected"
type Disabled string                    // Always "disabled"
type Error string                       // Always "error"
type Failed string                      // Always "failed"
type File string                        // Always "file"
type FileEdited string                  // Always "file.edited"
type FileWatcherUpdated string          // Always "file.watcher.updated"
type Idle string                        // Always "idle"
type InstallationUpdateAvailable string // Always "installation.update-available"
type InstallationUpdated string         // Always "installation.updated"
type Local string                       // Always "local"
type LspClientDiagnostics string        // Always "lsp.client.diagnostics"
type LspUpdated string                  // Always "lsp.updated"
type MessagePartRemoved string          // Always "message.part.removed"
type MessagePartUpdated string          // Always "message.part.updated"
type MessageRemoved string              // Always "message.removed"
type MessageUpdated string              // Always "message.updated"
type MessageAbortedError string         // Always "MessageAbortedError"
type MessageOutputLengthError string    // Always "MessageOutputLengthError"
type Notify string                      // Always "notify"
type OAuth string                       // Always "oauth"
type Patch string                       // Always "patch"
type Pending string                     // Always "pending"
type PermissionReplied string           // Always "permission.replied"
type PermissionUpdated string           // Always "permission.updated"
type ProviderAuthError string           // Always "ProviderAuthError"
type Reasoning string                   // Always "reasoning"
type Remote string                      // Always "remote"
type Retry string                       // Always "retry"
type Running string                     // Always "running"
type ServerConnected string             // Always "server.connected"
type SessionCompacted string            // Always "session.compacted"
type SessionCreated string              // Always "session.created"
type SessionDeleted string              // Always "session.deleted"
type SessionDiff string                 // Always "session.diff"
type SessionError string                // Always "session.error"
type SessionIdle string                 // Always "session.idle"
type SessionStatus string               // Always "session.status"
type SessionUpdated string              // Always "session.updated"
type Snapshot string                    // Always "snapshot"
type StepFinish string                  // Always "step-finish"
type StepStart string                   // Always "step-start"
type Subtask string                     // Always "subtask"
type Success string                     // Always "success"
type Symbol string                      // Always "symbol"
type Text string                        // Always "text"
type TodoUpdated string                 // Always "todo.updated"
type Tool string                        // Always "tool"
type TuiCommandExecute string           // Always "tui.command.execute"
type TuiPromptAppend string             // Always "tui.prompt.append"
type TuiToastShow string                // Always "tui.toast.show"
type UnknownError string                // Always "UnknownError"
type User string                        // Always "user"
type Wellknown string                   // Always "wellknown"
type WorkflowCancelled string           // Always "workflow.cancelled"
type WorkflowCompleted string           // Always "workflow.completed"
type WorkflowFailed string              // Always "workflow.failed"
type WorkflowPaused string              // Always "workflow.paused"
type WorkflowResumed string             // Always "workflow.resumed"
type WorkflowStarted string             // Always "workflow.started"
type WorkflowStepCompleted string       // Always "workflow.step.completed"
type WorkflowStepFailed string          // Always "workflow.step.failed"
type WorkflowStepStarted string         // Always "workflow.step.started"

func (c Agent) Default() Agent                                   { return "agent" }
func (c API) Default() API                                       { return "api" }
func (c APIError) Default() APIError                             { return "APIError" }
func (c Assistant) Default() Assistant                           { return "assistant" }
func (c Busy) Default() Busy                                     { return "busy" }
func (c ClientToolRequest) Default() ClientToolRequest           { return "client-tool-request" }
func (c ClientToolCompleted) Default() ClientToolCompleted       { return "client-tool.completed" }
func (c ClientToolExecuting) Default() ClientToolExecuting       { return "client-tool.executing" }
func (c ClientToolFailed) Default() ClientToolFailed             { return "client-tool.failed" }
func (c ClientToolRegistered) Default() ClientToolRegistered     { return "client-tool.registered" }
func (c ClientToolRequested) Default() ClientToolRequested       { return "client-tool.request" }
func (c ClientToolUnregistered) Default() ClientToolUnregistered { return "client-tool.unregistered" }
func (c CommandExecuted) Default() CommandExecuted               { return "command.executed" }
func (c Compaction) Default() Compaction                         { return "compaction" }
func (c Completed) Default() Completed                           { return "completed" }
func (c Connected) Default() Connected                           { return "connected" }
func (c Disabled) Default() Disabled                             { return "disabled" }
func (c Error) Default() Error                                   { return "error" }
func (c Failed) Default() Failed                                 { return "failed" }
func (c File) Default() File                                     { return "file" }
func (c FileEdited) Default() FileEdited                         { return "file.edited" }
func (c FileWatcherUpdated) Default() FileWatcherUpdated         { return "file.watcher.updated" }
func (c Idle) Default() Idle                                     { return "idle" }
func (c InstallationUpdateAvailable) Default() InstallationUpdateAvailable {
	return "installation.update-available"
}
func (c InstallationUpdated) Default() InstallationUpdated   { return "installation.updated" }
func (c Local) Default() Local                               { return "local" }
func (c LspClientDiagnostics) Default() LspClientDiagnostics { return "lsp.client.diagnostics" }
func (c LspUpdated) Default() LspUpdated                     { return "lsp.updated" }
func (c MessagePartRemoved) Default() MessagePartRemoved     { return "message.part.removed" }
func (c MessagePartUpdated) Default() MessagePartUpdated     { return "message.part.updated" }
func (c MessageRemoved) Default() MessageRemoved             { return "message.removed" }
func (c MessageUpdated) Default() MessageUpdated             { return "message.updated" }
func (c MessageAbortedError) Default() MessageAbortedError   { return "MessageAbortedError" }
func (c MessageOutputLengthError) Default() MessageOutputLengthError {
	return "MessageOutputLengthError"
}
func (c Notify) Default() Notify                               { return "notify" }
func (c OAuth) Default() OAuth                                 { return "oauth" }
func (c Patch) Default() Patch                                 { return "patch" }
func (c Pending) Default() Pending                             { return "pending" }
func (c PermissionReplied) Default() PermissionReplied         { return "permission.replied" }
func (c PermissionUpdated) Default() PermissionUpdated         { return "permission.updated" }
func (c ProviderAuthError) Default() ProviderAuthError         { return "ProviderAuthError" }
func (c Reasoning) Default() Reasoning                         { return "reasoning" }
func (c Remote) Default() Remote                               { return "remote" }
func (c Retry) Default() Retry                                 { return "retry" }
func (c Running) Default() Running                             { return "running" }
func (c ServerConnected) Default() ServerConnected             { return "server.connected" }
func (c SessionCompacted) Default() SessionCompacted           { return "session.compacted" }
func (c SessionCreated) Default() SessionCreated               { return "session.created" }
func (c SessionDeleted) Default() SessionDeleted               { return "session.deleted" }
func (c SessionDiff) Default() SessionDiff                     { return "session.diff" }
func (c SessionError) Default() SessionError                   { return "session.error" }
func (c SessionIdle) Default() SessionIdle                     { return "session.idle" }
func (c SessionStatus) Default() SessionStatus                 { return "session.status" }
func (c SessionUpdated) Default() SessionUpdated               { return "session.updated" }
func (c Snapshot) Default() Snapshot                           { return "snapshot" }
func (c StepFinish) Default() StepFinish                       { return "step-finish" }
func (c StepStart) Default() StepStart                         { return "step-start" }
func (c Subtask) Default() Subtask                             { return "subtask" }
func (c Success) Default() Success                             { return "success" }
func (c Symbol) Default() Symbol                               { return "symbol" }
func (c Text) Default() Text                                   { return "text" }
func (c TodoUpdated) Default() TodoUpdated                     { return "todo.updated" }
func (c Tool) Default() Tool                                   { return "tool" }
func (c TuiCommandExecute) Default() TuiCommandExecute         { return "tui.command.execute" }
func (c TuiPromptAppend) Default() TuiPromptAppend             { return "tui.prompt.append" }
func (c TuiToastShow) Default() TuiToastShow                   { return "tui.toast.show" }
func (c UnknownError) Default() UnknownError                   { return "UnknownError" }
func (c User) Default() User                                   { return "user" }
func (c Wellknown) Default() Wellknown                         { return "wellknown" }
func (c WorkflowCancelled) Default() WorkflowCancelled         { return "workflow.cancelled" }
func (c WorkflowCompleted) Default() WorkflowCompleted         { return "workflow.completed" }
func (c WorkflowFailed) Default() WorkflowFailed               { return "workflow.failed" }
func (c WorkflowPaused) Default() WorkflowPaused               { return "workflow.paused" }
func (c WorkflowResumed) Default() WorkflowResumed             { return "workflow.resumed" }
func (c WorkflowStarted) Default() WorkflowStarted             { return "workflow.started" }
func (c WorkflowStepCompleted) Default() WorkflowStepCompleted { return "workflow.step.completed" }
func (c WorkflowStepFailed) Default() WorkflowStepFailed       { return "workflow.step.failed" }
func (c WorkflowStepStarted) Default() WorkflowStepStarted     { return "workflow.step.started" }

func (c Agent) MarshalJSON() ([]byte, error)                       { return marshalString(c) }
func (c API) MarshalJSON() ([]byte, error)                         { return marshalString(c) }
func (c APIError) MarshalJSON() ([]byte, error)                    { return marshalString(c) }
func (c Assistant) MarshalJSON() ([]byte, error)                   { return marshalString(c) }
func (c Busy) MarshalJSON() ([]byte, error)                        { return marshalString(c) }
func (c ClientToolRequest) MarshalJSON() ([]byte, error)           { return marshalString(c) }
func (c ClientToolCompleted) MarshalJSON() ([]byte, error)         { return marshalString(c) }
func (c ClientToolExecuting) MarshalJSON() ([]byte, error)         { return marshalString(c) }
func (c ClientToolFailed) MarshalJSON() ([]byte, error)            { return marshalString(c) }
func (c ClientToolRegistered) MarshalJSON() ([]byte, error)        { return marshalString(c) }
func (c ClientToolRequested) MarshalJSON() ([]byte, error)         { return marshalString(c) }
func (c ClientToolUnregistered) MarshalJSON() ([]byte, error)      { return marshalString(c) }
func (c CommandExecuted) MarshalJSON() ([]byte, error)             { return marshalString(c) }
func (c Compaction) MarshalJSON() ([]byte, error)                  { return marshalString(c) }
func (c Completed) MarshalJSON() ([]byte, error)                   { return marshalString(c) }
func (c Connected) MarshalJSON() ([]byte, error)                   { return marshalString(c) }
func (c Disabled) MarshalJSON() ([]byte, error)                    { return marshalString(c) }
func (c Error) MarshalJSON() ([]byte, error)                       { return marshalString(c) }
func (c Failed) MarshalJSON() ([]byte, error)                      { return marshalString(c) }
func (c File) MarshalJSON() ([]byte, error)                        { return marshalString(c) }
func (c FileEdited) MarshalJSON() ([]byte, error)                  { return marshalString(c) }
func (c FileWatcherUpdated) MarshalJSON() ([]byte, error)          { return marshalString(c) }
func (c Idle) MarshalJSON() ([]byte, error)                        { return marshalString(c) }
func (c InstallationUpdateAvailable) MarshalJSON() ([]byte, error) { return marshalString(c) }
func (c InstallationUpdated) MarshalJSON() ([]byte, error)         { return marshalString(c) }
func (c Local) MarshalJSON() ([]byte, error)                       { return marshalString(c) }
func (c LspClientDiagnostics) MarshalJSON() ([]byte, error)        { return marshalString(c) }
func (c LspUpdated) MarshalJSON() ([]byte, error)                  { return marshalString(c) }
func (c MessagePartRemoved) MarshalJSON() ([]byte, error)          { return marshalString(c) }
func (c MessagePartUpdated) MarshalJSON() ([]byte, error)          { return marshalString(c) }
func (c MessageRemoved) MarshalJSON() ([]byte, error)              { return marshalString(c) }
func (c MessageUpdated) MarshalJSON() ([]byte, error)              { return marshalString(c) }
func (c MessageAbortedError) MarshalJSON() ([]byte, error)         { return marshalString(c) }
func (c MessageOutputLengthError) MarshalJSON() ([]byte, error)    { return marshalString(c) }
func (c Notify) MarshalJSON() ([]byte, error)                      { return marshalString(c) }
func (c OAuth) MarshalJSON() ([]byte, error)                       { return marshalString(c) }
func (c Patch) MarshalJSON() ([]byte, error)                       { return marshalString(c) }
func (c Pending) MarshalJSON() ([]byte, error)                     { return marshalString(c) }
func (c PermissionReplied) MarshalJSON() ([]byte, error)           { return marshalString(c) }
func (c PermissionUpdated) MarshalJSON() ([]byte, error)           { return marshalString(c) }
func (c ProviderAuthError) MarshalJSON() ([]byte, error)           { return marshalString(c) }
func (c Reasoning) MarshalJSON() ([]byte, error)                   { return marshalString(c) }
func (c Remote) MarshalJSON() ([]byte, error)                      { return marshalString(c) }
func (c Retry) MarshalJSON() ([]byte, error)                       { return marshalString(c) }
func (c Running) MarshalJSON() ([]byte, error)                     { return marshalString(c) }
func (c ServerConnected) MarshalJSON() ([]byte, error)             { return marshalString(c) }
func (c SessionCompacted) MarshalJSON() ([]byte, error)            { return marshalString(c) }
func (c SessionCreated) MarshalJSON() ([]byte, error)              { return marshalString(c) }
func (c SessionDeleted) MarshalJSON() ([]byte, error)              { return marshalString(c) }
func (c SessionDiff) MarshalJSON() ([]byte, error)                 { return marshalString(c) }
func (c SessionError) MarshalJSON() ([]byte, error)                { return marshalString(c) }
func (c SessionIdle) MarshalJSON() ([]byte, error)                 { return marshalString(c) }
func (c SessionStatus) MarshalJSON() ([]byte, error)               { return marshalString(c) }
func (c SessionUpdated) MarshalJSON() ([]byte, error)              { return marshalString(c) }
func (c Snapshot) MarshalJSON() ([]byte, error)                    { return marshalString(c) }
func (c StepFinish) MarshalJSON() ([]byte, error)                  { return marshalString(c) }
func (c StepStart) MarshalJSON() ([]byte, error)                   { return marshalString(c) }
func (c Subtask) MarshalJSON() ([]byte, error)                     { return marshalString(c) }
func (c Success) MarshalJSON() ([]byte, error)                     { return marshalString(c) }
func (c Symbol) MarshalJSON() ([]byte, error)                      { return marshalString(c) }
func (c Text) MarshalJSON() ([]byte, error)                        { return marshalString(c) }
func (c TodoUpdated) MarshalJSON() ([]byte, error)                 { return marshalString(c) }
func (c Tool) MarshalJSON() ([]byte, error)                        { return marshalString(c) }
func (c TuiCommandExecute) MarshalJSON() ([]byte, error)           { return marshalString(c) }
func (c TuiPromptAppend) MarshalJSON() ([]byte, error)             { return marshalString(c) }
func (c TuiToastShow) MarshalJSON() ([]byte, error)                { return marshalString(c) }
func (c UnknownError) MarshalJSON() ([]byte, error)                { return marshalString(c) }
func (c User) MarshalJSON() ([]byte, error)                        { return marshalString(c) }
func (c Wellknown) MarshalJSON() ([]byte, error)                   { return marshalString(c) }
func (c WorkflowCancelled) MarshalJSON() ([]byte, error)           { return marshalString(c) }
func (c WorkflowCompleted) MarshalJSON() ([]byte, error)           { return marshalString(c) }
func (c WorkflowFailed) MarshalJSON() ([]byte, error)              { return marshalString(c) }
func (c WorkflowPaused) MarshalJSON() ([]byte, error)              { return marshalString(c) }
func (c WorkflowResumed) MarshalJSON() ([]byte, error)             { return marshalString(c) }
func (c WorkflowStarted) MarshalJSON() ([]byte, error)             { return marshalString(c) }
func (c WorkflowStepCompleted) MarshalJSON() ([]byte, error)       { return marshalString(c) }
func (c WorkflowStepFailed) MarshalJSON() ([]byte, error)          { return marshalString(c) }
func (c WorkflowStepStarted) MarshalJSON() ([]byte, error)         { return marshalString(c) }

type constant[T any] interface {
	Constant[T]
	*T
}

func marshalString[T ~string, PT constant[T]](v T) ([]byte, error) {
	var zero T
	if v == zero {
		v = PT(&v).Default()
	}
	return shimjson.Marshal(string(v))
}
