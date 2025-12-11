import { z } from "zod"
import { defineTool } from "./tool.js"

/**
 * Invalid tool - receives redirected parse errors from experimental_repairToolCall.
 * This allows the LLM to see the error as a normal tool result and retry with corrected arguments.
 */
export const invalidTool = defineTool("invalid", {
  description: "Do not use",
  parameters: z.object({
    tool: z.string(),
    error: z.string(),
  }),
  async execute(params) {
    return {
      title: "Invalid Tool",
      output: `The arguments provided to the tool are invalid: ${params.error}`,
      metadata: {},
    }
  },
})
