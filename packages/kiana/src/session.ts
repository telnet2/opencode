import { streamText, generateText, type LanguageModel, tool, jsonSchema } from "ai"
import { z } from "zod"
import { spawn } from "node:child_process"
import * as fs from "node:fs"
import * as path from "node:path"
import { EventBus, type EventTypes } from "./event.js"
import { Config } from "./config.js"
import { createLanguageModel } from "./provider.js"
import { allTools, getTools, setSubagentExecutor } from "./tool/index.js"
import type { Tool, ToolContext, ToolResult } from "./tool/tool.js"
import type { SessionInfo } from "./types/session.js"
import type { MessageInfo } from "./types/message.js"
import type { Part, TextPart, ToolPart, ReasoningPart, StepStartPart, StepFinishPart } from "./types/part.js"

// Session storage for persistence
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

// ID generation
let idCounter = 0
function generateId(prefix: string): string {
  const timestamp = Date.now().toString(36)
  const counter = (idCounter++).toString(36).padStart(4, "0")
  return `${prefix}_${timestamp}${counter}`
}

// Session state
interface SessionState {
  id: string
  projectID: string
  config: Config
  model: LanguageModel
  tools: Record<string, Tool<any>>
  messages: MessageWithParts[]
  abortController: AbortController
  eventBus: EventBus
  workingDirectory: string
  storage: SessionStorage | null
}

interface MessageWithParts {
  info: MessageInfo
  parts: Part[]
}

export interface SessionOptions {
  sessionDir?: string // Directory for session persistence
}

export interface Session {
  id: string
  sendMessage(text: string): Promise<void>
  onEvent(callback: (event: EventTypes) => void): () => void
  getTools(): { name: string; description: string }[]
  abort(): void
}

export async function createSession(config: Config, options?: SessionOptions): Promise<Session> {
  const storage = options?.sessionDir ? createSessionStorage(options.sessionDir) : null

  // Try to load existing session
  const existing = storage?.load()

  const sessionID = existing?.session.id ?? generateId("session")
  const projectID = existing?.session.projectID ?? generateId("project")

  const model = createLanguageModel(config.provider)
  const tools = getTools(config.tools)
  const workingDirectory = config.workingDirectory || process.cwd()

  const eventBus = new EventBus()

  const state: SessionState = {
    id: sessionID,
    projectID,
    config,
    model,
    tools,
    messages: existing?.messages ?? [],
    abortController: new AbortController(),
    eventBus,
    workingDirectory,
    storage,
  }

  // Helper to save session state
  function saveSession() {
    if (state.storage) {
      const sessionInfo: SessionInfo = {
        id: state.id,
        projectID: state.projectID,
        directory: state.workingDirectory,
        title: "Session",
        version: "1.0.0",
        time: {
          created: Date.now(),
          updated: Date.now(),
        },
      }
      state.storage.save(sessionInfo, state.messages)
    }
  }

  // Setup subagent executor for Task tool
  setSubagentExecutor(async (params) => {
    // Create a subagent session
    const subConfig: Config = {
      ...config,
      systemPrompt: params.agentConfig.systemPrompt,
    }

    const subSession = await createSession(subConfig)

    // Build subagent context for event forwarding
    const subagentContext = {
      parentSessionID: params.parentSessionID,
      depth: params.depth,
      agentType: params.agentType,
    }

    // Forward events from subsession with context
    subSession.onEvent((event) => {
      // Add subagent context to the event
      const eventWithContext = {
        ...event,
        context: event.context
          ? {
              // If event already has context (nested subagent), increment depth
              ...event.context,
              depth: event.context.depth + params.depth,
            }
          : subagentContext,
      }
      eventBus.emit(eventWithContext as any)
    })

    // Send the prompt and wait for completion
    await subSession.sendMessage(params.prompt)

    // Get the last assistant message as output
    const lastMessage = state.messages.findLast((m) => m.info.role === "assistant")
    const textPart = lastMessage?.parts.find((p): p is TextPart => p.type === "text")

    return {
      output: textPart?.text || "No output from subagent",
      sessionID: subSession.id,
    }
  })

  // Emit session created event
  const sessionInfo: SessionInfo = {
    id: sessionID,
    projectID,
    directory: workingDirectory,
    title: "New Session",
    version: "1.0.0",
    time: {
      created: Date.now(),
      updated: Date.now(),
    },
  }

  eventBus.emit({
    type: "session.created",
    properties: { session: sessionInfo },
  })

  async function sendMessage(text: string): Promise<void> {
    // Create user message
    const userMessageID = generateId("message")
    const userMessage: MessageInfo = {
      id: userMessageID,
      sessionID,
      role: "user",
      time: { created: Date.now() },
    }

    const userTextPart: TextPart = {
      id: generateId("part"),
      sessionID,
      messageID: userMessageID,
      type: "text",
      text,
    }

    state.messages.push({
      info: userMessage,
      parts: [userTextPart],
    })

    eventBus.emit({
      type: "message.created",
      properties: { message: userMessage },
    })

    eventBus.emit({
      type: "message.part.updated",
      properties: { part: userTextPart },
    })

    // Run agent loop
    await runAgentLoop(state)

    // Save session after message completes
    saveSession()
  }

  function onEvent(callback: (event: EventTypes) => void): () => void {
    return eventBus.subscribe(callback)
  }

  function getToolsInfo(): { name: string; description: string }[] {
    return Object.entries(state.tools).map(([name, tool]) => ({
      name,
      description: tool.description,
    }))
  }

  function abort(): void {
    state.abortController.abort()
    state.abortController = new AbortController()

    eventBus.emit({
      type: "session.idle",
      properties: { sessionID },
    })
  }

  return {
    id: sessionID,
    sendMessage,
    onEvent,
    getTools: getToolsInfo,
    abort,
  }
}

/**
 * Update message tokens from usage data.
 * Handles both AI SDK v6 nested format and older flat format.
 */
function updateTokensFromUsage(message: MessageInfo, usage: any): void {
  if (!usage) return

  // AI SDK v6 with OpenAI provider uses nested format:
  // { inputTokens: { total, noCache, cacheRead, cacheWrite }, outputTokens: { total, text, reasoning } }
  // Older/simpler format: { inputTokens: number, outputTokens: number, cachedInputTokens: number }
  let inputTotal = 0
  let outputTotal = 0
  let cachedRead = 0
  let reasoning = 0

  if (typeof usage.inputTokens === "object" && usage.inputTokens !== null) {
    // New nested format from AI SDK v6 OpenAI provider
    inputTotal = usage.inputTokens.total ?? 0
    cachedRead = usage.inputTokens.cacheRead ?? 0
  } else {
    // Old flat format
    inputTotal = usage.inputTokens ?? usage.promptTokens ?? 0
    cachedRead = usage.cachedInputTokens ?? 0
  }

  if (typeof usage.outputTokens === "object" && usage.outputTokens !== null) {
    // New nested format
    outputTotal = usage.outputTokens.total ?? 0
    reasoning = usage.outputTokens.reasoning ?? 0
  } else {
    // Old flat format
    outputTotal = usage.outputTokens ?? usage.completionTokens ?? 0
    reasoning = usage.reasoningTokens ?? 0
  }

  ;(message as any).tokens = {
    input: inputTotal - cachedRead,
    output: outputTotal,
    reasoning: reasoning,
    cache: {
      read: cachedRead,
      write: 0, // Provider-specific
    },
  }
}

async function runAgentLoop(state: SessionState): Promise<void> {
  const { eventBus, config, model, tools, workingDirectory } = state

  while (true) {
    state.abortController.signal.throwIfAborted()

    // Get the last user message
    const lastUserMessage = state.messages.findLast((m) => m.info.role === "user")
    if (!lastUserMessage) break

    // Get the last assistant message
    const lastAssistantMessage = state.messages.findLast((m) => m.info.role === "assistant")

    // Check if we should continue
    if (lastAssistantMessage) {
      const finishReason = (lastAssistantMessage.info as any).finish
      if (finishReason && !["tool-calls", "unknown"].includes(finishReason)) {
        if (lastUserMessage.info.id < lastAssistantMessage.info.id) {
          break
        }
      }
    }

    // Create assistant message
    const assistantMessageID = generateId("message")
    const assistantMessage: MessageInfo = {
      id: assistantMessageID,
      sessionID: state.id,
      role: "assistant",
      parentID: lastUserMessage.info.id,
      modelID: config.provider.model,
      providerID: config.provider.type,
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
    state.messages.push({ info: assistantMessage, parts: assistantParts })

    eventBus.emit({
      type: "message.created",
      properties: { message: assistantMessage },
    })

    // Build system prompt with environment context
    const baseSystemPrompt = config.systemPrompt || getDefaultSystemPrompt()
    const envContext = await buildEnvironmentContext(workingDirectory)
    const systemPrompt = baseSystemPrompt + "\n" + envContext

    // Build messages for AI
    const aiMessages = buildAIMessages(state.messages)

    // Build tools for AI
    const aiTools = buildAITools(state, assistantMessageID, assistantParts)

    try {
      // Emit step start
      const stepStartPart: StepStartPart = {
        id: generateId("part"),
        sessionID: state.id,
        messageID: assistantMessageID,
        type: "step-start",
      }
      assistantParts.push(stepStartPart)
      eventBus.emit({
        type: "message.part.updated",
        properties: { part: stepStartPart },
      })

      // Check if streaming is enabled (default: true)
      const useStreaming = config.streaming !== false

      if (useStreaming) {
        // Streaming mode
        let currentTextPart: TextPart | undefined
        let currentReasoningPart: ReasoningPart | undefined
        const toolParts: Record<string, ToolPart> = {}

        const stream = streamText({
          model,
          system: systemPrompt,
          messages: aiMessages,
          tools: aiTools,
          maxRetries: config.maxRetries, // Retry with exponential backoff for rate limit errors
          abortSignal: state.abortController.signal,
        })

        for await (const chunk of stream.fullStream) {
          state.abortController.signal.throwIfAborted()

          switch (chunk.type) {
            case "text-start":
              currentTextPart = {
                id: generateId("part"),
                sessionID: state.id,
                messageID: assistantMessageID,
                type: "text",
                text: "",
                time: { start: Date.now() },
              }
              assistantParts.push(currentTextPart)
              break

            case "text-delta":
              if (currentTextPart) {
                currentTextPart.text += chunk.text
                eventBus.emit({
                  type: "message.part.updated",
                  properties: { part: currentTextPart, delta: chunk.text },
                })
              }
              break

            case "text-end":
              if (currentTextPart) {
                currentTextPart.text = currentTextPart.text.trimEnd()
                currentTextPart.time = {
                  ...currentTextPart.time!,
                  end: Date.now(),
                }
                eventBus.emit({
                  type: "message.part.updated",
                  properties: { part: currentTextPart },
                })
                currentTextPart = undefined
              }
              break

            case "reasoning-start":
              currentReasoningPart = {
                id: generateId("part"),
                sessionID: state.id,
                messageID: assistantMessageID,
                type: "reasoning",
                text: "",
                time: { start: Date.now() },
              }
              assistantParts.push(currentReasoningPart)
              break

            case "reasoning-delta":
              if (currentReasoningPart) {
                currentReasoningPart.text += chunk.text
                eventBus.emit({
                  type: "message.part.updated",
                  properties: { part: currentReasoningPart, delta: chunk.text },
                })
              }
              break

            case "reasoning-end":
              if (currentReasoningPart) {
                currentReasoningPart.text = currentReasoningPart.text.trimEnd()
                currentReasoningPart.time = {
                  ...currentReasoningPart.time!,
                  end: Date.now(),
                }
                eventBus.emit({
                  type: "message.part.updated",
                  properties: { part: currentReasoningPart },
                })
                currentReasoningPart = undefined
              }
              break

            case "tool-input-start": {
              const toolPart: ToolPart = {
                id: generateId("part"),
                sessionID: state.id,
                messageID: assistantMessageID,
                type: "tool",
                tool: chunk.toolName,
                callID: chunk.id,
                state: {
                  status: "pending",
                  input: {},
                  raw: "",
                },
              }
              assistantParts.push(toolPart)
              toolParts[chunk.id] = toolPart
              eventBus.emit({
                type: "message.part.updated",
                properties: { part: toolPart },
              })
              break
            }

            case "tool-call": {
              const toolPart = toolParts[chunk.toolCallId]
              if (toolPart) {
                toolPart.state = {
                  status: "running",
                  input: (chunk as any).input as Record<string, unknown> ?? {},
                  time: { start: Date.now() },
                }
                eventBus.emit({
                  type: "message.part.updated",
                  properties: { part: toolPart },
                })
              }
              break
            }

            case "tool-result": {
              const toolPart = toolParts[chunk.toolCallId]
              if (toolPart && toolPart.state.status === "running") {
                const output = chunk.output as ToolResult
                toolPart.state = {
                  status: "completed",
                  input: (chunk as any).input as Record<string, unknown> ?? {},
                  output: output?.output ?? "",
                  title: output?.title ?? "",
                  metadata: output?.metadata ?? {},
                  time: {
                    start: toolPart.state.time.start,
                    end: Date.now(),
                  },
                }
                eventBus.emit({
                  type: "message.part.updated",
                  properties: { part: toolPart },
                })
              }
              break
            }

            case "tool-error": {
              const toolPart = toolParts[chunk.toolCallId]
              if (toolPart && toolPart.state.status === "running") {
                toolPart.state = {
                  status: "error",
                  input: (chunk as any).input as Record<string, unknown> ?? {},
                  error: String(chunk.error),
                  time: {
                    start: toolPart.state.time.start,
                    end: Date.now(),
                  },
                }
                eventBus.emit({
                  type: "message.part.updated",
                  properties: { part: toolPart },
                })
              }
              break
            }

            case "finish-step": {
              // Update message with usage
              const usage = chunk.usage as any
              updateTokensFromUsage(assistantMessage, usage)
              ;(assistantMessage as any).finish = chunk.finishReason

              // Emit step finish
              const stepFinishPart: StepFinishPart = {
                id: generateId("part"),
                sessionID: state.id,
                messageID: assistantMessageID,
                type: "step-finish",
                reason: chunk.finishReason,
                tokens: (assistantMessage as any).tokens,
                cost: 0,
              }
              assistantParts.push(stepFinishPart)
              eventBus.emit({
                type: "message.part.updated",
                properties: { part: stepFinishPart },
              })
              break
            }

            case "error":
              throw chunk.error
          }
        }

        // After streaming completes, get final totalUsage for accurate token counts
        // This is the accumulated usage across all steps, which is more accurate
        // than per-step usage from finish-step events
        try {
          const totalUsage = await stream.totalUsage
          if (totalUsage) {
            updateTokensFromUsage(assistantMessage, totalUsage)
            // Update the message event with final tokens
            eventBus.emit({
              type: "message.updated",
              properties: { message: assistantMessage },
            })
          }
        } catch {
          // totalUsage may not be available if stream was aborted
        }
      } else {
        // Non-streaming mode - better for token counting with some providers
        // Use onStepFinish callback to emit events in real-time as each step completes
        const result = await generateText({
          model,
          system: systemPrompt,
          messages: aiMessages,
          tools: aiTools,
          maxRetries: config.maxRetries, // Retry with exponential backoff for rate limit errors
          abortSignal: state.abortController.signal,
          onStepFinish: async (step) => {
            // Process text content for this step
            if (step.text) {
              const textPart: TextPart = {
                id: generateId("part"),
                sessionID: state.id,
                messageID: assistantMessageID,
                type: "text",
                text: step.text.trimEnd(),
                time: { start: Date.now(), end: Date.now() },
              }
              assistantParts.push(textPart)
              eventBus.emit({
                type: "message.part.updated",
                properties: { part: textPart },
              })
            }

            // Process tool results - update existing tool parts to "completed"
            // Note: "running" state was already emitted by onInputAvailable
            for (const toolResult of step.toolResults ?? []) {
              const toolPart = assistantParts.find(
                (p): p is ToolPart => p.type === "tool" && p.callID === toolResult.toolCallId
              )
              if (toolPart) {
                const output = toolResult.output as ToolResult
                toolPart.state = {
                  status: "completed",
                  input: toolPart.state.input,
                  output: output?.output ?? "",
                  title: output?.title ?? toolPart.tool,
                  metadata: output?.metadata ?? {},
                  time: {
                    start: (toolPart.state as any).time?.start ?? Date.now(),
                    end: Date.now(),
                  },
                }
                // Emit completed state
                eventBus.emit({
                  type: "message.part.updated",
                  properties: { part: toolPart },
                })
              }
            }

            // Emit step finish for this step
            const stepFinishPart: StepFinishPart = {
              id: generateId("part"),
              sessionID: state.id,
              messageID: assistantMessageID,
              type: "step-finish",
              reason: step.finishReason,
              tokens: {
                input: step.usage?.inputTokens ?? 0,
                output: step.usage?.outputTokens ?? 0,
                reasoning: 0,
                cache: { read: 0, write: 0 },
              },
              cost: 0,
            }
            assistantParts.push(stepFinishPart)
            eventBus.emit({
              type: "message.part.updated",
              properties: { part: stepFinishPart },
            })
          },
        })

        // Update total token counts from usage
        updateTokensFromUsage(assistantMessage, result.usage as any)
        ;(assistantMessage as any).finish = result.finishReason
      }

      // Update message completion time
      assistantMessage.time.completed = Date.now()

      eventBus.emit({
        type: "message.updated",
        properties: { message: assistantMessage },
      })

      // Check if we should continue (tool calls)
      const finishReason = (assistantMessage as any).finish
      if (finishReason !== "tool-calls") {
        break
      }
    } catch (error) {
      // Handle abort
      if (error instanceof Error && error.name === "AbortError") {
        break
      }

      // Log error and break
      console.error("Agent loop error:", error)
      ;(assistantMessage as any).error = {
        name: error instanceof Error ? error.name : "Error",
        message: error instanceof Error ? error.message : String(error),
      }

      eventBus.emit({
        type: "message.updated",
        properties: { message: assistantMessage },
      })

      break
    }
  }

  eventBus.emit({
    type: "session.idle",
    properties: { sessionID: state.id },
  })
}

function buildAIMessages(messages: MessageWithParts[]): any[] {
  const aiMessages: any[] = []

  // Group consecutive assistant messages together (they represent a single turn with tool loop)
  let i = 0
  while (i < messages.length) {
    const msg = messages[i]

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

      while (i < messages.length && messages[i].info.role === "assistant") {
        const assistantMsg = messages[i]
        const textParts = assistantMsg.parts.filter((p): p is TextPart => p.type === "text" && !!p.text)
        // Include both completed and error tool calls - LLM needs to see errors to recover
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
            // AI SDK v6 expects input as stringified JSON
            input: JSON.stringify(toolPart.state.input),
          })
        }

        if (content.length > 0) {
          aiMessages.push({
            role: "assistant",
            content,
          })
        }

        // Add tool results as separate messages (AI SDK 6 format)
        // All tool results go in a single tool message
        const toolResultContent: any[] = []
        for (const toolPart of allToolParts) {
          // Handle both completed and error states
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
        // Just text, no tool calls
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

function buildAITools(
  state: SessionState,
  messageID: string,
  parts: Part[]
): Record<string, any> {
  const { tools, workingDirectory, eventBus } = state
  const aiTools: Record<string, any> = {}

  for (const [name, toolDef] of Object.entries(tools)) {
    // Use the actual tool schema - following OpenCode's approach
    // The AI SDK will handle validation errors via tool-error events
    const schema = z.toJSONSchema(toolDef.parameters)

    aiTools[name] = tool({
      description: toolDef.description,
      inputSchema: jsonSchema(schema as any),
      // Called before tool execution - emit "running" state event
      // In streaming mode, the part is already created by tool-input-start
      // In non-streaming mode, this is where we create the part
      async onInputAvailable({ input, toolCallId }) {
        // Check if part already exists (streaming mode creates it in tool-input-start)
        let toolPart = parts.find(
          (p): p is ToolPart => p.type === "tool" && p.callID === toolCallId
        )
        if (!toolPart) {
          // Non-streaming mode: create the part
          toolPart = {
            id: generateId("part"),
            sessionID: state.id,
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
          eventBus.emit({
            type: "message.part.updated",
            properties: { part: toolPart },
          })
        }
      },
      async execute(args, options) {
        const ctx: ToolContext = {
          sessionID: state.id,
          messageID,
          workingDirectory,
          abort: options.abortSignal ?? new AbortController().signal,
          metadata: (val) => {
            // Find the tool part and update it
            const toolPart = parts.find(
              (p): p is ToolPart => p.type === "tool" && p.callID === options.toolCallId
            )
            if (toolPart && toolPart.state.status === "running") {
              toolPart.state = {
                ...toolPart.state,
                title: val.title,
                metadata: val.metadata,
              }
              eventBus.emit({
                type: "message.part.updated",
                properties: { part: toolPart },
              })
            }
          },
        }

        // Execute the tool - let errors propagate naturally
        // The AI SDK will catch them and emit tool-error events
        // Validation is handled by the tool's defineTool wrapper
        const result = await toolDef.execute(args, ctx)
        return result
      },
      toModelOutput(result: ToolResult) {
        return {
          type: "text",
          value: result.output,
        }
      },
    })
  }

  return aiTools
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
 * Includes working directory, git info, platform, date, and file tree.
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

/**
 * Check if a directory is a git repository.
 */
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

/**
 * Get the current git branch name.
 */
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

/**
 * Get git status (staged/unstaged changes, untracked files).
 */
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
        // Limit to first 50 lines
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

/**
 * Get file tree using ripgrep (rg --files) or fallback to find.
 */
async function getFileTree(cwd: string, limit: number): Promise<string | null> {
  // Try ripgrep first (faster, respects .gitignore)
  const rgResult = await runCommand("rg", ["--files", "--sort", "path"], cwd, limit)
  if (rgResult) return rgResult

  // Fallback to git ls-files if in a git repo
  const gitResult = await runCommand("git", ["ls-files"], cwd, limit)
  if (gitResult) return gitResult

  // Last resort: find command
  const findResult = await runCommand(
    "find",
    [".", "-type", "f", "-not", "-path", "*/.*", "-not", "-path", "*/node_modules/*"],
    cwd,
    limit
  )
  return findResult
}

/**
 * Run a command and return stdout, limited to a number of lines.
 */
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
    proc.stdout.on("data", (data) => (output += data.toString()))
    proc.on("close", (code) => {
      if (code === 0 && output.trim()) {
        const lines = output.trim().split("\n").slice(0, lineLimit)
        if (output.trim().split("\n").length > lineLimit) {
          lines.push(`... (${output.trim().split("\n").length - lineLimit} more files)`)
        }
        resolve(lines.join("\n"))
      } else {
        resolve(null)
      }
    })
    proc.on("error", () => resolve(null))
  })
}


// Export types
export type { SessionState, MessageWithParts }
