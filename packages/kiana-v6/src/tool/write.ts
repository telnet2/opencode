import { z } from "zod"
import * as fs from "node:fs"
import * as path from "node:path"
import { defineTool } from "./tool.js"

const DESCRIPTION = `Writes a file to the local filesystem.

Usage:
- This tool will overwrite the existing file if there is one at the provided path.
- If this is an existing file, you MUST use the Read tool first to read the file's contents. This tool will fail if you did not read the file first.
- ALWAYS prefer editing existing files in the codebase. NEVER write new files unless explicitly required.
- NEVER proactively create documentation files (*.md) or README files. Only create documentation files if explicitly requested by the User.
- Only use emojis if the user explicitly requests it. Avoid writing emojis to files unless asked.`

export const writeTool = defineTool("write", {
  description: DESCRIPTION,
  parameters: z.object({
    content: z.string().describe("The content to write to the file"),
    filePath: z
      .string()
      .describe(
        "The absolute path to the file to write (must be absolute, not relative)"
      ),
  }),
  async execute(params, ctx) {
    const filepath = path.isAbsolute(params.filePath)
      ? params.filePath
      : path.join(ctx.workingDirectory, params.filePath)

    const exists = fs.existsSync(filepath)

    // Ensure parent directory exists
    const dir = path.dirname(filepath)
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true })
    }

    // Write file
    fs.writeFileSync(filepath, params.content, "utf-8")

    const title = path.relative(ctx.workingDirectory, filepath)

    return {
      title,
      metadata: {
        diagnostics: {},
        filepath,
        exists,
      },
      output: "",
    }
  },
})
