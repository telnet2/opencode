import { z } from "zod"
import { Tool } from "./tool"

const DEFAULT_READ_LIMIT = 2000
const MAX_LINE_LENGTH = 2000

const DESCRIPTION = `Reads a file from the memsh in-memory filesystem.

Usage:
- The filePath parameter should be an absolute path or relative to the current working directory
- By default, it reads up to 2000 lines starting from the beginning of the file
- You can optionally specify a line offset and limit (handy for long files)
- Any lines longer than 2000 characters will be truncated
- Results are returned with line numbers starting at 1`

interface ReadMetadata {
  preview: string
  filepath: string
  lines: number
  truncated: boolean
}

export const ReadTool = Tool.define<
  z.ZodObject<{
    filePath: z.ZodString
    offset: z.ZodOptional<z.ZodNumber>
    limit: z.ZodOptional<z.ZodNumber>
  }>,
  ReadMetadata
>("read", {
  description: DESCRIPTION,
  parameters: z.object({
    filePath: z.string().describe("The path to the file to read"),
    offset: z.coerce.number().describe("The line number to start reading from (0-based)").optional(),
    limit: z.coerce.number().describe("The number of lines to read (defaults to 2000)").optional(),
  }),
  async execute(params, ctx) {
    const filepath = params.filePath

    // Check if file exists
    const exists = await ctx.session.isFile(filepath)
    if (!exists) {
      // Check if it's a directory
      const isDir = await ctx.session.isDirectory(filepath)
      if (isDir) {
        throw new Error(`Cannot read directory: ${filepath}. Use the ls tool to list directory contents.`)
      }
      throw new Error(`File not found: ${filepath}`)
    }

    // Read the file content
    const content = await ctx.session.readFile(filepath)
    const allLines = content.split("\n")

    const limit = params.limit ?? DEFAULT_READ_LIMIT
    const offset = params.offset ?? 0

    // Slice the lines based on offset and limit
    const raw = allLines.slice(offset, offset + limit).map((line) => {
      return line.length > MAX_LINE_LENGTH ? line.substring(0, MAX_LINE_LENGTH) + "..." : line
    })

    // Format with line numbers
    const numbered = raw.map((line, index) => {
      return `${(index + offset + 1).toString().padStart(5, "0")}| ${line}`
    })

    const preview = raw.slice(0, 20).join("\n")

    let output = "<file>\n"
    output += numbered.join("\n")

    const totalLines = allLines.length
    const lastReadLine = offset + raw.length
    const hasMoreLines = totalLines > lastReadLine

    if (hasMoreLines) {
      output += `\n\n(File has more lines. Use 'offset' parameter to read beyond line ${lastReadLine})`
    } else {
      output += `\n\n(End of file - total ${totalLines} lines)`
    }
    output += "\n</file>"

    return {
      title: filepath,
      output,
      metadata: {
        preview,
        filepath,
        lines: raw.length,
        truncated: hasMoreLines,
      },
    }
  },
})
