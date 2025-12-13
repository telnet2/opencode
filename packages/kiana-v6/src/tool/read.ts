import { z } from "zod"
import * as fs from "node:fs"
import * as path from "node:path"
import { defineTool } from "./tool.js"

const DEFAULT_READ_LIMIT = 2000
const MAX_LINE_LENGTH = 2000

const DESCRIPTION = `Reads a file from the local filesystem. You can access any file directly by using this tool.
Assume this tool is able to read all files on the machine. If the User provides a path to a file assume that path is valid. It is okay to read a file that does not exist; an error will be returned.

Usage:
- The filePath parameter must be an absolute path, not a relative path
- By default, it reads up to 2000 lines starting from the beginning of the file
- You can optionally specify a line offset and limit (especially handy for long files), but it's recommended to read the whole file by not providing these parameters
- Any lines longer than 2000 characters will be truncated
- Results are returned using cat -n format, with line numbers starting at 1
- You have the capability to call multiple tools in a single response. It is always better to speculatively read multiple files as a batch that are potentially useful.
- If you read a file that exists but has empty contents you will receive a system reminder warning in place of file contents.
- You can read image files using this tool.`

export const readTool = defineTool("read", {
  description: DESCRIPTION,
  parameters: z.object({
    filePath: z.string().describe("The path to the file to read"),
    offset: z.coerce
      .number()
      .describe("The line number to start reading from (0-based)")
      .optional(),
    limit: z.coerce
      .number()
      .describe("The number of lines to read (defaults to 2000)")
      .optional(),
  }),
  async execute(params, ctx) {
    let filepath = params.filePath
    if (!path.isAbsolute(filepath)) {
      filepath = path.join(ctx.workingDirectory, filepath)
    }
    const title = path.relative(ctx.workingDirectory, filepath)

    // Block .env files (except samples/examples)
    const whitelist = [".env.sample", ".example"]
    const block = (() => {
      if (whitelist.some((w) => filepath.endsWith(w))) return false
      if (filepath.includes(".env")) return true
      return false
    })()

    if (block) {
      throw new Error(
        `The user has blocked you from reading ${filepath}, DO NOT make further attempts to read it`
      )
    }

    // Check file exists
    if (!fs.existsSync(filepath)) {
      const dir = path.dirname(filepath)
      const base = path.basename(filepath)

      try {
        const dirEntries = fs.readdirSync(dir)
        const suggestions = dirEntries
          .filter(
            (entry) =>
              entry.toLowerCase().includes(base.toLowerCase()) ||
              base.toLowerCase().includes(entry.toLowerCase())
          )
          .map((entry) => path.join(dir, entry))
          .slice(0, 3)

        if (suggestions.length > 0) {
          throw new Error(
            `File not found: ${filepath}\n\nDid you mean one of these?\n${suggestions.join("\n")}`
          )
        }
      } catch (e) {
        if ((e as Error).message.startsWith("File not found:")) throw e
        // Directory doesn't exist, fall through
      }

      throw new Error(`File not found: ${filepath}`)
    }

    const stats = fs.statSync(filepath)
    if (stats.isDirectory()) {
      throw new Error(`Path is a directory, not a file: ${filepath}`)
    }

    // Check if image
    const imageType = isImageFile(filepath)
    if (imageType) {
      const content = fs.readFileSync(filepath)
      const mime = getMimeType(filepath)
      const msg = "Image read successfully"
      return {
        title,
        output: msg,
        metadata: { preview: msg },
        attachments: [
          {
            id: `part_${Date.now()}`,
            sessionID: ctx.sessionID,
            messageID: ctx.messageID,
            type: "file" as const,
            mime,
            url: `data:${mime};base64,${content.toString("base64")}`,
          },
        ],
      }
    }

    // Check if binary
    if (isBinaryFile(filepath, stats.size)) {
      throw new Error(`Cannot read binary file: ${filepath}`)
    }

    // Read text file
    const content = fs.readFileSync(filepath, "utf-8")
    const lines = content.split("\n")
    const limit = params.limit ?? DEFAULT_READ_LIMIT
    const offset = params.offset || 0

    const raw = lines.slice(offset, offset + limit).map((line) => {
      return line.length > MAX_LINE_LENGTH
        ? line.substring(0, MAX_LINE_LENGTH) + "..."
        : line
    })

    // Format with line numbers (matching cat -n format: padded line number + tab)
    const numbered = raw.map((line, index) => {
      return `${(index + offset + 1).toString().padStart(5, " ")}\t${line}`
    })

    const preview = raw.slice(0, 20).join("\n")

    let output = "<file>\n"
    output += numbered.join("\n")

    const totalLines = lines.length
    const lastReadLine = offset + raw.length
    const hasMoreLines = totalLines > lastReadLine

    if (hasMoreLines) {
      output += `\n\n(File has more lines. Use 'offset' parameter to read beyond line ${lastReadLine})`
    } else {
      output += `\n\n(End of file - total ${totalLines} lines)`
    }
    output += "\n</file>"

    return {
      title,
      output,
      metadata: { preview },
    }
  },
})

function isImageFile(filePath: string): string | false {
  const ext = path.extname(filePath).toLowerCase()
  switch (ext) {
    case ".jpg":
    case ".jpeg":
      return "JPEG"
    case ".png":
      return "PNG"
    case ".gif":
      return "GIF"
    case ".bmp":
      return "BMP"
    case ".webp":
      return "WebP"
    default:
      return false
  }
}

function getMimeType(filePath: string): string {
  const ext = path.extname(filePath).toLowerCase()
  switch (ext) {
    case ".jpg":
    case ".jpeg":
      return "image/jpeg"
    case ".png":
      return "image/png"
    case ".gif":
      return "image/gif"
    case ".bmp":
      return "image/bmp"
    case ".webp":
      return "image/webp"
    default:
      return "application/octet-stream"
  }
}

function isBinaryFile(filepath: string, fileSize: number): boolean {
  const ext = path.extname(filepath).toLowerCase()

  // Known binary extensions
  const binaryExtensions = new Set([
    ".zip",
    ".tar",
    ".gz",
    ".exe",
    ".dll",
    ".so",
    ".class",
    ".jar",
    ".war",
    ".7z",
    ".doc",
    ".docx",
    ".xls",
    ".xlsx",
    ".ppt",
    ".pptx",
    ".odt",
    ".ods",
    ".odp",
    ".bin",
    ".dat",
    ".obj",
    ".o",
    ".a",
    ".lib",
    ".wasm",
    ".pyc",
    ".pyo",
  ])

  if (binaryExtensions.has(ext)) {
    return true
  }

  if (fileSize === 0) return false

  // Read first 4KB and check for binary content
  const bufferSize = Math.min(4096, fileSize)
  const buffer = Buffer.alloc(bufferSize)
  const fd = fs.openSync(filepath, "r")
  const bytesRead = fs.readSync(fd, buffer, 0, bufferSize, 0)
  fs.closeSync(fd)

  if (bytesRead === 0) return false

  let nonPrintableCount = 0
  for (let i = 0; i < bytesRead; i++) {
    // Null byte is definite binary indicator
    if (buffer[i] === 0) return true
    // Count non-printable characters (excluding common whitespace)
    if (buffer[i] < 9 || (buffer[i] > 13 && buffer[i] < 32)) {
      nonPrintableCount++
    }
  }

  // If >30% non-printable, consider binary
  return nonPrintableCount / bytesRead > 0.3
}
