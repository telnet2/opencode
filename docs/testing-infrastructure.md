# Testing Infrastructure & Strategies Analysis

**Date:** November 2024
**Status:** Comprehensive Analysis
**Scope:** OpenCode Repository Testing Infrastructure

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Testing Infrastructure Overview](#testing-infrastructure-overview)
3. [Test Categories by Package](#test-categories-by-package)
4. [Testing Frameworks & Libraries](#testing-frameworks--libraries)
5. [Testing Patterns & Strategies](#testing-patterns--strategies)
6. [CI/CD Pipeline](#cicd-pipeline)
7. [Mock Infrastructure](#mock-infrastructure)
8. [Critical Gap Analysis: Real Model Testing](#critical-gap-analysis-real-model-testing)
9. [Recommendations](#recommendations)

---

## Executive Summary

The OpenCode repository employs a **comprehensive but purely synthetic testing strategy**. The testing infrastructure spans multiple languages (TypeScript, Go, Python) with appropriate unit and integration tests. However, there is a **critical gap**: the codebase does **not test against real AI models** and does **not include agent performance evaluation**.

### Key Findings

| Aspect | Status | Notes |
|--------|--------|-------|
| Unit Testing | ✅ Present | Good coverage across all SDKs |
| Integration Testing | ✅ Present | Server startup and API validation |
| Mock-based Testing | ✅ Comprehensive | HTTP transport mocking, LSP fakes |
| Real Model Testing | ❌ **Absent** | No tests against live AI APIs |
| Agent Performance Evaluation | ❌ **Absent** | No benchmarks, evals, or quality metrics |
| End-to-End AI Workflow Tests | ❌ **Absent** | No complete agent task execution tests |
| CI/CD Pipeline | ✅ Present | GitHub Actions with Turbo orchestration |

---

## Testing Infrastructure Overview

### Repository Structure

```
opencode/
├── packages/
│   ├── opencode/test/           # Core TypeScript tests (22 files)
│   │   ├── config/              # Configuration tests
│   │   ├── file/                # File handling tests
│   │   ├── fixture/             # Test utilities
│   │   ├── ide/                 # IDE integration tests
│   │   ├── lsp/                 # LSP client tests
│   │   ├── patch/               # Patch system tests
│   │   ├── project/             # Project management tests
│   │   ├── provider/            # Provider transform tests
│   │   ├── session/             # Session management tests
│   │   ├── snapshot/            # Snapshot system tests
│   │   ├── tool/                # Tool execution tests
│   │   └── util/                # Utility function tests
│   ├── sdk/
│   │   ├── go/                  # Go SDK tests (18 files)
│   │   │   ├── *_test.go        # SDK entity tests
│   │   │   └── internal/        # API utilities tests
│   │   └── python/tests/        # Python SDK tests (2 files)
│   │       ├── test_wrapper.py  # Unit tests with mock transport
│   │       └── test_integration.py  # Integration tests
└── go-memsh/                    # In-memory shell tests (9 files)
    └── *_test.go                # Shell parsing and execution tests
```

### Test File Count by Package

| Package | Test Files | Test Type |
|---------|------------|-----------|
| `packages/opencode/test` | 22 | TypeScript (Bun) |
| `packages/sdk/go` | 18 | Go (native) |
| `packages/sdk/python/tests` | 2 | Python (pytest) |
| `go-memsh` | 9 | Go (native) |
| **Total** | **51** | - |

---

## Test Categories by Package

### TypeScript/Bun Tests (`packages/opencode/test/`)

These tests focus on the core application logic and infrastructure:

| Directory | Purpose | Test Focus |
|-----------|---------|------------|
| `config/` | Configuration management | YAML/JSON parsing, model config, agent colors |
| `file/` | File operations | .gitignore handling, file filtering |
| `fixture/` | Test utilities | Temporary directory creation, git initialization |
| `ide/` | IDE integration | IDE detection and integration |
| `lsp/` | Language Server Protocol | LSP client communication |
| `patch/` | Code patching | Diff/patch application |
| `project/` | Project management | Project initialization, directory handling |
| `provider/` | AI provider transforms | Token limits, provider-specific handling |
| `session/` | Session management | Session creation, events, retry logic |
| `snapshot/` | Git snapshot system | File tracking, revert, diff operations |
| `tool/` | Tool execution | Bash tool execution |
| `util/` | Utility functions | IIFE, lazy loading, timeouts, wildcards |

### Go SDK Tests (`packages/sdk/go/`)

Auto-generated tests from OpenAPI specification (Stainless):

| File | Coverage |
|------|----------|
| `agent_test.go` | Agent list API |
| `client_test.go` | HTTP client, retries, User-Agent, context handling |
| `session_test.go` | Session CRUD operations |
| `config_test.go` | Configuration retrieval |
| `file_test.go` | File status API |
| `tui_test.go` | TUI SSE events |
| `usage_test.go` | Usage tracking |
| `internal/apiform/` | Form encoding |
| `internal/apijson/` | JSON serialization |
| `internal/apiquery/` | Query string building |

### Python SDK Tests (`packages/sdk/python/tests/`)

| File | Type | Description |
|------|------|-------------|
| `test_wrapper.py` | Unit | Mock HTTP transport testing |
| `test_integration.py` | Integration | Live server subprocess testing |

### go-memsh Tests

Shell implementation tests:

| File | Coverage |
|------|----------|
| `sh_test.go` | Script execution |
| `shell_test.go` | Shell state management |
| `parser_test.go` | Command parsing |
| `posix_flags_test.go` | POSIX flag handling |
| `procsubst_test.go` | Process substitution |
| `httputils_test.go` | HTTP utilities |
| `textutils_test.go` | Text utilities |
| `import_export_test.go` | Environment handling |

---

## Testing Frameworks & Libraries

### TypeScript (Bun)

```typescript
import { describe, expect, test } from "bun:test"
```

- **Framework:** Bun's native test framework
- **Test Runner:** `bun test`
- **Assertions:** Built-in `expect` API
- **Features:** Async/await support, snapshot testing

### Go

```go
import "testing"
```

- **Framework:** Standard `testing` package
- **Test Runner:** `go test` or `./scripts/test`
- **Mock Server:** Prism (OpenAPI mock server)
- **Dependencies:**
  - `tidwall/gjson` - JSON parsing
  - `tidwall/sjson` - JSON manipulation
  - `spf13/afero` - Virtual filesystem

### Python

```python
import pytest
import httpx
```

- **Framework:** pytest with pytest-asyncio
- **Mock Transport:** `httpx.MockTransport`
- **SSE Testing:** `sseclient-py`
- **Test Runner:** `uv run --project packages/sdk/python pytest -q`

---

## Testing Patterns & Strategies

### 1. Mock-Based Unit Testing

**Pattern:** Replace HTTP transport with mock handlers that return predefined responses.

**Go Example (`client_test.go:19-45`):**
```go
type closureTransport struct {
    fn func(req *http.Request) (*http.Response, error)
}

func TestUserAgentHeader(t *testing.T) {
    client := opencode.NewClient(
        option.WithHTTPClient(&http.Client{
            Transport: &closureTransport{
                fn: func(req *http.Request) (*http.Response, error) {
                    userAgent = req.Header.Get("User-Agent")
                    return &http.Response{StatusCode: http.StatusOK}, nil
                },
            },
        }),
    )
    // ...
}
```

**Python Example (`test_wrapper.py:29-54`):**
```python
def test_get_path_with_mock_transport() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={...})

    transport = httpx.MockTransport(handler)
    w = OpenCodeClient(base_url="http://test")
    client = httpx.Client(base_url="http://test", transport=transport)
    w.client.set_httpx_client(client)
```

### 2. Temporary Directory Isolation

**Pattern:** Create isolated temporary directories for each test to prevent test pollution.

**TypeScript Example (`fixture/fixture.ts`):**
```typescript
async function tmpdir<T>(options?: TmpDirOptions<T>) {
    // Creates temporary directories
    // Supports git initialization
    // Automatic cleanup via Symbol.asyncDispose
}
```

### 3. Fake Server Implementation

**Pattern:** Implement minimal fake servers for protocol testing.

**Example:** `test/fixture/lsp/fake-lsp-server.js` - Minimal JSON-RPC LSP server for testing client communication.

### 4. Integration Testing with Real Subprocesses

**Pattern:** Start the actual server as a subprocess for integration tests.

**Python Example (`test_integration.py:16-93`):**
```python
def test_integration_live_server_endpoints() -> None:
    cmd = ["bun", "run", "./src/index.ts", "serve", "--port", "0"]
    proc = subprocess.Popen(cmd, ...)

    # Wait for server URL
    # Test actual API endpoints
    # Test SSE streaming

    proc.terminate()
```

### 5. Event Bus Testing

**Pattern:** Verify event emission order and payloads.

**TypeScript Example (`session/session.test.ts`):**
```typescript
test("session.started event should be emitted before session.updated", async () => {
    const events: string[] = []
    Bus.subscribe(Session.Event.Created, () => events.push("started"))
    Bus.subscribe(Session.Event.Updated, () => events.push("updated"))

    await Session.create({})

    expect(events.indexOf("started")).toBeLessThan(events.indexOf("updated"))
})
```

### 6. Retry Logic Testing

**Pattern:** Test exponential backoff and retry-after header handling.

**TypeScript Example (`session/retry.test.ts`):**
```typescript
test("caps delay at 30 seconds when headers missing", () => {
    const error = apiError()
    const delays = Array.from({ length: 10 }, (_, i) => SessionRetry.delay(error, i + 1))
    expect(delays).toStrictEqual([2000, 4000, 8000, 16000, 30000, 30000, ...])
})
```

---

## CI/CD Pipeline

### GitHub Actions Workflow (`.github/workflows/test.yml`)

```yaml
name: test
on:
  push:
    branches-ignore: [production]
  pull_request:
    branches-ignore: [production]
  workflow_dispatch:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ./.github/actions/setup-bun
      - run: |
          git config --global user.email "bot@opencode.ai"
          git config --global user.name "opencode"
          bun turbo typecheck
          bun turbo test
        env:
          CI: true
      - name: Check SDK is up to date
        run: |
          bun ./packages/sdk/js/script/build.ts
          git diff --exit-code packages/sdk/js/src/gen packages/sdk/js/dist
```

### Turbo Configuration (`turbo.json`)

```json
{
  "tasks": {
    "typecheck": {},
    "build": {
      "dependsOn": ["^build"],
      "outputs": ["dist/**"]
    },
    "opencode#test": {
      "dependsOn": ["^build"],
      "outputs": []
    }
  }
}
```

### Pipeline Stages

1. **Checkout** - Clone repository
2. **Setup Bun** - Install Bun runtime (v1.3.3+)
3. **Typecheck** - Run `bun turbo typecheck` (TypeScript validation)
4. **Test** - Run `bun turbo test` (all test suites)
5. **SDK Verification** - Ensure generated SDK is up-to-date

---

## Mock Infrastructure

### Prism Mock Server (Go SDK)

The Go SDK tests rely on Prism to mock the OpenAPI specification:

```bash
npx prism mock path/to/openapi.yml
```

**Configuration:**
- Default URL: `http://localhost:4010`
- Override: `TEST_API_BASE_URL` environment variable
- Skip tests: `SKIP_MOCK_TESTS=true`

### HTTP Transport Mocking

| Language | Library | Pattern |
|----------|---------|---------|
| Go | `http.RoundTripper` | Custom `closureTransport` struct |
| Python | `httpx.MockTransport` | Function-based request handler |
| TypeScript | Bun mocking | Direct module mocking (limited) |

### Virtual Filesystem (go-memsh)

```go
import "github.com/spf13/afero"

fs := afero.NewMemMapFs()
```

Used for testing file operations without touching the real filesystem.

---

## Critical Gap Analysis: Real Model Testing

### What's Missing

The OpenCode repository has **no testing against real AI models** and **no agent performance evaluation**. This is a significant gap for an AI-powered coding assistant.

#### 1. No Real Model API Calls

The tests mock all HTTP interactions. There are **zero tests** that:
- Make actual API calls to OpenAI, Anthropic, or other providers
- Validate model response parsing with real responses
- Test streaming behavior with real SSE from model providers

#### 2. No Agent Performance Evaluation

There are **no evaluation frameworks** that measure:
- Task completion accuracy
- Code quality of generated code
- Response latency and throughput
- Cost per task
- Tool selection accuracy
- Multi-turn conversation quality

#### 3. No End-to-End Agent Workflow Tests

Missing test scenarios:
- Complete task execution (prompt → tool calls → result)
- Error recovery in multi-step tasks
- Context window management with real conversations
- Model-specific behavior differences

### Evidence from Codebase

1. **Test search for "eval", "benchmark", "performance":**
   - Returns only documentation files
   - No actual evaluation code

2. **Test search for API key handling:**
   - Keys are only mentioned in environment type definitions
   - No test infrastructure for authenticated API calls

3. **Provider transform tests (`provider/transform.test.ts`):**
   - Only tests token limit calculations
   - No actual model interaction

4. **Session tests (`session/session.test.ts`):**
   - Tests event emission
   - Does not test actual AI message generation

### Comparison with Industry Standards

| Feature | OpenCode | Claude Code | GitHub Copilot |
|---------|----------|-------------|----------------|
| Unit Tests | ✅ | ✅ | ✅ |
| Integration Tests | ✅ | ✅ | ✅ |
| Real Model Tests | ❌ | Unknown | Unknown |
| Eval Benchmarks | ❌ | Yes (SWE-bench) | Yes (HumanEval) |
| Performance Metrics | ❌ | Yes | Yes |

---

## Recommendations

### Short-term Improvements

1. **Add Real API Integration Tests (Optional)**
   ```typescript
   // Mark as skip by default, run manually or in special CI jobs
   test.skip("real model response parsing", async () => {
       const client = new OpenAI({ apiKey: process.env.OPENAI_API_KEY })
       const response = await client.chat.completions.create({...})
       // Validate response structure
   })
   ```

2. **Add Response Schema Validation**
   - Validate that mock responses match real API schemas
   - Use recorded real responses as fixtures

### Medium-term Improvements

1. **Implement Evaluation Framework**
   ```typescript
   interface EvalResult {
       taskId: string
       success: boolean
       executionTime: number
       tokenUsage: { input: number, output: number }
       toolCalls: number
       errorRecoveries: number
   }

   async function runEval(task: EvalTask): Promise<EvalResult> {
       // Execute task with real model
       // Measure success and metrics
   }
   ```

2. **Create Benchmark Suite**
   - Code completion accuracy
   - Bug fixing success rate
   - Refactoring quality
   - Documentation generation

3. **Add Performance Regression Tests**
   - Track response latency over releases
   - Monitor token usage efficiency
   - Alert on cost increases

### Long-term Improvements

1. **Continuous Evaluation Pipeline**
   - Nightly eval runs against test repos
   - Automated quality tracking dashboard
   - A/B testing infrastructure for prompt changes

2. **Model Comparison Framework**
   - Compare GPT-4 vs Claude vs local models
   - Identify optimal model for each task type
   - Cost-performance optimization

3. **User Simulation Testing**
   - Synthetic user sessions
   - Common workflow coverage
   - Edge case discovery

---

## Conclusion

The OpenCode repository has a **solid foundation for traditional software testing** but **lacks AI-specific testing infrastructure**. The current tests validate:

- ✅ API client behavior
- ✅ Configuration handling
- ✅ Session management
- ✅ Tool execution
- ✅ File operations

But critically **do not validate**:

- ❌ AI model integration quality
- ❌ Agent task completion accuracy
- ❌ Response quality and correctness
- ❌ Performance under real conditions
- ❌ Cost efficiency

For an AI-powered coding assistant, **real model testing and evaluation benchmarks are essential** to ensure the product delivers value to users and to prevent regressions in AI behavior.
