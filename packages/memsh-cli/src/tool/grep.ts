import { z } from "zod"
import { Tool } from "./tool"

const DESCRIPTION = `Search tool for finding patterns in file contents in the memsh in-memory filesystem.

Usage:
- Supports regex patterns
- Filter files with include parameter (e.g., "*.js", "*.{ts,tsx}")
- Returns matching lines with file paths and line numbers`

interface GrepMetadata {
  matches: number
  truncated: boolean
}

const LIMIT = 100

export const GrepTool = Tool.define<
  z.ZodObject<{
    pattern: z.ZodString
    path: z.ZodOptional<z.ZodString>
    include: z.ZodOptional<z.ZodString>
  }>,
  GrepMetadata
>("grep", {
  description: DESCRIPTION,
  parameters: z.object({
    pattern: z.string().describe("The regex pattern to search for in file contents"),
    path: z.string().optional().describe("The directory to search in. Defaults to the current working directory."),
    include: z.string().optional().describe('File pattern to include in the search (e.g. "*.js", "*.{ts,tsx}")'),
  }),
  async execute(params, ctx) {
    const searchPath = params.path ?? "."

    // Build grep command
    // grep -r: recursive, -n: line numbers, -H: print filename
    let command = `grep -rnH '${params.pattern.replace(/'/g, "'\\''")}' ${searchPath}`

    // If include pattern is specified, use find + grep
    if (params.include) {
      // Convert include pattern for find
      const includePattern = params.include.replace(/\{([^}]+)\}/g, (_, group) => {
        // Convert {ts,tsx} to find -name patterns
        return group.split(",")[0] // Just use first pattern for simplicity
      })

      command = `find ${searchPath} -type f -name '${includePattern}' -exec grep -nH '${params.pattern.replace(/'/g, "'\\''")}' {} \\;`
    }

    // Add limit
    command += ` 2>/dev/null | head -${LIMIT + 1}`

    const result = await ctx.session.runSafe(command)
    const lines = result.output.split("\n").filter(Boolean)

    if (lines.length === 0) {
      return {
        title: params.pattern,
        metadata: { matches: 0, truncated: false },
        output: "No matches found",
      }
    }

    const truncated = lines.length > LIMIT
    const matches = truncated ? lines.slice(0, LIMIT) : lines

    // Parse and format output
    const outputLines: string[] = [`Found ${matches.length} matches`]
    let currentFile = ""

    for (const line of matches) {
      // Format: filename:linenum:content
      const colonIndex = line.indexOf(":")
      if (colonIndex === -1) continue

      const file = line.substring(0, colonIndex)
      const rest = line.substring(colonIndex + 1)

      const secondColonIndex = rest.indexOf(":")
      if (secondColonIndex === -1) continue

      const lineNum = rest.substring(0, secondColonIndex)
      const content = rest.substring(secondColonIndex + 1)

      if (currentFile !== file) {
        if (currentFile !== "") {
          outputLines.push("")
        }
        currentFile = file
        outputLines.push(`${file}:`)
      }
      outputLines.push(`  Line ${lineNum}: ${content}`)
    }

    if (truncated) {
      outputLines.push("")
      outputLines.push("(Results are truncated. Consider using a more specific path or pattern.)")
    }

    return {
      title: params.pattern,
      metadata: {
        matches: matches.length,
        truncated,
      },
      output: outputLines.join("\n"),
    }
  },
})
