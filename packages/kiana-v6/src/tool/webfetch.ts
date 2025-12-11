import { z } from "zod"
import { defineTool } from "./tool.js"

const MAX_RESPONSE_SIZE = 5 * 1024 * 1024 // 5MB
const DEFAULT_TIMEOUT = 30 * 1000 // 30 seconds
const MAX_TIMEOUT = 120 * 1000 // 2 minutes

const DESCRIPTION = `- Fetches content from a specified URL
- Takes a URL and a format as input
- Fetches the URL content, converts HTML to markdown
- Returns the content in the specified format
- Use this tool when you need to retrieve and analyze web content

Usage notes:
  - IMPORTANT: if another tool is present that offers better web fetching capabilities, is more targeted to the task, or has fewer restrictions, prefer using that tool instead of this one.
  - The URL must be a fully-formed valid URL
  - HTTP URLs will be automatically upgraded to HTTPS
  - This tool is read-only and does not modify any files
  - Results may be summarized if the content is very large
  - Includes a self-cleaning 15-minute cache for faster responses when repeatedly accessing the same URL`

// Simple cache for responses
const cache = new Map<string, { content: string; timestamp: number }>()
const CACHE_TTL = 15 * 60 * 1000 // 15 minutes

function cleanCache() {
  const now = Date.now()
  for (const [key, value] of cache) {
    if (now - value.timestamp > CACHE_TTL) {
      cache.delete(key)
    }
  }
}

export const webfetchTool = defineTool("webfetch", {
  description: DESCRIPTION,
  parameters: z.object({
    url: z.string().describe("The URL to fetch content from"),
    format: z
      .enum(["text", "markdown", "html"])
      .describe("The format to return the content in (text, markdown, or html)"),
    timeout: z.number().describe("Optional timeout in seconds (max 120)").optional(),
  }),
  async execute(params, ctx) {
    // Validate URL
    if (!params.url.startsWith("http://") && !params.url.startsWith("https://")) {
      throw new Error("URL must start with http:// or https://")
    }

    // Check cache
    cleanCache()
    const cacheKey = `${params.url}:${params.format}`
    const cached = cache.get(cacheKey)
    if (cached) {
      const contentType = "text/html" // Cached, assume HTML
      return {
        output: cached.content,
        title: `${params.url} (cached)`,
        metadata: {},
      }
    }

    const timeout = Math.min((params.timeout ?? DEFAULT_TIMEOUT / 1000) * 1000, MAX_TIMEOUT)

    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), timeout)

    // Build Accept header based on requested format
    let acceptHeader = "*/*"
    switch (params.format) {
      case "markdown":
        acceptHeader = "text/markdown;q=1.0, text/x-markdown;q=0.9, text/plain;q=0.8, text/html;q=0.7, */*;q=0.1"
        break
      case "text":
        acceptHeader = "text/plain;q=1.0, text/markdown;q=0.9, text/html;q=0.8, */*;q=0.1"
        break
      case "html":
        acceptHeader = "text/html;q=1.0, application/xhtml+xml;q=0.9, text/plain;q=0.8, text/markdown;q=0.7, */*;q=0.1"
        break
      default:
        acceptHeader =
          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8"
    }

    let response: Response
    try {
      response = await fetch(params.url, {
        signal: controller.signal,
        headers: {
          "User-Agent":
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
          Accept: acceptHeader,
          "Accept-Language": "en-US,en;q=0.9",
        },
      })
    } finally {
      clearTimeout(timeoutId)
    }

    if (!response.ok) {
      throw new Error(`Request failed with status code: ${response.status}`)
    }

    // Check content length
    const contentLength = response.headers.get("content-length")
    if (contentLength && parseInt(contentLength) > MAX_RESPONSE_SIZE) {
      throw new Error("Response too large (exceeds 5MB limit)")
    }

    const arrayBuffer = await response.arrayBuffer()
    if (arrayBuffer.byteLength > MAX_RESPONSE_SIZE) {
      throw new Error("Response too large (exceeds 5MB limit)")
    }

    const content = new TextDecoder().decode(arrayBuffer)
    const contentType = response.headers.get("content-type") || ""

    const title = `${params.url} (${contentType})`

    let output: string

    // Handle content based on requested format and actual content type
    switch (params.format) {
      case "markdown":
        if (contentType.includes("text/html")) {
          output = convertHTMLToMarkdown(content)
        } else {
          output = content
        }
        break

      case "text":
        if (contentType.includes("text/html")) {
          output = extractTextFromHTML(content)
        } else {
          output = content
        }
        break

      case "html":
      default:
        output = content
        break
    }

    // Cache the result
    cache.set(cacheKey, { content: output, timestamp: Date.now() })

    return {
      output,
      title,
      metadata: {},
    }
  },
})

function extractTextFromHTML(html: string): string {
  // Simple HTML to text conversion
  let text = html
    // Remove script and style tags with their content
    .replace(/<script[^>]*>[\s\S]*?<\/script>/gi, "")
    .replace(/<style[^>]*>[\s\S]*?<\/style>/gi, "")
    .replace(/<noscript[^>]*>[\s\S]*?<\/noscript>/gi, "")
    // Replace common block elements with newlines
    .replace(/<\/?(p|div|br|h[1-6]|li|tr)[^>]*>/gi, "\n")
    // Remove all remaining tags
    .replace(/<[^>]+>/g, "")
    // Decode HTML entities
    .replace(/&nbsp;/g, " ")
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#(\d+);/g, (_, num) => String.fromCharCode(parseInt(num)))
    // Clean up whitespace
    .replace(/\n\s*\n/g, "\n\n")
    .trim()

  return text
}

function convertHTMLToMarkdown(html: string): string {
  let md = html
    // Remove script and style tags
    .replace(/<script[^>]*>[\s\S]*?<\/script>/gi, "")
    .replace(/<style[^>]*>[\s\S]*?<\/style>/gi, "")
    .replace(/<noscript[^>]*>[\s\S]*?<\/noscript>/gi, "")

  // Convert headings
  md = md.replace(/<h1[^>]*>([\s\S]*?)<\/h1>/gi, "\n# $1\n")
  md = md.replace(/<h2[^>]*>([\s\S]*?)<\/h2>/gi, "\n## $1\n")
  md = md.replace(/<h3[^>]*>([\s\S]*?)<\/h3>/gi, "\n### $1\n")
  md = md.replace(/<h4[^>]*>([\s\S]*?)<\/h4>/gi, "\n#### $1\n")
  md = md.replace(/<h5[^>]*>([\s\S]*?)<\/h5>/gi, "\n##### $1\n")
  md = md.replace(/<h6[^>]*>([\s\S]*?)<\/h6>/gi, "\n###### $1\n")

  // Convert links
  md = md.replace(/<a[^>]+href="([^"]*)"[^>]*>([\s\S]*?)<\/a>/gi, "[$2]($1)")

  // Convert images
  md = md.replace(/<img[^>]+src="([^"]*)"[^>]*alt="([^"]*)"[^>]*>/gi, "![$2]($1)")
  md = md.replace(/<img[^>]+src="([^"]*)"[^>]*>/gi, "![]($1)")

  // Convert bold and italic
  md = md.replace(/<(strong|b)[^>]*>([\s\S]*?)<\/(strong|b)>/gi, "**$2**")
  md = md.replace(/<(em|i)[^>]*>([\s\S]*?)<\/(em|i)>/gi, "*$2*")

  // Convert code
  md = md.replace(/<code[^>]*>([\s\S]*?)<\/code>/gi, "`$1`")
  md = md.replace(/<pre[^>]*>([\s\S]*?)<\/pre>/gi, "\n```\n$1\n```\n")

  // Convert lists
  md = md.replace(/<li[^>]*>([\s\S]*?)<\/li>/gi, "\n- $1")
  md = md.replace(/<\/?[ou]l[^>]*>/gi, "\n")

  // Convert paragraphs and divs
  md = md.replace(/<\/p>/gi, "\n\n")
  md = md.replace(/<p[^>]*>/gi, "")
  md = md.replace(/<br\s*\/?>/gi, "\n")
  md = md.replace(/<\/div>/gi, "\n")
  md = md.replace(/<div[^>]*>/gi, "")

  // Convert horizontal rules
  md = md.replace(/<hr[^>]*>/gi, "\n---\n")

  // Remove remaining tags
  md = md.replace(/<[^>]+>/g, "")

  // Decode HTML entities
  md = md
    .replace(/&nbsp;/g, " ")
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#(\d+);/g, (_, num) => String.fromCharCode(parseInt(num)))

  // Clean up whitespace
  md = md.replace(/\n{3,}/g, "\n\n").trim()

  return md
}
