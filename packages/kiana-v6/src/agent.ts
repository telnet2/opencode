import { ToolLoopAgent, tool, jsonSchema, stepCountIs } from "ai"
import type { LanguageModel } from "ai"
import { z } from "zod"
import { spawn } from "node:child_process"
import * as fs from "node:fs"
import * as path from "node:path"
import type { Tool, ToolContext, ToolResult } from "./tool/tool.js"
import { ALL_TOOLS, DEFAULT_TOOLS, setSubagentExecutor } from "./tool/index.js"
import { initializeMCPServers, getMCPManager, type MCPServerConfig } from "./tool/mcp.js"
import type { SessionInfo } from "./types/session.js"
import type { MessageInfo } from "./types/message.js"
import type {
  Part,
  TextPart,
  ToolPart,
} from "./types/part.js"
import type {
  StreamPart,
  StartPart,
  TextStartPart,
  TextDeltaPart,
  TextEndPart,
  ToolInputAvailablePart,
  ToolOutputAvailablePart,
  ToolOutputErrorPart,
  StartStepPart,
  FinishStepPart,
  FinishPart,
  DataSessionPart,
  DataSessionIdlePart,
  DataTodoPart,
  TokenUsage,
} from "./stream.js"

// Re-export stream types
export type { StreamPart } from "./stream.js"
export { formatSSE, parseSSE, createSSEHeaders } from "./stream.js"

// ID generation
let idCounter = 0
function generateId(prefix: string): string {
  const timestamp = Date.now().toString(36)
  const counter = (idCounter++).toString(36).padStart(4, "0")
  return `${prefix}_${timestamp}${counter}`
}

// Storage for session persistence
interface SessionStorage {
  sessionDir: string
  save(session: SessionInfo, messages: MessageWithParts[]): void
  load(): { session: SessionInfo; messages: MessageWithParts[] } | null
}

function createSessionStorage(sessionDir: string): SessionStorage {
  const sessionFile = path.join(sessionDir, "session.json")
  const messagesFile = path.join(sessionDir, "messages.json")

  return {
    sessionDir,
    save(session: SessionInfo, messages: MessageWithParts[]) {
      fs.mkdirSync(sessionDir, { recursive: true })
      fs.writeFileSync(sessionFile, JSON.stringify(session, null, 2))
      fs.writeFileSync(messagesFile, JSON.stringify(messages, null, 2))
    },
    load() {
      if (!fs.existsSync(sessionFile) || !fs.existsSync(messagesFile)) {
        return null
      }
      try {
        const session = JSON.parse(fs.readFileSync(sessionFile, "utf-8")) as SessionInfo
        const messages = JSON.parse(fs.readFileSync(messagesFile, "utf-8")) as MessageWithParts[]
        return { session, messages }
      } catch {
        return null
      }
    },
  }
}

interface MessageWithParts {
  info: MessageInfo
  parts: Part[]
}

export interface CodingAgentConfig {
  /** The language model to use */
  model: LanguageModel
  /** Working directory for file operations */
  workingDirectory?: string
  /** System instructions for the agent */
  instructions?: string
  /** Subset of tools to enable (defaults to all) */
  tools?: string[]
  /** Maximum steps before stopping (default: 50) */
  maxSteps?: number
  /** Maximum retries for API calls (default: 3) */
  maxRetries?: number
  /** Session directory for persistence */
  sessionDir?: string
  /** MCP servers to connect to */
  mcpServers?: MCPServerConfig[]
}

export interface GenerateParams {
  /** The prompt/message to send */
  prompt: string
  /** Abort signal for cancellation */
  abortSignal?: AbortSignal
}

export interface StreamParams extends GenerateParams {}

/** Callback for receiving stream parts */
export type StreamCallback = (part: StreamPart) => void

export class CodingAgent {
  readonly id: string
  private messages: MessageWithParts[]
  private tools: Record<string, Tool>
  private workingDirectory: string
  private storage: SessionStorage | null
  private config: CodingAgentConfig
  private abortController: AbortController
  private streamCallbacks: Set<StreamCallback> = new Set()
  private mcpInitialized = false

  constructor(config: CodingAgentConfig) {
    this.config = config
    this.workingDirectory = config.workingDirectory || process.cwd()
    this.storage = config.sessionDir ? createSessionStorage(config.sessionDir) : null
    this.abortController = new AbortController()

    // Load existing session or create new
    const existing = this.storage?.load()
    this.id = existing?.session.id ?? generateId("session")
    this.messages = existing?.messages ?? []

    // Get the subset of tools
    const toolNames = config.tools ?? DEFAULT_TOOLS
    this.tools = {}
    for (const name of toolNames) {
      if (ALL_TOOLS[name]) {
        this.tools[name] = ALL_TOOLS[name]
      }
    }

    // Setup subagent executor for Task tool
    this.setupSubagentExecutor()
  }

  /**
   * Initialize MCP servers and add their tools
   */
  private async initializeMCP(): Promise<void> {
    if (this.mcpInitialized || !this.config.mcpServers?.length) {
      return
    }

    try {
      const mcpTools = await initializeMCPServers(this.config.mcpServers)
      
      // Add MCP tools to the agent's tool set
      for (const [name, tool] of Object.entries(mcpTools)) {
        this.tools[name] = tool
      }
      
      this.mcpInitialized = true
    } catch (error) {
      console.error("Failed to initialize MCP servers:", error)
      throw error
    }
  }

  /**
   * Cleanup MCP connections
   */
  async cleanup(): Promise<void> {
    if (this.mcpInitialized) {
      const manager = getMCPManager()
      await manager.disconnectAll()
    }
  }

  private emit(part: StreamPart): void {
    for (const cb of this.streamCallbacks) {
      try {
        cb(part)
      } catch (err) {
        console.error("Error in stream callback:", err)
      }
    }
  }

  private setupSubagentExecutor(): void {
    setSubagentExecutor(async (params) => {
      // Create a subagent with the specified agent config
      const subAgent = new CodingAgent({
        ...this.config,
        instructions: params.agentConfig.systemPrompt,
        // Disable task tool in subagents to prevent infinite recursion
        tools: (this.config.tools ?? DEFAULT_TOOLS).filter((t) => !["task", "todowrite", "todoread"].includes(t)),
      })

      // Forward stream parts from subagent with context
      subAgent.onStream((part) => {
        // Add subagent context via data part
        this.emit({
          type: "data-subagent-context",
          data: {
            parentSessionID: params.parentSessionID,
            depth: params.depth,
            agentType: params.agentType,
          },
        })
        this.emit(part)
      })

      // Generate response
      const result = await subAgent.generate({ prompt: params.prompt })

      return {
        output: result.text,
        sessionID: subAgent.id,
      }
    })
  }

  /**
   * Subscribe to stream parts
   */
  onStream(callback: StreamCallback): () => void {
    this.streamCallbacks.add(callback)
    return () => this.streamCallbacks.delete(callback)
  }

  /**
   * Generate a response (non-streaming) - still emits stream parts for compatibility
   */
  async generate(params: GenerateParams): Promise<{ text: string; usage?: any }> {
    // Initialize MCP servers if configured
    await this.initializeMCP()
    
    const { agent, callParams, messageId } = await this.buildAgent(params)

    // Emit session info
    this.emitSessionStart(messageId)

    const result = await agent.generate({
      ...callParams,
      abortSignal: params.abortSignal ?? this.abortController.signal,
    })

    // Emit finish
    this.emit({
      type: "finish",
      finishReason: "stop",
      usage: this.convertUsage(result.usage),
    })

    // Emit session idle
    this.emit({
      type: "data-session-idle",
      data: { sessionID: this.id },
    })

    this.saveSession()

    return {
      text: result.text,
      usage: result.usage,
    }
  }

  /**
   * Stream a response - returns async iterable that emits events via onStream callback
   */
  async stream(params: StreamParams): Promise<{
    textStream: AsyncIterable<string>
    text: Promise<string>
  }> {
    // Initialize MCP servers if configured
    await this.initializeMCP()
    
    const { agent, callParams, messageId, assistantParts } = await this.buildAgent(params)

    // Emit session info
    this.emitSessionStart(messageId)

    const result = await agent.stream({
      ...callParams,
      abortSignal: params.abortSignal ?? this.abortController.signal,
    })

    const self = this
    let textPartId: string | null = null

    // Wrap text stream to emit events AND yield text
    const textStream = (async function* (): AsyncGenerator<string> {
      for await (const chunk of result.textStream) {
        // Emit text-start on first chunk
        if (!textPartId) {
          textPartId = generateId("text")
          self.emit({ type: "text-start", id: textPartId })
        }

        // Emit text-delta
        self.emit({ type: "text-delta", id: textPartId, delta: chunk })

        yield chunk
      }

      // Emit text-end if we started
      if (textPartId) {
        self.emit({ type: "text-end", id: textPartId })
      }
    })()

    // When streaming completes, emit finish events
    const textPromise = result.text.then((text) => {
      self.emit({
        type: "finish",
        finishReason: "stop",
      })

      self.emit({
        type: "data-session-idle",
        data: { sessionID: self.id },
      })

      self.saveSession()
      return text
    })

    return {
      textStream,
      text: textPromise,
    }
  }

  /**
   * Abort the current generation
   */
  abort(): void {
    this.abortController.abort()
    this.abortController = new AbortController()

    this.emit({
      type: "data-session-idle",
      data: { sessionID: this.id },
    })
  }

  /**
   * Get list of available tools
   */
  getTools(): { name: string; description: string }[] {
    return Object.entries(this.tools).map(([name, tool]) => ({
      name,
      description: tool.description,
    }))
  }

  private emitSessionStart(messageId: string): void {
    const sessionInfo: SessionInfo = {
      id: this.id,
      projectID: generateId("project"),
      directory: this.workingDirectory,
      title: "Session",
      version: "1.0.0",
      time: {
        created: Date.now(),
        updated: Date.now(),
      },
    }

    // Emit session data
    this.emit({
      type: "data-session",
      data: sessionInfo,
    })

    // Emit message start
    this.emit({
      type: "start",
      messageId,
      metadata: { sessionID: this.id },
    })
  }

  private async buildAgent(params: GenerateParams): Promise<{
    agent: ToolLoopAgent<never, Record<string, any>, never>
    callParams: { prompt: string } | { messages: any[] }
    messageId: string
    assistantParts: Part[]
  }> {
    // Build system prompt with environment context
    const baseInstructions = this.config.instructions || getDefaultSystemPrompt()
    const envContext = await buildEnvironmentContext(this.workingDirectory)
    const instructions = baseInstructions + "\n" + envContext

    // Create message for tracking
    const userMessageID = generateId("message")
    const userMessage: MessageInfo = {
      id: userMessageID,
      sessionID: this.id,
      role: "user",
      time: { created: Date.now() },
    }

    const userTextPart: TextPart = {
      id: generateId("part"),
      sessionID: this.id,
      messageID: userMessageID,
      type: "text",
      text: params.prompt,
    }

    this.messages.push({
      info: userMessage,
      parts: [userTextPart],
    })

    // Create assistant message for tracking parts
    const assistantMessageID = generateId("message")
    const assistantMessage: MessageInfo = {
      id: assistantMessageID,
      sessionID: this.id,
      role: "assistant",
      parentID: userMessageID,
      time: { created: Date.now() },
      cost: 0,
      tokens: {
        input: 0,
        output: 0,
        reasoning: 0,
        cache: { read: 0, write: 0 },
      },
    }

    const assistantParts: Part[] = []
    this.messages.push({ info: assistantMessage, parts: assistantParts })

    // Build v6 tools
    const aiTools = this.buildAITools(assistantMessageID, assistantParts)

    // Build call params - use messages if we have history, otherwise just prompt
    const historyMessages = this.buildAIMessages()
    // Remove the last user message since that's the current prompt
    const previousMessages = historyMessages.slice(0, -1)

    // If we have previous messages, add current prompt to them
    // Otherwise just use the prompt directly
    const callParams: { prompt: string } | { messages: any[] } = previousMessages.length > 0
      ? { messages: [...previousMessages, { role: "user", content: params.prompt }] }
      : { prompt: params.prompt }

    const self = this

    const agent = new ToolLoopAgent({
      model: this.config.model,
      instructions,
      tools: aiTools,
      stopWhen: stepCountIs(this.config.maxSteps ?? 50),
      maxRetries: this.config.maxRetries ?? 3,
      onStepFinish: (step) => {
        // Emit step finish
        const finishStepPart: FinishStepPart = {
          type: "finish-step",
          usage: self.convertUsage(step.usage),
        }
        self.emit(finishStepPart)

        // Update message token counts
        self.updateTokensFromUsage(assistantMessage, step.usage)
      },
      onFinish: (event) => {
        // Update final token counts
        self.updateTokensFromUsage(assistantMessage, event.totalUsage)
        assistantMessage.time.completed = Date.now()
      },
    })

    return { agent, callParams, messageId: assistantMessageID, assistantParts }
  }

  private buildAITools(messageID: string, parts: Part[]): Record<string, any> {
    const aiTools: Record<string, any> = {}
    const self = this

    for (const [name, toolDef] of Object.entries(this.tools)) {
      const schema = z.toJSONSchema(toolDef.parameters)

      aiTools[name] = tool({
        description: toolDef.description,
        inputSchema: jsonSchema<any>(schema as any),
        // Called when tool input is fully available - emit tool-input-available
        onInputAvailable: async ({ input, toolCallId }: { input: any; toolCallId: string }) => {
          // Emit start-step for tool execution
          self.emit({ type: "start-step" })

          // Emit tool-input-available (AI SDK format)
          const toolInputPart: ToolInputAvailablePart = {
            type: "tool-input-available",
            toolCallId,
            toolName: name,
            input: input as Record<string, unknown>,
          }
          self.emit(toolInputPart)

          // Also track internally
          const toolPart: ToolPart = {
            id: generateId("part"),
            sessionID: self.id,
            messageID,
            type: "tool",
            tool: name,
            callID: toolCallId,
            state: {
              status: "running",
              input: input as Record<string, unknown>,
              time: { start: Date.now() },
            },
          }
          parts.push(toolPart)
        },
        execute: async (args: any, options: { abortSignal?: AbortSignal; toolCallId: string }) => {
          const ctx: ToolContext = {
            sessionID: self.id,
            messageID,
            workingDirectory: self.workingDirectory,
            abort: options.abortSignal ?? new AbortController().signal,
            metadata: (val) => {
              // Find and update the tool part
              const toolPart = parts.find(
                (p): p is ToolPart => p.type === "tool" && p.callID === options.toolCallId
              )
              if (toolPart && toolPart.state.status === "running") {
                toolPart.state = {
                  ...toolPart.state,
                  title: val.title,
                  metadata: val.metadata,
                }
              }
            },
          }

          try {
            const result = await toolDef.execute(args, ctx)

            // Emit tool-output-available (AI SDK format)
            const toolOutputPart: ToolOutputAvailablePart = {
              type: "tool-output-available",
              toolCallId: options.toolCallId,
              output: result.output,
              metadata: {
                title: result.title,
                ...result.metadata,
              },
            }
            self.emit(toolOutputPart)

            // Update internal tracking
            const toolPart = parts.find(
              (p): p is ToolPart => p.type === "tool" && p.callID === options.toolCallId
            )
            if (toolPart) {
              toolPart.state = {
                status: "completed",
                input: toolPart.state.input,
                output: result.output,
                title: result.title,
                metadata: result.metadata,
                time: {
                  start: (toolPart.state as any).time?.start ?? Date.now(),
                  end: Date.now(),
                },
              }
            }

            return result
          } catch (error) {
            const errorMsg = error instanceof Error ? error.message : String(error)

            // Emit tool-output-error (AI SDK format)
            const toolErrorPart: ToolOutputErrorPart = {
              type: "tool-output-error",
              toolCallId: options.toolCallId,
              error: errorMsg,
            }
            self.emit(toolErrorPart)

            // Update internal tracking
            const toolPart = parts.find(
              (p): p is ToolPart => p.type === "tool" && p.callID === options.toolCallId
            )
            if (toolPart) {
              toolPart.state = {
                status: "error",
                input: toolPart.state.input,
                error: errorMsg,
                time: {
                  start: (toolPart.state as any).time?.start ?? Date.now(),
                  end: Date.now(),
                },
              }
            }
            throw error
          }
        },
        toModelOutput: (result: ToolResult) => ({
          type: "text",
          value: result.output,
        }),
      })
    }

    return aiTools
  }

  private buildAIMessages(): any[] {
    const aiMessages: any[] = []

    let i = 0
    while (i < this.messages.length) {
      const msg = this.messages[i]

      if (msg.info.role === "user") {
        const textParts = msg.parts.filter((p): p is TextPart => p.type === "text")
        if (textParts.length > 0) {
          aiMessages.push({
            role: "user",
            content: textParts.map((p) => p.text).join("\n"),
          })
        }
        i++
      } else if (msg.info.role === "assistant") {
        // Collect all parts from consecutive assistant messages
        const allTextParts: TextPart[] = []
        const allToolParts: ToolPart[] = []

        while (i < this.messages.length && this.messages[i].info.role === "assistant") {
          const assistantMsg = this.messages[i]
          const textParts = assistantMsg.parts.filter(
            (p): p is TextPart => p.type === "text" && !!p.text
          )
          const toolParts = assistantMsg.parts.filter(
            (p): p is ToolPart =>
              p.type === "tool" && (p.state.status === "completed" || p.state.status === "error")
          )
          allTextParts.push(...textParts)
          allToolParts.push(...toolParts)
          i++
        }

        if (allToolParts.length > 0) {
          // Build assistant message with text and tool calls
          const content: any[] = []

          for (const textPart of allTextParts) {
            content.push({ type: "text", text: textPart.text })
          }

          for (const toolPart of allToolParts) {
            content.push({
              type: "tool-call",
              toolCallId: toolPart.callID,
              toolName: toolPart.tool,
              input: toolPart.state.input,
            })
          }

          if (content.length > 0) {
            aiMessages.push({
              role: "assistant",
              content,
            })
          }

          // Add tool results
          const toolResultContent: any[] = []
          for (const toolPart of allToolParts) {
            const isError = toolPart.state.status === "error"
            const outputValue = isError
              ? `Error: ${(toolPart.state as any).error}`
              : (toolPart.state as any).output ?? ""

            toolResultContent.push({
              type: "tool-result",
              toolCallId: toolPart.callID,
              toolName: toolPart.tool,
              output: {
                type: isError ? "error-text" : "text",
                value: String(outputValue),
              },
            })
          }

          if (toolResultContent.length > 0) {
            aiMessages.push({
              role: "tool",
              content: toolResultContent,
            })
          }
        } else if (allTextParts.length > 0) {
          aiMessages.push({
            role: "assistant",
            content: allTextParts.map((p) => p.text).join("\n"),
          })
        }
      } else {
        i++
      }
    }

    return aiMessages
  }

  private convertUsage(usage: any): TokenUsage | undefined {
    if (!usage) return undefined

    let inputTotal = 0
    let outputTotal = 0
    let cachedRead = 0
    let reasoning = 0

    if (typeof usage.inputTokens === "object" && usage.inputTokens !== null) {
      inputTotal = usage.inputTokens.total ?? 0
      cachedRead = usage.inputTokens.cacheRead ?? 0
    } else {
      inputTotal = usage.inputTokens ?? usage.promptTokens ?? 0
      cachedRead = usage.cachedInputTokens ?? 0
    }

    if (typeof usage.outputTokens === "object" && usage.outputTokens !== null) {
      outputTotal = usage.outputTokens.total ?? 0
      reasoning = usage.outputTokens.reasoning ?? 0
    } else {
      outputTotal = usage.outputTokens ?? usage.completionTokens ?? 0
      reasoning = usage.reasoningTokens ?? 0
    }

    return {
      inputTokens: inputTotal,
      outputTokens: outputTotal,
      reasoningTokens: reasoning,
      cacheReadTokens: cachedRead,
    }
  }

  private updateTokensFromUsage(message: MessageInfo, usage: any): void {
    if (!usage) return

    const converted = this.convertUsage(usage)
    if (!converted) return

    ;(message as any).tokens = {
      input: (converted.inputTokens ?? 0) - (converted.cacheReadTokens ?? 0),
      output: converted.outputTokens ?? 0,
      reasoning: converted.reasoningTokens ?? 0,
      cache: {
        read: converted.cacheReadTokens ?? 0,
        write: 0,
      },
    }
  }

  private saveSession(): void {
    if (this.storage) {
      const sessionInfo: SessionInfo = {
        id: this.id,
        projectID: generateId("project"),
        directory: this.workingDirectory,
        title: "Session",
        version: "1.0.0",
        time: {
          created: Date.now(),
          updated: Date.now(),
        },
      }
      this.storage.save(sessionInfo, this.messages)
    }
  }
}

function getDefaultSystemPrompt(): string {
  return `You are Kiana, a powerful coding agent running in headless mode.

You help users with software engineering tasks using the tools available to you. You are running non-interactively, so you cannot ask clarifying questions - make reasonable assumptions and proceed.

IMPORTANT: You must NEVER generate or guess URLs unless you are confident they are for programming help.

# Tone and style
- No emojis unless explicitly requested.
- Responses should be short and concise.
- Use Github-flavored markdown for formatting.
- Output text to communicate; use tools only to complete tasks.
- NEVER create files unless absolutely necessary. Prefer editing existing files.

# Professional objectivity
Prioritize technical accuracy over validation. Focus on facts and problem-solving with direct, objective technical info.

# Non-interactive mode
Since you are running headless without user interaction:
- Do not ask for clarification - make reasonable assumptions
- Complete tasks autonomously
- Report progress and results via tool outputs

# Tool usage
- Call multiple tools in parallel when independent.
- Use specialized tools (Read, Write, Edit, Glob, Grep) instead of bash equivalents.

# Code References
When referencing code, include \`file_path:line_number\` for navigation.`
}

/**
 * Build environment context for the system prompt.
 */
async function buildEnvironmentContext(workingDirectory: string): Promise<string> {
  const isGitRepo = await checkIsGitRepo(workingDirectory)
  const gitBranch = isGitRepo ? await getGitBranch(workingDirectory) : null
  const gitStatus = isGitRepo ? await getGitStatus(workingDirectory) : null
  const fileTree = await getFileTree(workingDirectory, 200)

  const lines = [
    `Here is useful information about the environment you are running in:`,
    `<env>`,
    `Working directory: ${workingDirectory}`,
    `Is directory a git repo: ${isGitRepo ? "Yes" : "No"}`,
  ]

  if (gitBranch) {
    lines.push(`Current branch: ${gitBranch}`)
  }

  lines.push(`Platform: ${process.platform}`)
  lines.push(`Today's date: ${new Date().toISOString().split("T")[0]}`)
  lines.push(`</env>`)

  if (gitStatus) {
    lines.push(``)
    lines.push(`<git_status>`)
    lines.push(gitStatus)
    lines.push(`</git_status>`)
  }

  if (fileTree) {
    lines.push(``)
    lines.push(`<files>`)
    lines.push(fileTree)
    lines.push(`</files>`)
  }

  return lines.join("\n")
}

async function checkIsGitRepo(cwd: string): Promise<boolean> {
  return new Promise((resolve) => {
    const proc = spawn("git", ["rev-parse", "--is-inside-work-tree"], {
      cwd,
      stdio: ["ignore", "pipe", "pipe"],
    })
    proc.on("close", (code) => resolve(code === 0))
    proc.on("error", () => resolve(false))
  })
}

async function getGitBranch(cwd: string): Promise<string | null> {
  return new Promise((resolve) => {
    const proc = spawn("git", ["branch", "--show-current"], {
      cwd,
      stdio: ["ignore", "pipe", "pipe"],
    })
    let output = ""
    proc.stdout.on("data", (data) => (output += data.toString()))
    proc.on("close", (code) => {
      if (code === 0 && output.trim()) {
        resolve(output.trim())
      } else {
        resolve(null)
      }
    })
    proc.on("error", () => resolve(null))
  })
}

async function getGitStatus(cwd: string): Promise<string | null> {
  return new Promise((resolve) => {
    const proc = spawn("git", ["status", "--short"], {
      cwd,
      stdio: ["ignore", "pipe", "pipe"],
    })
    let output = ""
    proc.stdout.on("data", (data) => (output += data.toString()))
    proc.on("close", (code) => {
      if (code === 0 && output.trim()) {
        const lines = output.trim().split("\n").slice(0, 50)
        if (output.trim().split("\n").length > 50) {
          lines.push("... (truncated)")
        }
        resolve(lines.join("\n"))
      } else {
        resolve(null)
      }
    })
    proc.on("error", () => resolve(null))
  })
}

async function getFileTree(cwd: string, limit: number): Promise<string | null> {
  // Try ripgrep first
  const rgResult = await runCommand("rg", ["--files", "--sort", "path"], cwd, limit)
  if (rgResult) return rgResult

  // Fallback to git ls-files
  const gitResult = await runCommand("git", ["ls-files"], cwd, limit)
  if (gitResult) return gitResult

  // Last resort: find
  const findResult = await runCommand(
    "find",
    [".", "-type", "f", "-not", "-path", "*/.*", "-not", "-path", "*/node_modules/*"],
    cwd,
    limit
  )
  return findResult
}

async function runCommand(
  cmd: string,
  args: string[],
  cwd: string,
  lineLimit: number
): Promise<string | null> {
  return new Promise((resolve) => {
    const proc = spawn(cmd, args, {
      cwd,
      stdio: ["ignore", "pipe", "pipe"],
    })
    let output = ""
    let lineCount = 0
    proc.stdout.on("data", (data) => {
      const text = data.toString()
      const lines = text.split("\n")
      for (const line of lines) {
        if (lineCount >= lineLimit) {
          proc.kill()
          return
        }
        if (line) {
          output += line + "\n"
          lineCount++
        }
      }
    })
    proc.on("close", () => {
      if (output.trim()) {
        resolve(output.trim())
      } else {
        resolve(null)
      }
    })
    proc.on("error", () => resolve(null))
  })
}
