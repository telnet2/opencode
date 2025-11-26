import { z } from "zod"
import { Tool } from "./tool"

const DESCRIPTION = `Lists directory contents in the memsh in-memory filesystem.

Usage:
- Lists files and directories in the specified path
- By default, shows a structured tree view of the directory
- Can show hidden files with the 'all' parameter
- Can show detailed file information with the 'long' parameter`

export interface LsMetadata {
  count: number
  truncated: boolean
  [key: string]: unknown
}

const LIMIT = 100

export const LsTool = Tool.define<
  z.ZodObject<{
    path: z.ZodOptional<z.ZodString>
    all: z.ZodOptional<z.ZodBoolean>
    long: z.ZodOptional<z.ZodBoolean>
  }>,
  LsMetadata
>("ls", {
  description: DESCRIPTION,
  parameters: z.object({
    path: z.string().optional().describe("The directory path to list. Defaults to current working directory."),
    all: z.boolean().optional().describe("Show hidden files (files starting with .)"),
    long: z.boolean().optional().describe("Show detailed file information (permissions, size, date)"),
  }),
  async execute(params, ctx) {
    const searchPath = params.path ?? "."

    // Build ls command with appropriate flags
    const flags: string[] = []
    if (params.all) flags.push("-a")
    if (params.long) flags.push("-l")

    const flagStr = flags.length > 0 ? flags.join("") : ""
    const command = `ls ${flagStr} ${searchPath}`

    const result = await ctx.session.runSafe(command)

    if (result.error) {
      throw new Error(result.error)
    }

    const lines = result.output.split("\n").filter(Boolean)

    // For long format, the first line might be "total X", skip it
    const startIndex = params.long && lines[0]?.startsWith("total ") ? 1 : 0
    const entries = lines.slice(startIndex)

    const truncated = entries.length > LIMIT
    const finalEntries = truncated ? entries.slice(0, LIMIT) : entries

    // Build output
    let output = `${searchPath}/\n`
    output += finalEntries.join("\n")

    if (truncated) {
      output += "\n\n(Results are truncated. Consider using a more specific path.)"
    }

    return {
      title: searchPath,
      metadata: {
        count: finalEntries.length,
        truncated,
      },
      output,
    }
  },
})
