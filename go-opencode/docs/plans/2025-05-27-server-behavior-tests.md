# Plan: Server HTTP/SSE Behavior Tests

## Overview

This plan covers tests for HTTP and SSE-specific behaviors of the server. Unlike service tests that focus on *what* the system does, these tests focus on *how* the HTTP layer behaves.

### Location
`citest/server/`

### Focus Areas
1. **SSE Streaming** - Event stream format, heartbeats, session filtering
2. **HTTP Response Format** - Status codes, error structures, headers
3. **Chunked Streaming** - Message response streaming behavior
4. **CORS and Headers** - Cross-origin support, content types

---

## Part 1: SSE Event Streaming (`sse_test.go`)

### Test Structure

```ginkgo
var _ = Describe("SSE Event Streaming", func() {

    Describe("GET /event", func() {
        It("should return SSE content-type header", func() {
            // GET /event?sessionID=xxx
            // Expect: Content-Type: text/event-stream
        })

        It("should set cache control headers", func() {
            // Expect: Cache-Control: no-cache
            // Expect: Connection: keep-alive
        })

        It("should send heartbeat within 30 seconds", func() {
            // Connect to /event?sessionID=xxx
            // Wait for heartbeat (: heartbeat\n\n)
            // Verify received within 35 seconds
        })

        It("should format events correctly", func() {
            // Trigger an event (create session)
            // Expect format:
            // event: session.created
            // data: {"type":"session.created","data":{...}}
            // (blank line)
        })

        It("should filter events by session ID", func() {
            // Create 2 sessions
            // Connect to /event?sessionID=session1
            // Send message to session2
            // Verify NO event received for session2
        })

        It("should deliver events for matching session", func() {
            // Connect to /event?sessionID=xxx
            // Send message to session xxx
            // Expect: message.created and message.updated events
        })

        It("should return 400 without sessionID", func() {
            // GET /event (no query param)
            // Expect: 400 Bad Request
        })
    })

    Describe("GET /global/event", func() {
        It("should stream all events without filtering", func() {
            // Connect to /global/event
            // Create session, send message
            // Expect: All events received
        })

        It("should include events from multiple sessions", func() {
            // Create session1, session2
            // Connect to /global/event
            // Send message to both sessions
            // Expect: Events from both sessions
        })
    })

    Describe("SSE Connection Lifecycle", func() {
        It("should handle client disconnect gracefully", func() {
            // Connect, then close connection
            // Verify server doesn't crash/leak
        })

        It("should stop sending after context cancel", func() {
            // Connect with context
            // Cancel context
            // Verify connection closes cleanly
        })
    })

    Describe("Event Types", func() {
        Context("Session Events", func() {
            It("should emit session.created", func() {})
            It("should emit session.updated", func() {})
            It("should emit session.deleted", func() {})
        })

        Context("Message Events", func() {
            It("should emit message.created for user message", func() {})
            It("should emit message.updated during streaming", func() {})
            It("should emit part.updated for content chunks", func() {})
        })

        Context("Tool Events", func() {
            It("should emit part.updated for tool calls", func() {})
            It("should emit file.edited when file is modified", func() {})
        })
    })
})
```

---

## Part 2: HTTP Response Behavior (`response_test.go`)

```ginkgo
var _ = Describe("HTTP Response Behavior", func() {

    Describe("Success Responses", func() {
        It("should return 200 with JSON body for GET", func() {
            // GET /session
            // Expect: 200, Content-Type: application/json
        })

        It("should return 201 for resource creation", func() {
            // POST /session
            // Expect: 200 (or 201 if implemented)
        })

        It("should return empty body for DELETE", func() {
            // DELETE /session/{id}
            // Expect: 200, {"success": true}
        })
    })

    Describe("Error Responses", func() {
        It("should return structured error for 400", func() {
            // Send malformed JSON
            // Expect: {"error": {"code": "INVALID_REQUEST", "message": "..."}}
        })

        It("should return 404 for unknown resource", func() {
            // GET /session/nonexistent
            // Expect: 404, {"error": {"code": "NOT_FOUND", ...}}
        })

        It("should return 500 for internal errors", func() {
            // Trigger internal error
            // Expect: 500, {"error": {"code": "INTERNAL_ERROR", ...}}
        })

        It("should include error details when available", func() {
            // Expect: {"error": {..., "details": {...}}}
        })
    })

    Describe("Error Codes", func() {
        It("should use INVALID_REQUEST for bad input", func() {})
        It("should use NOT_FOUND for missing resources", func() {})
        It("should use PROVIDER_ERROR for LLM failures", func() {})
    })
})
```

---

## Part 3: Chunked Message Streaming (`streaming_test.go`)

```ginkgo
var _ = Describe("Message Streaming", func() {

    Describe("POST /session/{id}/message", func() {
        It("should use chunked transfer encoding", func() {
            // Verify Transfer-Encoding: chunked
        })

        It("should send user message as first chunk", func() {
            // First JSON object should be user message echo
        })

        It("should send multiple chunks during generation", func() {
            // Count chunks received
            // Expect: > 1 chunk for non-trivial response
        })

        It("should send final message with complete content", func() {
            // Last chunk should have full assistant message
        })

        It("should flush chunks immediately", func() {
            // Verify no buffering (chunks arrive as generated)
        })

        It("should handle connection drop mid-stream", func() {
            // Close connection while streaming
            // Verify server handles gracefully
        })
    })

    Describe("Streaming Error Handling", func() {
        It("should return error chunk on LLM failure", func() {
            // Trigger provider error
            // Expect: Error object in stream
        })

        It("should close stream after error", func() {
            // After error chunk, stream should end
        })
    })
})
```

---

## Part 4: CORS and Headers (`headers_test.go`)

```ginkgo
var _ = Describe("CORS and Headers", func() {

    Describe("CORS Preflight", func() {
        It("should respond to OPTIONS request", func() {
            // OPTIONS /session
            // Expect: 200 with CORS headers
        })

        It("should allow all origins", func() {
            // Origin: http://example.com
            // Expect: Access-Control-Allow-Origin: *
        })

        It("should allow required methods", func() {
            // Expect: Access-Control-Allow-Methods includes GET, POST, PUT, PATCH, DELETE
        })

        It("should allow required headers", func() {
            // Expect: Access-Control-Allow-Headers includes Content-Type, Authorization
        })
    })

    Describe("Request Headers", func() {
        It("should generate request ID", func() {
            // Any request
            // Expect: X-Request-ID in response
        })

        It("should accept JSON content-type", func() {
            // Content-Type: application/json
            // Expect: Request processed
        })
    })

    Describe("Response Headers", func() {
        It("should set JSON content-type for API responses", func() {
            // GET /session
            // Expect: Content-Type: application/json
        })

        It("should disable buffering for SSE", func() {
            // GET /event
            // Expect: X-Accel-Buffering: no
        })
    })
})
```

---

## Part 5: File and Search Endpoints (`endpoints_test.go`)

```ginkgo
var _ = Describe("File Endpoints", func() {

    Describe("GET /file", func() {
        It("should list directory contents", func() {
            // GET /file?path=/tmp
            // Expect: Array of file entries
        })

        It("should return 400 for invalid path", func() {})
    })

    Describe("GET /file/content", func() {
        It("should return file content with line numbers", func() {
            // GET /file/content?path=/tmp/test.txt
            // Expect: {"content": "...", "lines": N}
        })

        It("should support offset and limit", func() {
            // GET /file/content?path=...&offset=10&limit=5
            // Expect: Lines 10-14
        })

        It("should indicate truncation", func() {
            // Large file with limit
            // Expect: {"truncated": true}
        })
    })

    Describe("GET /file/status", func() {
        It("should return git status", func() {
            // Expect: {"branch": "...", "staged": [...], ...}
        })
    })
})

var _ = Describe("Search Endpoints", func() {

    Describe("GET /find", func() {
        It("should search text in files", func() {
            // GET /find?query=function
            // Expect: {"matches": [...]}
        })

        It("should limit results to 100", func() {})
    })

    Describe("GET /find/file", func() {
        It("should search files by pattern", func() {
            // GET /find/file?pattern=*.go
            // Expect: Array of matching files
        })
    })
})
```

---

## Test Utilities

### SSE Client Helper (`citest/testutil/sse.go`)

```go
package testutil

type SSEClient struct {
    URL    string
    Events chan SSEEvent
    Errors chan error
}

type SSEEvent struct {
    Type string
    Data json.RawMessage
}

func NewSSEClient(url string) *SSEClient
func (c *SSEClient) Connect(ctx context.Context) error
func (c *SSEClient) Close()
func (c *SSEClient) WaitForEvent(eventType string, timeout time.Duration) (*SSEEvent, error)
func (c *SSEClient) WaitForHeartbeat(timeout time.Duration) error
```

---

## Running Tests

```bash
# Run server behavior tests
cd citest && ginkgo -v ./server/

# Run SSE tests only
cd citest && ginkgo -v --focus="SSE" ./server/

# Run with race detection
cd citest && ginkgo -v -race ./server/

# Verbose output
cd citest && ginkgo -v --progress ./server/
```

---

## Success Criteria

1. SSE events are properly formatted and filtered by session
2. Heartbeats are sent within 30-second interval
3. Error responses follow standard structure
4. CORS headers are present and correct
5. Chunked streaming delivers content progressively
6. Server handles connection drops gracefully
