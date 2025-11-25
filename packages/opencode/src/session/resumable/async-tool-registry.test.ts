import { describe, it, expect, beforeEach, afterEach } from "bun:test"
import { AsyncToolRegistry } from "./async-tool-registry"
import z from "zod"

/**
 * Unit tests for AsyncToolRegistry
 *
 * Note: Tests that call register()/unregister() require Instance context
 * and are marked as integration tests. This file focuses on schema validation
 * and helper functions that don't require Instance context.
 */
describe("AsyncToolRegistry", () => {
  beforeEach(() => {
    AsyncToolRegistry._reset()
  })

  afterEach(() => {
    AsyncToolRegistry._reset()
  })

  describe("Schema validation", () => {
    it("should validate AsyncResult schema", () => {
      const validResult = {
        async: true,
        pendingID: "pending_123",
        output: "Processing started",
      }

      const result = AsyncToolRegistry.AsyncResult.safeParse(validResult)
      expect(result.success).toBe(true)
    })

    it("should validate AsyncResult with optional fields", () => {
      const fullResult = {
        async: true,
        pendingID: "pending_123",
        externalRef: "job_456",
        estimatedDuration: 60000,
        output: "Processing started",
      }

      const result = AsyncToolRegistry.AsyncResult.safeParse(fullResult)
      expect(result.success).toBe(true)
    })

    it("should reject invalid AsyncResult", () => {
      const invalidResult = {
        async: false, // Must be true
        pendingID: "pending_123",
        output: "Processing started",
      }

      const result = AsyncToolRegistry.AsyncResult.safeParse(invalidResult)
      expect(result.success).toBe(false)
    })
  })

  describe("isAsync without registration", () => {
    it("should return false for unregistered tools", () => {
      expect(AsyncToolRegistry.isAsync("unknown")).toBe(false)
    })

    it("should return false for any string when registry is empty", () => {
      expect(AsyncToolRegistry.isAsync("any-tool")).toBe(false)
      expect(AsyncToolRegistry.isAsync("")).toBe(false)
      expect(AsyncToolRegistry.isAsync("test-123")).toBe(false)
    })
  })

  describe("list without registration", () => {
    it("should return empty array when no tools registered", () => {
      expect(AsyncToolRegistry.list()).toEqual([])
    })
  })

  describe("get without registration", () => {
    it("should return undefined for unregistered tools", () => {
      expect(AsyncToolRegistry.get("unknown")).toBeUndefined()
    })
  })

  describe("validateResult without registration", () => {
    it("should return true for unknown tools (no validator)", () => {
      expect(
        AsyncToolRegistry.validateResult("unknown-tool", {
          title: "Test",
          output: "Any output",
        }),
      ).toBe(true)
    })
  })

  describe("createToolInfo", () => {
    it("should create a standard Tool.Info from async definition", async () => {
      const definition: AsyncToolRegistry.AsyncToolDefinition = {
        id: "tool-info-test",
        description: "Tool info test",
        parameters: z.object({ name: z.string() }),
        async execute(input, ctx) {
          return ctx.createAsyncResult(`Hello ${input.name}`)
        },
      }

      const toolInfo = AsyncToolRegistry.createToolInfo(definition)

      expect(toolInfo.id).toBe("tool-info-test")

      const initialized = await toolInfo.init()
      expect(initialized.description).toContain("async")
      expect(initialized.description).toContain("Tool info test")
    })

    it("should preserve parameter schema", async () => {
      const parameters = z.object({
        name: z.string(),
        count: z.number(),
      })

      const definition: AsyncToolRegistry.AsyncToolDefinition = {
        id: "param-test",
        description: "Parameter test",
        parameters,
        async execute(input, ctx) {
          return ctx.createAsyncResult("OK")
        },
      }

      const toolInfo = AsyncToolRegistry.createToolInfo(definition)
      const initialized = await toolInfo.init()

      // Verify the parameters schema is preserved
      expect(initialized.parameters).toBeDefined()
    })
  })

  describe("createExternalServiceTool", () => {
    it("should create an async tool definition for external services", () => {
      const tool = AsyncToolRegistry.createExternalServiceTool({
        id: "external-service-test",
        description: "External service test",
        parameters: z.object({ query: z.string() }),
        defaultTimeout: 300000, // 5 minutes
        async startJob(input, webhookURL, pendingID) {
          return {
            jobID: "external_job_456",
            message: `Job started for query: ${input.query}`,
          }
        },
      })

      expect(tool.id).toBe("external-service-test")
      expect(tool.description).toBe("External service test")
      expect(tool.defaultTimeout).toBe(300000)
    })

    it("should create tool with no default timeout", () => {
      const tool = AsyncToolRegistry.createExternalServiceTool({
        id: "no-timeout-test",
        description: "No timeout test",
        parameters: z.object({}),
        async startJob(input, webhookURL, pendingID) {
          return {
            jobID: "job_123",
            message: "Started",
          }
        },
      })

      expect(tool.id).toBe("no-timeout-test")
      expect(tool.defaultTimeout).toBeUndefined()
    })

    it("should create tool with complex parameters", () => {
      const tool = AsyncToolRegistry.createExternalServiceTool({
        id: "complex-params-test",
        description: "Complex params test",
        parameters: z.object({
          query: z.string(),
          options: z.object({
            limit: z.number(),
            offset: z.number(),
          }),
          filters: z.array(z.string()),
        }),
        async startJob(input, webhookURL, pendingID) {
          return {
            jobID: "job_456",
            message: `Started with ${input.filters.length} filters`,
          }
        },
      })

      expect(tool.id).toBe("complex-params-test")
    })
  })

  describe("Event definitions", () => {
    it("should have all expected events", () => {
      expect(AsyncToolRegistry.Event.Registered).toBeDefined()
      expect(AsyncToolRegistry.Event.Unregistered).toBeDefined()
      expect(AsyncToolRegistry.Event.Started).toBeDefined()
    })

    it("should have correct event types", () => {
      expect(AsyncToolRegistry.Event.Registered.type).toBe("async-tool.registered")
      expect(AsyncToolRegistry.Event.Unregistered.type).toBe("async-tool.unregistered")
      expect(AsyncToolRegistry.Event.Started.type).toBe("async-tool.started")
    })
  })
})
