# MockLLM-Based Comparative Testing Architecture

## Executive Summary

This document outlines the architecture for using MockLLM to create deterministic comparative tests between the TypeScript and Go implementations of OpenCode. By using a mock LLM server, we can achieve:

1. **Deterministic responses** - Same input always produces same output
2. **Fast execution** - No network latency or rate limits
3. **Cost-free testing** - No API usage charges
4. **Controlled tool calls** - Predictable tool use patterns

---

## 1. MockLLM Overview

### 1.1 What is MockLLM?

[MockLLM](https://github.com/StacklokLabs/mockllm) is a Python-based mock server that mimics OpenAI and Anthropic API formats. It provides:

- **OpenAI-compatible endpoint**: `POST /v1/chat/completions`
- **Anthropic-compatible endpoint**: `POST /v1/messages`
- **Streaming support**: Server-sent events for both formats
- **Response configuration**: YAML-based predefined responses
- **Fuzzy matching**: Prompts matched by keyword presence

### 1.2 Limitations of External MockLLM

While the external MockLLM project is useful, it has limitations for our use case:

1. **Python dependency** - Requires Python runtime
2. **Limited Go integration** - No native Go bindings
3. **Configuration complexity** - YAML config must be written to disk
4. **Process management** - Must spawn external process

### 1.3 Our Approach: Native Go Mock Server

Instead of using the Python MockLLM directly, we've implemented a **native Go mock server** that provides the same functionality:

```go
// Package comparative provides MockLLM server implementation
type MockLLMServer struct {
    server    *httptest.Server
    config    *MockLLMConfig
    requests  []MockRequest      // Record all requests
    streaming bool
}
```

**Benefits:**
- No external dependencies
- In-process testing (faster)
- Direct request inspection
- Configurable via Go code
- Full streaming support

---

## 2. Architecture Overview

### 2.1 Component Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Comparative Testing Harness                       │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────────┐    ┌──────────────────┐    ┌──────────────┐  │
│  │   Test Suite     │───▶│   DualClient     │───▶│  Comparator  │  │
│  │   (Ginkgo)       │    │  (Parallel Req)  │    │  (JSON Diff) │  │
│  └──────────────────┘    └──────────────────┘    └──────────────┘  │
│           │                      │                       │          │
│           │              ┌───────┴───────┐               │          │
│           │              │               │               │          │
│           ▼              ▼               ▼               ▼          │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                      MockLLM Server                           │  │
│  │  ┌────────────────────┐  ┌────────────────────────────────┐  │  │
│  │  │ /v1/chat/completions│  │  /v1/messages (Anthropic)     │  │  │
│  │  │     (OpenAI)        │  │                               │  │  │
│  │  └────────────────────┘  └────────────────────────────────┘  │  │
│  │                                                               │  │
│  │  ┌────────────────────────────────────────────────────────┐  │  │
│  │  │              Response Configuration                     │  │  │
│  │  │   "hello" → "Hello! How can I help you?"               │  │  │
│  │  │   "read file" → tool_call(read_file, path)             │  │  │
│  │  │   "create" → tool_call(write_file, path, content)      │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                              │                                      │
│              ┌───────────────┴───────────────┐                     │
│              │                               │                      │
│              ▼                               ▼                      │
│  ┌──────────────────────┐    ┌──────────────────────┐              │
│  │   TypeScript Server  │    │     Go Server        │              │
│  │  (packages/opencode) │    │   (go-opencode)      │              │
│  │                       │    │                      │              │
│  │  OPENAI_BASE_URL=     │    │  OPENAI_BASE_URL=    │              │
│  │  http://mock:8888/v1  │    │  http://mock:8888/v1 │              │
│  └──────────────────────┘    └──────────────────────┘              │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 Data Flow

```
1. Test Case sends request to DualClient
2. DualClient sends parallel requests to TS and Go servers
3. Both servers call MockLLM for LLM responses
4. MockLLM returns deterministic responses based on prompt matching
5. Servers process responses and return to DualClient
6. DualClient collects responses
7. Comparator diffs the responses
8. Test Case asserts on differences
```

---

## 3. Mock Configuration

### 3.1 Response Configuration

```go
config := &MockLLMConfig{
    Responses: map[string]MockResponse{
        // Simple text responses
        "hello": {
            Content: "Hello! How can I help you today?",
        },
        "what is 2+2": {
            Content: "4",
        },

        // Tool call responses
        "read file": {
            Content: "I'll read that file for you.",
            ToolCalls: []MockToolCall{
                {
                    ID:   "call_read_123",
                    Type: "function",
                    Function: MockFunctionCall{
                        Name:      "read_file",
                        Arguments: `{"path": "/test.txt"}`,
                    },
                },
            },
        },

        // Multi-tool responses
        "create and run": {
            Content: "I'll create the file and run the command.",
            ToolCalls: []MockToolCall{
                {
                    ID:   "call_write",
                    Function: MockFunctionCall{
                        Name:      "write_file",
                        Arguments: `{"path": "/script.sh", "content": "#!/bin/bash\necho hello"}`,
                    },
                },
                {
                    ID:   "call_bash",
                    Function: MockFunctionCall{
                        Name:      "bash",
                        Arguments: `{"command": "chmod +x /script.sh && /script.sh"}`,
                    },
                },
            },
        },
    },

    Defaults: MockDefaults{
        Fallback: "I understand your request and will help you with that.",
    },

    Settings: MockSettings{
        LagMS:           0,      // No artificial lag for tests
        EnableStreaming: true,  // Enable SSE streaming
    },
}
```

### 3.2 Response Matching Algorithm

```go
func (m *MockLLMServer) findResponse(prompt string) *MockResponse {
    prompt = strings.ToLower(strings.TrimSpace(prompt))

    // Check for keyword matches (order matters - first match wins)
    for key, resp := range m.config.Responses {
        if strings.Contains(prompt, strings.ToLower(key)) {
            return &resp
        }
    }

    // Return fallback for unmatched prompts
    return &MockResponse{
        Content: m.config.Defaults.Fallback,
    }
}
```

---

## 4. Test Scenarios

### 4.1 Basic Message Flow

```go
var _ = Describe("Message Parity", func() {
    var harness *ComparativeHarness

    BeforeEach(func() {
        harness, _ = NewComparativeHarness(&HarnessConfig{
            MockResponses: map[string]MockResponse{
                "hello": {Content: "Hello! How can I help?"},
            },
        })
        harness.Start(ctx)
    })

    It("should return identical message structure", func() {
        // Create sessions on both servers
        tsSession := createSession(harness.TSServer.URL())
        goSession := createSession(harness.GoServer.URL())

        // Send identical messages
        tsResp := sendMessage(harness.TSServer.URL(), tsSession.ID, "hello")
        goResp := sendMessage(harness.GoServer.URL(), goSession.ID, "hello")

        // Compare responses
        diffs, _ := CompareJSON(tsResp, goResp, DefaultTolerances())
        criticalDiffs := FilterBySeverity(diffs, SeverityCritical)

        Expect(criticalDiffs).To(BeEmpty())
    })
})
```

### 4.2 Tool Execution Flow

```go
var _ = Describe("Tool Execution Parity", func() {
    It("should execute tools identically", func() {
        harness, _ := NewComparativeHarness(&HarnessConfig{
            MockResponses: map[string]MockResponse{
                "list files": {
                    Content: "I'll list the files in the directory.",
                    ToolCalls: []MockToolCall{
                        {
                            ID: "call_bash",
                            Function: MockFunctionCall{
                                Name:      "bash",
                                Arguments: `{"command": "ls -la"}`,
                            },
                        },
                    },
                },
            },
        })

        // Execute and compare tool results
        tsResult := executeWithTools(harness.TSServer.URL(), "list files")
        goResult := executeWithTools(harness.GoServer.URL(), "list files")

        // Tool call sequences should match
        Expect(tsResult.ToolCalls).To(HaveLen(len(goResult.ToolCalls)))

        // Tool results should match (same command, same output)
        Expect(tsResult.ToolResults[0].Output).To(Equal(goResult.ToolResults[0].Output))
    })
})
```

### 4.3 Streaming Parity

```go
var _ = Describe("Streaming Parity", func() {
    It("should stream chunks in same order", func() {
        // Collect streaming chunks from both servers
        tsChunks := collectStreamingChunks(harness.TSServer.URL(), sessionID, "hello")
        goChunks := collectStreamingChunks(harness.GoServer.URL(), sessionID, "hello")

        // Both should produce chunks
        Expect(tsChunks).NotTo(BeEmpty())
        Expect(goChunks).NotTo(BeEmpty())

        // Final content should match
        tsFinal := concatenateChunks(tsChunks)
        goFinal := concatenateChunks(goChunks)
        Expect(tsFinal).To(Equal(goFinal))
    })
})
```

---

## 5. Existing Mock Utilities in Codebase

### 5.1 Found Mock Implementations

| Location | Type | Purpose |
|----------|------|---------|
| `packages/sdk/python/tests/test_wrapper.py` | httpx.MockTransport | HTTP response mocking for Python SDK |
| `packages/sdk/go/scripts/mock` | Prism CLI | OpenAPI-based mock server generation |
| `packages/opencode/test/fixture/lsp/fake-lsp-server.js` | Node.js | Fake LSP server for testing |
| `packages/memsh-cli/src/client/client.test.ts` | Bun mock() | Fetch mocking for client tests |

### 5.2 Integration Opportunities

1. **Prism for API Contract Testing**
   - Use Prism to validate both servers against OpenAPI spec
   - Ensures API contract compliance

2. **Native Mock Server for LLM**
   - Our Go MockLLM server for deterministic LLM responses
   - No external dependencies

3. **Shared Test Fixtures**
   - Reuse test data between TS and Go tests
   - Common response configurations

---

## 6. Environment Configuration

### 6.1 Environment Variables for Mock Mode

```bash
# Point both servers at MockLLM
export OPENAI_API_KEY="mock-key-not-used"
export OPENAI_BASE_URL="http://localhost:8888/v1"

# Or for Anthropic
export ANTHROPIC_API_KEY="mock-key-not-used"
export ANTHROPIC_BASE_URL="http://localhost:8888/v1"

# Server-specific directories (isolated state)
export OPENCODE_STATE_DIR="/tmp/compare/ts-state"    # For TS
export OPENCODE_STATE_DIR="/tmp/compare/go-state"    # For Go

# Same working directory (shared test files)
export OPENCODE_DIRECTORY="/tmp/compare/workspace"
```

### 6.2 Config File Override

For TypeScript server, create `opencode.json`:
```json
{
  "model": "mock/gpt-4",
  "provider": {
    "mock": {
      "npm": "@ai-sdk/openai-compatible",
      "api": "http://localhost:8888/v1",
      "models": {
        "gpt-4": {
          "name": "Mock GPT-4",
          "tool_call": true
        }
      }
    }
  }
}
```

---

## 7. Implementation Roadmap

### Phase 1: Core Infrastructure (Done)
- [x] MockLLM server implementation in Go
- [x] OpenAI-compatible endpoint
- [x] Anthropic-compatible endpoint
- [x] Streaming support
- [x] Tool call responses
- [x] Request recording

### Phase 2: Comparative Harness (In Progress)
- [x] DualClient for parallel requests
- [x] JSON comparison with tolerances
- [x] Severity classification
- [ ] Server lifecycle management
- [ ] Shared workspace setup

### Phase 3: Test Coverage
- [ ] Session CRUD parity tests
- [ ] Message flow parity tests
- [ ] Tool execution parity tests
- [ ] Streaming parity tests
- [ ] Error handling parity tests

### Phase 4: CI Integration
- [ ] GitHub Actions workflow
- [ ] Test matrix (Go versions, providers)
- [ ] Report generation
- [ ] PR commenting

---

## 8. Running the Tests

### 8.1 Quick Start

```bash
# Run MockLLM comparative tests
cd go-opencode
go test -v ./citest/comparative/... -count=1

# Run with TS server comparison (requires bun)
COMPARE_WITH_TS=true go test -v ./citest/comparative/...
```

### 8.2 Makefile Targets

```makefile
# In go-opencode/Makefile

.PHONY: test-comparative test-comparative-full

# Run Go-only comparative tests (fast)
test-comparative:
	go test -v ./citest/comparative/... -count=1

# Run full comparative tests with TS server
test-comparative-full:
	COMPARE_WITH_TS=true go test -v ./citest/comparative/... -timeout=5m
```

---

## 9. Benefits of This Approach

### 9.1 Determinism
- Same prompt always yields same response
- No flakiness from LLM variability
- Reproducible test failures

### 9.2 Speed
- In-process mock server (no network)
- Sub-millisecond response times
- Parallel test execution

### 9.3 Cost
- Zero API costs
- Unlimited test runs
- No rate limiting

### 9.4 Control
- Precise tool call sequences
- Error injection capability
- Timing simulation

### 9.5 Debugging
- Request recording
- Response inspection
- Diff reporting

---

## 10. Future Enhancements

### 10.1 Response Recording Mode
Record real LLM responses and replay them:
```go
// Record mode - call real LLM and save responses
mockServer.SetMode(RecordMode)
mockServer.SetBackend("https://api.openai.com/v1")

// Replay mode - use saved responses
mockServer.SetMode(ReplayMode)
mockServer.LoadResponses("./fixtures/recorded.json")
```

### 10.2 Fuzzy Response Matching
More sophisticated prompt matching:
```go
// Semantic similarity matching
mockServer.SetMatcher(SemanticMatcher{
    Threshold: 0.8,
    Embeddings: "./embeddings.bin",
})
```

### 10.3 Chaos Testing
Inject failures and delays:
```go
mockServer.SetChaos(ChaosConfig{
    ErrorRate:     0.1,  // 10% random errors
    MaxLatencyMS:  5000, // Up to 5s delay
    TokenDropRate: 0.05, // 5% dropped tokens in streaming
})
```

---

## Appendix A: MockLLM API Reference

### OpenAI Endpoint

**Request:**
```http
POST /v1/chat/completions
Content-Type: application/json

{
  "model": "gpt-4",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "stream": false
}
```

**Response:**
```json
{
  "id": "chatcmpl-mock-123",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "mock-gpt-4",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "Hello! How can I help you?"
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 100,
    "completion_tokens": 50,
    "total_tokens": 150
  }
}
```

### Anthropic Endpoint

**Request:**
```http
POST /v1/messages
Content-Type: application/json
X-API-Key: mock-key
anthropic-version: 2023-06-01

{
  "model": "claude-3-opus",
  "max_tokens": 1024,
  "messages": [
    {"role": "user", "content": "Hello"}
  ]
}
```

**Response:**
```json
{
  "id": "msg_mock_123",
  "type": "message",
  "role": "assistant",
  "model": "mock-claude-3",
  "content": [{
    "type": "text",
    "text": "Hello! How can I help you?"
  }],
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 100,
    "output_tokens": 50
  }
}
```
