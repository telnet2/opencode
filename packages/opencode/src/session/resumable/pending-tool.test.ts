import { describe, it, expect } from "bun:test"
import { PendingToolCall } from "./pending-tool"

/**
 * Unit tests for PendingToolCall schemas and types
 *
 * Note: Integration tests requiring Storage and Instance context
 * should be run separately with proper test fixtures.
 */
describe("PendingToolCall", () => {
  describe("Schema validation", () => {
    it("should validate status enum", () => {
      const validStatuses = ["waiting", "processing", "completed", "failed", "expired", "cancelled"]

      for (const status of validStatuses) {
        const result = PendingToolCall.Status.safeParse(status)
        expect(result.success).toBe(true)
      }

      const invalidResult = PendingToolCall.Status.safeParse("invalid")
      expect(invalidResult.success).toBe(false)
    })

    it("should validate result schema", () => {
      const validResult = {
        title: "Test Title",
        output: "Test Output",
        metadata: { key: "value" },
      }

      const result = PendingToolCall.Result.safeParse(validResult)
      expect(result.success).toBe(true)
    })

    it("should validate result schema without optional metadata", () => {
      const minimalResult = {
        title: "Test",
        output: "Output",
      }

      const result = PendingToolCall.Result.safeParse(minimalResult)
      expect(result.success).toBe(true)
    })

    it("should validate info schema structure", () => {
      const validInfo = {
        id: "pending_123",
        sessionID: "session_456",
        messageID: "message_789",
        partID: "part_abc",
        callID: "call_def",
        tool: "test-tool",
        input: { foo: "bar" },
        status: "waiting",
        time: {
          created: Date.now(),
        },
      }

      const result = PendingToolCall.Info.safeParse(validInfo)
      expect(result.success).toBe(true)
    })

    it("should validate info schema with optional fields", () => {
      const fullInfo = {
        id: "pending_123",
        sessionID: "session_456",
        messageID: "message_789",
        partID: "part_abc",
        callID: "call_def",
        tool: "test-tool",
        input: { foo: "bar" },
        status: "completed",
        webhookURL: "https://example.com/webhook",
        webhookSecret: "secret123",
        externalRef: "job_xyz",
        timeout: Date.now() + 60000,
        time: {
          created: Date.now() - 10000,
          started: Date.now() - 5000,
          completed: Date.now(),
        },
        result: {
          title: "Done",
          output: "Success",
          metadata: { duration: 5000 },
        },
      }

      const result = PendingToolCall.Info.safeParse(fullInfo)
      expect(result.success).toBe(true)
    })

    it("should reject invalid info schema", () => {
      const invalidInfo = {
        id: "pending_123",
        // Missing required fields
      }

      const result = PendingToolCall.Info.safeParse(invalidInfo)
      expect(result.success).toBe(false)
    })

    it("should validate index schema", () => {
      const validIndex = {
        bySession: {
          session_1: ["pending_1", "pending_2"],
          session_2: ["pending_3"],
        },
        byStatus: {
          waiting: ["pending_1"],
          processing: ["pending_2"],
          completed: ["pending_3"],
          failed: [],
          expired: [],
          cancelled: [],
        },
        byExpiration: [
          { id: "pending_1", expiresAt: Date.now() + 60000 },
          { id: "pending_2", expiresAt: Date.now() + 120000 },
        ],
        lastUpdated: Date.now(),
      }

      const result = PendingToolCall.Index.safeParse(validIndex)
      expect(result.success).toBe(true)
    })
  })

  describe("Event definitions", () => {
    it("should have all expected events", () => {
      expect(PendingToolCall.Event.Created).toBeDefined()
      expect(PendingToolCall.Event.Updated).toBeDefined()
      expect(PendingToolCall.Event.Completed).toBeDefined()
      expect(PendingToolCall.Event.Failed).toBeDefined()
      expect(PendingToolCall.Event.Expired).toBeDefined()
    })

    it("should have correct event types", () => {
      expect(PendingToolCall.Event.Created.type).toBe("pending-tool.created")
      expect(PendingToolCall.Event.Updated.type).toBe("pending-tool.updated")
      expect(PendingToolCall.Event.Completed.type).toBe("pending-tool.completed")
      expect(PendingToolCall.Event.Failed.type).toBe("pending-tool.failed")
      expect(PendingToolCall.Event.Expired.type).toBe("pending-tool.expired")
    })
  })
})
