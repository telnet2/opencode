# Test Plan: Go OpenCode Server

## Overview

This document outlines the comprehensive testing strategy for the Go OpenCode server, modeled after the existing TypeScript test infrastructure.

---

## 1. Test Framework

### Go Testing Stack

```go
// go.mod testing dependencies
require (
    github.com/stretchr/testify v1.9.0    // Assertions and mocking
)
```

**Why Go's Standard Testing + Testify:**
- Native `go test` integration
- Fast parallel test execution
- Testify provides familiar assertion patterns matching Bun's `expect()`
- Built-in benchmarking support

### Test Commands

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/tool/...

# Run specific test
go test -run TestBashTool ./internal/tool/

# Run with race detector
go test -race ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## 2. Test Structure

```
test/
├── fixture/
│   ├── fixture.go           # Temp directory helper (like TypeScript fixture.ts)
│   ├── mock_provider.go     # Mock LLM provider for testing
│   ├── mock_lsp_server.go   # Fake LSP server
│   └── testdata/            # Static test files
│       ├── config/
│       ├── sessions/
│       └── tools/
├── unit/
│   ├── storage_test.go
│   ├── event_test.go
│   ├── config_test.go
│   ├── permission_test.go
│   ├── bash_parser_test.go
│   ├── wildcard_test.go
│   ├── message_test.go
│   └── transform_test.go
├── integration/
│   ├── session_test.go
│   ├── tool_bash_test.go
│   ├── tool_edit_test.go
│   ├── tool_read_test.go
│   ├── provider_test.go
│   ├── lsp_test.go
│   └── server_test.go
└── e2e/
    ├── client_test.go       # Test with actual TUI client
    └── api_test.go          # Full API endpoint tests
```

---

## 3. Test Fixtures

### Temporary Directory Helper

Matching the TypeScript `tmpdir()` pattern:

```go
// test/fixture/fixture.go
package fixture

import (
    "context"
    "os"
    "os/exec"
    "path/filepath"
    "testing"
)

// TmpDir provides a temporary directory for tests with automatic cleanup
type TmpDir struct {
    Path    string
    t       *testing.T
    cleanup []func()
}

type TmpDirOption func(*TmpDir) error

// WithGit initializes a git repository in the temp directory
func WithGit() TmpDirOption {
    return func(td *TmpDir) error {
        cmd := exec.Command("git", "init")
        cmd.Dir = td.Path
        if err := cmd.Run(); err != nil {
            return err
        }

        // Configure git user for commits
        exec.Command("git", "-C", td.Path, "config", "user.email", "test@opencode.ai").Run()
        exec.Command("git", "-C", td.Path, "config", "user.name", "opencode-test").Run()

        return nil
    }
}

// WithFile creates a file with content in the temp directory
func WithFile(path, content string) TmpDirOption {
    return func(td *TmpDir) error {
        fullPath := filepath.Join(td.Path, path)
        if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
            return err
        }
        return os.WriteFile(fullPath, []byte(content), 0644)
    }
}

// WithInit runs custom initialization
func WithInit(fn func(dir string) error) TmpDirOption {
    return func(td *TmpDir) error {
        return fn(td.Path)
    }
}

// NewTmpDir creates a new temporary directory with options
func NewTmpDir(t *testing.T, opts ...TmpDirOption) *TmpDir {
    t.Helper()

    dir, err := os.MkdirTemp("", "opencode-test-*")
    if err != nil {
        t.Fatalf("failed to create temp dir: %v", err)
    }

    td := &TmpDir{
        Path: dir,
        t:    t,
    }

    // Register cleanup
    t.Cleanup(func() {
        for _, fn := range td.cleanup {
            fn()
        }
        os.RemoveAll(dir)
    })

    // Apply options
    for _, opt := range opts {
        if err := opt(td); err != nil {
            t.Fatalf("failed to apply option: %v", err)
        }
    }

    return td
}

// OnCleanup registers a cleanup function
func (td *TmpDir) OnCleanup(fn func()) {
    td.cleanup = append(td.cleanup, fn)
}

// WriteFile writes a file in the temp directory
func (td *TmpDir) WriteFile(path, content string) {
    td.t.Helper()
    fullPath := filepath.Join(td.Path, path)
    if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
        td.t.Fatalf("failed to create dir: %v", err)
    }
    if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
        td.t.Fatalf("failed to write file: %v", err)
    }
}

// ReadFile reads a file from the temp directory
func (td *TmpDir) ReadFile(path string) string {
    td.t.Helper()
    fullPath := filepath.Join(td.Path, path)
    data, err := os.ReadFile(fullPath)
    if err != nil {
        td.t.Fatalf("failed to read file: %v", err)
    }
    return string(data)
}
```

### Usage Example

```go
func TestEditTool_BasicEdit(t *testing.T) {
    tmp := fixture.NewTmpDir(t,
        fixture.WithGit(),
        fixture.WithFile("test.txt", "hello world"),
    )

    tool := NewEditTool(tmp.Path)
    ctx := context.Background()

    result, err := tool.Execute(ctx, EditInput{
        FilePath:  filepath.Join(tmp.Path, "test.txt"),
        OldString: "hello",
        NewString: "goodbye",
    }, testToolContext())

    require.NoError(t, err)
    assert.Equal(t, "goodbye world", tmp.ReadFile("test.txt"))
}
```

---

## 4. Test Categories

### 4.1 Unit Tests

Isolated tests for individual functions and components.

#### Storage Tests
```go
// test/unit/storage_test.go
func TestStorage_PutGet(t *testing.T) { /* ... */ }
func TestStorage_List(t *testing.T) { /* ... */ }
func TestStorage_Delete(t *testing.T) { /* ... */ }
func TestStorage_Scan(t *testing.T) { /* ... */ }
func TestStorage_ConcurrentWrite(t *testing.T) { /* ... */ }
```

#### Event Bus Tests
```go
// test/unit/event_test.go
func TestBus_Subscribe(t *testing.T) { /* ... */ }
func TestBus_Unsubscribe(t *testing.T) { /* ... */ }
func TestBus_PublishAsync(t *testing.T) { /* ... */ }
func TestBus_SubscribeAll(t *testing.T) { /* ... */ }
```

#### Configuration Tests
```go
// test/unit/config_test.go
func TestConfig_LoadDefaults(t *testing.T) { /* ... */ }
func TestConfig_MergeConfigs(t *testing.T) { /* ... */ }
func TestConfig_EnvOverrides(t *testing.T) { /* ... */ }
func TestConfig_JSONCParsing(t *testing.T) { /* ... */ }
```

#### Permission Tests
```go
// test/unit/permission_test.go
func TestWildcard_MatchPattern(t *testing.T) { /* ... */ }
func TestDoomLoop_Detection(t *testing.T) { /* ... */ }
func TestBashParser_SimpleCommand(t *testing.T) { /* ... */ }
func TestBashParser_Pipeline(t *testing.T) { /* ... */ }
func TestBashParser_AndChain(t *testing.T) { /* ... */ }
```

#### Message Tests
```go
// test/unit/message_test.go
func TestMessage_ToModelMessage(t *testing.T) { /* ... */ }
func TestMessage_PartSerialization(t *testing.T) { /* ... */ }
func TestMessage_TokenCounting(t *testing.T) { /* ... */ }
```

#### Provider Transform Tests
```go
// test/unit/transform_test.go
func TestTransform_MaxTokens(t *testing.T) {
    tests := []struct {
        provider string
        model    string
        expected int
    }{
        {"anthropic", "claude-sonnet-4", 64000},
        {"openai", "gpt-4o", 16384},
        {"google", "gemini-2.5-pro", 65536},
    }
    // ...
}
```

### 4.2 Integration Tests

Tests that involve multiple components working together.

#### Session Integration
```go
// test/integration/session_test.go
func TestSession_Create(t *testing.T) {
    tmp := fixture.NewTmpDir(t, fixture.WithGit())
    store := session.NewStore(storage.New(tmp.Path))

    sess, err := store.Create(context.Background(), &types.Session{
        ID:        ulid.Make().String(),
        ProjectID: "test-project",
        Directory: tmp.Path,
        Title:     "Test Session",
    })
    require.NoError(t, err)

    // Verify event was emitted
    // ...
}

func TestSession_MessageFlow(t *testing.T) { /* ... */ }
func TestSession_Fork(t *testing.T) { /* ... */ }
func TestSession_Revert(t *testing.T) { /* ... */ }
```

#### Tool Integration
```go
// test/integration/tool_bash_test.go
func TestBashTool_Execute(t *testing.T) {
    tmp := fixture.NewTmpDir(t)
    tool := NewBashTool(tmp.Path, permission.NewChecker())

    result, err := tool.Execute(context.Background(), BashInput{
        Command:     "echo 'hello'",
        Description: "Echo hello",
    }, testToolContext())

    require.NoError(t, err)
    assert.Contains(t, result.Output, "hello")
}

func TestBashTool_Timeout(t *testing.T) {
    tmp := fixture.NewTmpDir(t)
    tool := NewBashTool(tmp.Path, permission.NewChecker())

    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()

    _, err := tool.Execute(ctx, BashInput{
        Command:     "sleep 10",
        Description: "Sleep",
    }, testToolContext())

    assert.Error(t, err)
}

func TestBashTool_OutputTruncation(t *testing.T) { /* ... */ }
func TestBashTool_ExternalDirBlocking(t *testing.T) { /* ... */ }
```

```go
// test/integration/tool_edit_test.go
func TestEditTool_ExactMatch(t *testing.T) { /* ... */ }
func TestEditTool_FuzzyMatch(t *testing.T) { /* ... */ }
func TestEditTool_ReplaceAll(t *testing.T) { /* ... */ }
func TestEditTool_NotFound(t *testing.T) { /* ... */ }
func TestEditTool_MultipleMatches(t *testing.T) { /* ... */ }
```

```go
// test/integration/tool_read_test.go
func TestReadTool_TextFile(t *testing.T) { /* ... */ }
func TestReadTool_BinaryDetection(t *testing.T) { /* ... */ }
func TestReadTool_ImageFile(t *testing.T) { /* ... */ }
func TestReadTool_Pagination(t *testing.T) { /* ... */ }
func TestReadTool_EnvBlocking(t *testing.T) { /* ... */ }
```

#### Provider Integration
```go
// test/integration/provider_test.go
func TestProvider_Anthropic(t *testing.T) {
    if os.Getenv("ANTHROPIC_API_KEY") == "" {
        t.Skip("ANTHROPIC_API_KEY not set")
    }
    // ...
}

func TestProvider_OpenAI(t *testing.T) {
    if os.Getenv("OPENAI_API_KEY") == "" {
        t.Skip("OPENAI_API_KEY not set")
    }
    // ...
}

func TestProvider_Streaming(t *testing.T) { /* ... */ }
func TestProvider_ToolCalling(t *testing.T) { /* ... */ }
```

#### LSP Integration
```go
// test/integration/lsp_test.go
func TestLSP_Initialize(t *testing.T) {
    server := fixture.StartFakeLSPServer(t)
    defer server.Stop()

    client := lsp.NewClient(server.Stdin, server.Stdout)
    err := client.Initialize(context.Background())
    require.NoError(t, err)
}

func TestLSP_Diagnostics(t *testing.T) { /* ... */ }
func TestLSP_Hover(t *testing.T) { /* ... */ }
```

### 4.3 API Tests

HTTP endpoint tests matching OpenCode's 60+ endpoints.

```go
// test/integration/server_test.go
package integration

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/opencode-ai/opencode-server/internal/server"
)

func TestAPI_SessionCRUD(t *testing.T) {
    srv := server.New(testConfig())
    ts := httptest.NewServer(srv.Handler())
    defer ts.Close()

    // Create session
    resp, err := http.Post(ts.URL+"/session", "application/json",
        strings.NewReader(`{"directory": "/tmp/test"}`))
    require.NoError(t, err)
    require.Equal(t, http.StatusOK, resp.StatusCode)

    var session types.Session
    json.NewDecoder(resp.Body).Decode(&session)

    // Get session
    resp, err = http.Get(ts.URL + "/session/" + session.ID)
    require.NoError(t, err)
    require.Equal(t, http.StatusOK, resp.StatusCode)

    // Delete session
    req, _ := http.NewRequest("DELETE", ts.URL+"/session/"+session.ID, nil)
    resp, err = http.DefaultClient.Do(req)
    require.NoError(t, err)
    require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAPI_FileOperations(t *testing.T) { /* ... */ }
func TestAPI_ConfigEndpoints(t *testing.T) { /* ... */ }
func TestAPI_SSEStreaming(t *testing.T) { /* ... */ }
```

### 4.4 E2E Tests

Full end-to-end tests with actual TUI client.

```go
// test/e2e/client_test.go
func TestE2E_TUIClientCompatibility(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }

    // Start Go server
    srv := startTestServer(t)
    defer srv.Stop()

    // Use OpenCode SDK to interact
    client := sdk.NewClient(srv.URL)

    // Create session
    session, err := client.Session.Create(context.Background(), sdk.SessionCreateParams{
        Directory: t.TempDir(),
    })
    require.NoError(t, err)

    // Send message
    events := make(chan sdk.Event)
    go client.Event.Stream(context.Background(), events)

    _, err = client.Session.Message(context.Background(), session.ID, sdk.MessageParams{
        Content: "Hello, world!",
    })
    require.NoError(t, err)

    // Verify events received
    // ...
}
```

---

## 5. Mock Implementations

### Mock LLM Provider

```go
// test/fixture/mock_provider.go
package fixture

import (
    "context"

    "github.com/opencode-ai/opencode-server/internal/provider"
)

type MockProvider struct {
    Responses []MockResponse
    CallCount int
}

type MockResponse struct {
    Text      string
    ToolCalls []provider.ToolCall
    Error     error
}

func (m *MockProvider) CreateCompletion(ctx context.Context, req provider.CompletionRequest) (*provider.CompletionStream, error) {
    if m.CallCount >= len(m.Responses) {
        return nil, fmt.Errorf("no more mock responses")
    }

    resp := m.Responses[m.CallCount]
    m.CallCount++

    if resp.Error != nil {
        return nil, resp.Error
    }

    return &MockCompletionStream{
        text:      resp.Text,
        toolCalls: resp.ToolCalls,
    }, nil
}

type MockCompletionStream struct {
    text      string
    toolCalls []provider.ToolCall
    position  int
}

func (s *MockCompletionStream) Next() (provider.StreamEvent, error) {
    // Return text deltas, tool calls, finish
}

func (s *MockCompletionStream) Close() error {
    return nil
}
```

### Mock LSP Server

```go
// test/fixture/mock_lsp_server.go
package fixture

import (
    "bufio"
    "encoding/json"
    "io"
    "testing"
)

type MockLSPServer struct {
    Stdin  io.WriteCloser
    Stdout io.ReadCloser
    t      *testing.T
}

func StartMockLSPServer(t *testing.T) *MockLSPServer {
    // Create pipes for communication
    stdinR, stdinW := io.Pipe()
    stdoutR, stdoutW := io.Pipe()

    srv := &MockLSPServer{
        Stdin:  stdinW,
        Stdout: stdoutR,
        t:      t,
    }

    // Start goroutine to handle requests
    go srv.handleRequests(stdinR, stdoutW)

    return srv
}

func (s *MockLSPServer) handleRequests(in io.Reader, out io.Writer) {
    scanner := bufio.NewScanner(in)
    for scanner.Scan() {
        // Parse JSON-RPC request
        // Respond with appropriate mock response
    }
}

func (s *MockLSPServer) Stop() {
    s.Stdin.Close()
    s.Stdout.Close()
}
```

---

## 6. CI/CD Integration

### GitHub Actions Workflow

```yaml
# .github/workflows/test-go.yml
name: Go Tests

on:
  push:
    branches: [main]
    paths:
      - 'go-opencode/**'
  pull_request:
    paths:
      - 'go-opencode/**'

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Install dependencies
        run: go mod download
        working-directory: go-opencode

      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...
        working-directory: go-opencode
        env:
          CI: true

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          file: go-opencode/coverage.out

  integration:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Run integration tests
        run: go test -v -tags=integration ./test/integration/...
        working-directory: go-opencode
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
```

### Makefile

```makefile
# Makefile
.PHONY: test test-unit test-integration test-e2e cover lint

test:
	go test -v -race ./...

test-unit:
	go test -v -race ./test/unit/...

test-integration:
	go test -v -tags=integration ./test/integration/...

test-e2e:
	go test -v -tags=e2e ./test/e2e/...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

bench:
	go test -bench=. -benchmem ./...
```

---

## 7. Test Coverage Targets

| Package | Target Coverage | Priority |
|---------|-----------------|----------|
| `internal/storage` | 90% | P0 |
| `internal/event` | 90% | P0 |
| `internal/permission` | 95% | P0 |
| `internal/tool` | 85% | P0 |
| `internal/session` | 80% | P0 |
| `internal/provider` | 75% | P1 |
| `internal/server` | 70% | P1 |
| `internal/config` | 80% | P1 |
| `internal/lsp` | 60% | P2 |
| `internal/mcp` | 60% | P2 |

---

## 8. Test Porting from TypeScript

### Mapping TypeScript Tests to Go

| TypeScript Test | Go Test |
|-----------------|---------|
| `test/tool/bash.test.ts` | `test/integration/tool_bash_test.go` |
| `test/tool/patch.test.ts` | `test/integration/tool_edit_test.go` |
| `test/session/session.test.ts` | `test/integration/session_test.go` |
| `test/config/config.test.ts` | `test/unit/config_test.go` |
| `test/util/wildcard.test.ts` | `test/unit/wildcard_test.go` |
| `test/provider/transform.test.ts` | `test/unit/transform_test.go` |
| `test/lsp/client.test.ts` | `test/integration/lsp_test.go` |

### Pattern Translations

| Bun/TypeScript | Go |
|----------------|-----|
| `describe("name", () => {...})` | `func TestName(t *testing.T) {...}` with subtests |
| `test("should...", async () => {...})` | `t.Run("should...", func(t *testing.T) {...})` |
| `expect(x).toBe(y)` | `assert.Equal(t, y, x)` |
| `expect(x).toContain(y)` | `assert.Contains(t, x, y)` |
| `await using tmp = await tmpdir()` | `tmp := fixture.NewTmpDir(t)` |
| `beforeEach(() => {...})` | Test setup in each test or `TestMain` |
| `afterEach(() => {...})` | `t.Cleanup(func() {...})` |

---

## 9. Performance Benchmarks

```go
// test/benchmark/storage_bench_test.go
func BenchmarkStorage_Put(b *testing.B) {
    tmp, _ := os.MkdirTemp("", "bench")
    defer os.RemoveAll(tmp)

    s := storage.New(tmp)
    ctx := context.Background()
    data := map[string]string{"key": "value"}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        s.Put(ctx, []string{"bench", fmt.Sprintf("item%d", i)}, data)
    }
}

func BenchmarkBashParser_Parse(b *testing.B) {
    command := `git add . && git commit -m "message" && git push`

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        permission.ParseBashCommand(command)
    }
}

func BenchmarkEditTool_FuzzyMatch(b *testing.B) {
    // Benchmark fuzzy matching algorithm
}
```

---

## 10. Summary

### Test Execution Order

1. **Unit Tests** - Fast, run first, catch basic errors
2. **Integration Tests** - Medium speed, test component interactions
3. **API Tests** - Test HTTP endpoints
4. **E2E Tests** - Slow, run last, verify full system

### Key Principles

1. **Match TypeScript behavior** - Port existing tests, same assertions
2. **Use fixtures** - Consistent temp directory management
3. **Parallel by default** - Go tests run in parallel unless `-parallel 1`
4. **Skip expensive tests** - Use build tags for integration/e2e
5. **Mock external services** - Don't depend on real LLM APIs in unit tests
6. **CI/CD integration** - All tests must pass in GitHub Actions
