export { Tool, ToolRegistry, registry } from "./tool"
export { BashTool } from "./bash"
export { ReadTool } from "./read"
export { WriteTool } from "./write"
export { EditTool } from "./edit"
export { GlobTool } from "./glob"
export { GrepTool } from "./grep"
export { LsTool } from "./ls"

// Import and register all tools
import { registry } from "./tool"
import { BashTool } from "./bash"
import { ReadTool } from "./read"
import { WriteTool } from "./write"
import { EditTool } from "./edit"
import { GlobTool } from "./glob"
import { GrepTool } from "./grep"
import { LsTool } from "./ls"

/**
 * Register all default tools
 */
export function registerDefaultTools(): void {
  registry.registerAll(BashTool, ReadTool, WriteTool, EditTool, GlobTool, GrepTool, LsTool)
}

/**
 * Get list of all available tools
 */
export const allTools = [BashTool, ReadTool, WriteTool, EditTool, GlobTool, GrepTool, LsTool]
