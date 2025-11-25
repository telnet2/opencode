import { describe, it, expect } from "bun:test"
import { SessionResumer } from "./session-resumer"

/**
 * Unit tests for SessionResumer schemas and types
 *
 * Note: Integration tests requiring Storage and Instance context
 * should be run separately with proper test fixtures.
 */
describe("SessionResumer", () => {
  describe("Schema validation", () => {
    it("should validate SubmitResultInput schema", () => {
      const validInput = {
        pendingID: "pending_123",
        result: {
          title: "Test",
          output: "Output",
        },
      }

      const result = SessionResumer.SubmitResultInput.safeParse(validInput)
      expect(result.success).toBe(true)
    })

    it("should validate SubmitErrorInput schema", () => {
      const validInput = {
        pendingID: "pending_123",
        error: "Something went wrong",
      }

      const result = SessionResumer.SubmitErrorInput.safeParse(validInput)
      expect(result.success).toBe(true)
    })

    it("should validate WebhookPayload result type", () => {
      const resultPayload = {
        type: "result",
        pendingID: "pending_123",
        result: {
          title: "Done",
          output: "Success",
        },
      }

      const result = SessionResumer.WebhookPayload.safeParse(resultPayload)
      expect(result.success).toBe(true)
    })

    it("should validate WebhookPayload error type", () => {
      const errorPayload = {
        type: "error",
        pendingID: "pending_123",
        error: "Failed",
      }

      const result = SessionResumer.WebhookPayload.safeParse(errorPayload)
      expect(result.success).toBe(true)
    })

    it("should validate WebhookPayload progress type", () => {
      const progressPayload = {
        type: "progress",
        pendingID: "pending_123",
        progress: {
          percent: 50,
          message: "Halfway done",
        },
      }

      const result = SessionResumer.WebhookPayload.safeParse(progressPayload)
      expect(result.success).toBe(true)
    })

    it("should reject invalid WebhookPayload type", () => {
      const invalidPayload = {
        type: "unknown",
        pendingID: "pending_123",
      }

      const result = SessionResumer.WebhookPayload.safeParse(invalidPayload)
      expect(result.success).toBe(false)
    })
  })

  describe("Event definitions", () => {
    it("should have all expected events", () => {
      expect(SessionResumer.Event.SessionResuming).toBeDefined()
      expect(SessionResumer.Event.SessionResumed).toBeDefined()
      expect(SessionResumer.Event.ResultSubmitted).toBeDefined()
    })

    it("should have correct event types", () => {
      expect(SessionResumer.Event.SessionResuming.type).toBe("session.resuming")
      expect(SessionResumer.Event.SessionResumed.type).toBe("session.resumed")
      expect(SessionResumer.Event.ResultSubmitted.type).toBe("async-tool.result-submitted")
    })
  })
})
