import { Hono } from "hono"
import { describeRoute, resolver, validator } from "hono-openapi"
import { streamSSE } from "hono/streaming"
import { z } from "zod"
import { Bus } from "../bus"
import { Log } from "../util/log"
import { PendingToolCall, SessionResumer, Recovery, AsyncToolRegistry } from "../session/resumable"

const log = Log.create({ service: "async-tools-route" })

// ============================================================================
// Request/Response Schemas
// ============================================================================

const SubmitResultRequest = z.object({
  pendingID: z.string(),
  result: PendingToolCall.Result,
})

const SubmitErrorRequest = z.object({
  pendingID: z.string(),
  error: z.string(),
})

const SubmitResultResponse = z.object({
  success: z.boolean(),
  sessionID: z.string().optional(),
})

// Define webhook payload inline to avoid circular imports
const WebhookRequest = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("result"),
    pendingID: z.string(),
    result: PendingToolCall.Result,
  }),
  z.object({
    type: z.literal("error"),
    pendingID: z.string(),
    error: z.string(),
  }),
  z.object({
    type: z.literal("progress"),
    pendingID: z.string(),
    progress: z.object({
      percent: z.number().min(0).max(100).optional(),
      message: z.string().optional(),
    }),
  }),
])

const PendingToolResponse = z.object({
  pending: PendingToolCall.Info,
})

const PendingListResponse = z.object({
  pending: z.array(PendingToolCall.Info),
  count: z.number(),
})

const RecoveryStatusResponse = z.object({
  waitingSessions: z.number(),
  waitingCalls: z.number(),
  processingCalls: z.number(),
  sessions: z.array(
    z.object({
      sessionID: z.string(),
      pendingCalls: z.number(),
      oldestCall: z.number(),
    }),
  ),
})

const ErrorResponse = z.object({
  error: z.string(),
})

// ============================================================================
// Routes
// ============================================================================

export const AsyncToolsRoute = new Hono()
  // Submit result for async tool
  .post(
    "/result",
    describeRoute({
      description: "Submit result for an async tool call and resume session",
      operationId: "asyncTools.submitResult",
      requestBody: {
        content: {
          "application/json": {
            schema: resolver(SubmitResultRequest),
          },
        },
      },
      responses: {
        200: {
          description: "Result submitted successfully",
          content: {
            "application/json": {
              schema: resolver(SubmitResultResponse),
            },
          },
        },
        400: {
          description: "Invalid request",
          content: {
            "application/json": {
              schema: resolver(ErrorResponse),
            },
          },
        },
        404: {
          description: "Pending tool call not found",
          content: {
            "application/json": {
              schema: resolver(ErrorResponse),
            },
          },
        },
      },
    }),
    validator("json", SubmitResultRequest),
    async (c) => {
      const { pendingID, result } = c.req.valid("json")
      log.info("submitting async result", { pendingID })

      try {
        // Validate result if validator exists
        const pending = await PendingToolCall.get(pendingID)
        if (!pending) {
          return c.json({ error: "Pending tool call not found" }, 404)
        }

        if (!AsyncToolRegistry.validateResult(pending.tool, result)) {
          return c.json({ error: "Result validation failed" }, 400)
        }

        await SessionResumer.submitResult({ pendingID, result })

        return c.json({
          success: true,
          sessionID: pending.sessionID,
        })
      } catch (e) {
        log.error("failed to submit result", { pendingID, error: e })
        const message = e instanceof Error ? e.message : String(e)
        return c.json({ error: message }, 400)
      }
    },
  )

  // Submit error for async tool
  .post(
    "/error",
    describeRoute({
      description: "Submit error for an async tool call and resume session",
      operationId: "asyncTools.submitError",
      requestBody: {
        content: {
          "application/json": {
            schema: resolver(SubmitErrorRequest),
          },
        },
      },
      responses: {
        200: {
          description: "Error submitted successfully",
          content: {
            "application/json": {
              schema: resolver(SubmitResultResponse),
            },
          },
        },
        400: {
          description: "Invalid request",
          content: {
            "application/json": {
              schema: resolver(ErrorResponse),
            },
          },
        },
        404: {
          description: "Pending tool call not found",
          content: {
            "application/json": {
              schema: resolver(ErrorResponse),
            },
          },
        },
      },
    }),
    validator("json", SubmitErrorRequest),
    async (c) => {
      const { pendingID, error } = c.req.valid("json")
      log.info("submitting async error", { pendingID, error })

      try {
        const pending = await PendingToolCall.get(pendingID)
        if (!pending) {
          return c.json({ error: "Pending tool call not found" }, 404)
        }

        await SessionResumer.submitError({ pendingID, error })

        return c.json({
          success: true,
          sessionID: pending.sessionID,
        })
      } catch (e) {
        log.error("failed to submit error", { pendingID, error: e })
        const message = e instanceof Error ? e.message : String(e)
        return c.json({ error: message }, 400)
      }
    },
  )

  // Webhook endpoint for external systems
  .post(
    "/webhook",
    describeRoute({
      description: "Webhook endpoint for external systems to submit results/errors",
      operationId: "asyncTools.webhook",
      requestBody: {
        content: {
          "application/json": {
            schema: resolver(WebhookRequest),
          },
        },
      },
      responses: {
        200: {
          description: "Webhook processed successfully",
          content: {
            "application/json": {
              schema: resolver(SubmitResultResponse),
            },
          },
        },
        400: {
          description: "Invalid webhook payload",
          content: {
            "application/json": {
              schema: resolver(ErrorResponse),
            },
          },
        },
        401: {
          description: "Invalid signature",
          content: {
            "application/json": {
              schema: resolver(ErrorResponse),
            },
          },
        },
      },
    }),
    validator("json", WebhookRequest),
    async (c) => {
      const payload = c.req.valid("json")
      const signature = c.req.header("X-Webhook-Signature")

      log.info("received webhook", { type: payload.type, pendingID: payload.pendingID })

      try {
        await SessionResumer.handleWebhook(payload, signature)

        return c.json({
          success: true,
        })
      } catch (e) {
        log.error("webhook processing failed", { error: e })
        const message = e instanceof Error ? e.message : String(e)

        if (message.includes("signature")) {
          return c.json({ error: message }, 401)
        }
        return c.json({ error: message }, 400)
      }
    },
  )

  // Get pending tool call status
  .get(
    "/pending/:id",
    describeRoute({
      description: "Get status of a pending tool call",
      operationId: "asyncTools.getPending",
      responses: {
        200: {
          description: "Pending tool call info",
          content: {
            "application/json": {
              schema: resolver(PendingToolResponse),
            },
          },
        },
        404: {
          description: "Pending tool call not found",
          content: {
            "application/json": {
              schema: resolver(ErrorResponse),
            },
          },
        },
      },
    }),
    async (c) => {
      const id = c.req.param("id")
      const pending = await PendingToolCall.get(id)

      if (!pending) {
        return c.json({ error: "Pending tool call not found" }, 404)
      }

      return c.json({ pending })
    },
  )

  // List all pending tool calls
  .get(
    "/pending",
    describeRoute({
      description: "List all pending async tool calls",
      operationId: "asyncTools.listPending",
      responses: {
        200: {
          description: "List of pending tool calls",
          content: {
            "application/json": {
              schema: resolver(PendingListResponse),
            },
          },
        },
      },
    }),
    async (c) => {
      const status = c.req.query("status") as PendingToolCall.Status | undefined
      const sessionID = c.req.query("sessionID")

      let pending: PendingToolCall.Info[]

      if (sessionID) {
        pending = await PendingToolCall.listBySession(sessionID)
      } else if (status) {
        pending = await PendingToolCall.listByStatus(status)
      } else {
        pending = await PendingToolCall.listAll()
      }

      return c.json({
        pending,
        count: pending.length,
      })
    },
  )

  // Cancel a pending tool call
  .delete(
    "/pending/:id",
    describeRoute({
      description: "Cancel a pending tool call",
      operationId: "asyncTools.cancelPending",
      responses: {
        200: {
          description: "Pending tool call cancelled",
          content: {
            "application/json": {
              schema: resolver(SubmitResultResponse),
            },
          },
        },
        400: {
          description: "Cannot cancel tool call",
          content: {
            "application/json": {
              schema: resolver(ErrorResponse),
            },
          },
        },
        404: {
          description: "Pending tool call not found",
          content: {
            "application/json": {
              schema: resolver(ErrorResponse),
            },
          },
        },
      },
    }),
    async (c) => {
      const id = c.req.param("id")
      const pending = await PendingToolCall.get(id)

      if (!pending) {
        return c.json({ error: "Pending tool call not found" }, 404)
      }

      if (pending.status !== "waiting" && pending.status !== "processing") {
        return c.json({ error: `Cannot cancel tool call with status: ${pending.status}` }, 400)
      }

      try {
        await PendingToolCall.cancel(id)
        await SessionResumer.resumeWithError(pending, "Cancelled by user")

        return c.json({
          success: true,
          sessionID: pending.sessionID,
        })
      } catch (e) {
        log.error("failed to cancel pending call", { id, error: e })
        const message = e instanceof Error ? e.message : String(e)
        return c.json({ error: message }, 400)
      }
    },
  )

  // Get recovery status
  .get(
    "/recovery/status",
    describeRoute({
      description: "Get status of sessions waiting for async tool results",
      operationId: "asyncTools.recoveryStatus",
      responses: {
        200: {
          description: "Recovery status",
          content: {
            "application/json": {
              schema: resolver(RecoveryStatusResponse),
            },
          },
        },
      },
    }),
    async (c) => {
      const status = await Recovery.getStatus()
      return c.json(status)
    },
  )

  // Trigger recovery check
  .post(
    "/recovery/check",
    describeRoute({
      description: "Check for expired tool calls and recover sessions",
      operationId: "asyncTools.recoveryCheck",
      responses: {
        200: {
          description: "Recovery check completed",
          content: {
            "application/json": {
              schema: resolver(
                z.object({
                  expired: z.number(),
                }),
              ),
            },
          },
        },
      },
    }),
    async (c) => {
      const expired = await Recovery.checkExpired()
      return c.json({ expired })
    },
  )

  // SSE endpoint for monitoring async tool status changes
  .get(
    "/events",
    describeRoute({
      description: "Stream async tool events",
      operationId: "asyncTools.events",
      responses: {
        200: {
          description: "SSE stream of async tool events",
          content: {
            "text/event-stream": {
              schema: resolver(
                z.object({
                  type: z.string(),
                  data: z.any(),
                }),
              ),
            },
          },
        },
      },
    }),
    async (c) => {
      const sessionID = c.req.query("sessionID")

      log.info("async tool events connected", { sessionID })

      return streamSSE(c, async (stream) => {
        const unsubscribers: (() => void)[] = []

        // Subscribe to pending tool events
        unsubscribers.push(
          Bus.subscribe(PendingToolCall.Event.Created, async (event) => {
            if (!sessionID || event.properties.pending.sessionID === sessionID) {
              await stream.writeSSE({
                event: "created",
                data: JSON.stringify(event.properties.pending),
              })
            }
          }),
        )

        unsubscribers.push(
          Bus.subscribe(PendingToolCall.Event.Completed, async (event) => {
            if (!sessionID || event.properties.pending.sessionID === sessionID) {
              await stream.writeSSE({
                event: "completed",
                data: JSON.stringify(event.properties.pending),
              })
            }
          }),
        )

        unsubscribers.push(
          Bus.subscribe(PendingToolCall.Event.Failed, async (event) => {
            if (!sessionID || event.properties.pending.sessionID === sessionID) {
              await stream.writeSSE({
                event: "failed",
                data: JSON.stringify({
                  pending: event.properties.pending,
                  error: event.properties.error,
                }),
              })
            }
          }),
        )

        unsubscribers.push(
          Bus.subscribe(PendingToolCall.Event.Expired, async (event) => {
            if (!sessionID || event.properties.pending.sessionID === sessionID) {
              await stream.writeSSE({
                event: "expired",
                data: JSON.stringify(event.properties.pending),
              })
            }
          }),
        )

        unsubscribers.push(
          Bus.subscribe(SessionResumer.Event.SessionResuming, async (event) => {
            if (!sessionID || event.properties.sessionID === sessionID) {
              await stream.writeSSE({
                event: "session-resuming",
                data: JSON.stringify(event.properties),
              })
            }
          }),
        )

        unsubscribers.push(
          Bus.subscribe(SessionResumer.Event.SessionResumed, async (event) => {
            if (!sessionID || event.properties.sessionID === sessionID) {
              await stream.writeSSE({
                event: "session-resumed",
                data: JSON.stringify(event.properties),
              })
            }
          }),
        )

        // Keep alive
        const keepAlive = setInterval(async () => {
          try {
            await stream.writeSSE({ event: "ping", data: "" })
          } catch {
            // Connection closed
          }
        }, 30000)

        // Wait for disconnect
        await new Promise<void>((resolve) => {
          stream.onAbort(() => {
            log.info("async tool events disconnected", { sessionID })
            unsubscribers.forEach((unsub) => unsub())
            clearInterval(keepAlive)
            resolve()
          })
        })
      })
    },
  )

  // Get session async waiting status
  .get(
    "/session/:sessionID/status",
    describeRoute({
      description: "Get async waiting status for a session",
      operationId: "asyncTools.sessionStatus",
      responses: {
        200: {
          description: "Session async status",
          content: {
            "application/json": {
              schema: resolver(
                z.discriminatedUnion("waiting", [
                  z.object({
                    waiting: z.literal(true),
                    pendingCalls: z.array(
                      z.object({
                        id: z.string(),
                        tool: z.string(),
                        createdAt: z.number(),
                        externalRef: z.string().optional(),
                      }),
                    ),
                  }),
                  z.object({ waiting: z.literal(false) }),
                ]),
              ),
            },
          },
        },
      },
    }),
    async (c) => {
      const sessionID = c.req.param("sessionID")
      const status = await SessionResumer.getWaitingStatus(sessionID)
      return c.json(status)
    },
  )

  // List registered async tools
  .get(
    "/tools",
    describeRoute({
      description: "List registered async tools",
      operationId: "asyncTools.listTools",
      responses: {
        200: {
          description: "List of async tool IDs",
          content: {
            "application/json": {
              schema: resolver(
                z.object({
                  tools: z.array(z.string()),
                }),
              ),
            },
          },
        },
      },
    }),
    async (c) => {
      const tools = AsyncToolRegistry.list()
      return c.json({ tools })
    },
  )
