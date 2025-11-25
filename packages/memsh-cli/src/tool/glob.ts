import { z } from "zod"
import { Tool } from "./tool"

const DESCRIPTION = `Fast file pattern matching tool in the memsh in-memory filesystem.

Usage:
- Supports glob patterns like "**/*.js" or "src/**/*.ts"
- Returns matching file paths
- Use this tool when you need to find files by name patterns`

interface GlobMetadata {
  count: number
  truncated: boolean
}

const LIMIT = 100

/**
 * Convert glob pattern to find command pattern
 */
function globToFindPattern(pattern: string): string {
  // For simple patterns, use -name
  // For patterns with directory parts, use -path
  if (pattern.includes("/") || pattern.includes("**")) {
    // Convert ** to find's wildcard syntax
    // find uses -path which matches the full path
    return pattern.replace(/\*\*/g, "*")
  }
  return pattern
}

export const GlobTool = Tool.define<
  z.ZodObject<{
    pattern: z.ZodString
    path: z.ZodOptional<z.ZodString>
  }>,
  GlobMetadata
>("glob", {
  description: DESCRIPTION,
  parameters: z.object({
    pattern: z.string().describe("The glob pattern to match files against"),
    path: z
      .string()
      .optional()
      .describe("The directory to search in. If not specified, the current working directory will be used."),
  }),
  async execute(params, ctx) {
    const searchPath = params.path ?? "."
    const findPattern = globToFindPattern(params.pattern)

    // Build find command
    // Use -name for simple patterns, -path for patterns with directories
    const usePath = params.pattern.includes("/") || params.pattern.includes("**")
    const findFlag = usePath ? "-path" : "-name"

    // Find files (not directories)
    const command = `find ${searchPath} -type f ${findFlag} '${findPattern}' 2>/dev/null | head -${LIMIT + 1}`

    const result = await ctx.session.runSafe(command)
    const lines = result.output.split("\n").filter(Boolean)

    const truncated = lines.length > LIMIT
    const files = truncated ? lines.slice(0, LIMIT) : lines

    const output: string[] = []
    if (files.length === 0) {
      output.push("No files found")
    } else {
      output.push(...files)
      if (truncated) {
        output.push("")
        output.push("(Results are truncated. Consider using a more specific path or pattern.)")
      }
    }

    return {
      title: searchPath,
      metadata: {
        count: files.length,
        truncated,
      },
      output: output.join("\n"),
    }
  },
})
