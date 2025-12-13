import kleur from "kleur"
import type { AssistantMessage, Event, Message, Part, ToolPart, ToolState } from "@opencode-ai/sdk/v2"

function colorize(noColor: boolean) {
  const k = kleur
  k.enabled = !noColor
  return k
}

export interface RendererOptions {
  noColor?: boolean
  quiet?: boolean
  json?: boolean
  verbose?: boolean
}

export class Renderer {
  private readonly c: typeof kleur
  constructor(private readonly options: RendererOptions = {}) {
    this.c = colorize(options.noColor ?? false)
  }

  banner(url: string) {
    if (this.options.quiet) return
    console.error(this.c.gray(`Connected to ${url}`))
  }

  help(text: string) {
    if (this.options.quiet) return
    console.log(text)
  }

  user(input: string) {
    if (this.options.json) {
      console.log(JSON.stringify({ type: "user", text: input }))
    } else {
      console.log(this.c.bold(this.c.cyan(`you ›`)), input)
    }
  }

  assistant(message: string) {
    if (this.options.json) {
      console.log(JSON.stringify({ type: "assistant", text: message }))
    } else {
      console.log(this.c.bold(this.c.green(`assistant ›`)), message)
    }
  }

  trace(message: string, details?: Record<string, unknown>) {
    if (!this.options.verbose) return
    const text = details ? `${message} ${JSON.stringify(details)}` : message
    console.error(this.c.gray(`[trace] ${text}`))
  }

  tool(part: ToolPart) {
    const summary = this.describeToolState(part.state)
    if (this.options.json) {
      console.log(JSON.stringify({ type: "tool", tool: part.tool, callID: part.callID, state: part.state }))
      return
    }
    console.log(this.c.yellow(`→ tool ${part.tool} (${summary})`))
    if (part.state.status === "completed" && part.state.output) {
      console.log(this.c.gray(String(part.state.output)))
    }
    if (part.state.status === "error") {
      console.error(this.c.red(`  error: ${part.state.error}`))
    }
  }

  event(event: Event) {
    if (this.options.json) {
      console.log(JSON.stringify({ type: "event", event }))
      return
    }
    switch (event.type) {
      case "message.updated":
        this.renderMessage(event.properties.info)
        break
      case "message.part.updated":
        this.renderPart(event.properties.part)
        break
      default:
        this.trace("event", { type: event.type })
    }
  }

  renderMessage(message: Message | AssistantMessage, parts?: Part[]) {
    const textParts = parts?.filter((p): p is Part & { type: "text" } => p.type === "text")
    const toolParts = parts?.filter((p): p is ToolPart => p.type === "tool")
    if (textParts?.length) {
      const combined = textParts.map((p) => p.text).join("\n")
      this.assistant(combined)
    } else if (message.role === "assistant") {
      const assistant = message as AssistantMessage
      this.assistant(`message ${assistant.id} updated`)
    }
    if (toolParts?.length) {
      for (const part of toolParts) {
        this.tool(part)
      }
    }
  }

  renderPart(part: Part) {
    if (part.type === "text") {
      this.assistant(part.text)
    } else if (part.type === "tool") {
      this.tool(part)
    }
  }

  private describeToolState(state: ToolState): string {
    switch (state.status) {
      case "pending":
        return "pending"
      case "running":
        return "running"
      case "completed":
        return "done"
      case "error":
        return "error"
      default:
        return "unknown"
    }
  }
}
