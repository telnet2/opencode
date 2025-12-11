import { z } from "zod"
import * as fs from "node:fs"
import * as path from "node:path"
import { defineTool } from "./tool.js"
import { searchContent } from "../util/ripgrep.js"

const MAX_LINE_LENGTH = 2000

const DESCRIPTION = `- Fast content search tool that works with any codebase size
- Searches file contents using regular expressions
- Supports full regex syntax (eg. "log.*Error", "function\\s+\\w+", etc.)
- Filter files by pattern with the include parameter (eg. "*.js", "*.{ts,tsx}")
- Returns file paths with at least one match sorted by modification time
- Use this tool when you need to find files containing specific patterns
- If you need to identify/count the number of matches within files, use the Bash tool with \`rg\` (ripgrep) directly. Do NOT use \`grep\`.
- When you are doing an open ended search that may require multiple rounds of globbing and grepping, use the Task tool instead`

export const grepTool = defineTool("grep", {
  description: DESCRIPTION,
  parameters: z.object({
    pattern: z.string().describe("The regex pattern to search for in file contents"),
    path: z
      .string()
      .optional()
      .describe("The directory to search in. Defaults to the current working directory."),
    include: z
      .string()
      .optional()
      .describe('File pattern to include in the search (e.g. "*.js", "*.{ts,tsx}")'),
  }),
  async execute(params, ctx) {
    if (!params.pattern) {
      throw new Error("pattern is required")
    }

    const searchPath = params.path || ctx.workingDirectory

    const rawMatches = await searchContent({
      cwd: searchPath,
      pattern: params.pattern,
      glob: params.include,
    })

    // Add modification time to matches
    const matches: Array<{
      path: string
      modTime: number
      lineNum: number
      lineText: string
    }> = []

    for (const match of rawMatches) {
      let modTime = 0
      try {
        const stats = fs.statSync(match.path)
        modTime = stats.mtime.getTime()
      } catch {
        // File might have been deleted
      }
      matches.push({
        ...match,
        modTime,
      })
    }

    // Sort by modification time (newest first)
    matches.sort((a, b) => b.modTime - a.modTime)

    const limit = 100
    const truncated = matches.length > limit
    const finalMatches = truncated ? matches.slice(0, limit) : matches

    if (finalMatches.length === 0) {
      return {
        title: params.pattern,
        metadata: { matches: 0, truncated: false },
        output: "No files found",
      }
    }

    const outputLines = [`Found ${finalMatches.length} matches`]

    let currentFile = ""
    for (const match of finalMatches) {
      if (currentFile !== match.path) {
        if (currentFile !== "") {
          outputLines.push("")
        }
        currentFile = match.path
        outputLines.push(`${match.path}:`)
      }
      const truncatedLineText =
        match.lineText.length > MAX_LINE_LENGTH
          ? match.lineText.substring(0, MAX_LINE_LENGTH) + "..."
          : match.lineText
      outputLines.push(`  Line ${match.lineNum}: ${truncatedLineText}`)
    }

    if (truncated) {
      outputLines.push("")
      outputLines.push("(Results are truncated. Consider using a more specific path or pattern.)")
    }

    return {
      title: params.pattern,
      metadata: {
        matches: finalMatches.length,
        truncated,
      },
      output: outputLines.join("\n"),
    }
  },
})
