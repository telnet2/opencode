import { describe, expect, test, beforeEach, afterEach } from "bun:test"
import path from "path"
import { ClientToolRegistry } from "../../src/tool/client-registry"
import { Bus } from "../../src/bus"
import { Log } from "../../src/util/log"
import { Instance } from "../../src/project/instance"

const projectRoot = path.join(__dirname, "../..")
Log.init({ print: false })

describe("ClientToolRegistry", () => {
  beforeEach(() => {
    ClientToolRegistry._reset()
  })

  afterEach(() => {
    ClientToolRegistry._reset()
  })

  describe("register", () => {
    test("should register tools with prefixed IDs", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const tools = [{ id: "test_tool", description: "A test tool", parameters: {} }]

          const registered = ClientToolRegistry.register("client-123", tools)

          expect(registered).toEqual(["client_client-123_test_tool"])
          expect(ClientToolRegistry.getTools("client-123")).toHaveLength(1)
        },
      })
    })

    test("should handle multiple tools", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const tools = [
            { id: "tool1", description: "Tool 1", parameters: {} },
            { id: "tool2", description: "Tool 2", parameters: { type: "object" } },
          ]

          const registered = ClientToolRegistry.register("client-123", tools)

          expect(registered).toHaveLength(2)
          expect(registered).toContain("client_client-123_tool1")
          expect(registered).toContain("client_client-123_tool2")
        },
      })
    })

    test("should emit Registered event", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          let eventReceived = false
          let receivedClientID: string | undefined
          let receivedToolIDs: string[] | undefined

          const unsub = Bus.subscribe(ClientToolRegistry.Event.Registered, (event) => {
            eventReceived = true
            receivedClientID = event.properties.clientID
            receivedToolIDs = event.properties.toolIDs
          })

          const tools = [{ id: "test_tool", description: "Test", parameters: {} }]
          ClientToolRegistry.register("client-abc", tools)

          await new Promise((resolve) => setTimeout(resolve, 50))
          unsub()

          expect(eventReceived).toBe(true)
          expect(receivedClientID).toBe("client-abc")
          expect(receivedToolIDs).toContain("client_client-abc_test_tool")
        },
      })
    })

    test("should overwrite existing tool with same ID", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const tools1 = [{ id: "tool", description: "Original", parameters: {} }]
          const tools2 = [{ id: "tool", description: "Updated", parameters: { foo: "bar" } }]

          ClientToolRegistry.register("client-123", tools1)
          ClientToolRegistry.register("client-123", tools2)

          const tools = ClientToolRegistry.getTools("client-123")
          expect(tools).toHaveLength(1)
          expect(tools[0].description).toBe("Updated")
        },
      })
    })
  })

  describe("unregister", () => {
    test("should remove specific tools", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const tools = [
            { id: "tool1", description: "Tool 1", parameters: {} },
            { id: "tool2", description: "Tool 2", parameters: {} },
          ]

          ClientToolRegistry.register("client-123", tools)
          const unregistered = ClientToolRegistry.unregister("client-123", ["tool1"])

          expect(unregistered).toContain("client_client-123_tool1")
          expect(ClientToolRegistry.getTools("client-123")).toHaveLength(1)
        },
      })
    })

    test("should remove all tools for client when no toolIDs provided", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const tools = [
            { id: "tool1", description: "Tool 1", parameters: {} },
            { id: "tool2", description: "Tool 2", parameters: {} },
          ]

          ClientToolRegistry.register("client-123", tools)
          const unregistered = ClientToolRegistry.unregister("client-123")

          expect(unregistered).toHaveLength(2)
          expect(ClientToolRegistry.getTools("client-123")).toHaveLength(0)
        },
      })
    })

    test("should return empty array for non-existent client", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const unregistered = ClientToolRegistry.unregister("non-existent")
          expect(unregistered).toHaveLength(0)
        },
      })
    })

    test("should emit Unregistered event", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          let eventReceived = false

          const tools = [{ id: "test_tool", description: "Test", parameters: {} }]
          ClientToolRegistry.register("client-abc", tools)

          const unsub = Bus.subscribe(ClientToolRegistry.Event.Unregistered, (event) => {
            eventReceived = true
          })

          ClientToolRegistry.unregister("client-abc")

          await new Promise((resolve) => setTimeout(resolve, 50))
          unsub()

          expect(eventReceived).toBe(true)
        },
      })
    })
  })

  describe("getTools", () => {
    test("should return tools for client", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const tools = [
            { id: "tool1", description: "Tool 1", parameters: {} },
            { id: "tool2", description: "Tool 2", parameters: {} },
          ]

          ClientToolRegistry.register("client-123", tools)
          const result = ClientToolRegistry.getTools("client-123")

          expect(result).toHaveLength(2)
        },
      })
    })

    test("should return empty array for unknown client", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const result = ClientToolRegistry.getTools("unknown-client")
          expect(result).toHaveLength(0)
        },
      })
    })
  })

  describe("getAllTools", () => {
    test("should return tools from all clients", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          ClientToolRegistry.register("client-1", [{ id: "tool1", description: "Tool 1", parameters: {} }])
          ClientToolRegistry.register("client-2", [{ id: "tool2", description: "Tool 2", parameters: {} }])

          const all = ClientToolRegistry.getAllTools()

          expect(all.size).toBe(2)
          expect(all.has("client_client-1_tool1")).toBe(true)
          expect(all.has("client_client-2_tool2")).toBe(true)
        },
      })
    })
  })

  describe("isClientTool", () => {
    test("should return true for client tool IDs", () => {
      expect(ClientToolRegistry.isClientTool("client_abc_tool")).toBe(true)
    })

    test("should return false for non-client tool IDs", () => {
      expect(ClientToolRegistry.isClientTool("bash")).toBe(false)
      expect(ClientToolRegistry.isClientTool("read")).toBe(false)
    })
  })

  describe("findClientForTool", () => {
    test("should find client owning a tool", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          ClientToolRegistry.register("client-123", [{ id: "my_tool", description: "Test", parameters: {} }])

          const clientID = ClientToolRegistry.findClientForTool("client_client-123_my_tool")
          expect(clientID).toBe("client-123")
        },
      })
    })

    test("should return undefined for non-existent tool", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const clientID = ClientToolRegistry.findClientForTool("non_existent_tool")
          expect(clientID).toBeUndefined()
        },
      })
    })
  })

  describe("getTool", () => {
    test("should return tool definition", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          ClientToolRegistry.register("client-123", [
            { id: "my_tool", description: "My tool description", parameters: { type: "object" } },
          ])

          const tool = ClientToolRegistry.getTool("client_client-123_my_tool")
          expect(tool).toBeDefined()
          expect(tool?.description).toBe("My tool description")
        },
      })
    })

    test("should return undefined for non-existent tool", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const tool = ClientToolRegistry.getTool("non_existent")
          expect(tool).toBeUndefined()
        },
      })
    })
  })

  describe("execute", () => {
    test("should emit ToolRequest event", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          let eventReceived = false
          let receivedRequest: any

          ClientToolRegistry.register("client-123", [{ id: "test", description: "test", parameters: {} }])

          const unsub = Bus.subscribe(ClientToolRegistry.Event.ToolRequest, (event) => {
            eventReceived = true
            receivedRequest = event.properties.request
          })

          // Start execution (will timeout, but we just want to check event emission)
          const executePromise = ClientToolRegistry.execute(
            "client-123",
            {
              requestID: "req-1",
              sessionID: "sess-1",
              messageID: "msg-1",
              callID: "call-1",
              tool: "client_client-123_test",
              input: { foo: "bar" },
            },
            100,
          ).catch(() => {}) // Ignore timeout

          await new Promise((resolve) => setTimeout(resolve, 50))
          unsub()

          expect(eventReceived).toBe(true)
          expect(receivedRequest.tool).toBe("client_client-123_test")
          expect(receivedRequest.input).toEqual({ foo: "bar" })
        },
      })
    })

    test("should timeout if no response", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          ClientToolRegistry.register("client-123", [{ id: "slow_tool", description: "Slow tool", parameters: {} }])

          const startTime = Date.now()

          await expect(
            ClientToolRegistry.execute(
              "client-123",
              {
                requestID: "req-1",
                sessionID: "sess-1",
                messageID: "msg-1",
                callID: "call-1",
                tool: "client_client-123_slow_tool",
                input: {},
              },
              100,
            ),
          ).rejects.toThrow("timed out")

          const elapsed = Date.now() - startTime
          expect(elapsed).toBeGreaterThanOrEqual(100)
          expect(elapsed).toBeLessThan(200)
        },
      })
    })

    test("should emit Executing event", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          let eventReceived = false

          ClientToolRegistry.register("client-123", [{ id: "test", description: "test", parameters: {} }])

          const unsub = Bus.subscribe(ClientToolRegistry.Event.Executing, (event) => {
            eventReceived = true
          })

          ClientToolRegistry.execute(
            "client-123",
            {
              requestID: "req-1",
              sessionID: "sess-1",
              messageID: "msg-1",
              callID: "call-1",
              tool: "client_client-123_test",
              input: {},
            },
            100,
          ).catch(() => {})

          await new Promise((resolve) => setTimeout(resolve, 50))
          unsub()

          expect(eventReceived).toBe(true)
        },
      })
    })
  })

  describe("submitResult", () => {
    test("should resolve pending request on success", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          ClientToolRegistry.register("client-123", [{ id: "test", description: "test", parameters: {} }])

          const executePromise = ClientToolRegistry.execute(
            "client-123",
            {
              requestID: "req-success",
              sessionID: "sess-1",
              messageID: "msg-1",
              callID: "call-1",
              tool: "client_client-123_test",
              input: {},
            },
            5000,
          )

          // Submit result after a short delay
          await new Promise((resolve) => setTimeout(resolve, 10))
          const submitted = ClientToolRegistry.submitResult("req-success", {
            status: "success",
            title: "Success",
            output: "Result output",
          })

          expect(submitted).toBe(true)

          const result = await executePromise
          expect(result.status).toBe("success")
          expect(result.output).toBe("Result output")
        },
      })
    })

    test("should reject pending request on error", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          ClientToolRegistry.register("client-123", [{ id: "test", description: "test", parameters: {} }])

          const executePromise = ClientToolRegistry.execute(
            "client-123",
            {
              requestID: "req-error",
              sessionID: "sess-1",
              messageID: "msg-1",
              callID: "call-1",
              tool: "client_client-123_test",
              input: {},
            },
            5000,
          )

          await new Promise((resolve) => setTimeout(resolve, 10))
          ClientToolRegistry.submitResult("req-error", {
            status: "error",
            error: "Something went wrong",
          })

          await expect(executePromise).rejects.toThrow("Something went wrong")
        },
      })
    })

    test("should return false for unknown request ID", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const result = ClientToolRegistry.submitResult("unknown-request", {
            status: "success",
            title: "Test",
            output: "Output",
          })

          expect(result).toBe(false)
        },
      })
    })
  })

  describe("cleanup", () => {
    test("should cancel pending requests", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          ClientToolRegistry.register("client-123", [{ id: "test", description: "test", parameters: {} }])

          const executePromise = ClientToolRegistry.execute(
            "client-123",
            {
              requestID: "req-cleanup",
              sessionID: "sess-1",
              messageID: "msg-1",
              callID: "call-1",
              tool: "client_client-123_test",
              input: {},
            },
            5000,
          )

          await new Promise((resolve) => setTimeout(resolve, 10))
          ClientToolRegistry.cleanup("client-123")

          await expect(executePromise).rejects.toThrow("Client disconnected")
        },
      })
    })

    test("should remove all client tools", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          ClientToolRegistry.register("client-123", [
            { id: "tool1", description: "Tool 1", parameters: {} },
            { id: "tool2", description: "Tool 2", parameters: {} },
          ])

          expect(ClientToolRegistry.getTools("client-123")).toHaveLength(2)

          ClientToolRegistry.cleanup("client-123")

          expect(ClientToolRegistry.getTools("client-123")).toHaveLength(0)
        },
      })
    })
  })

  describe("extractOriginalToolID", () => {
    test("should extract original tool ID from prefixed ID", () => {
      const original = ClientToolRegistry.extractOriginalToolID("client_abc123_my_tool", "abc123")
      expect(original).toBe("my_tool")
    })

    test("should return as-is if not prefixed", () => {
      const original = ClientToolRegistry.extractOriginalToolID("my_tool", "abc123")
      expect(original).toBe("my_tool")
    })
  })

  describe("hasTools", () => {
    test("should return true when client has tools", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          ClientToolRegistry.register("client-123", [{ id: "tool", description: "Tool", parameters: {} }])
          expect(ClientToolRegistry.hasTools("client-123")).toBe(true)
        },
      })
    })

    test("should return false when client has no tools", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          expect(ClientToolRegistry.hasTools("unknown-client")).toBe(false)
        },
      })
    })
  })

  describe("getClientIDs", () => {
    test("should return all client IDs", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          ClientToolRegistry.register("client-1", [{ id: "tool1", description: "Tool 1", parameters: {} }])
          ClientToolRegistry.register("client-2", [{ id: "tool2", description: "Tool 2", parameters: {} }])

          const clientIDs = ClientToolRegistry.getClientIDs()
          expect(clientIDs).toContain("client-1")
          expect(clientIDs).toContain("client-2")
        },
      })
    })
  })
})
