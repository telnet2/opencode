# Implementation Plan: Symbol Search Endpoint

## Overview

**Endpoint:** `GET /find/symbol`
**Current Status:** Returns 501 Not Implemented
**Priority:** Medium
**Effort:** Low (infrastructure already exists)

---

## 1. Current State Analysis

### TypeScript Reference Implementation

**Location:** `packages/opencode/src/lsp/index.ts:322-331`

```typescript
export async function workspaceSymbol(query: string) {
  return run((client) =>
    client.connection
      .sendRequest("workspace/symbol", { query })
      .then((result: any) => result.filter((x: LSP.Symbol) => kinds.includes(x.kind)))
      .then((result: any) => result.slice(0, 10))
      .catch(() => []),
  ).then((result) => result.flat() as LSP.Symbol[])
}
```

**Server endpoint:** `packages/opencode/src/server/server.ts:1399-1425`
- Accepts `query` query parameter
- Returns `LSP.Symbol[]` (limited to 10 results)
- Filters by specific symbol kinds (Class, Function, Method, Interface, Variable, Constant, Struct, Enum)

### Go Current State

**Handler:** `go-opencode/internal/server/handlers_file.go:247-250`
```go
func (s *Server) searchSymbols(w http.ResponseWriter, r *http.Request) {
    notImplemented(w)
}
```

**LSP Client (ALREADY IMPLEMENTED):** `go-opencode/internal/lsp/operations.go:11-26`
```go
func (c *Client) WorkspaceSymbol(ctx context.Context, query string) ([]Symbol, error) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    var allSymbols []Symbol
    for _, client := range c.clients {
        symbols, err := client.workspaceSymbol(ctx, query)
        if err != nil {
            continue
        }
        allSymbols = append(allSymbols, symbols...)
    }
    return allSymbols, nil
}
```

**The LSP infrastructure is complete.** Only the HTTP handler wiring is missing.

---

## 2. Implementation Tasks

### Task 1: Add LSP Client to Server

**File:** `go-opencode/internal/server/server.go`

Add LSP client to Server struct and initialization:

```go
type Server struct {
    // existing fields...
    lspClient *lsp.Client  // Add this
}

func New(config Config, appConfig *types.Config, ...) *Server {
    // existing initialization...

    // Initialize LSP client if not disabled
    var lspClient *lsp.Client
    if appConfig.LSP == nil || !appConfig.LSP.Disabled {
        lspClient = lsp.NewClient(config.Directory)
    }

    return &Server{
        // existing fields...
        lspClient: lspClient,
    }
}
```

### Task 2: Implement Handler

**File:** `go-opencode/internal/server/handlers_file.go`

Replace the stub with actual implementation:

```go
// Symbol kinds to include in results (matching TypeScript)
var symbolKindsFilter = map[int]bool{
    5:  true, // Class
    6:  true, // Method
    11: true, // Interface
    12: true, // Function
    13: true, // Variable
    14: true, // Constant
    23: true, // Struct
    10: true, // Enum
}

// searchSymbols handles GET /find/symbol
func (s *Server) searchSymbols(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("query")
    if query == "" {
        writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "query parameter required")
        return
    }

    // Check if LSP is available
    if s.lspClient == nil {
        writeJSON(w, http.StatusOK, []any{})
        return
    }

    ctx := r.Context()
    symbols, err := s.lspClient.WorkspaceSymbol(ctx, query)
    if err != nil {
        // Log error but return empty array (matching TS behavior)
        writeJSON(w, http.StatusOK, []any{})
        return
    }

    // Filter by symbol kinds (matching TypeScript)
    filtered := make([]lsp.Symbol, 0, len(symbols))
    for _, s := range symbols {
        if symbolKindsFilter[s.Kind] {
            filtered = append(filtered, s)
        }
    }

    // Limit to 10 results (matching TypeScript)
    if len(filtered) > 10 {
        filtered = filtered[:10]
    }

    writeJSON(w, http.StatusOK, filtered)
}
```

### Task 3: Response Schema Alignment

**Ensure Go LSP Symbol type matches TypeScript:**

TypeScript (`packages/opencode/src/lsp/index.ts:34-46`):
```typescript
interface Symbol {
  name: string
  kind: number
  location: {
    uri: string
    range: {
      start: { line: number, character: number }
      end: { line: number, character: number }
    }
  }
}
```

Go (`go-opencode/internal/lsp/types.go`):
```go
type Symbol struct {
    Name     string         `json:"name"`
    Kind     int            `json:"kind"`
    Location SymbolLocation `json:"location"`
}

type SymbolLocation struct {
    URI   string `json:"uri"`
    Range Range  `json:"range"`
}

type Range struct {
    Start Position `json:"start"`
    End   Position `json:"end"`
}

type Position struct {
    Line      int `json:"line"`
    Character int `json:"character"`
}
```

**Verify JSON tags match** - they already do!

---

## 3. External Configuration

### Required LSP Server Configuration

**File:** `~/.config/opencode/config.json` or project `.opencode/config.json`

```json
{
  "lsp": {
    "typescript": {
      "command": ["typescript-language-server", "--stdio"],
      "extensions": [".ts", ".tsx", ".js", ".jsx"]
    },
    "go": {
      "command": ["gopls"],
      "extensions": [".go"]
    },
    "python": {
      "command": ["pylsp"],
      "extensions": [".py"]
    }
  }
}
```

### Environment Variables

None required - LSP configuration is file-based.

### Dependencies

Go LSP server packages should already be running. The implementation uses existing LSP infrastructure at `go-opencode/internal/lsp/`.

---

## 4. Integration Test Plan

### Test File Location

`go-opencode/citest/service/symbol_test.go`

### Test Cases

```go
package service_test

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var _ = Describe("GET /find/symbol", func() {
    Describe("Query Parameter Validation", func() {
        It("should return 400 when query is missing", func() {
            resp, err := client.Get(ctx, "/find/symbol")
            Expect(err).NotTo(HaveOccurred())
            Expect(resp.StatusCode).To(Equal(400))

            var errResp struct {
                Error struct {
                    Code string `json:"code"`
                } `json:"error"`
            }
            Expect(resp.JSON(&errResp)).To(Succeed())
            Expect(errResp.Error.Code).To(Equal("INVALID_REQUEST"))
        })
    })

    Describe("Basic Functionality", func() {
        It("should return empty array when no symbols match", func() {
            resp, err := client.Get(ctx, "/find/symbol",
                testutil.WithQuery(map[string]string{"query": "nonexistent_xyz_123"}))
            Expect(err).NotTo(HaveOccurred())
            Expect(resp.StatusCode).To(Equal(200))

            var symbols []any
            Expect(resp.JSON(&symbols)).To(Succeed())
            Expect(symbols).To(BeEmpty())
        })

        It("should return empty array when LSP is disabled", func() {
            // Test with LSP disabled in config
            resp, err := client.Get(ctx, "/find/symbol",
                testutil.WithQuery(map[string]string{"query": "test"}))
            Expect(err).NotTo(HaveOccurred())
            Expect(resp.StatusCode).To(Equal(200))
        })
    })

    Describe("Response Format", func() {
        It("should return symbols with correct structure", func() {
            // Create a test file with known symbols
            tempDir.WriteFile("test.go", `
package main

func TestFunction() {}
type TestStruct struct{}
`)

            // Wait for LSP to index
            time.Sleep(500 * time.Millisecond)

            resp, err := client.Get(ctx, "/find/symbol",
                testutil.WithQuery(map[string]string{"query": "Test"}))
            Expect(err).NotTo(HaveOccurred())
            Expect(resp.StatusCode).To(Equal(200))

            var symbols []map[string]any
            Expect(resp.JSON(&symbols)).To(Succeed())

            // Verify structure if symbols found
            if len(symbols) > 0 {
                symbol := symbols[0]
                Expect(symbol).To(HaveKey("name"))
                Expect(symbol).To(HaveKey("kind"))
                Expect(symbol).To(HaveKey("location"))

                location := symbol["location"].(map[string]any)
                Expect(location).To(HaveKey("uri"))
                Expect(location).To(HaveKey("range"))
            }
        })

        It("should limit results to 10 symbols", func() {
            // Create file with many symbols
            var code strings.Builder
            code.WriteString("package main\n\n")
            for i := 0; i < 20; i++ {
                code.WriteString(fmt.Sprintf("func Symbol%d() {}\n", i))
            }
            tempDir.WriteFile("many.go", code.String())

            time.Sleep(500 * time.Millisecond)

            resp, err := client.Get(ctx, "/find/symbol",
                testutil.WithQuery(map[string]string{"query": "Symbol"}))
            Expect(err).NotTo(HaveOccurred())

            var symbols []any
            Expect(resp.JSON(&symbols)).To(Succeed())
            Expect(len(symbols)).To(BeNumerically("<=", 10))
        })
    })

    Describe("Symbol Kind Filtering", func() {
        It("should only return allowed symbol kinds", func() {
            tempDir.WriteFile("kinds.go", `
package main

import "fmt"

const MyConst = 1
var MyVar = 2
type MyStruct struct { Field int }
func MyFunction() { fmt.Println("test") }
type MyInterface interface { Method() }
`)

            time.Sleep(500 * time.Millisecond)

            resp, err := client.Get(ctx, "/find/symbol",
                testutil.WithQuery(map[string]string{"query": "My"}))
            Expect(err).NotTo(HaveOccurred())

            var symbols []map[string]any
            Expect(resp.JSON(&symbols)).To(Succeed())

            // All returned symbols should be of allowed kinds
            allowedKinds := []float64{5, 6, 10, 11, 12, 13, 14, 23}
            for _, sym := range symbols {
                kind := sym["kind"].(float64)
                Expect(allowedKinds).To(ContainElement(kind))
            }
        })
    })
})
```

### Comparative Test

`go-opencode/citest/comparative/symbol_test.go`

```go
func TestSymbolSearch_Comparative(t *testing.T) {
    // Start both servers with same LSP config
    harness := StartComparativeHarness(t)
    defer harness.Stop()

    // Create test file
    testFile := filepath.Join(harness.WorkDir, "test.go")
    os.WriteFile(testFile, []byte(`
package main
func HelloWorld() {}
`), 0644)

    // Wait for LSP indexing
    time.Sleep(time.Second)

    // Query both servers
    resp := harness.Client().Get(ctx, "/find/symbol?query=Hello")

    // Both should succeed
    assert.Equal(t, resp.TS.StatusCode, resp.Go.StatusCode)

    // Compare response structure (not exact values due to timing)
    var tsSymbols, goSymbols []map[string]any
    json.Unmarshal(resp.TS.Body, &tsSymbols)
    json.Unmarshal(resp.Go.Body, &goSymbols)

    // Both should have same structure for returned symbols
    if len(tsSymbols) > 0 && len(goSymbols) > 0 {
        assertSymbolStructure(t, tsSymbols[0])
        assertSymbolStructure(t, goSymbols[0])
    }
}
```

---

## 5. Implementation Checklist

- [ ] Add `lspClient` field to Server struct
- [ ] Initialize LSP client in `server.New()`
- [ ] Implement `searchSymbols` handler with filtering
- [ ] Add query parameter validation
- [ ] Add result limiting (max 10)
- [ ] Add symbol kind filtering
- [ ] Write unit tests
- [ ] Write comparative tests
- [ ] Update OpenAPI spec

---

## 6. Rollout

1. **Week 1:** Implement handler and wire LSP client
2. **Week 1:** Add integration tests
3. **Week 2:** Run comparative tests
4. **Week 2:** Documentation update

---

## References

- TypeScript LSP: `packages/opencode/src/lsp/index.ts`
- TypeScript Server: `packages/opencode/src/server/server.ts:1399-1425`
- Go LSP Client: `go-opencode/internal/lsp/operations.go`
- Go Handler Stub: `go-opencode/internal/server/handlers_file.go:247-250`
