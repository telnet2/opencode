import { describe, test, expect, mock, beforeEach } from "bun:test"
import { MemshClient } from "./client"
import type { CreateSessionResponse, ListSessionsResponse, RemoveSessionResponse } from "./types"

describe("MemshClient", () => {
  describe("constructor", () => {
    test("should create client with default options", () => {
      const client = new MemshClient({ baseUrl: "http://localhost:8080" })
      expect(client.state).toBe("disconnected")
    })

    test("should create client with custom options", () => {
      const client = new MemshClient({
        baseUrl: "http://localhost:8080",
        timeout: 60000,
        autoReconnect: true,
        maxReconnectAttempts: 10,
        reconnectDelay: 2000,
      })
      expect(client.state).toBe("disconnected")
    })
  })

  describe("REST API", () => {
    let client: MemshClient
    const mockFetch = mock(() => Promise.resolve(new Response()))

    beforeEach(() => {
      client = new MemshClient({ baseUrl: "http://localhost:8080" })
      global.fetch = mockFetch as unknown as typeof fetch
    })

    test("createSession should POST to correct endpoint", async () => {
      const mockResponse: CreateSessionResponse = {
        session: {
          id: "test-session-id",
          created_at: "2024-01-01T00:00:00Z",
          last_used: "2024-01-01T00:00:00Z",
          cwd: "/",
        },
      }

      mockFetch.mockResolvedValueOnce(new Response(JSON.stringify(mockResponse), { status: 200 }))

      const result = await client.createSession()

      expect(mockFetch).toHaveBeenCalledWith("http://localhost:8080/api/v1/session/create", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      })
      expect(result.session.id).toBe("test-session-id")
    })

    test("listSessions should POST to correct endpoint", async () => {
      const mockResponse: ListSessionsResponse = {
        sessions: [
          {
            id: "session-1",
            created_at: "2024-01-01T00:00:00Z",
            last_used: "2024-01-01T00:00:00Z",
            cwd: "/",
          },
        ],
      }

      mockFetch.mockResolvedValueOnce(new Response(JSON.stringify(mockResponse), { status: 200 }))

      const result = await client.listSessions()

      expect(mockFetch).toHaveBeenCalledWith("http://localhost:8080/api/v1/session/list", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      })
      expect(result.sessions.length).toBe(1)
    })

    test("removeSession should POST to correct endpoint with session_id", async () => {
      const mockResponse: RemoveSessionResponse = {
        success: true,
        message: "Session removed successfully",
      }

      mockFetch.mockResolvedValueOnce(new Response(JSON.stringify(mockResponse), { status: 200 }))

      const result = await client.removeSession("test-session-id")

      expect(mockFetch).toHaveBeenCalledWith("http://localhost:8080/api/v1/session/remove", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ session_id: "test-session-id" }),
      })
      expect(result.success).toBe(true)
    })

    test("should throw error on non-OK response", async () => {
      mockFetch.mockResolvedValueOnce(new Response("Not Found", { status: 404, statusText: "Not Found" }))

      await expect(client.createSession()).rejects.toThrow("Failed to create session: Not Found")
    })
  })

  describe("disconnect", () => {
    test("should set state to disconnected", () => {
      const client = new MemshClient({ baseUrl: "http://localhost:8080" })
      client.disconnect()
      expect(client.state).toBe("disconnected")
    })
  })
})
