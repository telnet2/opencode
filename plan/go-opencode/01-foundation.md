# Phase 1: Foundation (Weeks 1-2)

## Overview

Establish the core infrastructure: types, storage layer, and event system. These components are dependencies for all other phases.

---

## 1.1 Core Types

### Session Types

```go
// pkg/types/session.go
package types

import "time"

type Session struct {
    ID          string          `json:"id"`
    ProjectID   string          `json:"projectID"`
    Directory   string          `json:"directory"`
    ParentID    *string         `json:"parentID,omitempty"`
    Title       string          `json:"title"`
    Version     string          `json:"version"`
    Summary     SessionSummary  `json:"summary"`
    Share       *SessionShare   `json:"share,omitempty"`
    Time        SessionTime     `json:"time"`
    Revert      *SessionRevert  `json:"revert,omitempty"`
    CustomPrompt *CustomPrompt  `json:"customPrompt,omitempty"`
}

type SessionSummary struct {
    Additions int        `json:"additions"`
    Deletions int        `json:"deletions"`
    Files     int        `json:"files"`
    Diffs     []FileDiff `json:"diffs,omitempty"`
}

type SessionTime struct {
    Created    int64  `json:"created"`
    Updated    int64  `json:"updated"`
    Compacting *int64 `json:"compacting,omitempty"`
}

type SessionShare struct {
    URL string `json:"url"`
}

type SessionRevert struct {
    MessageID string  `json:"messageID"`
    PartID    *string `json:"partID,omitempty"`
    Snapshot  *string `json:"snapshot,omitempty"`
    Diff      *string `json:"diff,omitempty"`
}

type CustomPrompt struct {
    Type      string            `json:"type"` // "file" | "inline"
    Value     string            `json:"value"`
    LoadedAt  *int64            `json:"loadedAt,omitempty"`
    Variables map[string]string `json:"variables,omitempty"`
}
```

### Message Types

```go
// pkg/types/message.go
package types

// Message represents either a User or Assistant message
type Message struct {
    ID        string       `json:"id"`
    SessionID string       `json:"sessionID"`
    Role      string       `json:"role"` // "user" | "assistant"
    Time      MessageTime  `json:"time"`

    // User-specific fields
    Agent   string            `json:"agent,omitempty"`
    Model   *ModelRef         `json:"model,omitempty"`
    System  *string           `json:"system,omitempty"`
    Tools   map[string]bool   `json:"tools,omitempty"`

    // Assistant-specific fields
    ModelID    string           `json:"modelID,omitempty"`
    ProviderID string           `json:"providerID,omitempty"`
    Mode       string           `json:"mode,omitempty"`
    Finish     *string          `json:"finish,omitempty"`
    Cost       float64          `json:"cost,omitempty"`
    Tokens     *TokenUsage      `json:"tokens,omitempty"`
    Error      *MessageError    `json:"error,omitempty"`
}

type MessageTime struct {
    Created int64  `json:"created"`
    Updated *int64 `json:"updated,omitempty"`
}

type ModelRef struct {
    ProviderID string `json:"providerID"`
    ModelID    string `json:"modelID"`
}

type TokenUsage struct {
    Input     int        `json:"input"`
    Output    int        `json:"output"`
    Reasoning int        `json:"reasoning,omitempty"`
    Cache     CacheUsage `json:"cache,omitempty"`
}

type CacheUsage struct {
    Read  int `json:"read"`
    Write int `json:"write"`
}

type MessageError struct {
    Type    string `json:"type"` // "api" | "auth" | "output_length"
    Message string `json:"message"`
}
```

### Message Parts

```go
// pkg/types/parts.go
package types

// Part represents a component of an assistant message
type Part interface {
    PartType() string
    PartID() string
}

type TextPart struct {
    ID       string          `json:"id"`
    Type     string          `json:"type"` // always "text"
    Text     string          `json:"text"`
    Time     PartTime        `json:"time,omitempty"`
    Metadata map[string]any  `json:"metadata,omitempty"`
}

func (p TextPart) PartType() string { return "text" }
func (p TextPart) PartID() string   { return p.ID }

type ReasoningPart struct {
    ID       string   `json:"id"`
    Type     string   `json:"type"` // always "reasoning"
    Text     string   `json:"text"`
    Time     PartTime `json:"time,omitempty"`
}

func (p ReasoningPart) PartType() string { return "reasoning" }
func (p ReasoningPart) PartID() string   { return p.ID }

type ToolPart struct {
    ID         string         `json:"id"`
    Type       string         `json:"type"` // always "tool"
    ToolCallID string         `json:"toolCallID"`
    ToolName   string         `json:"toolName"`
    Input      map[string]any `json:"input"`
    State      string         `json:"state"` // "pending" | "running" | "completed" | "error"
    Output     *string        `json:"output,omitempty"`
    Error      *string        `json:"error,omitempty"`
    Title      *string        `json:"title,omitempty"`
    Metadata   map[string]any `json:"metadata,omitempty"`
    Time       PartTime       `json:"time,omitempty"`
}

func (p ToolPart) PartType() string { return "tool" }
func (p ToolPart) PartID() string   { return p.ID }

type FilePart struct {
    ID       string `json:"id"`
    Type     string `json:"type"` // always "file"
    Filename string `json:"filename"`
    MediaType string `json:"mediaType"`
    URL      string `json:"url"`
}

func (p FilePart) PartType() string { return "file" }
func (p FilePart) PartID() string   { return p.ID }

type PartTime struct {
    Start *int64 `json:"start,omitempty"`
    End   *int64 `json:"end,omitempty"`
}
```

---

## 1.2 Storage Layer

### Interface

```go
// internal/storage/storage.go
package storage

import (
    "context"
    "encoding/json"
)

// Storage provides file-based JSON storage matching TypeScript implementation
type Storage struct {
    basePath string
}

func New(basePath string) *Storage {
    return &Storage{basePath: basePath}
}

// Path structure: storage/{type}/{id1}/{id2}/...
// Examples:
//   - storage/session/{projectID}/{sessionID}.json
//   - storage/message/{sessionID}/{messageID}.json
//   - storage/part/{messageID}/{partID}.json

func (s *Storage) Get(ctx context.Context, path []string, v any) error {
    // Read JSON file at path
    // Unmarshal into v
}

func (s *Storage) Put(ctx context.Context, path []string, v any) error {
    // Marshal v to JSON
    // Write to file at path with locking
}

func (s *Storage) Delete(ctx context.Context, path []string) error {
    // Delete file at path
}

func (s *Storage) List(ctx context.Context, path []string) ([]string, error) {
    // List files/directories at path
}

func (s *Storage) Scan(ctx context.Context, path []string, fn func(key string, data json.RawMessage) error) error {
    // Iterate over all items at path
}
```

### File Locking

```go
// internal/storage/lock.go
package storage

import (
    "os"
    "syscall"
)

type FileLock struct {
    path string
    file *os.File
}

func NewFileLock(path string) *FileLock {
    return &FileLock{path: path}
}

func (l *FileLock) Lock() error {
    var err error
    l.file, err = os.OpenFile(l.path+".lock", os.O_CREATE|os.O_RDWR, 0600)
    if err != nil {
        return err
    }
    return syscall.Flock(int(l.file.Fd()), syscall.LOCK_EX)
}

func (l *FileLock) Unlock() error {
    if l.file == nil {
        return nil
    }
    syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
    l.file.Close()
    os.Remove(l.path + ".lock")
    return nil
}
```

### Session Storage

```go
// internal/session/storage.go
package session

import (
    "context"
    "github.com/opencode-ai/opencode-server/internal/storage"
    "github.com/opencode-ai/opencode-server/pkg/types"
)

type Store struct {
    storage *storage.Storage
}

func NewStore(s *storage.Storage) *Store {
    return &Store{storage: s}
}

func (s *Store) Create(ctx context.Context, session *types.Session) error {
    return s.storage.Put(ctx, []string{"session", session.ProjectID, session.ID}, session)
}

func (s *Store) Get(ctx context.Context, projectID, sessionID string) (*types.Session, error) {
    var session types.Session
    err := s.storage.Get(ctx, []string{"session", projectID, sessionID}, &session)
    return &session, err
}

func (s *Store) Update(ctx context.Context, session *types.Session) error {
    return s.storage.Put(ctx, []string{"session", session.ProjectID, session.ID}, session)
}

func (s *Store) Delete(ctx context.Context, projectID, sessionID string) error {
    return s.storage.Delete(ctx, []string{"session", projectID, sessionID})
}

func (s *Store) List(ctx context.Context, projectID string) ([]*types.Session, error) {
    var sessions []*types.Session
    err := s.storage.Scan(ctx, []string{"session", projectID}, func(key string, data json.RawMessage) error {
        var session types.Session
        if err := json.Unmarshal(data, &session); err != nil {
            return err
        }
        sessions = append(sessions, &session)
        return nil
    })
    return sessions, err
}
```

---

## 1.3 Event System

### Event Bus

```go
// internal/event/bus.go
package event

import (
    "sync"
)

type EventType string

const (
    SessionCreated  EventType = "session.created"
    SessionUpdated  EventType = "session.updated"
    SessionDeleted  EventType = "session.deleted"
    MessageUpdated  EventType = "message.updated"
    MessageRemoved  EventType = "message.removed"
    PartUpdated     EventType = "part.updated"
    FileEdited      EventType = "file.edited"
)

type Event struct {
    Type EventType `json:"type"`
    Data any       `json:"data"`
}

type Subscriber func(event Event)

type Bus struct {
    mu          sync.RWMutex
    subscribers map[EventType][]Subscriber
    global      []Subscriber
}

var globalBus = &Bus{
    subscribers: make(map[EventType][]Subscriber),
}

func Subscribe(eventType EventType, fn Subscriber) func() {
    globalBus.mu.Lock()
    defer globalBus.mu.Unlock()

    globalBus.subscribers[eventType] = append(globalBus.subscribers[eventType], fn)

    // Return unsubscribe function
    return func() {
        globalBus.mu.Lock()
        defer globalBus.mu.Unlock()
        subs := globalBus.subscribers[eventType]
        for i, sub := range subs {
            // Compare function pointers (simplified)
            if &sub == &fn {
                globalBus.subscribers[eventType] = append(subs[:i], subs[i+1:]...)
                break
            }
        }
    }
}

func SubscribeAll(fn Subscriber) func() {
    globalBus.mu.Lock()
    defer globalBus.mu.Unlock()

    globalBus.global = append(globalBus.global, fn)

    return func() {
        globalBus.mu.Lock()
        defer globalBus.mu.Unlock()
        for i, sub := range globalBus.global {
            if &sub == &fn {
                globalBus.global = append(globalBus.global[:i], globalBus.global[i+1:]...)
                break
            }
        }
    }
}

func Publish(event Event) {
    globalBus.mu.RLock()
    subs := globalBus.subscribers[event.Type]
    global := globalBus.global
    globalBus.mu.RUnlock()

    // Publish to type-specific subscribers
    for _, sub := range subs {
        go sub(event)
    }

    // Publish to global subscribers
    for _, sub := range global {
        go sub(event)
    }
}
```

### Event Types

```go
// internal/event/types.go
package event

type SessionCreatedData struct {
    Session *types.Session `json:"session"`
}

type SessionUpdatedData struct {
    Session *types.Session `json:"session"`
}

type SessionDeletedData struct {
    SessionID string `json:"sessionID"`
}

type MessageUpdatedData struct {
    Message *types.Message `json:"message"`
}

type PartUpdatedData struct {
    SessionID string     `json:"sessionID"`
    MessageID string     `json:"messageID"`
    Part      types.Part `json:"part"`
    Delta     *string    `json:"delta,omitempty"` // For streaming text
}
```

---

## 1.4 Configuration

### Config Loading

```go
// internal/config/config.go
package config

import (
    "encoding/json"
    "os"
    "path/filepath"
)

type Config struct {
    Model        string                 `json:"model,omitempty"`
    SmallModel   string                 `json:"small_model,omitempty"`
    Provider     map[string]ProviderCfg `json:"provider,omitempty"`
    LSP          *LSPConfig             `json:"lsp,omitempty"`
    Watcher      *WatcherConfig         `json:"watcher,omitempty"`
    Experimental *ExperimentalConfig    `json:"experimental,omitempty"`
}

type ProviderCfg struct {
    APIKey  string `json:"apiKey,omitempty"`
    BaseURL string `json:"baseUrl,omitempty"`
}

type LSPConfig struct {
    Disabled bool `json:"disabled,omitempty"`
}

type WatcherConfig struct {
    Ignore []string `json:"ignore,omitempty"`
}

type ExperimentalConfig struct {
    BatchTool bool `json:"batch_tool,omitempty"`
}

// Load configuration from multiple sources (priority order)
func Load(directory string) (*Config, error) {
    config := &Config{}

    // 1. Global config (~/.config/opencode/)
    if globalPath, err := globalConfigPath(); err == nil {
        loadConfigFile(filepath.Join(globalPath, "opencode.json"), config)
        loadConfigFile(filepath.Join(globalPath, "opencode.jsonc"), config)
    }

    // 2. Project config (.opencode/)
    loadConfigFile(filepath.Join(directory, ".opencode", "opencode.json"), config)
    loadConfigFile(filepath.Join(directory, ".opencode", "opencode.jsonc"), config)

    // 3. Environment variables
    applyEnvOverrides(config)

    return config, nil
}

func loadConfigFile(path string, config *Config) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return err // File doesn't exist, skip
    }

    // Strip JSONC comments if needed
    data = stripJSONComments(data)

    var fileConfig Config
    if err := json.Unmarshal(data, &fileConfig); err != nil {
        return err
    }

    mergeConfig(config, &fileConfig)
    return nil
}
```

### Paths

```go
// internal/config/paths.go
package config

import (
    "os"
    "path/filepath"
    "runtime"
)

type Paths struct {
    Data   string // ~/.local/share/opencode
    Config string // ~/.config/opencode
    Cache  string // ~/.cache/opencode
    State  string // ~/.local/state/opencode
}

func GetPaths() *Paths {
    return &Paths{
        Data:   getEnvOrDefault("XDG_DATA_HOME", defaultDataHome()) + "/opencode",
        Config: getEnvOrDefault("XDG_CONFIG_HOME", defaultConfigHome()) + "/opencode",
        Cache:  getEnvOrDefault("XDG_CACHE_HOME", defaultCacheHome()) + "/opencode",
        State:  getEnvOrDefault("XDG_STATE_HOME", defaultStateHome()) + "/opencode",
    }
}

func defaultDataHome() string {
    if runtime.GOOS == "windows" {
        return os.Getenv("APPDATA")
    }
    return filepath.Join(os.Getenv("HOME"), ".local", "share")
}

func defaultConfigHome() string {
    if runtime.GOOS == "windows" {
        return os.Getenv("APPDATA")
    }
    return filepath.Join(os.Getenv("HOME"), ".config")
}
```

---

## 1.5 Deliverables

### Unit Tests

```go
// test/unit/storage_test.go
package unit

import (
    "context"
    "os"
    "testing"

    "github.com/opencode-ai/opencode-server/internal/storage"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestStorage_PutGet(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "storage-test")
    require.NoError(t, err)
    defer os.RemoveAll(tmpDir)

    s := storage.New(tmpDir)
    ctx := context.Background()

    // Test Put
    data := map[string]string{"key": "value"}
    err = s.Put(ctx, []string{"test", "item"}, data)
    require.NoError(t, err)

    // Test Get
    var result map[string]string
    err = s.Get(ctx, []string{"test", "item"}, &result)
    require.NoError(t, err)
    assert.Equal(t, "value", result["key"])
}

func TestStorage_List(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "storage-test")
    require.NoError(t, err)
    defer os.RemoveAll(tmpDir)

    s := storage.New(tmpDir)
    ctx := context.Background()

    // Create multiple items
    s.Put(ctx, []string{"sessions", "proj1", "sess1"}, map[string]string{})
    s.Put(ctx, []string{"sessions", "proj1", "sess2"}, map[string]string{})

    // List items
    items, err := s.List(ctx, []string{"sessions", "proj1"})
    require.NoError(t, err)
    assert.Len(t, items, 2)
}
```

```go
// test/unit/event_test.go
package unit

import (
    "sync"
    "testing"
    "time"

    "github.com/opencode-ai/opencode-server/internal/event"
    "github.com/stretchr/testify/assert"
)

func TestBus_Subscribe(t *testing.T) {
    var received event.Event
    var wg sync.WaitGroup
    wg.Add(1)

    unsub := event.Subscribe(event.SessionCreated, func(e event.Event) {
        received = e
        wg.Done()
    })
    defer unsub()

    event.Publish(event.Event{
        Type: event.SessionCreated,
        Data: event.SessionCreatedData{},
    })

    // Wait with timeout
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        assert.Equal(t, event.SessionCreated, received.Type)
    case <-time.After(time.Second):
        t.Fatal("timeout waiting for event")
    }
}

func TestBus_Unsubscribe(t *testing.T) {
    callCount := 0

    unsub := event.Subscribe(event.SessionCreated, func(e event.Event) {
        callCount++
    })

    // Publish before unsubscribe
    event.Publish(event.Event{Type: event.SessionCreated})
    time.Sleep(10 * time.Millisecond)

    // Unsubscribe
    unsub()

    // Publish after unsubscribe
    event.Publish(event.Event{Type: event.SessionCreated})
    time.Sleep(10 * time.Millisecond)

    assert.Equal(t, 1, callCount)
}
```

### Acceptance Criteria

- [ ] All core types match TypeScript definitions
- [ ] Storage layer passes read/write/list/delete tests
- [ ] Event bus supports subscribe/unsubscribe/publish
- [ ] Configuration loads from global + project paths
- [ ] File locking prevents concurrent write corruption
- [ ] All tests pass with `go test ./...`
