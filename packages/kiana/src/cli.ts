#!/usr/bin/env node

import * as fs from "node:fs"
import * as path from "node:path"
import * as readline from "node:readline"
import { createSession } from "./session.js"
import { loadConfig, writeConfigTemplate } from "./config.js"
import { VERSION } from "./index.js"

interface CliArgs {
  config?: string
  session?: string
  cwd?: string
  createConfig?: string
  humanReadable?: boolean
  verbose?: boolean
  prompt?: string
  help?: boolean
  version?: boolean
}

function parseArgs(args: string[]): CliArgs {
  const result: CliArgs = {}

  for (let i = 0; i < args.length; i++) {
    const arg = args[i]

    if (arg === "--config" || arg === "-c") {
      result.config = args[++i]
    } else if (arg === "--session" || arg === "-s") {
      result.session = args[++i]
    } else if (arg === "--cwd") {
      result.cwd = args[++i]
    } else if (arg === "--create-config") {
      result.createConfig = args[++i]
    } else if (arg === "--human-readable" || arg === "-H") {
      result.humanReadable = true
    } else if (arg === "--prompt" || arg === "-p") {
      result.prompt = args[++i]
    } else if (arg === "--verbose") {
      result.verbose = true
    } else if (arg === "--help" || arg === "-h") {
      result.help = true
    } else if (arg === "--version" || arg === "-V") {
      result.version = true
    }
  }

  return result
}

function printHelp(): void {
  console.log(`
Kiana - Minimal Headless Coding Agent

Usage:
  kiana --config <path> [options]
  kiana --create-config <path>

Options:
  --config, -c <path>       Path to JSONC config file (required)
  --prompt, -p <text>       Initial prompt to send (exits after completion)
  --session, -s <dir>       Directory to persist session
  --cwd <dir>               Working directory for tools
  --human-readable, -H      Output in human-readable format (streaming text)
  --verbose                 Show detailed tool inputs/outputs (with -H)
  --create-config <path>    Write config template and exit
  --help, -h                Show this help message
  --version, -V             Show version

Stdin/Stdout Protocol:
  Input (stdin):  Line-delimited JSON commands
    {"type": "message", "text": "Fix the bug in auth.ts"}
    {"type": "abort"}

  Output (stdout): Line-delimited JSON events
    {"type": "session.created", "properties": {...}}
    {"type": "message.part.updated", "properties": {...}}

Example:
  kiana --config ./kiana.jsonc
  echo '{"type":"message","text":"Hello"}' | kiana --config ./kiana.jsonc
`)
}

function printVersion(): void {
  console.log(`kiana v${VERSION}`)
}

interface StdinCommand {
  type: "message" | "abort"
  text?: string
}

// ANSI color codes
const colors = {
  reset: "\x1b[0m",
  bold: "\x1b[1m",
  dim: "\x1b[2m",
  cyan: "\x1b[36m",
  green: "\x1b[32m",
  yellow: "\x1b[33m",
  blue: "\x1b[34m",
  magenta: "\x1b[35m",
  red: "\x1b[31m",
  gray: "\x1b[90m",
}

// Human-readable event formatter
function formatEventHumanReadable(event: any, verbose: boolean = false): string | null {
  const indent = event.context ? "  ".repeat(event.context.depth) : ""
  const agentPrefix = event.context
    ? `${colors.magenta}[${event.context.agentType || "subagent"}]${colors.reset} `
    : ""

  switch (event.type) {
    case "session.created":
      return `${colors.dim}Session started: ${event.properties.session.id}${colors.reset}\n`

    case "session.idle":
      return `${colors.dim}--- Session idle ---${colors.reset}\n`

    case "message.created":
      if (event.properties.message.role === "user") {
        return null // User message text is shown via part.updated
      }
      return null // Don't show assistant message creation

    case "message.part.updated": {
      const part = event.properties.part
      const delta = event.properties.delta

      if (part.type === "text") {
        if (delta) {
          // Streaming text delta
          return `${indent}${agentPrefix}${delta}`
        }
        return null // Final update without delta
      }

      if (part.type === "reasoning") {
        if (delta) {
          return `${indent}${agentPrefix}${colors.dim}${delta}${colors.reset}`
        }
        return null
      }

      if (part.type === "tool") {
        const state = part.state
        if (state.status === "running") {
          const toolName = part.tool
          const title = state.title || formatToolInput(toolName, state.input)
          let output = `\n${indent}${agentPrefix}${colors.cyan}▶ ${toolName}${colors.reset} ${colors.gray}${title}${colors.reset}\n`
          // In verbose mode, show the full input
          if (verbose && state.input && Object.keys(state.input).length > 0) {
            const inputStr = JSON.stringify(state.input, null, 2)
              .split("\n")
              .map((line) => `${indent}  ${colors.dim}${line}${colors.reset}`)
              .join("\n")
            output += `${inputStr}\n`
          }
          return output
        }
        if (state.status === "completed") {
          const title = state.title || part.tool
          // Check if this is actually an error wrapped as a completed result
          const isError = state.metadata?.error || state.metadata?.validationError || state.metadata?.parseError
          const icon = isError ? "✗" : "✓"
          const color = isError ? colors.red : colors.green
          let output = `${indent}${agentPrefix}${color}${icon} ${title}${colors.reset}\n`
          // In verbose mode, show the output (truncated)
          if (verbose && state.output) {
            const outputLines = String(state.output).split("\n").slice(0, 10)
            if (String(state.output).split("\n").length > 10) {
              outputLines.push("... (truncated)")
            }
            const outputStr = outputLines
              .map((line) => `${indent}  ${colors.dim}${line}${colors.reset}`)
              .join("\n")
            output += `${outputStr}\n`
          }
          return output
        }
        if (state.status === "error") {
          let errorOutput = `${indent}${agentPrefix}${colors.red}✗ ${part.tool}: ${state.error}${colors.reset}\n`
          // Always show the input that caused the error for debugging
          if (state.input && Object.keys(state.input).length > 0) {
            const inputStr = JSON.stringify(state.input, null, 2)
              .split("\n")
              .map((line) => `${indent}  ${colors.dim}${line}${colors.reset}`)
              .join("\n")
            errorOutput += `${inputStr}\n`
          }
          return errorOutput
        }
        return null
      }

      if (part.type === "step-start") {
        return null // Don't show step start
      }

      if (part.type === "step-finish") {
        return `\n` // Add newline after step
      }

      return null
    }

    case "message.updated": {
      const msg = event.properties.message
      if (msg.error) {
        return `${colors.red}Error: ${msg.error.message || msg.error}${colors.reset}\n`
      }
      return null
    }

    case "todo.updated": {
      const todos = event.properties.todos
      if (!todos || todos.length === 0) return null

      let output = `\n${colors.yellow}📋 Todos:${colors.reset}\n`
      for (const todo of todos) {
        const icon = todo.status === "completed" ? "✓" : todo.status === "in_progress" ? "▶" : "○"
        const color = todo.status === "completed" ? colors.green : todo.status === "in_progress" ? colors.cyan : colors.gray
        output += `  ${color}${icon} ${todo.content}${colors.reset}\n`
      }
      return output
    }

    default:
      return null
  }
}

function formatToolInput(toolName: string, input: Record<string, unknown>): string {
  switch (toolName) {
    case "read":
      return String(input.filePath || input.file_path || "")
    case "write":
      return String(input.filePath || input.file_path || "")
    case "edit":
      return String(input.filePath || input.file_path || "")
    case "bash":
      return String(input.command || "").slice(0, 50) + (String(input.command || "").length > 50 ? "..." : "")
    case "glob":
      return String(input.pattern || "")
    case "grep":
      return String(input.pattern || "")
    case "list":
      return String(input.path || ".")
    case "task":
      return String(input.description || "")
    default:
      return ""
  }
}

async function main(): Promise<void> {
  const args = parseArgs(process.argv.slice(2))

  if (args.help) {
    printHelp()
    process.exit(0)
  }

  if (args.version) {
    printVersion()
    process.exit(0)
  }

  if (args.createConfig) {
    const configPath = path.resolve(args.createConfig)
    writeConfigTemplate(configPath)
    console.error(`Config template written to: ${configPath}`)
    process.exit(0)
  }

  if (!args.config) {
    console.error("Error: --config is required")
    console.error("Use --help for usage information")
    process.exit(1)
  }

  const configPath = path.resolve(args.config)

  if (!fs.existsSync(configPath)) {
    console.error(`Error: Config file not found: ${configPath}`)
    process.exit(1)
  }

  // Load config
  const config = loadConfig(configPath)

  // Override working directory if specified
  if (args.cwd) {
    config.workingDirectory = path.resolve(args.cwd)
  }

  // Create session
  const session = await createSession(config)

  // Subscribe to events and output
  if (args.humanReadable) {
    const verbose = args.verbose ?? false
    session.onEvent((event) => {
      const formatted = formatEventHumanReadable(event, verbose)
      if (formatted) {
        process.stdout.write(formatted)
      }
    })
  } else {
    session.onEvent((event) => {
      console.log(JSON.stringify(event))
    })
  }

  // Handle session persistence
  if (args.session) {
    const sessionDir = path.resolve(args.session)
    fs.mkdirSync(sessionDir, { recursive: true })

    // Save session info
    const sessionFile = path.join(sessionDir, "session.json")
    fs.writeFileSync(
      sessionFile,
      JSON.stringify(
        {
          id: session.id,
          created: Date.now(),
          config: configPath,
        },
        null,
        2
      )
    )

    // TODO: Implement full session persistence with messages/parts
  }

  // If --prompt is provided, send it and exit after completion
  if (args.prompt) {
    await session.sendMessage(args.prompt)
    process.exit(0)
  }

  // Otherwise, read commands from stdin
  let pendingMessage: Promise<void> | null = null
  let stdinClosed = false

  const rl = readline.createInterface({
    input: process.stdin,
    output: undefined,
    terminal: false,
  })

  rl.on("line", async (line) => {
    if (!line.trim()) return

    try {
      const command: StdinCommand = JSON.parse(line)

      switch (command.type) {
        case "message":
          if (command.text) {
            pendingMessage = session.sendMessage(command.text)
            await pendingMessage
            pendingMessage = null
            // If stdin already closed, exit after message completes
            if (stdinClosed) {
              process.exit(0)
            }
          }
          break

        case "abort":
          session.abort()
          break

        default:
          console.error(JSON.stringify({ error: `Unknown command type: ${(command as any).type}` }))
      }
    } catch (error) {
      console.error(
        JSON.stringify({
          error: `Failed to parse command: ${error instanceof Error ? error.message : String(error)}`,
        })
      )
    }
  })

  rl.on("close", async () => {
    stdinClosed = true
    // Wait for any pending message to complete before exiting
    if (pendingMessage) {
      await pendingMessage
    }
    process.exit(0)
  })

  // Handle signals
  process.on("SIGINT", () => {
    session.abort()
    process.exit(0)
  })

  process.on("SIGTERM", () => {
    session.abort()
    process.exit(0)
  })
}

main().catch((error) => {
  console.error(`Fatal error: ${error instanceof Error ? error.message : String(error)}`)
  process.exit(1)
})
