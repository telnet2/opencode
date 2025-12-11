import { z } from "zod"
import * as fs from "node:fs"
import * as path from "node:path"
import { defineTool } from "./tool.js"
import { listFiles } from "../util/ripgrep.js"

const DESCRIPTION = `- Fast file pattern matching tool that works with any codebase size
- Supports glob patterns like "**/*.js" or "src/**/*.ts"
- Returns matching file paths sorted by modification time
- Use this tool when you need to find files by name patterns
- When you are doing an open ended search that may require multiple rounds of globbing and grepping, use the Task tool instead
- You have the capability to call multiple tools in a single response. It is always better to speculatively perform multiple searches as a batch that are potentially useful.`

export const globTool = defineTool("glob", {
  description: DESCRIPTION,
  parameters: z.object({
    pattern: z.string().describe("The glob pattern to match files against"),
    path: z
      .string()
      .optional()
      .describe(
        `The directory to search in. If not specified, the current working directory will be used. IMPORTANT: Omit this field to use the default directory. DO NOT enter "undefined" or "null" - simply omit it for the default behavior. Must be a valid directory path if provided.`
      ),
  }),
  async execute(params, ctx) {
    let search = params.path ?? ctx.workingDirectory
    search = path.isAbsolute(search) ? search : path.resolve(ctx.workingDirectory, search)

    const limit = 100
    const files: Array<{ path: string; mtime: number }> = []
    let truncated = false

    for await (const file of listFiles({
      cwd: search,
      glob: [params.pattern],
    })) {
      if (files.length >= limit) {
        truncated = true
        break
      }
      const full = path.resolve(search, file)
      let mtime = 0
      try {
        const stats = fs.statSync(full)
        mtime = stats.mtime.getTime()
      } catch {
        // File might have been deleted
      }
      files.push({
        path: full,
        mtime,
      })
    }

    // Sort by modification time (newest first)
    files.sort((a, b) => b.mtime - a.mtime)

    const output: string[] = []
    if (files.length === 0) {
      output.push("No files found")
    } else {
      output.push(...files.map((f) => f.path))
      if (truncated) {
        output.push("")
        output.push("(Results are truncated. Consider using a more specific path or pattern.)")
      }
    }

    return {
      title: path.relative(ctx.workingDirectory, search),
      metadata: {
        count: files.length,
        truncated,
      },
      output: output.join("\n"),
    }
  },
})
