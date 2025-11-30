import path from "path"
import fs from "fs/promises"
import { Global } from "../global"

/**
 * Traffic logging module for capturing client-server and server-provider communication.
 *
 * Environment variables:
 * - OPENCODE_TRAFFIC_LOG: Enable traffic logging (set to "1" or "true")
 * - OPENCODE_CLIENT_TRAFFIC_LOG: Path to client-server traffic log file
 * - OPENCODE_PROVIDER_TRAFFIC_LOG: Path to server-provider traffic log file
 */
export namespace Traffic {
  export type TrafficType = "client" | "provider"

  export interface RequestEntry {
    id: string
    type: "request"
    timestamp: string
    method: string
    url: string
    headers?: Record<string, string>
    body?: string
  }

  export interface ResponseEntry {
    id: string
    type: "response"
    timestamp: string
    status: number
    statusText?: string
    headers?: Record<string, string>
    duration: number
  }

  export interface StreamChunkEntry {
    id: string
    type: "stream-chunk"
    timestamp: string
    chunk: string
    index: number
  }

  export interface StreamEndEntry {
    id: string
    type: "stream-end"
    timestamp: string
    totalChunks: number
    totalBytes: number
    duration: number
  }

  export interface ErrorEntry {
    id: string
    type: "error"
    timestamp: string
    error: string
    duration?: number
  }

  export type LogEntry = RequestEntry | ResponseEntry | StreamChunkEntry | StreamEndEntry | ErrorEntry

  // Default log directory
  const defaultLogDir = path.join(Global.Path.data, "traffic")

  // Check if traffic logging is enabled
  export function isEnabled(): boolean {
    const env = process.env["OPENCODE_TRAFFIC_LOG"]
    return env === "1" || env === "true"
  }

  // Get the log file path for a traffic type
  function getLogPath(trafficType: TrafficType): string {
    if (trafficType === "client") {
      return process.env["OPENCODE_CLIENT_TRAFFIC_LOG"] ?? path.join(defaultLogDir, "client.log")
    } else {
      return process.env["OPENCODE_PROVIDER_TRAFFIC_LOG"] ?? path.join(defaultLogDir, "provider.log")
    }
  }

  // File writers cache
  const writers: Map<string, ReturnType<ReturnType<typeof Bun.file>["writer"]>> = new Map()

  // Ensure log directory exists
  let initialized = false
  async function ensureInit(): Promise<void> {
    if (initialized) return
    initialized = true

    const clientPath = getLogPath("client")
    const providerPath = getLogPath("provider")

    await Promise.all([
      fs.mkdir(path.dirname(clientPath), { recursive: true }),
      fs.mkdir(path.dirname(providerPath), { recursive: true }),
    ])
  }

  // Get or create a file writer
  async function getWriter(trafficType: TrafficType): Promise<ReturnType<ReturnType<typeof Bun.file>["writer"]>> {
    await ensureInit()

    const logPath = getLogPath(trafficType)
    let writer = writers.get(logPath)

    if (!writer) {
      const file = Bun.file(logPath)
      writer = file.writer()
      writers.set(logPath, writer)
    }

    return writer
  }

  // Write a log entry
  async function writeEntry(trafficType: TrafficType, entry: LogEntry): Promise<void> {
    if (!isEnabled()) return

    try {
      const writer = await getWriter(trafficType)
      const line = JSON.stringify(entry) + "\n"
      writer.write(line)
      writer.flush()
    } catch (error) {
      // Silently fail to avoid affecting main behavior
      console.error("[Traffic] Failed to write log entry:", error)
    }
  }

  // Generate a unique request ID
  export function generateRequestId(): string {
    return `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`
  }

  // Truncate body for logging (to avoid huge log files)
  const MAX_BODY_SIZE = 50 * 1024 // 50KB

  function truncateBody(body: string | undefined): string | undefined {
    if (!body) return undefined
    if (body.length <= MAX_BODY_SIZE) return body
    return body.substring(0, MAX_BODY_SIZE) + `\n... [truncated, total ${body.length} bytes]`
  }

  // Serialize headers for logging
  function serializeHeaders(headers: Headers | HeadersInit | undefined): Record<string, string> | undefined {
    if (!headers) return undefined

    if (headers instanceof Headers) {
      const result: Record<string, string> = {}
      headers.forEach((value, key) => {
        // Mask sensitive headers
        if (key.toLowerCase() === "authorization" || key.toLowerCase() === "x-api-key") {
          result[key] = "[REDACTED]"
        } else {
          result[key] = value
        }
      })
      return result
    }

    if (Array.isArray(headers)) {
      const result: Record<string, string> = {}
      for (const [key, value] of headers) {
        if (key.toLowerCase() === "authorization" || key.toLowerCase() === "x-api-key") {
          result[key] = "[REDACTED]"
        } else {
          result[key] = value
        }
      }
      return result
    }

    const result: Record<string, string> = {}
    for (const [key, value] of Object.entries(headers)) {
      if (key.toLowerCase() === "authorization" || key.toLowerCase() === "x-api-key") {
        result[key] = "[REDACTED]"
      } else {
        result[key] = String(value)
      }
    }
    return result
  }

  // Log a request
  export async function logRequest(
    trafficType: TrafficType,
    requestId: string,
    input: {
      method: string
      url: string
      headers?: Headers | HeadersInit
      body?: string
    },
  ): Promise<void> {
    const entry: RequestEntry = {
      id: requestId,
      type: "request",
      timestamp: new Date().toISOString(),
      method: input.method,
      url: input.url,
      headers: serializeHeaders(input.headers),
      body: truncateBody(input.body),
    }
    await writeEntry(trafficType, entry)
  }

  // Log a response
  export async function logResponse(
    trafficType: TrafficType,
    requestId: string,
    input: {
      status: number
      statusText?: string
      headers?: Headers
      duration: number
    },
  ): Promise<void> {
    const entry: ResponseEntry = {
      id: requestId,
      type: "response",
      timestamp: new Date().toISOString(),
      status: input.status,
      statusText: input.statusText,
      headers: serializeHeaders(input.headers),
      duration: input.duration,
    }
    await writeEntry(trafficType, entry)
  }

  // Log a stream chunk
  export async function logStreamChunk(
    trafficType: TrafficType,
    requestId: string,
    chunk: string,
    index: number,
  ): Promise<void> {
    const entry: StreamChunkEntry = {
      id: requestId,
      type: "stream-chunk",
      timestamp: new Date().toISOString(),
      chunk: truncateBody(chunk) ?? "",
      index,
    }
    await writeEntry(trafficType, entry)
  }

  // Log stream end
  export async function logStreamEnd(
    trafficType: TrafficType,
    requestId: string,
    input: {
      totalChunks: number
      totalBytes: number
      duration: number
    },
  ): Promise<void> {
    const entry: StreamEndEntry = {
      id: requestId,
      type: "stream-end",
      timestamp: new Date().toISOString(),
      totalChunks: input.totalChunks,
      totalBytes: input.totalBytes,
      duration: input.duration,
    }
    await writeEntry(trafficType, entry)
  }

  // Log an error
  export async function logError(
    trafficType: TrafficType,
    requestId: string,
    error: Error | string,
    duration?: number,
  ): Promise<void> {
    const entry: ErrorEntry = {
      id: requestId,
      type: "error",
      timestamp: new Date().toISOString(),
      error: error instanceof Error ? error.message : error,
      duration,
    }
    await writeEntry(trafficType, entry)
  }

  /**
   * Create a wrapped fetch function that logs all traffic.
   * This wraps the response body stream to log chunks without affecting behavior.
   */
  export function createLoggingFetch(
    trafficType: TrafficType,
    baseFetch: typeof fetch = fetch,
  ): typeof fetch {
    if (!isEnabled()) {
      return baseFetch
    }

    return async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
      const requestId = generateRequestId()
      const startTime = Date.now()

      // Extract request details
      const url = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url
      const method = init?.method ?? (typeof input === "object" && "method" in input ? input.method : "GET")

      // Log request
      let bodyStr: string | undefined
      if (init?.body) {
        if (typeof init.body === "string") {
          bodyStr = init.body
        } else if (init.body instanceof ArrayBuffer) {
          bodyStr = new TextDecoder().decode(init.body)
        } else if (init.body instanceof Uint8Array) {
          bodyStr = new TextDecoder().decode(init.body)
        }
      }

      await logRequest(trafficType, requestId, {
        method,
        url,
        headers: init?.headers,
        body: bodyStr,
      })

      try {
        const response = await baseFetch(input, init)
        const duration = Date.now() - startTime

        // Log response headers
        await logResponse(trafficType, requestId, {
          status: response.status,
          statusText: response.statusText,
          headers: response.headers,
          duration,
        })

        // If response has a body, wrap it to log stream chunks
        if (response.body) {
          const originalBody = response.body
          let chunkIndex = 0
          let totalBytes = 0
          const streamStartTime = Date.now()

          const wrappedStream = new ReadableStream({
            async start(controller) {
              const reader = originalBody.getReader()
              try {
                while (true) {
                  const { done, value } = await reader.read()
                  if (done) {
                    // Log stream end
                    await logStreamEnd(trafficType, requestId, {
                      totalChunks: chunkIndex,
                      totalBytes,
                      duration: Date.now() - streamStartTime,
                    })
                    controller.close()
                    break
                  }

                  // Log the chunk
                  const chunkStr = new TextDecoder().decode(value)
                  totalBytes += value.byteLength
                  await logStreamChunk(trafficType, requestId, chunkStr, chunkIndex++)

                  // Forward the chunk
                  controller.enqueue(value)
                }
              } catch (error) {
                await logError(trafficType, requestId, error as Error, Date.now() - streamStartTime)
                controller.error(error)
              }
            },
          })

          // Create new response with wrapped body
          return new Response(wrappedStream, {
            status: response.status,
            statusText: response.statusText,
            headers: response.headers,
          })
        }

        return response
      } catch (error) {
        const duration = Date.now() - startTime
        await logError(trafficType, requestId, error as Error, duration)
        throw error
      }
    }
  }

  /**
   * Get log file paths for debugging/info purposes.
   */
  export function getLogPaths(): { client: string; provider: string } {
    return {
      client: getLogPath("client"),
      provider: getLogPath("provider"),
    }
  }

  /**
   * Flush all writers and close them.
   */
  export async function flush(): Promise<void> {
    for (const writer of writers.values()) {
      try {
        writer.flush()
      } catch {
        // Ignore flush errors
      }
    }
  }
}
