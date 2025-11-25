import { describe, expect, test, beforeAll, afterAll } from "bun:test"
import path from "path"
import { spawn, type Subprocess } from "bun"
import { Log } from "../../src/util/log"

Log.init({ print: false })

/**
 * Integration tests for the Client Tools API endpoints.
 *
 * These tests start a real OpenCode server and validate the HTTP API
 * for client tool registration, execution, and result submission.
 *
 * Note: These tests require network access and a working server.
 * They are skipped by default in CI environments without network.
 */
describe.skip("Client Tools API (requires live server)", () => {
  let serverProcess: Subprocess | null = null
  let serverUrl: string | null = null

  // Start server before all tests
  beforeAll(async () => {
    const opencodePath = path.join(__dirname, "../..")

    serverProcess = spawn({
      cmd: ["bun", "run", "--conditions=development", "./src/index.ts", "serve", "--port", "0", "--hostname", "127.0.0.1"],
      cwd: opencodePath,
      stdout: "pipe",
      stderr: "pipe",
    })

    // Wait for server to start and extract URL
    const timeout = 15000
    const startTime = Date.now()

    const reader = serverProcess.stdout.getReader()
    let buffer = ""

    while (Date.now() - startTime < timeout) {
      const { value, done } = await reader.read()
      if (done) break

      buffer += new TextDecoder().decode(value)
      const match = buffer.match(/opencode server listening on (http:\/\/[^\s]+)/)
      if (match) {
        serverUrl = match[1]
        break
      }
    }

    reader.releaseLock()

    if (!serverUrl) {
      serverProcess?.kill()
      throw new Error("Server did not start within timeout")
    }
  }, 20000)

  // Stop server after all tests
  afterAll(async () => {
    if (serverProcess) {
      serverProcess.kill()
      await serverProcess.exited
    }
  })

  describe("POST /client-tools/register", () => {
    test("should register tools successfully", async () => {
      const response = await fetch(`${serverUrl}/client-tools/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          clientID: "test-client-1",
          tools: [
            {
              id: "test_tool",
              description: "A test tool for integration testing",
              parameters: {
                type: "object",
                properties: {
                  input: { type: "string" },
                },
              },
            },
          ],
        }),
      })

      expect(response.status).toBe(200)

      const data = await response.json()
      expect(data.registered).toBeDefined()
      expect(data.registered).toContain("client_test-client-1_test_tool")
    })

    test("should register multiple tools", async () => {
      const response = await fetch(`${serverUrl}/client-tools/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          clientID: "test-client-2",
          tools: [
            { id: "tool1", description: "Tool 1", parameters: {} },
            { id: "tool2", description: "Tool 2", parameters: {} },
            { id: "tool3", description: "Tool 3", parameters: {} },
          ],
        }),
      })

      expect(response.status).toBe(200)

      const data = await response.json()
      expect(data.registered).toHaveLength(3)
    })
  })

  describe("GET /client-tools/tools/:clientID", () => {
    test("should return registered tools for client", async () => {
      // First register a tool
      await fetch(`${serverUrl}/client-tools/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          clientID: "test-client-get",
          tools: [{ id: "get_test", description: "Get test tool", parameters: {} }],
        }),
      })

      // Then retrieve tools
      const response = await fetch(`${serverUrl}/client-tools/tools/test-client-get`)

      expect(response.status).toBe(200)

      const tools = await response.json()
      expect(Array.isArray(tools)).toBe(true)
      expect(tools.length).toBeGreaterThan(0)
      expect(tools[0].description).toBe("Get test tool")
    })

    test("should return empty array for unknown client", async () => {
      const response = await fetch(`${serverUrl}/client-tools/tools/unknown-client-xyz`)

      expect(response.status).toBe(200)

      const tools = await response.json()
      expect(Array.isArray(tools)).toBe(true)
      expect(tools).toHaveLength(0)
    })
  })

  describe("GET /client-tools/tools", () => {
    test("should return all registered tools", async () => {
      // Register tools for multiple clients
      await fetch(`${serverUrl}/client-tools/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          clientID: "all-tools-client-1",
          tools: [{ id: "all_tool_1", description: "All tool 1", parameters: {} }],
        }),
      })

      await fetch(`${serverUrl}/client-tools/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          clientID: "all-tools-client-2",
          tools: [{ id: "all_tool_2", description: "All tool 2", parameters: {} }],
        }),
      })

      const response = await fetch(`${serverUrl}/client-tools/tools`)

      expect(response.status).toBe(200)

      const tools = await response.json()
      expect(typeof tools).toBe("object")
      // Should contain tools from both clients
      expect(Object.keys(tools).some((k) => k.includes("all_tool_1"))).toBe(true)
      expect(Object.keys(tools).some((k) => k.includes("all_tool_2"))).toBe(true)
    })
  })

  describe("DELETE /client-tools/unregister", () => {
    test("should unregister specific tools", async () => {
      // Register tools first
      await fetch(`${serverUrl}/client-tools/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          clientID: "unregister-client",
          tools: [
            { id: "keep_tool", description: "Keep this", parameters: {} },
            { id: "remove_tool", description: "Remove this", parameters: {} },
          ],
        }),
      })

      // Unregister specific tool
      const response = await fetch(`${serverUrl}/client-tools/unregister`, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          clientID: "unregister-client",
          toolIDs: ["remove_tool"],
        }),
      })

      expect(response.status).toBe(200)

      const data = await response.json()
      expect(data.success).toBe(true)
      expect(data.unregistered).toContain("client_unregister-client_remove_tool")

      // Verify remaining tools
      const toolsResponse = await fetch(`${serverUrl}/client-tools/tools/unregister-client`)
      const tools = await toolsResponse.json()
      expect(tools).toHaveLength(1)
    })

    test("should unregister all tools when no toolIDs provided", async () => {
      // Register tools
      await fetch(`${serverUrl}/client-tools/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          clientID: "unregister-all-client",
          tools: [
            { id: "tool1", description: "Tool 1", parameters: {} },
            { id: "tool2", description: "Tool 2", parameters: {} },
          ],
        }),
      })

      // Unregister all
      const response = await fetch(`${serverUrl}/client-tools/unregister`, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          clientID: "unregister-all-client",
        }),
      })

      expect(response.status).toBe(200)

      // Verify no remaining tools
      const toolsResponse = await fetch(`${serverUrl}/client-tools/tools/unregister-all-client`)
      const tools = await toolsResponse.json()
      expect(tools).toHaveLength(0)
    })
  })

  describe("POST /client-tools/result", () => {
    test("should return 404 for unknown request ID", async () => {
      const response = await fetch(`${serverUrl}/client-tools/result`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          requestID: "unknown-request-id",
          result: {
            status: "success",
            title: "Test",
            output: "Test output",
          },
        }),
      })

      expect(response.status).toBe(404)

      const data = await response.json()
      expect(data.error).toBe("Unknown request ID")
    })
  })

  describe("SSE /client-tools/pending/:clientID", () => {
    test("should establish SSE connection", async () => {
      // Register a tool first
      await fetch(`${serverUrl}/client-tools/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          clientID: "sse-test-client",
          tools: [{ id: "sse_tool", description: "SSE test tool", parameters: {} }],
        }),
      })

      // Connect to SSE endpoint
      const response = await fetch(`${serverUrl}/client-tools/pending/sse-test-client`)

      expect(response.status).toBe(200)
      expect(response.headers.get("content-type")).toContain("text/event-stream")

      // Close the connection
      await response.body?.cancel()
    })
  })
})
