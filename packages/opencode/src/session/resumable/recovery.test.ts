import { describe, it, expect } from "bun:test"
import { Recovery } from "./recovery"

/**
 * Unit tests for Recovery module
 *
 * Note: Integration tests requiring Storage and Instance context
 * should be run separately with proper test fixtures.
 */
describe("Recovery", () => {
  describe("expiration checker", () => {
    it("should start and stop without errors", () => {
      expect(() => {
        Recovery.startExpirationChecker(60000)
        Recovery.stopExpirationChecker()
      }).not.toThrow()
    })

    it("should handle multiple start calls", () => {
      expect(() => {
        Recovery.startExpirationChecker(60000)
        Recovery.startExpirationChecker(30000) // Should replace previous
        Recovery.stopExpirationChecker()
      }).not.toThrow()
    })

    it("should handle stop without start", () => {
      expect(() => {
        Recovery.stopExpirationChecker()
      }).not.toThrow()
    })
  })

  describe("Type definitions", () => {
    it("should have RecoveryResult interface", () => {
      // Type check - RecoveryResult structure
      const result: Recovery.RecoveryResult = {
        recovered: [],
        expired: [],
        errors: [],
        cleanedUp: 0,
      }

      expect(result.recovered).toEqual([])
      expect(result.expired).toEqual([])
      expect(result.errors).toEqual([])
      expect(result.cleanedUp).toBe(0)
    })

    it("should have RecoveryStatus interface", () => {
      // Type check - RecoveryStatus structure
      const status: Recovery.RecoveryStatus = {
        waitingSessions: 0,
        waitingCalls: 0,
        processingCalls: 0,
        sessions: [],
      }

      expect(status.waitingSessions).toBe(0)
      expect(status.waitingCalls).toBe(0)
      expect(status.processingCalls).toBe(0)
      expect(status.sessions).toEqual([])
    })

    it("should allow session details in RecoveryStatus", () => {
      const status: Recovery.RecoveryStatus = {
        waitingSessions: 2,
        waitingCalls: 5,
        processingCalls: 1,
        sessions: [
          {
            sessionID: "session_1",
            pendingCalls: 3,
            oldestCall: Date.now() - 60000,
          },
          {
            sessionID: "session_2",
            pendingCalls: 2,
            oldestCall: Date.now() - 30000,
          },
        ],
      }

      expect(status.sessions.length).toBe(2)
      expect(status.sessions[0].sessionID).toBe("session_1")
      expect(status.sessions[0].pendingCalls).toBe(3)
    })
  })
})
