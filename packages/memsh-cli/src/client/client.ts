import type {
  MemshClientOptions,
  ConnectionState,
  JSONRPCRequest,
  JSONRPCResponse,
  CreateSessionResponse,
  ListSessionsResponse,
  RemoveSessionRequest,
  RemoveSessionResponse,
  ExecuteCommandParams,
  ExecuteCommandResult,
} from "./types"

/**
 * MemshClient - WebSocket JSON-RPC client for go-memsh service
 *
 * Provides methods to:
 * - Manage sessions (create, list, remove)
 * - Execute shell commands in sessions
 * - Handle connection lifecycle
 */
export class MemshClient {
  private options: Required<MemshClientOptions>
  private ws: WebSocket | null = null
  private requestId = 0
  private pendingRequests: Map<
    number | string,
    {
      resolve: (value: unknown) => void
      reject: (error: Error) => void
      timeout: ReturnType<typeof setTimeout>
    }
  > = new Map()
  private connectionState: ConnectionState = "disconnected"
  private reconnectAttempts = 0

  constructor(options: MemshClientOptions) {
    this.options = {
      baseUrl: options.baseUrl,
      timeout: options.timeout ?? 30000,
      autoReconnect: options.autoReconnect ?? false,
      maxReconnectAttempts: options.maxReconnectAttempts ?? 5,
      reconnectDelay: options.reconnectDelay ?? 1000,
    }
  }

  /**
   * Get current connection state
   */
  get state(): ConnectionState {
    return this.connectionState
  }

  /**
   * Get base URL for REST API calls
   */
  private get restBaseUrl(): string {
    return this.options.baseUrl
  }

  /**
   * Get WebSocket URL for REPL connection
   */
  private get wsUrl(): string {
    const url = new URL(this.options.baseUrl)
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:"
    url.pathname = "/api/v1/session/repl"
    return url.toString()
  }

  /**
   * Create a new session
   */
  async createSession(): Promise<CreateSessionResponse> {
    const response = await fetch(`${this.restBaseUrl}/api/v1/session/create`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
    })

    if (!response.ok) {
      throw new Error(`Failed to create session: ${response.statusText}`)
    }

    return response.json() as Promise<CreateSessionResponse>
  }

  /**
   * List all active sessions
   */
  async listSessions(): Promise<ListSessionsResponse> {
    const response = await fetch(`${this.restBaseUrl}/api/v1/session/list`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
    })

    if (!response.ok) {
      throw new Error(`Failed to list sessions: ${response.statusText}`)
    }

    return response.json() as Promise<ListSessionsResponse>
  }

  /**
   * Remove a session
   */
  async removeSession(sessionId: string): Promise<RemoveSessionResponse> {
    const request: RemoveSessionRequest = { session_id: sessionId }

    const response = await fetch(`${this.restBaseUrl}/api/v1/session/remove`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(request),
    })

    if (!response.ok) {
      throw new Error(`Failed to remove session: ${response.statusText}`)
    }

    return response.json() as Promise<RemoveSessionResponse>
  }

  /**
   * Connect to the WebSocket REPL endpoint
   */
  async connect(): Promise<void> {
    if (this.connectionState === "connected" || this.connectionState === "connecting") {
      return
    }

    this.connectionState = "connecting"

    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.wsUrl)

        const connectionTimeout = setTimeout(() => {
          if (this.connectionState === "connecting") {
            this.ws?.close()
            reject(new Error("Connection timeout"))
          }
        }, this.options.timeout)

        this.ws.onopen = () => {
          clearTimeout(connectionTimeout)
          this.connectionState = "connected"
          this.reconnectAttempts = 0
          resolve()
        }

        this.ws.onclose = () => {
          this.handleDisconnect()
        }

        this.ws.onerror = (error) => {
          clearTimeout(connectionTimeout)
          if (this.connectionState === "connecting") {
            reject(new Error(`WebSocket connection error: ${error}`))
          }
        }

        this.ws.onmessage = (event) => {
          this.handleMessage(event.data as string)
        }
      } catch (error) {
        this.connectionState = "disconnected"
        reject(error)
      }
    })
  }

  /**
   * Disconnect from the WebSocket
   */
  disconnect(): void {
    if (this.ws) {
      this.options.autoReconnect = false // Prevent reconnection
      this.ws.close()
      this.ws = null
    }
    this.connectionState = "disconnected"
    this.clearPendingRequests(new Error("Client disconnected"))
  }

  /**
   * Execute a shell command in a session
   */
  async execute(params: ExecuteCommandParams): Promise<ExecuteCommandResult> {
    if (this.connectionState !== "connected") {
      await this.connect()
    }

    return this.sendRequest<ExecuteCommandResult>("shell.execute", params)
  }

  /**
   * Execute a shell command with raw command string
   * Parses the command string into command and args
   */
  async executeCommand(sessionId: string, commandString: string): Promise<ExecuteCommandResult> {
    // For complex commands with pipes, redirections, etc., pass the whole thing as the command
    // The shell will handle parsing
    return this.execute({
      session_id: sessionId,
      command: commandString,
      args: [],
    })
  }

  /**
   * Send a JSON-RPC request
   */
  private sendRequest<T>(method: string, params: Record<string, unknown>): Promise<T> {
    return new Promise((resolve, reject) => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        reject(new Error("WebSocket is not connected"))
        return
      }

      const id = ++this.requestId
      const request: JSONRPCRequest = {
        jsonrpc: "2.0",
        method,
        params,
        id,
      }

      const timeout = setTimeout(() => {
        this.pendingRequests.delete(id)
        reject(new Error(`Request timeout for method: ${method}`))
      }, this.options.timeout)

      this.pendingRequests.set(id, {
        resolve: resolve as (value: unknown) => void,
        reject,
        timeout,
      })

      this.ws.send(JSON.stringify(request))
    })
  }

  /**
   * Handle incoming WebSocket messages
   */
  private handleMessage(data: string): void {
    try {
      const response: JSONRPCResponse = JSON.parse(data)

      if (response.id === null) {
        // Notification, ignore
        return
      }

      const pending = this.pendingRequests.get(response.id)
      if (!pending) {
        return
      }

      this.pendingRequests.delete(response.id)
      clearTimeout(pending.timeout)

      if (response.error) {
        pending.reject(new Error(`JSON-RPC Error [${response.error.code}]: ${response.error.message}`))
      } else {
        pending.resolve(response.result)
      }
    } catch (error) {
      console.error("Failed to parse WebSocket message:", error)
    }
  }

  /**
   * Handle WebSocket disconnect
   */
  private handleDisconnect(): void {
    const wasConnected = this.connectionState === "connected"
    this.connectionState = "disconnected"
    this.ws = null

    // Reject all pending requests
    this.clearPendingRequests(new Error("Connection lost"))

    // Attempt reconnection if enabled
    if (wasConnected && this.options.autoReconnect && this.reconnectAttempts < this.options.maxReconnectAttempts) {
      this.reconnectAttempts++
      this.connectionState = "reconnecting"

      setTimeout(() => {
        this.connect().catch(() => {
          // Reconnection failed, will be handled by next attempt
        })
      }, this.options.reconnectDelay * this.reconnectAttempts)
    }
  }

  /**
   * Clear all pending requests with an error
   */
  private clearPendingRequests(error: Error): void {
    for (const [id, pending] of this.pendingRequests) {
      clearTimeout(pending.timeout)
      pending.reject(error)
    }
    this.pendingRequests.clear()
  }
}

/**
 * Create a new MemshClient instance
 */
export function createClient(options: MemshClientOptions): MemshClient {
  return new MemshClient(options)
}
