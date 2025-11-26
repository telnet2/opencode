import { Hono } from "hono"
import { describeRoute, resolver, validator } from "hono-openapi"
import { streamSSE } from "hono/streaming"
import { z } from "zod"
import { ClientToolRegistry } from "../tool/client-registry"
import { Bus } from "../bus"
import { Log } from "../util/log"

const log = Log.create({ service: "client-tools-route" })

// ============================================================================
// Request/Response Schemas
// ============================================================================

const RegisterRequest = z.object({
  clientID: z.string(),
  tools: z.array(ClientToolRegistry.ClientToolDefinition),
})

const RegisterResponse = z.object({
  registered: z.array(z.string()),
})

const UnregisterRequest = z.object({
  clientID: z.string(),
  toolIDs: z.array(z.string()).optional(),
})

const UnregisterResponse = z.object({
  success: z.boolean(),
  unregistered: z.array(z.string()),
})

const SubmitResultRequest = z.object({
  requestID: z.string(),
  result: ClientToolRegistry.ClientToolResponse,
})

const SubmitResultResponse = z.object({
  success: z.boolean(),
})

const ErrorResponse = z.object({
  error: z.string(),
})

// ============================================================================
// Routes
// ============================================================================

export const ClientToolsRoute = new Hono()
  // Register client tools
  .post(
    "/register",
    describeRoute({
      description: "Register client tools for a client",
      operationId: "clientTools.register",
      requestBody: {
        content: {
          "application/json": {
            schema: resolver(RegisterRequest) as any,
          },
        },
      },
      responses: {
        200: {
          description: "Tools registered successfully",
          content: {
            "application/json": {
              schema: resolver(RegisterResponse),
            },
          },
        },
      },
    }),
    validator("json", RegisterRequest),
    async (c) => {
      const { clientID, tools } = c.req.valid("json")
      log.info("registering tools", { clientID, count: tools.length })

      const registered = ClientToolRegistry.register(clientID, tools)

      return c.json({ registered })
    },
  )

  // Unregister client tools
  .delete(
    "/unregister",
    describeRoute({
      description: "Unregister client tools",
      operationId: "clientTools.unregister",
      requestBody: {
        content: {
          "application/json": {
            schema: resolver(UnregisterRequest) as any,
          },
        },
      },
      responses: {
        200: {
          description: "Tools unregistered successfully",
          content: {
            "application/json": {
              schema: resolver(UnregisterResponse),
            },
          },
        },
      },
    }),
    validator("json", UnregisterRequest),
    async (c) => {
      const { clientID, toolIDs } = c.req.valid("json")
      log.info("unregistering tools", { clientID, toolIDs })

      const unregistered = ClientToolRegistry.unregister(clientID, toolIDs)

      return c.json({ success: true, unregistered })
    },
  )

  // Submit tool execution result
  .post(
    "/result",
    describeRoute({
      description: "Submit tool execution result from client",
      operationId: "clientTools.result",
      requestBody: {
        content: {
          "application/json": {
            schema: resolver(SubmitResultRequest) as any,
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
        404: {
          description: "Unknown request ID",
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
      const { requestID, result } = c.req.valid("json")
      log.info("submitting result", { requestID, status: result.status })

      const success = ClientToolRegistry.submitResult(requestID, result)

      if (!success) {
        return c.json({ error: "Unknown request ID" }, 404)
      }

      return c.json({ success: true })
    },
  )

  // SSE endpoint for tool execution requests
  .get(
    "/pending/:clientID",
    describeRoute({
      description: "Stream pending tool execution requests to client",
      operationId: "clientTools.pending",
      responses: {
        200: {
          description: "SSE stream of tool requests",
          content: {
            "text/event-stream": {
              schema: resolver(ClientToolRegistry.ClientToolExecutionRequest),
            },
          },
        },
      },
    }),
    async (c) => {
      const clientID = c.req.param("clientID")
      log.info("client connected for tool requests", { clientID })

      return streamSSE(c, async (stream) => {
        // Subscribe to tool request events for this client
        const unsubscribe = Bus.subscribe(ClientToolRegistry.Event.ToolRequest, async (event) => {
          if (event.properties.clientID === clientID) {
            log.info("sending tool request to client", {
              clientID,
              requestID: event.properties.request.requestID,
              tool: event.properties.request.tool,
            })
            await stream.writeSSE({
              event: "tool-request",
              data: JSON.stringify(event.properties.request),
            })
          }
        })

        // Keep connection alive with periodic pings
        const keepAlive = setInterval(async () => {
          try {
            await stream.writeSSE({
              event: "ping",
              data: "",
            })
          } catch {
            // Connection closed
          }
        }, 30000)

        // Wait for disconnect
        await new Promise<void>((resolve) => {
          stream.onAbort(() => {
            log.info("client disconnected", { clientID })
            unsubscribe()
            clearInterval(keepAlive)
            ClientToolRegistry.cleanup(clientID)
            resolve()
          })
        })
      })
    },
  )

  // Get registered tools for a client
  .get(
    "/tools/:clientID",
    describeRoute({
      description: "Get registered tools for a client",
      operationId: "clientTools.getTools",
      responses: {
        200: {
          description: "List of registered tools",
          content: {
            "application/json": {
              schema: resolver(z.array(ClientToolRegistry.ClientToolDefinition)),
            },
          },
        },
      },
    }),
    async (c) => {
      const clientID = c.req.param("clientID")
      const tools = ClientToolRegistry.getTools(clientID)
      return c.json(tools)
    },
  )

  // Get all registered client tools
  .get(
    "/tools",
    describeRoute({
      description: "Get all registered client tools across all clients",
      operationId: "clientTools.getAllTools",
      responses: {
        200: {
          description: "Map of all registered client tools",
          content: {
            "application/json": {
              schema: resolver(z.record(z.string(), ClientToolRegistry.ClientToolDefinition)),
            },
          },
        },
      },
    }),
    async (c) => {
      const tools = ClientToolRegistry.getAllTools()
      return c.json(Object.fromEntries(tools))
    },
  )
