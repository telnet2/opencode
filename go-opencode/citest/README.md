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

## YAML Configuration

MockLLM supports YAML-based configuration for defining response scenarios. This makes it easy to customize behavior without modifying Go code.

### Configuration File Location

- Default config: `citest/config/mockllm.yaml`
- Custom config can be loaded programmatically

### Configuration Schema

```yaml
settings:
  lag_ms: 0              # Artificial delay before responding
  enable_streaming: true # Support SSE streaming
  chunk_delay_ms: 5      # Delay between stream chunks

defaults:
  fallback: "Default response when no rules match"

responses:
  - name: rule-name        # Optional identifier
    match:                 # Match criteria (pick one)
      contains: "text"     # Case-insensitive substring
      contains_all: [...]  # All must be present
      contains_any: [...]  # Any one must be present
      exact: "text"        # Exact match
      regex: "pattern"     # Regex pattern (future)
    response: "Text to return"
    priority: 10           # Higher priority wins

tool_rules:
  - name: tool-rule-name
    match:
      contains: "trigger phrase"
    tool: bash             # Tool name (must be in request)
    tool_call:
      id: "call_id"        # Optional, auto-generated if empty
      arguments:
        command: "echo hi" # Tool-specific arguments
    response: "Optional text alongside tool call"
    priority: 10
```

### Loading Custom Configuration

```go
// From file
mockLLM, err := testutil.NewMockLLMServerFromFile("path/to/config.yaml")

// From directory (looks for mockllm.yaml or mockllm.yml)
config, err := testutil.LoadMockLLMConfigFromDir("path/to/dir")
mockLLM := testutil.NewMockLLMServerWithConfig(config)

// Runtime configuration
mockLLM := testutil.NewMockLLMServer()
mockLLM.AddResponse(testutil.ResponseRule{
    Name:     "custom-rule",
    Match:    testutil.MatchConfig{Contains: "custom prompt"},
    Response: "Custom response",
    Priority: 100,
})
mockLLM.AddToolRule(testutil.ToolRule{
    Name:  "custom-tool-rule",
    Match: testutil.MatchConfig{Contains: "run custom"},
    Tool:  "bash",
    ToolCall: testutil.ToolCallConfig{
        Arguments: map[string]string{"command": "echo custom"},
    },
    Priority: 100,
})
```

### Example YAML Configuration

```yaml
settings:
  lag_ms: 0
  enable_streaming: true
  chunk_delay_ms: 5

defaults:
  fallback: "I understand your request."

responses:
  - name: greeting
    match:
      contains: "hello"
    response: "Hello! How can I help?"
    priority: 1

  - name: math
    match:
      contains_any:
        - "2+2"
        - "two plus two"
    response: "4"
    priority: 10

  - name: context-aware
    match:
      contains_all:
        - "remember"
        - "number"
    response: "42"
    priority: 5

tool_rules:
  - name: list-files
    match:
      contains: "list files"
    tool: bash
    tool_call:
      arguments:
        command: "ls -la"
    response: "Listing files..."
    priority: 10
```

## Adding New Mock Responses

### Option 1: YAML Configuration (Recommended)

Edit `citest/config/mockllm.yaml` to add new response patterns:

```yaml
responses:
  - name: your-new-rule
    match:
      contains: "your-pattern"
    response: "Your response"
    priority: 10
```

For tool calls:

```yaml
tool_rules:
  - name: your-tool-rule
    match:
      contains: "trigger-phrase"
    tool: your_tool
    tool_call:
      arguments:
        param: "value"
    response: "I'll execute that for you."
    priority: 10
```

### Option 2: Runtime Configuration

Add rules programmatically in your test:

```go
mockLLM := testutil.NewMockLLMServer()
mockLLM.AddResponse(testutil.ResponseRule{
    Name:     "dynamic-rule",
    Match:    testutil.MatchConfig{Contains: "dynamic"},
    Response: "Dynamic response",
    Priority: 100,
})
```

### Option 3: Code Modification (Legacy)

For backward compatibility, you can still edit `citest/testutil/mockllm.go`:

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
