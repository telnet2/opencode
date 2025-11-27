# CI Test Infrastructure

This directory contains integration tests for the OpenCode server. Tests can run against real LLM providers or a mock server for CI/CD environments.

## Test Suites

| Directory | Description |
|-----------|-------------|
| `e2e/` | End-to-end tests using the OpenCode SDK |
| `server/` | HTTP API endpoint tests |
| `service/` | Service-level integration tests |
| `comparative/` | Comparative testing between providers |
| `testutil/` | Shared test utilities and mock servers |

## Running Tests

### With Real Providers

Tests require API keys for real LLM providers:

```bash
# Using ARK provider (default)
export ARK_API_KEY="your-api-key"
export ARK_MODEL_ID="your-model-id"
export ARK_BASE_URL="https://ark.cn-beijing.volces.com/api/v3"
go test ./citest/...

# Using OpenAI provider
export TEST_PROVIDER=openai
export OPENAI_API_KEY="your-api-key"
go test ./citest/...
```

### With MockLLM (No API Keys Required)

For CI/CD environments or local development without API keys, use the MockLLM provider:

```bash
TEST_PROVIDER=mockllm go test ./citest/...
```

This starts a local mock server that simulates OpenAI-compatible responses.

## MockLLM Provider

### Overview

MockLLM is a lightweight HTTP server that mimics the OpenAI chat completions API. It provides deterministic responses based on prompt matching, making tests reproducible and fast.

### Features

- **OpenAI-compatible API**: Implements `/v1/chat/completions` and `/chat/completions`
- **Streaming support**: Full SSE streaming response format
- **Tool calls**: Supports function calling for `bash` and `read` tools
- **Request recording**: Captures all requests for verification in tests
- **Deterministic responses**: Same prompt always returns same response

### How It Works

When `TEST_PROVIDER=mockllm` is set:

1. `StartTestServer()` creates a `MockLLMServer` instance
2. The OpenAI provider is configured with the mock server's URL as `BaseURL`
3. All LLM requests are routed to the mock server
4. The mock server returns predefined responses based on prompt content

### Response Mapping

The mock server matches prompts (case-insensitive) and returns appropriate responses:

| Prompt Contains | Response |
|-----------------|----------|
| `hello, world` | `Hello, World!` |
| `2+2` or `2 + 2` | `4` |
| `remember` + `42` | `OK` |
| `what number` + `remember` | `42` |
| `alice` + `name` | `Nice to meet you, Alice` |
| `what` + `name` | `Alice` |
| `hello` | `Hello! How can I help you today?` |
| (default) | `I understand your request. Let me help you with that.` |

### Tool Call Support

When tools are available in the request, the mock server can generate tool calls:

**Bash Tool:**
- Prompt: `run echo hello world` → Tool call: `bash({"command": "echo hello world"})`
- Prompt: `ls '/some/path'` → Tool call: `bash({"command": "ls /some/path"})`

**Read Tool:**
- Prompt: `read /path/to/file.txt` → Tool call: `read({"file_path": "/path/to/file.txt"})`

### Example Usage in Tests

```go
package mytest

import (
    "os"
    "testing"

    "github.com/opencode-ai/opencode/citest/testutil"
)

func TestWithMockLLM(t *testing.T) {
    os.Setenv("TEST_PROVIDER", "mockllm")

    server, err := testutil.StartTestServer()
    if err != nil {
        t.Fatal(err)
    }
    defer server.Stop()

    client := server.Client()
    // ... run tests against client
}
```

### Standalone MockLLM Server

You can also use MockLLM directly in unit tests:

```go
package mytest

import (
    "testing"

    "github.com/opencode-ai/opencode/citest/testutil"
)

func TestDirectMockLLM(t *testing.T) {
    mockLLM := testutil.NewMockLLMServer()
    defer mockLLM.Close()

    // mockLLM.URL() returns the server URL (e.g., "http://127.0.0.1:12345")
    // Use this URL as BaseURL for your LLM provider

    // After requests, verify what was sent:
    requests := mockLLM.GetRequests()
    for _, req := range requests {
        t.Logf("Request: %s %s", req.Method, req.Path)
        t.Logf("Body: %+v", req.Body)
    }
}
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TEST_PROVIDER` | Provider to use: `ark`, `openai`, `mockllm` | `openai` |
| `ARK_API_KEY` | ARK provider API key | - |
| `ARK_MODEL_ID` | ARK model/endpoint ID | - |
| `ARK_BASE_URL` | ARK API base URL | - |
| `OPENAI_API_KEY` | OpenAI API key | - |
| `OPENAI_MODEL_ID` | OpenAI model ID | `gpt-4o-mini` |

### Test Server Options

```go
// Custom working directory
server, _ := testutil.StartTestServer(
    testutil.WithWorkDir("/path/to/workdir"),
)

// Custom .env file
server, _ := testutil.StartTestServer(
    testutil.WithEnvFile("/path/to/.env"),
)
```

## Test Results with MockLLM

Current test pass rates using `TEST_PROVIDER=mockllm`:

| Suite | Passed | Total | Rate |
|-------|--------|-------|------|
| e2e | 14 | 14 | 100% |
| server | 23 | 24 | 95.8% |
| service | 79 | 88 | 89.8% |

Note: Some failures in `server` and `service` suites are related to MCP and client-tools infrastructure, not MockLLM functionality.

## Adding New Mock Responses

To add new response patterns, edit `citest/testutil/mockllm.go`:

```go
func (m *MockLLMServer) generateResponse(prompt string, tools []string) *mockResponse {
    promptLower := strings.ToLower(prompt)

    // Add your pattern here
    switch {
    case strings.Contains(promptLower, "your-pattern"):
        return &mockResponse{content: "Your response"}

    // ... existing patterns
    }
}
```

For tool calls:

```go
if hasYourTool && strings.Contains(promptLower, "trigger-phrase") {
    return &mockResponse{
        content: "I'll execute that for you.",
        toolCalls: []toolCall{
            {
                id:        "call_your_tool_001",
                name:      "your_tool",
                arguments: `{"param": "value"}`,
            },
        },
    }
}
```

## Troubleshooting

### Tests Skip with "ARK environment variables not set"

Make sure `TEST_PROVIDER=mockllm` is set:
```bash
TEST_PROVIDER=mockllm go test ./citest/...
```

### Model Not Found Error

The MockLLM uses `gpt-4o-mini` as the model ID because the OpenAI provider validates model names against a known list. If you see "model not found" errors, ensure the config uses a valid model ID.

### Provider Not Found Error

Check that the provider is being initialized correctly. The MockLLM configures an OpenAI provider with a custom BaseURL pointing to the mock server.

### Debugging Mock Requests

Use `mockLLM.GetRequests()` to inspect what requests were sent:

```go
requests := mockLLM.GetRequests()
for _, req := range requests {
    fmt.Printf("Path: %s\n", req.Path)
    fmt.Printf("Body: %+v\n", req.Body)
}
```
