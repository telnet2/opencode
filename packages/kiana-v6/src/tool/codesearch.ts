import { z } from "zod"
import { defineTool } from "./tool.js"

const DESCRIPTION = `- Search and get relevant context for any programming task using Exa Code API
- Provides the highest quality and freshest context for libraries, SDKs, and APIs
- Use this tool for ANY question or task related to programming
- Returns comprehensive code examples, documentation, and API references
- Optimized for finding specific programming patterns and solutions

Usage notes:
  - Adjustable token count (1000-50000) for focused or comprehensive results
  - Default 5000 tokens provides balanced context for most queries
  - Use lower values for specific questions, higher values for comprehensive documentation
  - Supports queries about frameworks, libraries, APIs, and programming concepts
  - Examples: 'React useState hook examples', 'Python pandas dataframe filtering', 'Express.js middleware'`

const API_CONFIG = {
  BASE_URL: "https://mcp.exa.ai",
  ENDPOINTS: {
    CONTEXT: "/mcp",
  },
} as const

interface McpCodeRequest {
  jsonrpc: string
  id: number
  method: string
  params: {
    name: string
    arguments: {
      query: string
      tokensNum: number
    }
  }
}

interface McpCodeResponse {
  jsonrpc: string
  result: {
    content: Array<{
      type: string
      text: string
    }>
  }
}

export const codesearchTool = defineTool("codesearch", {
  description: DESCRIPTION,
  parameters: z.object({
    query: z
      .string()
      .describe(
        "Search query to find relevant context for APIs, Libraries, and SDKs. For example, 'React useState hook examples', 'Python pandas dataframe filtering', 'Express.js middleware', 'Next js partial prerendering configuration'"
      ),
    tokensNum: z
      .number()
      .min(1000)
      .max(50000)
      .default(5000)
      .describe(
        "Number of tokens to return (1000-50000). Default is 5000 tokens. Adjust this value based on how much context you need - use lower values for focused queries and higher values for comprehensive documentation."
      ),
  }),
  async execute(params, ctx) {
    const codeRequest: McpCodeRequest = {
      jsonrpc: "2.0",
      id: 1,
      method: "tools/call",
      params: {
        name: "get_code_context_exa",
        arguments: {
          query: params.query,
          tokensNum: params.tokensNum || 5000,
        },
      },
    }

    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 30000)

    try {
      const headers: Record<string, string> = {
        accept: "application/json, text/event-stream",
        "content-type": "application/json",
      }

      const signals = ctx.abort ? [controller.signal, ctx.abort] : [controller.signal]
      const response = await fetch(`${API_CONFIG.BASE_URL}${API_CONFIG.ENDPOINTS.CONTEXT}`, {
        method: "POST",
        headers,
        body: JSON.stringify(codeRequest),
        signal: AbortSignal.any(signals),
      })

      clearTimeout(timeoutId)

      if (!response.ok) {
        const errorText = await response.text()
        throw new Error(`Code search error (${response.status}): ${errorText}`)
      }

      const responseText = await response.text()

      // Parse SSE response
      const lines = responseText.split("\n")
      for (const line of lines) {
        if (line.startsWith("data: ")) {
          const data: McpCodeResponse = JSON.parse(line.substring(6))
          if (data.result && data.result.content && data.result.content.length > 0) {
            return {
              output: data.result.content[0].text,
              title: `Code search: ${params.query}`,
              metadata: {},
            }
          }
        }
      }

      return {
        output:
          "No code snippets or documentation found. Please try a different query, be more specific about the library or programming concept, or check the spelling of framework names.",
        title: `Code search: ${params.query}`,
        metadata: {},
      }
    } catch (error) {
      clearTimeout(timeoutId)

      if (error instanceof Error && error.name === "AbortError") {
        throw new Error("Code search request timed out")
      }

      throw error
    }
  },
})
