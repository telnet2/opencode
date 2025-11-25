import { z } from "zod"
import { Tool } from "./tool"

const DEFAULT_MAX_OUTPUT_LENGTH = 30_000

const DESCRIPTION = `Executes a shell command in the memsh in-memory filesystem.

Usage notes:
- Commands are executed in the session's current working directory
- The shell supports standard POSIX commands (ls, cat, mkdir, rm, etc.)
- Pipes and redirections are supported
- Environment variables can be set and used
- Use this tool for file system operations and command execution

Available built-in commands:
- File operations: pwd, cd, ls, cat, mkdir, rm, touch, cp, mv
- Text processing: grep, head, tail, wc, sort, uniq, echo
- File search: find
- HTTP & JSON: curl, jq
- Environment: env, export, set, unset
- Control flow: if/then/else, for, while
- Test expressions: test, [
- Utilities: help, sleep, true, false, exit
- Import/Export: import-file, import-dir, export-file, export-dir`

interface BashMetadata {
  output: string
  exit?: number
  error?: string
  cwd: string
  description: string
}

export const BashTool = Tool.define<
  z.ZodObject<{
    command: z.ZodString
    timeout: z.ZodOptional<z.ZodNumber>
    description: z.ZodString
  }>,
  BashMetadata
>("bash", {
  description: DESCRIPTION,
  parameters: z.object({
    command: z.string().describe("The shell command to execute"),
    timeout: z.number().optional().describe("Optional timeout in milliseconds (default: 60000)"),
    description: z.string().describe("Clear, concise description of what this command does in 5-10 words"),
  }),
  async execute(params, ctx) {
    const result = await ctx.session.execute(params.command)

    let output = result.output.join("\n")

    // Truncate if too long
    if (output.length > DEFAULT_MAX_OUTPUT_LENGTH) {
      output = output.slice(0, DEFAULT_MAX_OUTPUT_LENGTH)
      output += "\n\n(Output was truncated due to length limit)"
    }

    // Add error to output if present
    if (result.error) {
      output += `\n\nError: ${result.error}`
    }

    // Update metadata during execution
    ctx.metadata({
      metadata: {
        output,
        cwd: result.cwd,
        error: result.error,
        description: params.description,
      },
    })

    return {
      title: params.description,
      metadata: {
        output,
        cwd: result.cwd,
        error: result.error,
        description: params.description,
      },
      output,
    }
  },
})
