# Client-Side Tools Testing Strategy

**Related Document:** [Client-Side Tools Design](./client-side-tools.md)

---

## Executive Summary

**Yes, the current test infrastructure can be utilized to implement and validate the client tool feature.** This document details how each testing pattern can be applied and identifies any gaps that need to be addressed.

### Test Coverage Matrix

| Component | Test Type | Infrastructure Available | Additional Needs |
|-----------|-----------|-------------------------|------------------|
| ClientToolRegistry | Unit | ✅ Bun test framework | None |
| API Routes | Integration | ✅ Python integration tests | None |
| Event Bus Integration | Unit | ✅ Bus subscribe/publish tests | None |
| SSE Streaming | Integration | ✅ Python SSE client tests | None |
| WebSocket Handler | Integration | ⚠️ Partial (needs WebSocket client) | WebSocket test utility |
| SDK ClientToolsManager | Unit | ✅ Mock transport pattern | None |
| End-to-End Flow | Integration | ✅ Python subprocess pattern | None |
| Tool Execution Timeout | Unit | ✅ Retry/timeout test patterns | None |

---

## Applicable Test Infrastructure

### 1. Bun Test Framework (TypeScript Unit Tests)

**Location:** `packages/opencode/test/`

**Applicable For:**
- `ClientToolRegistry` module testing
- Tool registration/unregistration logic
- Event emission verification
- Timeout and error handling

**Example Pattern (from `session/session.test.ts:11-41`):**

```typescript
import { describe, expect, test } from "bun:test"
import { ClientToolRegistry } from "../../src/tool/client-registry"
import { Instance } from "../../src/project/instance"
import { Bus } from "../../src/bus"

describe("ClientToolRegistry", () => {
  test("should register tools for a client", async () => {
    await Instance.provide({
      directory: process.cwd(),
      fn: async () => {
        const tools = [
          { id: "test_tool", description: "A test tool", parameters: {} }
        ]

        const registered = ClientToolRegistry.register("client-123", tools)

        expect(registered).toEqual(["client_client-123_test_tool"])
        expect(ClientToolRegistry.getTools("client-123")).toHaveLength(1)
      },
    })
  })

  test("should emit ToolRequest event when executing", async () => {
    await Instance.provide({
      directory: process.cwd(),
      fn: async () => {
        let eventReceived = false
        const unsub = Bus.subscribe(ClientToolRegistry.Event.ToolRequest, () => {
          eventReceived = true
        })

        // Register tool first
        ClientToolRegistry.register("client-123", [
          { id: "test", description: "test", parameters: {} }
        ])

        // Start execution (will emit event)
        const executePromise = ClientToolRegistry.execute("client-123", {
          requestID: "req-1",
          sessionID: "sess-1",
          messageID: "msg-1",
          callID: "call-1",
          tool: "client_client-123_test",
          input: {},
        }, 100) // Short timeout for test

        await new Promise(resolve => setTimeout(resolve, 50))
        unsub()

        expect(eventReceived).toBe(true)
      },
    })
  })
})
```

### 2. Instance.provide() Pattern

**Purpose:** Provides isolated project context for each test.

**Applicable For:**
- Tests that require project/session context
- Event bus isolation between tests
- Tool registry state isolation

**Usage:**
```typescript
await Instance.provide({
  directory: projectRoot,
  fn: async () => {
    // Test code runs in isolated instance context
    // Bus subscriptions are scoped to this instance
  },
})
```

### 3. Temporary Directory Fixture

**Location:** `packages/opencode/test/fixture/fixture.ts`

**Applicable For:**
- Tests that need file system operations
- Integration tests with real server subprocess

**Usage:**
```typescript
import { tmpdir } from "../fixture/fixture"

test("client tool with file access", async () => {
  await using tmp = await tmpdir({ git: true })

  // tmp.path is the isolated directory
  // Automatically cleaned up after test
})
```

### 4. Fake Server Pattern (LSP Tests)

**Location:** `packages/opencode/test/fixture/lsp/fake-lsp-server.js`

**Applicable For:**
- Testing client-server communication protocols
- SSE and WebSocket message exchange
- JSON-RPC style request/response testing

**New Fake Client Tool Server:**

```javascript
// packages/opencode/test/fixture/client-tools/fake-client.js
// Simulates an SDK client that handles tool requests

const EventSource = require("eventsource")

class FakeToolClient {
  constructor(baseUrl, clientId) {
    this.baseUrl = baseUrl
    this.clientId = clientId
    this.handlers = new Map()
  }

  registerTool(id, handler) {
    this.handlers.set(id, handler)
  }

  async register(tools) {
    const response = await fetch(`${this.baseUrl}/client-tools/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        clientID: this.clientId,
        tools,
      }),
    })
    return response.json()
  }

  connect() {
    this.es = new EventSource(
      `${this.baseUrl}/client-tools/pending/${this.clientId}`
    )

    this.es.addEventListener("tool-request", async (event) => {
      const request = JSON.parse(event.data)
      const handler = this.handlers.get(
        request.tool.replace(`client_${this.clientId}_`, "")
      )

      let result
      if (handler) {
        try {
          result = { status: "success", ...await handler(request.input) }
        } catch (error) {
          result = { status: "error", error: error.message }
        }
      } else {
        result = { status: "error", error: "Unknown tool" }
      }

      await fetch(`${this.baseUrl}/client-tools/result`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ requestID: request.requestID, result }),
      })
    })
  }

  disconnect() {
    this.es?.close()
  }
}

module.exports = { FakeToolClient }
```

### 5. Python Integration Tests (Subprocess Server)

**Location:** `packages/sdk/python/tests/test_integration.py`

**Applicable For:**
- Full server startup and API validation
- SSE streaming tests
- End-to-end client tool flow

**Extended Integration Test:**

```python
# packages/sdk/python/tests/test_client_tools.py

import json
import subprocess
import time
import threading
import pytest
from pathlib import Path
from sseclient import SSEClient
import httpx

@pytest.mark.timeout(60)
def test_client_tool_registration_and_execution():
    """Test full client tool flow: register -> execute -> result"""

    # Start server (reuse pattern from test_integration.py)
    repo_root = find_repo_root()
    pkg_opencode = repo_root / "packages" / "opencode"

    proc = subprocess.Popen(
        ["bun", "run", "./src/index.ts", "serve", "--port", "0"],
        cwd=str(pkg_opencode),
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
    )

    url = wait_for_server_url(proc, timeout=15)
    client_id = "test-client-123"

    try:
        # 1. Register client tool
        register_response = httpx.post(
            f"{url}/client-tools/register",
            json={
                "clientID": client_id,
                "tools": [{
                    "id": "echo",
                    "description": "Echo input back",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "message": {"type": "string"}
                        }
                    }
                }]
            }
        )
        assert register_response.status_code == 200
        registered = register_response.json()["registered"]
        assert len(registered) == 1
        assert "echo" in registered[0]

        # 2. Start SSE listener for tool requests
        tool_requests = []
        def listen_for_requests():
            response = httpx.get(
                f"{url}/client-tools/pending/{client_id}",
                timeout=30.0
            )
            client = SSEClient(response)
            for event in client.events():
                if event.event == "tool-request":
                    tool_requests.append(json.loads(event.data))
                    break

        listener_thread = threading.Thread(target=listen_for_requests)
        listener_thread.start()
        time.sleep(0.5)  # Wait for SSE connection

        # 3. Create session and send prompt that would trigger tool
        # (This would require actual AI model - skip for unit test)
        # Instead, simulate tool request via internal API if available

        # 4. Submit result
        if tool_requests:
            result_response = httpx.post(
                f"{url}/client-tools/result",
                json={
                    "requestID": tool_requests[0]["requestID"],
                    "result": {
                        "status": "success",
                        "title": "Echo result",
                        "output": "Hello, World!"
                    }
                }
            )
            assert result_response.status_code == 200

    finally:
        terminate_process(proc)


@pytest.mark.timeout(30)
def test_client_tool_unregister():
    """Test tool unregistration"""
    # Similar setup...
    pass


@pytest.mark.timeout(30)
def test_client_tool_timeout():
    """Test that tool execution times out if client doesn't respond"""
    # Register tool, trigger execution, don't respond, verify timeout
    pass
```

### 6. Mock HTTP Transport (SDK Tests)

**Location:** `packages/sdk/python/tests/test_wrapper.py`, `packages/sdk/go/client_test.go`

**Applicable For:**
- SDK ClientToolsManager unit tests
- Isolated testing without real server

**Python Example:**
```python
def test_client_tools_manager_register():
    """Test ClientToolsManager registration without server"""

    registered_tools = []

    def handler(request: httpx.Request) -> httpx.Response:
        if request.url.path == "/client-tools/register":
            body = json.loads(request.content)
            registered_tools.extend(body["tools"])
            return httpx.Response(200, json={
                "registered": [f"client_{body['clientID']}_{t['id']}" for t in body["tools"]]
            })
        return httpx.Response(404)

    transport = httpx.MockTransport(handler)
    client = httpx.Client(base_url="http://test", transport=transport)

    manager = ClientToolsManager("test-client", "http://test")
    manager._http_client = client

    result = manager.register_sync([
        {"id": "tool1", "description": "Test", "parameters": {}}
    ])

    assert len(registered_tools) == 1
    assert "tool1" in result[0]
```

**Go Example:**
```go
func TestClientToolRegistration(t *testing.T) {
    var registeredTools []map[string]interface{}

    client := opencode.NewClient(
        option.WithHTTPClient(&http.Client{
            Transport: &closureTransport{
                fn: func(req *http.Request) (*http.Response, error) {
                    if req.URL.Path == "/client-tools/register" {
                        body, _ := io.ReadAll(req.Body)
                        var payload map[string]interface{}
                        json.Unmarshal(body, &payload)
                        registeredTools = payload["tools"].([]map[string]interface{})

                        return &http.Response{
                            StatusCode: 200,
                            Body: io.NopCloser(strings.NewReader(`{"registered":["client_test_tool1"]}`)),
                        }, nil
                    }
                    return &http.Response{StatusCode: 404}, nil
                },
            },
        }),
    )

    // Test registration
    // ...
}
```

### 7. Event Bus Testing Pattern

**Applicable For:**
- Testing ClientToolRegistry event emission
- Testing event subscription/unsubscription
- Testing event ordering

**Pattern (from `session/session.test.ts`):**
```typescript
test("tool request event should be emitted", async () => {
  await Instance.provide({
    directory: projectRoot,
    fn: async () => {
      const events: any[] = []

      const unsub = Bus.subscribe(ClientToolRegistry.Event.ToolRequest, (event) => {
        events.push(event)
      })

      // Trigger tool execution
      ClientToolRegistry.register("client-1", [
        { id: "test", description: "test", parameters: {} }
      ])

      const executePromise = ClientToolRegistry.execute("client-1", {
        requestID: "req-1",
        sessionID: "sess-1",
        messageID: "msg-1",
        callID: "call-1",
        tool: "client_client-1_test",
        input: { foo: "bar" },
      }, 1000)

      await new Promise(resolve => setTimeout(resolve, 100))
      unsub()

      expect(events).toHaveLength(1)
      expect(events[0].properties.clientID).toBe("client-1")
      expect(events[0].properties.request.input).toEqual({ foo: "bar" })
    },
  })
})
```

### 8. Retry and Timeout Testing Pattern

**Location:** `packages/opencode/test/session/retry.test.ts`

**Applicable For:**
- Client tool execution timeout testing
- Retry logic for failed tool executions
- Exponential backoff validation

**Example:**
```typescript
describe("ClientToolRegistry.execute timeout", () => {
  test("should timeout after specified duration", async () => {
    await Instance.provide({
      directory: projectRoot,
      fn: async () => {
        ClientToolRegistry.register("client-1", [
          { id: "slow_tool", description: "Slow tool", parameters: {} }
        ])

        const startTime = Date.now()

        await expect(
          ClientToolRegistry.execute("client-1", {
            requestID: "req-1",
            sessionID: "sess-1",
            messageID: "msg-1",
            callID: "call-1",
            tool: "client_client-1_slow_tool",
            input: {},
          }, 100) // 100ms timeout
        ).rejects.toThrow("timed out")

        const elapsed = Date.now() - startTime
        expect(elapsed).toBeGreaterThanOrEqual(100)
        expect(elapsed).toBeLessThan(200)
      },
    })
  })
})
```

---

## Proposed Test Structure

```
packages/opencode/test/
├── tool/
│   ├── bash.test.ts              # Existing
│   ├── patch.test.ts             # Existing
│   ├── client-registry.test.ts   # NEW: ClientToolRegistry unit tests
│   └── client-tools-api.test.ts  # NEW: API route tests
├── fixture/
│   ├── fixture.ts                # Existing
│   ├── lsp/
│   │   └── fake-lsp-server.js    # Existing
│   └── client-tools/             # NEW
│       └── fake-client.js        # Fake SDK client for testing

packages/sdk/
├── js/test/                      # NEW
│   └── client-tools.test.ts      # ClientToolsManager tests
├── python/tests/
│   ├── test_wrapper.py           # Existing
│   ├── test_integration.py       # Existing
│   └── test_client_tools.py      # NEW: Client tools integration
└── go/
    ├── client_test.go            # Existing
    └── clienttools_test.go       # NEW: Client tools tests
```

---

## Test Categories

### Unit Tests (No External Dependencies)

| Test File | Coverage |
|-----------|----------|
| `client-registry.test.ts` | Registration, unregistration, tool lookup |
| `client-registry.test.ts` | Event emission, pending request management |
| `client-registry.test.ts` | Timeout handling, cleanup |
| `js/client-tools.test.ts` | ClientToolsManager with mock transport |

### Integration Tests (Server Subprocess)

| Test File | Coverage |
|-----------|----------|
| `test_client_tools.py` | Full registration/execution flow via SSE |
| `test_client_tools.py` | WebSocket communication (if implemented) |
| `test_client_tools.py` | Multi-client scenarios |

### End-to-End Tests (Requires Real Model)

| Test | Coverage | Feasibility |
|------|----------|-------------|
| AI triggers client tool | Complete flow | **Not feasible without real model** |
| Tool result used in response | Complete flow | **Not feasible without real model** |

---

## Implementation Recommendations

### 1. Start with Unit Tests

```typescript
// packages/opencode/test/tool/client-registry.test.ts

describe("ClientToolRegistry", () => {
  describe("register", () => {
    test("registers tools with prefixed IDs")
    test("handles multiple tools")
    test("handles duplicate registration")
  })

  describe("unregister", () => {
    test("removes specific tools")
    test("removes all tools for client")
    test("handles non-existent client")
  })

  describe("getTools", () => {
    test("returns tools for client")
    test("returns empty array for unknown client")
  })

  describe("execute", () => {
    test("emits ToolRequest event")
    test("times out if no response")
    test("resolves on successful result")
    test("rejects on error result")
  })

  describe("submitResult", () => {
    test("resolves pending request")
    test("returns false for unknown request")
    test("clears timeout on submission")
  })

  describe("cleanup", () => {
    test("cancels pending requests")
    test("removes all client tools")
  })
})
```

### 2. Add API Route Tests

```typescript
// packages/opencode/test/tool/client-tools-api.test.ts

describe("Client Tools API Routes", () => {
  // Use Python integration test pattern: start server subprocess

  test("POST /client-tools/register creates tools")
  test("DELETE /client-tools/unregister removes tools")
  test("POST /client-tools/result submits execution result")
  test("GET /client-tools/pending/:clientID streams requests")
})
```

### 3. Add SDK Tests

```typescript
// packages/sdk/js/test/client-tools.test.ts

describe("ClientToolsManager", () => {
  test("register sends HTTP request to server")
  test("connect establishes SSE connection")
  test("handles incoming tool requests")
  test("submits tool results")
  test("disconnect cleans up connections")
})
```

---

## Gaps and Additional Infrastructure Needed

### 1. WebSocket Test Utility

The current infrastructure doesn't have WebSocket testing utilities. Options:

**Option A: Skip WebSocket in initial tests**
- Focus on SSE which is already testable
- WebSocket is optional in the design

**Option B: Add WebSocket test helper**
```typescript
// packages/opencode/test/fixture/websocket.ts
import WebSocket from "ws"

export function createTestWebSocket(url: string): Promise<{
  ws: WebSocket
  messages: any[]
  send: (data: any) => void
  waitForMessage: (predicate: (msg: any) => boolean) => Promise<any>
  close: () => void
}> {
  // Implementation
}
```

### 2. SSE Test Utility for TypeScript

Python has `sseclient-py`, but TypeScript tests may need:

```typescript
// packages/opencode/test/fixture/sse-client.ts
export async function* sseStream(url: string): AsyncGenerator<{
  event: string
  data: string
}> {
  const response = await fetch(url)
  const reader = response.body!.getReader()
  // Parse SSE format
}
```

### 3. Test Server Startup Helper

Consolidate server startup logic:

```typescript
// packages/opencode/test/fixture/server.ts
export async function startTestServer(): Promise<{
  url: string
  close: () => Promise<void>
}> {
  // Start server with random port
  // Wait for startup
  // Return URL and cleanup function
}
```

---

## CI/CD Considerations

The existing CI pipeline (`test.yml`) will automatically run new tests:

```yaml
- run: |
    bun turbo typecheck
    bun turbo test  # This runs all tests including new client-tools tests
```

For Python tests:
```yaml
- run: uv run --project packages/sdk/python pytest -q
```

**No changes needed to CI configuration.**

---

## Conclusion

The current test infrastructure **fully supports** implementing and validating the client tools feature:

| Requirement | Available Infrastructure | Confidence |
|-------------|-------------------------|------------|
| Unit testing ClientToolRegistry | Bun test + Instance.provide | ✅ High |
| Event bus integration testing | Bus.subscribe pattern | ✅ High |
| API route testing | Python subprocess pattern | ✅ High |
| SSE streaming testing | Python sseclient | ✅ High |
| SDK unit testing | Mock HTTP transport | ✅ High |
| WebSocket testing | Needs utility addition | ⚠️ Medium |
| End-to-end with real AI | Not possible without model | ❌ N/A |

**Recommendation:** Proceed with implementation using the existing patterns. Add WebSocket test utility only if WebSocket support is prioritized over SSE.
