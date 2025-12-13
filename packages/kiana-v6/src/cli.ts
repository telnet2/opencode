#!/usr/bin/env node
/**
 * Kiana v6 CLI - AI SDK UI Stream Protocol Edition
 *
 * Usage:
 *   kiana-v6 [options]
 *
 * Options:
 *   --config, -c     Path to config file (default: ./kiana.jsonc)
 *   --prompt, -p     Send a single prompt and exit
 *   --session, -s    Session directory for persistence
 *   --create-config  Generate a template config file
 *   -H               Human-readable output (instead of SSE)
 *   -v, --verbose    Show verbose output (tool inputs/outputs)
 *   --help, -h       Show help
 */

import * as fs from "node:fs"
import * as readline from "node:readline"
import { loadConfig, writeConfigTemplate } from "./config.js"
import { createLanguageModel } from "./provider.js"
import { CodingAgent, formatSSE, type StreamPart } from "./agent.js"

// Parse command line arguments
interface Args {
  config?: string
  prompt?: string
  session?: string
  createConfig?: boolean
  humanReadable: boolean
  verbose: boolean
  help: boolean
}

function parseArgs(argv: string[]): Args {
  const args: Args = {
    humanReadable: false,
    verbose: false,
    help: false,
  }

  for (let i = 2; i < argv.length; i++) {
    const arg = argv[i]
    switch (arg) {
      case "--config":
      case "-c":
        args.config = argv[++i]
        break
      case "--prompt":
      case "-p":
        args.prompt = argv[++i]
        break
      case "--session":
      case "-s":
        args.session = argv[++i]
        break
      case "--create-config":
        args.createConfig = true
        break
      case "-H":
        args.humanReadable = true
        break
      case "-v":
      case "--verbose":
        args.verbose = true
        break
      case "--help":
      case "-h":
        args.help = true
        break
    }
  }

  return args
}

function printHelp(): void {
  console.log(`
Kiana v6 - Minimal Headless Coding Agent (AI SDK UI Stream Protocol)

Usage:
  kiana-v6 [options]

Options:
  --config, -c     Path to config file (default: ./kiana.jsonc)
  --prompt, -p     Send a single prompt and exit
  --session, -s    Session directory for persistence
  --create-config  Generate a template config file
  -H               Human-readable output (instead of SSE)
  -v, --verbose    Show verbose output (tool inputs/outputs)
  --help, -h       Show help

Examples:
  # Run with default config (./kiana.jsonc)
  kiana-v6 -H -p "List all TypeScript files"

  # Run with a specific config
  kiana-v6 -c config.jsonc -p "List all TypeScript files"

  # Human-readable streaming output
  kiana-v6 -H -p "Analyze this codebase"

  # Interactive mode (read JSON from stdin)
  kiana-v6

  # Generate config template
  kiana-v6 --create-config > kiana.jsonc
`)
}

// ANSI color codes for human-readable mode
const colors = {
  reset: "\x1b[0m",
  dim: "\x1b[2m",
  cyan: "\x1b[36m",
  green: "\x1b[32m",
  red: "\x1b[31m",
  yellow: "\x1b[33m",
  gray: "\x1b[90m",
}

// Track if we're mid-line (text streaming without newline)
let needsNewlineBeforeBlock = false

/**
 * Format a unified diff with colors for terminal display.
 * - Red for removed lines (-)
 * - Green for added lines (+)
 * - Cyan for hunk headers (@@)
 * - Dim for context lines
 */
function formatDiff(diff: string): string {
  return diff
    .split("\n")
    .map((line) => {
      if (line.startsWith("+++") || line.startsWith("---")) {
        // File headers - dim
        return `${colors.dim}${line}${colors.reset}`
      } else if (line.startsWith("@@")) {
        // Hunk headers - cyan
        return `${colors.cyan}${line}${colors.reset}`
      } else if (line.startsWith("+")) {
        // Added lines - green
        return `${colors.green}${line}${colors.reset}`
      } else if (line.startsWith("-")) {
        // Removed lines - red
        return `${colors.red}${line}${colors.reset}`
      } else {
        // Context lines - dim
        return `${colors.dim}${line}${colors.reset}`
      }
    })
    .join("\n")
}

// Human-readable formatter for stream parts
function formatPartHumanReadable(part: StreamPart, verbose: boolean = false): string | null {
  switch (part.type) {
    case "data-session":
      return `${colors.dim}Session: ${part.data.id}${colors.reset}\n`

    case "start":
      return null // Suppress start message

    case "text-start":
      return null // Text will be shown via deltas

    case "text-delta":
      needsNewlineBeforeBlock = true // We're mid-text
      return part.delta

    case "text-end":
      needsNewlineBeforeBlock = false
      return "\n"

    case "tool-input-available": {
      // Add newline if we were mid-text
      const prefix = needsNewlineBeforeBlock ? "\n" : ""
      needsNewlineBeforeBlock = false

      // Format tool-specific preview
      let preview = ""
      const input = part.input as Record<string, unknown>

      switch (part.toolName) {
        case "bash":
          // Show the command being run
          if (input.command) {
            const cmd = String(input.command)
            const truncated = cmd.length > 80 ? cmd.slice(0, 77) + "..." : cmd
            preview = ` ${colors.gray}${truncated}${colors.reset}`
          }
          break
        case "read":
        case "write":
        case "edit":
        case "glob":
          // Show the file path or pattern
          if (input.file_path) {
            preview = ` ${colors.gray}${input.file_path}${colors.reset}`
          } else if (input.pattern) {
            preview = ` ${colors.gray}${input.pattern}${colors.reset}`
          }
          break
        case "list":
          // Show the directory being listed
          if (input.path) {
            preview = ` ${colors.gray}${input.path}${colors.reset}`
          } else {
            preview = ` ${colors.gray}.${colors.reset}`
          }
          break
        case "grep":
          // Show the pattern being searched
          if (input.pattern) {
            preview = ` ${colors.gray}${input.pattern}${colors.reset}`
          }
          break
        default:
          // For other tools, show truncated JSON in verbose mode
          if (verbose) {
            preview = ` ${colors.gray}${JSON.stringify(input).slice(0, 80)}${colors.reset}`
          }
      }

      return `${prefix}${colors.cyan}▶ ${part.toolName}${colors.reset}${preview}\n`
    }

    case "tool-output-available": {
      const title = part.metadata?.title || part.toolCallId
      let extra = ""

      // Show diff for edit operations (always, not just verbose)
      if (part.metadata?.diff) {
        extra = "\n" + formatDiff(String(part.metadata.diff))
      } else if (verbose && part.output) {
        extra = `\n${colors.gray}${String(part.output).slice(0, 200)}${colors.reset}`
      }

      return `${colors.green}✓ ${title}${colors.reset}${extra}\n`
    }

    case "tool-output-error":
      return `${colors.red}✗ Error: ${part.error}${colors.reset}\n`

    case "start-step":
      return null // Suppress step markers

    case "finish-step":
      return null // Suppress step markers

    case "finish":
      return null // Final finish is handled by idle

    case "data-session-idle": {
      // Add newline if we were mid-text
      const prefix = needsNewlineBeforeBlock ? "\n" : ""
      needsNewlineBeforeBlock = false
      return `${prefix}${colors.dim}--- Session idle ---${colors.reset}\n`
    }

    case "data-todo":
      if (verbose) {
        const todoList = part.data.todos
          .map((t) => `  [${t.status}] ${t.content}`)
          .join("\n")
        return `${colors.yellow}Todos:${colors.reset}\n${todoList}\n`
      }
      return null

    case "data-subagent-context":
      if (verbose) {
        return `${colors.dim}[Subagent depth=${part.data.depth} type=${part.data.agentType}]${colors.reset}\n`
      }
      return null

    case "error":
      return `${colors.red}Error: ${part.error}${colors.reset}\n`

    default:
      return null
  }
}

async function main(): Promise<void> {
  const args = parseArgs(process.argv)

  if (args.help) {
    printHelp()
    process.exit(0)
  }

  if (args.createConfig) {
    console.log(writeConfigTemplate())
    process.exit(0)
  }

  // Default to ./kiana.jsonc if no config specified
  const configPath = args.config || "./kiana.jsonc"

  if (!fs.existsSync(configPath)) {
    if (args.config) {
      console.error(`Error: Config file not found: ${configPath}`)
    } else {
      console.error("Error: No config file found. Either:")
      console.error("  - Create ./kiana.jsonc in the current directory")
      console.error("  - Specify a config file with --config <path>")
      console.error("  - Generate a template with --create-config > kiana.jsonc")
    }
    process.exit(1)
  }

  // Load config
  const config = loadConfig(configPath)

  // Create language model
  const model = createLanguageModel(config.provider)

  // Create agent
  const agent = new CodingAgent({
    model,
    workingDirectory: config.workingDirectory || process.cwd(),
    instructions: config.systemPrompt,
    tools: config.tools ?? undefined,
    maxSteps: config.maxSteps,
    maxRetries: config.maxRetries,
    sessionDir: args.session,
  })

  // Set up output handler
  const outputPart = (part: StreamPart) => {
    if (args.humanReadable) {
      const formatted = formatPartHumanReadable(part, args.verbose)
      if (formatted) {
        process.stdout.write(formatted)
      }
    } else {
      // SSE format
      process.stdout.write(formatSSE(part))
    }
  }

  // Subscribe to stream parts
  agent.onStream(outputPart)

  // Handle signals
  process.on("SIGINT", () => {
    agent.abort()
    process.exit(0)
  })

  process.on("SIGTERM", () => {
    agent.abort()
    process.exit(0)
  })

  // If --prompt is provided, send it and exit after completion
  if (args.prompt) {
    // Print prompt in human-readable mode
    if (args.humanReadable) {
      console.log(args.prompt)
    }

    if (config.streaming !== false) {
      const result = await agent.stream({ prompt: args.prompt })
      // Consume the text stream to trigger events
      for await (const _chunk of result.textStream) {
        // Events are emitted via onStream callback
      }
      // Wait for completion
      await result.text
    } else {
      await agent.generate({ prompt: args.prompt })
    }

    // Output [DONE] in SSE mode
    if (!args.humanReadable) {
      process.stdout.write("data: [DONE]\n\n")
    }

    process.exit(0)
  }

  // Otherwise, read commands from stdin (JSON protocol)
  const rl = readline.createInterface({
    input: process.stdin,
    output: undefined,
    terminal: false,
  })

  let pendingPromise: Promise<any> | null = null

  rl.on("line", async (line) => {
    if (!line.trim()) return

    try {
      const command = JSON.parse(line)

      switch (command.type) {
        case "message":
          if (pendingPromise) {
            console.error(JSON.stringify({ error: "Already processing a message" }))
            return
          }

          pendingPromise = (async () => {
            try {
              if (config.streaming !== false) {
                const result = await agent.stream({ prompt: command.text })
                for await (const _chunk of result.textStream) {
                  // Events emitted via callback
                }
                await result.text
              } else {
                await agent.generate({ prompt: command.text })
              }
            } catch (err) {
              const errorPart: StreamPart = {
                type: "error",
                error: err instanceof Error ? err.message : String(err),
              }
              outputPart(errorPart)
            } finally {
              pendingPromise = null
            }
          })()
          break

        case "abort":
          agent.abort()
          break

        default:
          console.error(JSON.stringify({ error: `Unknown command type: ${command.type}` }))
      }
    } catch (err) {
      console.error(JSON.stringify({ error: `Invalid JSON: ${err}` }))
    }
  })

  rl.on("close", () => {
    agent.abort()
    process.exit(0)
  })
}

main().catch((err) => {
  console.error("Fatal error:", err)
  process.exit(1)
})
