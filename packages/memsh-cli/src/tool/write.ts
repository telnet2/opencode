import { z } from "zod"
import { Tool } from "./tool"

const DESCRIPTION = `Writes content to a file in the memsh in-memory filesystem.

Usage:
- This tool will overwrite the existing file if there is one at the provided path
- If the parent directory doesn't exist, it will be created
- The filePath should be an absolute path or relative to the current working directory`

interface WriteMetadata {
  filepath: string
  exists: boolean
  size: number
}

export const WriteTool = Tool.define<
  z.ZodObject<{
    content: z.ZodString
    filePath: z.ZodString
  }>,
  WriteMetadata
>("write", {
  description: DESCRIPTION,
  parameters: z.object({
    content: z.string().describe("The content to write to the file"),
    filePath: z.string().describe("The path to the file to write"),
  }),
  async execute(params, ctx) {
    const filepath = params.filePath

    // Check if file already exists
    const exists = await ctx.session.exists(filepath)

    // Ensure parent directory exists
    const parentDir = filepath.split("/").slice(0, -1).join("/")
    if (parentDir) {
      const parentExists = await ctx.session.exists(parentDir)
      if (!parentExists) {
        await ctx.session.mkdir(parentDir, { recursive: true })
      }
    }

    // Write the file
    await ctx.session.writeFile(filepath, params.content)

    const output = exists ? `File overwritten: ${filepath}` : `File created: ${filepath}`

    return {
      title: filepath,
      metadata: {
        filepath,
        exists,
        size: params.content.length,
      },
      output,
    }
  },
})
