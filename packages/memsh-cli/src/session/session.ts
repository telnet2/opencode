import { MemshClient, type SessionInfo, type ExecuteCommandResult, type MemshClientOptions } from "../client"

/**
 * Session configuration options
 */
export interface SessionOptions extends MemshClientOptions {
  /** Session ID to use (if not provided, a new session will be created) */
  sessionId?: string
}

/**
 * Session represents an active shell session in go-memsh
 *
 * Provides a high-level interface for:
 * - Executing commands
 * - Managing working directory
 * - Reading and writing files
 */
export class Session {
  private client: MemshClient
  private sessionInfo: SessionInfo | null = null
  private _connected = false

  constructor(private options: SessionOptions) {
    this.client = new MemshClient(options)
  }

  /**
   * Get session ID
   */
  get id(): string | null {
    return this.sessionInfo?.id ?? this.options.sessionId ?? null
  }

  /**
   * Get current working directory
   */
  get cwd(): string {
    return this.sessionInfo?.cwd ?? "/"
  }

  /**
   * Check if session is connected
   */
  get connected(): boolean {
    return this._connected && this.client.state === "connected"
  }

  /**
   * Get session info
   */
  get info(): SessionInfo | null {
    return this.sessionInfo
  }

  /**
   * Initialize the session
   * Creates a new session or connects to an existing one
   */
  async init(): Promise<void> {
    if (this.options.sessionId) {
      // Use existing session
      const sessions = await this.client.listSessions()
      const existing = sessions.sessions.find((s) => s.id === this.options.sessionId)
      if (!existing) {
        throw new Error(`Session not found: ${this.options.sessionId}`)
      }
      this.sessionInfo = existing
    } else {
      // Create new session
      const response = await this.client.createSession()
      this.sessionInfo = response.session
    }

    // Connect to WebSocket
    await this.client.connect()
    this._connected = true
  }

  /**
   * Execute a shell command
   */
  async execute(command: string): Promise<ExecuteCommandResult> {
    if (!this.sessionInfo) {
      throw new Error("Session not initialized. Call init() first.")
    }

    const result = await this.client.executeCommand(this.sessionInfo.id, command)

    // Update cwd from result
    if (result.cwd) {
      this.sessionInfo = {
        ...this.sessionInfo,
        cwd: result.cwd,
        last_used: new Date().toISOString(),
      }
    }

    return result
  }

  /**
   * Execute a command and return the output as a string
   */
  async run(command: string): Promise<string> {
    const result = await this.execute(command)
    if (result.error) {
      throw new Error(result.error)
    }
    return result.output.join("\n")
  }

  /**
   * Execute a command and return both output and error
   */
  async runSafe(command: string): Promise<{ output: string; error?: string; cwd: string }> {
    const result = await this.execute(command)
    return {
      output: result.output.join("\n"),
      error: result.error,
      cwd: result.cwd,
    }
  }

  /**
   * Change working directory
   */
  async cd(path: string): Promise<string> {
    const result = await this.execute(`cd ${this.escapePath(path)}`)
    if (result.error) {
      throw new Error(result.error)
    }
    return result.cwd
  }

  /**
   * Get current working directory
   */
  async pwd(): Promise<string> {
    const result = await this.run("pwd")
    return result.trim()
  }

  /**
   * Read a file
   */
  async readFile(path: string): Promise<string> {
    return this.run(`cat ${this.escapePath(path)}`)
  }

  /**
   * Write content to a file
   */
  async writeFile(path: string, content: string): Promise<void> {
    // Use a heredoc to write multi-line content
    const escapedContent = content.replace(/'/g, "'\\''")
    await this.run(`cat > ${this.escapePath(path)} << 'MEMSH_EOF'\n${content}\nMEMSH_EOF`)
  }

  /**
   * Append content to a file
   */
  async appendFile(path: string, content: string): Promise<void> {
    await this.run(`cat >> ${this.escapePath(path)} << 'MEMSH_EOF'\n${content}\nMEMSH_EOF`)
  }

  /**
   * Check if a file exists
   */
  async exists(path: string): Promise<boolean> {
    const result = await this.runSafe(`test -e ${this.escapePath(path)} && echo "exists"`)
    return result.output.trim() === "exists"
  }

  /**
   * Check if a path is a directory
   */
  async isDirectory(path: string): Promise<boolean> {
    const result = await this.runSafe(`test -d ${this.escapePath(path)} && echo "dir"`)
    return result.output.trim() === "dir"
  }

  /**
   * Check if a path is a file
   */
  async isFile(path: string): Promise<boolean> {
    const result = await this.runSafe(`test -f ${this.escapePath(path)} && echo "file"`)
    return result.output.trim() === "file"
  }

  /**
   * Create a directory
   */
  async mkdir(path: string, options?: { recursive?: boolean }): Promise<void> {
    const flags = options?.recursive ? "-p" : ""
    await this.run(`mkdir ${flags} ${this.escapePath(path)}`)
  }

  /**
   * Remove a file or directory
   */
  async rm(path: string, options?: { recursive?: boolean; force?: boolean }): Promise<void> {
    const flags = [options?.recursive ? "-r" : "", options?.force ? "-f" : ""].filter(Boolean).join("")
    await this.run(`rm ${flags} ${this.escapePath(path)}`)
  }

  /**
   * List directory contents
   */
  async ls(path?: string, options?: { all?: boolean; long?: boolean }): Promise<string[]> {
    const flags = [options?.all ? "-a" : "", options?.long ? "-l" : ""].filter(Boolean).join("")
    const targetPath = path ? this.escapePath(path) : "."
    const result = await this.run(`ls ${flags} ${targetPath}`)
    return result.split("\n").filter(Boolean)
  }

  /**
   * Close the session
   */
  async close(removeSession = false): Promise<void> {
    if (removeSession && this.sessionInfo) {
      await this.client.removeSession(this.sessionInfo.id)
    }
    this.client.disconnect()
    this._connected = false
  }

  /**
   * Escape a path for shell usage
   */
  private escapePath(path: string): string {
    // Simple escaping - wrap in single quotes and escape single quotes within
    return `'${path.replace(/'/g, "'\\''")}'`
  }
}

/**
 * Create and initialize a new session
 */
export async function createSession(options: SessionOptions): Promise<Session> {
  const session = new Session(options)
  await session.init()
  return session
}
