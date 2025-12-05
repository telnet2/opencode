import { z } from "zod"
import { Tool } from "./tool"

const DESCRIPTION = `Performs exact string replacements in files in the memsh in-memory filesystem.

Usage:
- The edit will FAIL if oldString is not found in the file
- The edit will FAIL if oldString is not unique in the file (unless replaceAll is true)
- Use replaceAll for replacing and renaming strings across the file
- If oldString is empty, the file will be created with newString as content`

export interface EditMetadata {
  filepath: string
  diff: string
  additions: number
  deletions: number
  [key: string]: unknown
}

/**
 * Create a simple unified diff
 */
function createDiff(filepath: string, oldContent: string, newContent: string): string {
  const oldLines = oldContent.split("\n")
  const newLines = newContent.split("\n")

  let diff = `--- ${filepath}\n+++ ${filepath}\n`

  // Simple diff: show removed and added lines
  const maxLines = Math.max(oldLines.length, newLines.length)
  let inHunk = false
  let hunkStart = 0
  let hunkLines: string[] = []

  const flushHunk = () => {
    if (hunkLines.length > 0) {
      diff += `@@ -${hunkStart + 1} +${hunkStart + 1} @@\n`
      diff += hunkLines.join("\n") + "\n"
      hunkLines = []
    }
    inHunk = false
  }

  for (let i = 0; i < maxLines; i++) {
    const oldLine = oldLines[i]
    const newLine = newLines[i]

    if (oldLine === newLine) {
      if (inHunk) {
        hunkLines.push(` ${oldLine ?? ""}`)
        if (hunkLines.filter((l) => l.startsWith(" ")).length > 3) {
          flushHunk()
        }
      }
    } else {
      if (!inHunk) {
        inHunk = true
        hunkStart = Math.max(0, i - 3)
        // Add context before
        for (let j = hunkStart; j < i; j++) {
          if (oldLines[j] !== undefined) {
            hunkLines.push(` ${oldLines[j]}`)
          }
        }
      }
      if (oldLine !== undefined && (newLine === undefined || oldLine !== newLine)) {
        hunkLines.push(`-${oldLine}`)
      }
      if (newLine !== undefined && (oldLine === undefined || oldLine !== newLine)) {
        hunkLines.push(`+${newLine}`)
      }
    }
  }

  flushHunk()

  return diff
}

/**
 * Count additions and deletions
 */
function countChanges(oldContent: string, newContent: string): { additions: number; deletions: number } {
  const oldLines = oldContent.split("\n")
  const newLines = newContent.split("\n")

  let additions = 0
  let deletions = 0

  // Simple counting: lines that differ
  const maxLines = Math.max(oldLines.length, newLines.length)
  for (let i = 0; i < maxLines; i++) {
    if (oldLines[i] !== newLines[i]) {
      if (oldLines[i] !== undefined) deletions++
      if (newLines[i] !== undefined) additions++
    }
  }

  return { additions, deletions }
}

export const EditTool = Tool.define<
  z.ZodObject<{
    filePath: z.ZodString
    oldString: z.ZodString
    newString: z.ZodString
    replaceAll: z.ZodOptional<z.ZodBoolean>
  }>,
  EditMetadata
>("edit", {
  description: DESCRIPTION,
  parameters: z.object({
    filePath: z.string().describe("The path to the file to modify"),
    oldString: z.string().describe("The text to replace"),
    newString: z.string().describe("The text to replace it with (must be different from oldString)"),
    replaceAll: z.boolean().optional().describe("Replace all occurrences of oldString (default false)"),
  }),
  async execute(params, ctx) {
    const filepath = params.filePath

    if (params.oldString === params.newString) {
      throw new Error("oldString and newString must be different")
    }

    // Handle creating new file when oldString is empty
    if (params.oldString === "") {
      // Create or overwrite file
      const parentDir = filepath.split("/").slice(0, -1).join("/")
      if (parentDir) {
        const parentExists = await ctx.session.exists(parentDir)
        if (!parentExists) {
          await ctx.session.mkdir(parentDir, { recursive: true })
        }
      }

      await ctx.session.writeFile(filepath, params.newString)

      const diff = createDiff(filepath, "", params.newString)
      const { additions, deletions } = countChanges("", params.newString)

      return {
        title: filepath,
        metadata: {
          filepath,
          diff,
          additions,
          deletions,
        },
        output: diff,
      }
    }

    // Check if file exists
    const exists = await ctx.session.exists(filepath)
    if (!exists) {
      throw new Error(`File not found: ${filepath}`)
    }

    // Check if it's a directory
    const isDir = await ctx.session.isDirectory(filepath)
    if (isDir) {
      throw new Error(`Path is a directory, not a file: ${filepath}`)
    }

    // Read current content
    const oldContent = await ctx.session.readFile(filepath)

    // Check if oldString exists in content
    if (!oldContent.includes(params.oldString)) {
      throw new Error("oldString not found in content")
    }

    // Check for multiple occurrences if not replaceAll
    if (!params.replaceAll) {
      const firstIndex = oldContent.indexOf(params.oldString)
      const lastIndex = oldContent.lastIndexOf(params.oldString)

      if (firstIndex !== lastIndex) {
        throw new Error(
          "Found multiple matches for oldString. Provide more surrounding lines in oldString to identify the correct match, or use replaceAll to replace all occurrences.",
        )
      }
    }

    // Perform replacement
    const newContent = params.replaceAll
      ? oldContent.replaceAll(params.oldString, params.newString)
      : oldContent.replace(params.oldString, params.newString)

    // Write updated content
    await ctx.session.writeFile(filepath, newContent)

    // Generate diff
    const diff = createDiff(filepath, oldContent, newContent)
    const { additions, deletions } = countChanges(oldContent, newContent)

    return {
      title: filepath,
      metadata: {
        filepath,
        diff,
        additions,
        deletions,
      },
      output: diff,
    }
  },
})
