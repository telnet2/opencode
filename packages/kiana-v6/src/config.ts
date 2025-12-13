import { z } from "zod"
import { readFileSync, writeFileSync } from "node:fs"
import { parse as parseJsonc } from "jsonc-parser"

export const ProviderConfigSchema = z.object({
  type: z.enum(["anthropic", "openai", "openai-compatible", "google"]),
  apiKey: z.string(),
  model: z.string(),
  baseUrl: z.string().nullable().optional(),
})

export type ProviderConfig = z.infer<typeof ProviderConfigSchema>

export const MCPServerConfigSchema = z.object({
  name: z.string(),
  command: z.string(),
  args: z.array(z.string()),
  env: z.record(z.string(), z.string()).optional(),
})

export type MCPServerConfig = z.infer<typeof MCPServerConfigSchema>

export const ConfigSchema = z.object({
  provider: ProviderConfigSchema,
  systemPrompt: z.string().optional(),
  workingDirectory: z.string().nullable().optional(),
  tools: z.array(z.string()).nullable().optional(),
  // Maximum number of steps in the agent loop (default: 50)
  maxSteps: z.number().int().min(1).max(100).optional().default(50),
  // Maximum number of retries for rate limit errors (default: 5)
  // Uses exponential backoff and respects Retry-After headers
  maxRetries: z.number().int().min(0).max(10).optional().default(5),
  // Enable streaming mode (default: true)
  streaming: z.boolean().optional().default(true),
  // MCP servers to connect to
  mcpServers: z.array(MCPServerConfigSchema).optional(),
})

export type Config = z.infer<typeof ConfigSchema>

/**
 * Load and validate config from a JSONC file.
 */
export function loadConfig(path: string): Config {
  const content = readFileSync(path, "utf-8")
  const parsed = parseJsonc(content)

  const result = ConfigSchema.safeParse(parsed)
  if (!result.success) {
    const errors = result.error.issues
      .map((e: z.ZodIssue) => `  ${e.path.join(".")}: ${e.message}`)
      .join("\n")
    throw new Error(`Invalid config at ${path}:\n${errors}`)
  }

  return result.data
}

/**
 * Default system prompt for Kiana headless mode.
 */
export const DEFAULT_SYSTEM_PROMPT = `You are Kiana, a powerful coding agent running in headless mode.

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
- CRITICAL: When calling tools, pass arguments as a proper JSON object, NOT as a string.
  - Correct: {"command": "git status", "description": "Check git status"}
  - Wrong: "{\"command\": \"git status\", \"description\": \"Check git status\"}"
- Never stringify or escape the arguments object - pass it directly as structured data.

# Code References
When referencing code, include \`file_path:line_number\` for navigation.`

/**
 * Config template for --create-config flag.
 */
export const CONFIG_TEMPLATE = `{
  // Provider configuration
  "provider": {
    "type": "anthropic",  // "anthropic" | "openai" | "openai-compatible" | "google"
    "apiKey": "YOUR_API_KEY",
    "model": "claude-sonnet-4-20250514",
    "baseUrl": null  // Required for openai-compatible
  },

  // System prompt (optional, has sensible default)
  "systemPrompt": null,

  // Working directory for tools (optional, defaults to current dir)
  "workingDirectory": null,

  // Tool whitelist (optional, null = all tools enabled)
  // Available: bash, read, write, edit, glob, grep, list, webfetch, websearch, codesearch, todowrite, todoread, task
  "tools": null,

  // Maximum steps in agent loop (default: 50)
  "maxSteps": 50,

  // Maximum retries for rate limit errors (default: 5)
  // Uses exponential backoff and respects Retry-After headers
  "maxRetries": 5,

  // MCP servers to connect to (optional)
  "mcpServers": [
    // Example: File system server
    // {
    //   "name": "filesystem",
    //   "command": "npx",
    //   "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/directory"],
    //   "env": {}
    // }
  ]
}
`

/**
 * Get config template string (or write to file if path provided).
 */
export function writeConfigTemplate(path?: string): string {
  if (path) {
    writeFileSync(path, CONFIG_TEMPLATE, "utf-8")
  }
  return CONFIG_TEMPLATE
}
