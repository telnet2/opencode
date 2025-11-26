# Technical Specifications

## Overview

This document provides detailed technical specifications for the Go OpenCode server implementation, including API contracts, data formats, and integration requirements.

---

## 1. API Specification

### Base URL
```
http://localhost:{port}
```

### Authentication
```
Header: Authorization: Bearer {api_key}
```

### Content Types
- Request: `application/json`
- Response: `application/json`
- Streaming: `text/event-stream`

---

## 2. REST API Endpoints

### Session Management

#### Create Session
```http
POST /session
Content-Type: application/json

{
  "directory": "/path/to/project"
}

Response 200:
{
  "id": "01HYQXYZ...",
  "projectID": "...",
  "directory": "/path/to/project",
  "title": "New Session",
  "time": { "created": 1700000000000, "updated": 1700000000000 },
  "summary": { "additions": 0, "deletions": 0, "files": 0 }
}
```

#### List Sessions
```http
GET /session?directory=/path/to/project

Response 200:
[
  { "id": "...", "title": "...", ... }
]
```

#### Get Session
```http
GET /session/{id}

Response 200:
{ "id": "...", "title": "...", ... }
```

#### Update Session
```http
PATCH /session/{id}
Content-Type: application/json

{
  "title": "Updated Title"
}

Response 200:
{ "id": "...", "title": "Updated Title", ... }
```

#### Delete Session
```http
DELETE /session/{id}

Response 200:
{ "success": true }
```

#### Send Message
```http
POST /session/{id}/message
Content-Type: application/json

{
  "content": "Hello, can you help me?",
  "agent": "build",
  "model": { "providerID": "anthropic", "modelID": "claude-sonnet-4" }
}

Response 200 (streaming via SSE):
{ "id": "...", "role": "assistant", ... }
```

#### Abort Session
```http
POST /session/{id}/abort

Response 200:
{ "success": true }
```

#### Fork Session
```http
POST /session/{id}/fork
Content-Type: application/json

{
  "messageID": "..."
}

Response 200:
{ "id": "new-session-id", ... }
```

#### Revert Session
```http
POST /session/{id}/revert
Content-Type: application/json

{
  "messageID": "..."
}

Response 200:
{ "success": true }
```

### File Operations

#### List Directory
```http
GET /file?path=/path/to/dir

Response 200:
{
  "files": [
    { "name": "file.txt", "isDirectory": false, "size": 1234 }
  ]
}
```

#### Read File
```http
GET /file/content?path=/path/to/file.txt&offset=0&limit=2000

Response 200:
{
  "content": "file contents...",
  "lines": 100,
  "truncated": false
}
```

#### Git Status
```http
GET /file/status?directory=/path/to/project

Response 200:
{
  "branch": "main",
  "staged": ["file1.txt"],
  "unstaged": ["file2.txt"],
  "untracked": ["file3.txt"]
}
```

### Search

#### Text Search (Grep)
```http
GET /find?pattern=TODO&path=/path/to/dir&include=*.ts

Response 200:
{
  "matches": [
    { "file": "src/app.ts", "line": 42, "content": "// TODO: fix this" }
  ],
  "count": 1
}
```

#### File Search
```http
GET /find/file?pattern=*.ts&path=/path/to/dir

Response 200:
{
  "files": ["src/app.ts", "src/util.ts"],
  "count": 2
}
```

### Configuration

#### Get Config
```http
GET /config?directory=/path/to/project

Response 200:
{
  "model": "anthropic/claude-sonnet-4",
  "provider": { ... },
  "experimental": { ... }
}
```

#### List Providers
```http
GET /provider

Response 200:
[
  {
    "id": "anthropic",
    "name": "Anthropic",
    "models": [
      { "id": "claude-sonnet-4", "name": "Claude Sonnet 4" }
    ]
  }
]
```

### Event Streaming

#### Global Events
```http
GET /global/event
Accept: text/event-stream

Response (SSE):
data: {"type":"session.created","data":{...}}

data: {"type":"message.updated","data":{...}}
```

#### Session Events
```http
GET /event?sessionID={id}
Accept: text/event-stream

Response (SSE):
data: {"type":"part.updated","data":{...}}
```

---

## 3. Event Types

### Session Events
```go
type SessionCreated struct {
    Type string   `json:"type"` // "session.created"
    Data Session  `json:"data"`
}

type SessionUpdated struct {
    Type string   `json:"type"` // "session.updated"
    Data Session  `json:"data"`
}

type SessionDeleted struct {
    Type string `json:"type"` // "session.deleted"
    Data struct {
        SessionID string `json:"sessionID"`
    } `json:"data"`
}
```

### Message Events
```go
type MessageUpdated struct {
    Type string  `json:"type"` // "message.updated"
    Data Message `json:"data"`
}

type PartUpdated struct {
    Type string `json:"type"` // "part.updated"
    Data struct {
        SessionID string `json:"sessionID"`
        MessageID string `json:"messageID"`
        Part      Part   `json:"part"`
        Delta     string `json:"delta,omitempty"` // For streaming text
    } `json:"data"`
}
```

### Permission Events
```go
type PermissionRequired struct {
    Type string `json:"type"` // "permission.required"
    Data struct {
        ID        string   `json:"id"`
        Type      string   `json:"permissionType"`
        Pattern   []string `json:"pattern"`
        SessionID string   `json:"sessionID"`
        Title     string   `json:"title"`
    } `json:"data"`
}
```

---

## 4. Data Storage Format

### Directory Structure
```
~/.local/share/opencode/
├── storage/
│   ├── session/
│   │   └── {projectID}/
│   │       └── {sessionID}.json
│   ├── message/
│   │   └── {sessionID}/
│   │       └── {messageID}.json
│   └── part/
│       └── {messageID}/
│           └── {partID}.json
├── auth.json
└── cache/
    └── models.json
```

### Session JSON
```json
{
  "id": "01HYQXYZ...",
  "projectID": "proj_abc123",
  "directory": "/path/to/project",
  "parentID": null,
  "title": "Debug authentication issue",
  "version": "2",
  "summary": {
    "additions": 42,
    "deletions": 10,
    "files": 3,
    "diffs": []
  },
  "time": {
    "created": 1700000000000,
    "updated": 1700001000000
  }
}
```

### Message JSON
```json
{
  "id": "01HYQABC...",
  "sessionID": "01HYQXYZ...",
  "role": "assistant",
  "time": { "created": 1700000500000 },
  "modelID": "claude-sonnet-4",
  "providerID": "anthropic",
  "mode": "build",
  "finish": "stop",
  "cost": 0.0123,
  "tokens": {
    "input": 1000,
    "output": 500,
    "reasoning": 0,
    "cache": { "read": 100, "write": 50 }
  }
}
```

### Part JSON
```json
{
  "id": "part_001",
  "type": "tool",
  "toolCallID": "call_abc",
  "toolName": "edit",
  "input": {
    "file_path": "/path/to/file.ts",
    "old_string": "foo",
    "new_string": "bar"
  },
  "state": "completed",
  "output": "Replaced 1 occurrence",
  "title": "Edit file.ts",
  "time": { "start": 1700000600000, "end": 1700000601000 }
}
```

---

## 5. LLM Provider Integration

### Provider Interface
```go
type Provider interface {
    ID() string
    Models() []Model
    CreateCompletion(ctx context.Context, req CompletionRequest) (CompletionStream, error)
}

type CompletionRequest struct {
    Model       string
    Messages    []Message
    Tools       []Tool
    MaxTokens   int
    Temperature float64
    TopP        float64
    StopWords   []string
}

type CompletionStream interface {
    Next() (StreamEvent, error)
    Close() error
}

type StreamEvent interface {
    eventType() string
}

type TextDeltaEvent struct {
    Text string
}

type ToolCallStartEvent struct {
    ID       string
    Name     string
}

type ToolCallDeltaEvent struct {
    ID    string
    Delta string // JSON fragment
}

type ToolCallEndEvent struct {
    ID    string
    Input json.RawMessage
}

type FinishEvent struct {
    Reason string // "stop", "tool_calls", "max_tokens"
    Usage  TokenUsage
}
```

### Provider Implementations

#### Anthropic
```go
// Uses: github.com/anthropics/anthropic-sdk-go
provider := anthropic.NewProvider(anthropic.Config{
    APIKey: os.Getenv("ANTHROPIC_API_KEY"),
    // Beta headers for streaming tool use
    BetaHeaders: []string{"prompt-caching-2024-07-31"},
})
```

#### OpenAI
```go
// Uses: github.com/openai/openai-go
provider := openai.NewProvider(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})
```

#### Google
```go
// Uses: google.golang.org/genai
provider := google.NewProvider(google.Config{
    APIKey: os.Getenv("GOOGLE_API_KEY"),
})
```

---

## 6. Tool JSON Schema

### Read Tool
```json
{
  "name": "read",
  "description": "Reads a file from the local filesystem...",
  "parameters": {
    "type": "object",
    "properties": {
      "file_path": {
        "type": "string",
        "description": "The absolute path to the file to read"
      },
      "offset": {
        "type": "integer",
        "description": "Line number to start reading from"
      },
      "limit": {
        "type": "integer",
        "description": "Number of lines to read (default: 2000)"
      }
    },
    "required": ["file_path"]
  }
}
```

### Edit Tool
```json
{
  "name": "edit",
  "description": "Performs exact string replacements in files...",
  "parameters": {
    "type": "object",
    "properties": {
      "file_path": {
        "type": "string",
        "description": "The absolute path to the file to modify"
      },
      "old_string": {
        "type": "string",
        "description": "The text to replace"
      },
      "new_string": {
        "type": "string",
        "description": "The text to replace it with"
      },
      "replace_all": {
        "type": "boolean",
        "description": "Replace all occurrences (default: false)"
      }
    },
    "required": ["file_path", "old_string", "new_string"]
  }
}
```

### Bash Tool
```json
{
  "name": "bash",
  "description": "Executes a bash command...",
  "parameters": {
    "type": "object",
    "properties": {
      "command": {
        "type": "string",
        "description": "The command to execute"
      },
      "timeout": {
        "type": "integer",
        "description": "Optional timeout in milliseconds (max 600000)"
      },
      "description": {
        "type": "string",
        "description": "Brief description of what this command does"
      }
    },
    "required": ["command", "description"]
  }
}
```

---

## 7. Bash Parsing with mvdan/sh

### Grammar Support

mvdan/sh supports:
- POSIX shell
- Bash (default)
- mksh

### Node Types

```go
// Key AST node types from mvdan.cc/sh/v3/syntax

// File - root node
type File struct {
    Stmts []*Stmt
}

// Stmt - a statement
type Stmt struct {
    Cmd      Command
    Negated  bool
    Position Pos
}

// CallExpr - a simple command call
type CallExpr struct {
    Args []*Word
}

// Word - a shell word (can be quoted, expanded, etc.)
type Word struct {
    Parts []WordPart
}

// WordPart types:
// - Lit: literal text
// - SglQuoted: 'single quoted'
// - DblQuoted: "double quoted"
// - ParamExp: $VAR or ${VAR}
// - CmdSubst: $(command) or `command`
// - ArithExp: $((expr))
```

### Parsing Example

```go
import (
    "mvdan.cc/sh/v3/syntax"
)

func parseCommand(cmd string) ([]Command, error) {
    parser := syntax.NewParser(syntax.Variant(syntax.LangBash))
    file, err := parser.Parse(strings.NewReader(cmd), "")
    if err != nil {
        return nil, err
    }

    var commands []Command
    syntax.Walk(file, func(node syntax.Node) bool {
        if call, ok := node.(*syntax.CallExpr); ok {
            commands = append(commands, extractCommand(call))
        }
        return true
    })
    return commands, nil
}
```

### Comparison with TypeScript

| Feature | TypeScript (tree-sitter) | Go (mvdan/sh) |
|---------|--------------------------|---------------|
| Parser init | Load WASM file | `syntax.NewParser()` |
| Parse | `parser.parse(command)` | `parser.Parse(reader, "")` |
| Walk AST | `tree.rootNode.descendantsOfType()` | `syntax.Walk(file, fn)` |
| Get text | `node.text` | `wordToString(word)` |
| Node types | `"command"`, `"word"` | `*CallExpr`, `*Word` |

---

## 8. Configuration Schema

```go
type Config struct {
    // Model selection
    Model      string `json:"model,omitempty"`       // "anthropic/claude-sonnet-4"
    SmallModel string `json:"small_model,omitempty"` // For fast tasks

    // Provider configs
    Provider map[string]ProviderConfig `json:"provider,omitempty"`

    // Agent configs
    Agent map[string]AgentConfig `json:"agent,omitempty"`

    // LSP
    LSP LSPConfig `json:"lsp,omitempty"`

    // File watcher
    Watcher WatcherConfig `json:"watcher,omitempty"`

    // Experimental features
    Experimental ExperimentalConfig `json:"experimental,omitempty"`
}

type ProviderConfig struct {
    APIKey  string `json:"apiKey,omitempty"`
    BaseURL string `json:"baseUrl,omitempty"`
    Disable bool   `json:"disable,omitempty"`
}

type AgentConfig struct {
    Tools      map[string]bool       `json:"tools,omitempty"`
    Permission AgentPermissionConfig `json:"permission,omitempty"`
}

type AgentPermissionConfig struct {
    Edit            string            `json:"edit,omitempty"`    // "allow"|"deny"|"ask"
    Bash            map[string]string `json:"bash,omitempty"`    // pattern -> action
    WebFetch        string            `json:"webfetch,omitempty"`
    ExternalDir     string            `json:"external_directory,omitempty"`
    DoomLoop        string            `json:"doom_loop,omitempty"`
}

type LSPConfig struct {
    Disabled bool              `json:"disabled,omitempty"`
    Servers  map[string]string `json:"servers,omitempty"` // language -> command
}

type WatcherConfig struct {
    Ignore []string `json:"ignore,omitempty"`
}

type ExperimentalConfig struct {
    BatchTool bool `json:"batch_tool,omitempty"`
}
```

---

## 9. Error Handling

### HTTP Error Responses

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Missing required field: directory",
    "details": { "field": "directory" }
  }
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Invalid request body/params |
| `NOT_FOUND` | 404 | Resource not found |
| `PERMISSION_DENIED` | 403 | Permission rejected |
| `PROVIDER_ERROR` | 502 | LLM provider error |
| `RATE_LIMITED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Server error |

### Go Error Types

```go
type APIError struct {
    Code    string         `json:"code"`
    Message string         `json:"message"`
    Details map[string]any `json:"details,omitempty"`
}

func (e *APIError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Predefined errors
var (
    ErrSessionNotFound = &APIError{Code: "NOT_FOUND", Message: "Session not found"}
    ErrInvalidInput    = &APIError{Code: "INVALID_REQUEST", Message: "Invalid input"}
)
```

---

## 10. Performance Requirements

### Response Times

| Operation | Target | Max |
|-----------|--------|-----|
| Create session | <50ms | 200ms |
| List sessions | <100ms | 500ms |
| Read file | <50ms | 200ms |
| Edit file | <100ms | 500ms |
| Glob search | <200ms | 1s |
| Grep search | <500ms | 5s |
| LLM first token | <2s | 10s |

### Resource Limits

| Resource | Limit |
|----------|-------|
| Max message content | 1MB |
| Max file read | 10MB |
| Max bash output | 30KB |
| Max SSE connections | 100 |
| Max concurrent sessions | 50 |

### Memory Budget

| Component | Target |
|-----------|--------|
| Idle server | <50MB |
| Per session | <10MB |
| Per active LLM stream | <5MB |
| Total server | <500MB |

---

## 11. Testing Requirements

### Unit Test Coverage

| Package | Target |
|---------|--------|
| `internal/storage` | >90% |
| `internal/permission` | >95% |
| `internal/tool` | >85% |
| `internal/provider` | >75% |
| `internal/session` | >80% |

### Integration Tests

- All 60+ API endpoints
- LLM streaming with mock provider
- Tool execution with real file system
- Permission flow end-to-end

### E2E Tests

- TUI client compatibility
- Full conversation flow
- Multi-tool execution
- Session persistence

---

*Document Version: 1.0*
*Last Updated: 2025-11-26*
