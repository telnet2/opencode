# Plan: Fundamental API Tests (Provider + Service Layer)

## Overview

This plan covers integration tests for the fundamental API layers:
1. **Provider tests** - Unit tests in `internal/provider/` for direct LLM API calls
2. **Service tests** - Behavioral tests in `citest/service/` via HTTP against real server

Tests are designed bottom-up: provider tests validate LLM connectivity, service tests validate business logic through HTTP.

---

## Part 1: Provider Tests (internal/provider/)

### Location
`internal/provider/ark_test.go` (enhance existing)

### Test Scenarios

#### 1.1 Basic Completion
```ginkgo
Describe("ArkProvider", func() {
    Describe("CreateCompletion", func() {
        It("should return a response for simple prompt", func() {
            // Send: "Say hello"
            // Expect: Non-empty response
        })

        It("should stream response chunks", func() {
            // Verify multiple chunks received before completion
        })

        It("should respect max_tokens limit", func() {
            // Set max_tokens=10, verify response is truncated
        })

        It("should handle temperature parameter", func() {
            // temperature=0 should give deterministic output
        })
    })
})
```

#### 1.2 Tool Calling
```ginkgo
Describe("Tool Binding", func() {
    It("should bind tools to chat model", func() {
        // Bind a simple tool, verify no error
    })

    It("should generate tool calls when appropriate", func() {
        // Prompt: "What is 2+2? Use the calculator tool"
        // Tool: calculator with add function
        // Expect: ToolCall in response
    })

    It("should include tool call arguments", func() {
        // Verify arguments are properly JSON encoded
    })
})
```

#### 1.3 Error Handling
```ginkgo
Describe("Error Handling", func() {
    It("should return error for invalid API key", func() {
        // Create provider with bad key, expect error
    })

    It("should handle context cancellation", func() {
        // Cancel context mid-stream, verify graceful handling
    })

    It("should handle empty response", func() {
        // Edge case: model returns no content
    })
})
```

---

## Part 2: Service Tests (citest/service/)

### Location
`citest/service/`

### Prerequisites
- Test server running on localhost (started in BeforeSuite)
- ARK provider configured via environment variables
- HTTP client for raw requests

### Test Structure

```
citest/service/
├── service_suite_test.go    # Ginkgo bootstrap, server lifecycle
├── session_test.go          # Session CRUD operations
├── message_test.go          # Message send/receive, streaming
└── tools_test.go            # Tool execution (bash, file)
```

### 2.1 Session Lifecycle (`session_test.go`)

```ginkgo
var _ = Describe("Session Management", func() {
    Describe("POST /session", func() {
        It("should create a new session", func() {
            // POST /session with directory
            // Expect: 200, session object with ID
        })

        It("should create session with title", func() {
            // POST /session with title
            // Expect: session.title matches
        })

        It("should reject invalid directory", func() {
            // POST /session with non-existent directory
            // Expect: 400 error
        })
    })

    Describe("GET /session", func() {
        It("should list all sessions", func() {
            // Create 3 sessions
            // GET /session
            // Expect: array with 3 sessions
        })
    })

    Describe("GET /session/{id}", func() {
        It("should return session by ID", func() {
            // Create session, get by ID
            // Expect: matching session
        })

        It("should return 404 for unknown session", func() {
            // GET /session/unknown-id
            // Expect: 404
        })
    })

    Describe("DELETE /session/{id}", func() {
        It("should delete session", func() {
            // Create, delete, verify gone
        })
    })
})
```

### 2.2 Message Flow (`message_test.go`)

```ginkgo
var _ = Describe("Message Flow", func() {
    var sessionID string

    BeforeEach(func() {
        // Create fresh session
    })

    AfterEach(func() {
        // Cleanup session
    })

    Describe("POST /session/{id}/message", func() {
        It("should send message and receive streaming response", func() {
            // POST message: "Say hello"
            // Read chunked response
            // Expect: assistant message with content
        })

        It("should echo user message first", func() {
            // First chunk should be user message
        })

        It("should include token usage in final response", func() {
            // Verify tokens.input, tokens.output present
        })

        It("should handle multi-turn conversation", func() {
            // Send message 1, wait for response
            // Send message 2 referencing message 1
            // Verify context is maintained
        })
    })

    Describe("GET /session/{id}/message", func() {
        It("should return all messages in session", func() {
            // Send 2 messages
            // GET messages
            // Expect: 4 messages (2 user + 2 assistant)
        })
    })
})
```

### 2.3 Tool Execution (`tools_test.go`)

```ginkgo
var _ = Describe("Tool Execution", func() {
    var sessionID string

    BeforeEach(func() {
        // Create session with tools enabled
    })

    Describe("Bash Tool", func() {
        It("should execute simple bash command", func() {
            // Prompt: "Run 'echo hello' in bash"
            // Expect: Tool call executed, output contains "hello"
        })

        It("should capture command output", func() {
            // Prompt: "List files in current directory"
            // Expect: Tool result with file listing
        })

        It("should handle command failure", func() {
            // Prompt: "Run 'exit 1'"
            // Expect: Tool shows error status
        })
    })

    Describe("File Read Tool", func() {
        It("should read file content", func() {
            // Create temp file with known content
            // Prompt: "Read the file /tmp/test.txt"
            // Expect: Tool returns file content
        })

        It("should handle non-existent file", func() {
            // Prompt: "Read /nonexistent/file.txt"
            // Expect: Error in tool result
        })
    })

    Describe("File Write Tool", func() {
        It("should write content to file", func() {
            // Prompt: "Write 'test content' to /tmp/output.txt"
            // Verify file created with content
        })
    })

    Describe("Tool Chain", func() {
        It("should execute multiple tools in sequence", func() {
            // Prompt: "Create a file, then read it back"
            // Expect: Both tools executed successfully
        })
    })
})
```

---

## Test Utilities

### Server Lifecycle (`citest/testutil/server.go`)

```go
package testutil

type TestServer struct {
    Server   *server.Server
    BaseURL  string
    Config   *types.Config
}

func StartTestServer() (*TestServer, error) {
    // Load config from citest/config/opencode.json
    // Initialize providers, storage, tools
    // Start server on random available port
    // Return TestServer with BaseURL
}

func (ts *TestServer) Stop() error {
    // Graceful shutdown
}
```

### HTTP Client Helpers (`citest/testutil/client.go`)

```go
package testutil

type TestClient struct {
    BaseURL    string
    HTTPClient *http.Client
}

func (c *TestClient) CreateSession(dir string) (*Session, error)
func (c *TestClient) SendMessage(sessionID, content string) (*MessageResponse, error)
func (c *TestClient) SendMessageStreaming(sessionID, content string) (<-chan Chunk, error)
func (c *TestClient) GetMessages(sessionID string) ([]Message, error)
func (c *TestClient) DeleteSession(sessionID string) error
```

---

## Running Tests

```bash
# Run provider tests only
go test -v ./internal/provider/... -run TestArk

# Run service tests only
cd citest && ginkgo -v ./service/

# Run all integration tests
cd citest && ginkgo -v ./...

# Run with focus
cd citest && ginkgo -v --focus="Session Management" ./service/
```

---

## Environment Requirements

```bash
# Required environment variables
export ARK_API_KEY="your-api-key"
export ARK_MODEL_ID="ep-xxx"
export ARK_BASE_URL="https://ark-ap-southeast.byteintl.net/api/v3"

# Or use .env file in project root
```

---

## Success Criteria

1. All provider tests pass with real ARK endpoint
2. All service tests pass with real server + ARK
3. Tool execution (bash, file read/write) works end-to-end
4. Streaming responses are properly chunked
5. Error cases return appropriate HTTP status codes
