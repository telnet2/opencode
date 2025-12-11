import { Tool } from "./tool.js"
import { bashTool } from "./bash.js"
import { readTool } from "./read.js"
import { writeTool } from "./write.js"
import { editTool } from "./edit.js"
import { globTool } from "./glob.js"
import { grepTool } from "./grep.js"
import { listTool } from "./list.js"
import { webfetchTool } from "./webfetch.js"
import { websearchTool } from "./websearch.js"
import { codesearchTool } from "./codesearch.js"
import { todoWriteTool, todoReadTool } from "./todo.js"
import { taskTool, setSubagentExecutor } from "./task.js"
import { invalidTool } from "./invalid.js"

// Export individual tools
export { bashTool } from "./bash.js"
export { readTool } from "./read.js"
export { writeTool } from "./write.js"
export { editTool } from "./edit.js"
export { globTool } from "./glob.js"
export { grepTool } from "./grep.js"
export { listTool } from "./list.js"
export { webfetchTool } from "./webfetch.js"
export { websearchTool } from "./websearch.js"
export { codesearchTool } from "./codesearch.js"
export { todoWriteTool, todoReadTool, getTodos, setTodos } from "./todo.js"
export { taskTool, setSubagentExecutor } from "./task.js"
export { invalidTool } from "./invalid.js"

// Export tool types
export type { Tool, ToolContext, ToolResult } from "./tool.js"
export { defineTool } from "./tool.js"

// All available tools
export const allTools: Record<string, Tool<any>> = {
  bash: bashTool,
  read: readTool,
  write: writeTool,
  edit: editTool,
  glob: globTool,
  grep: grepTool,
  list: listTool,
  webfetch: webfetchTool,
  websearch: websearchTool,
  codesearch: codesearchTool,
  todowrite: todoWriteTool,
  todoread: todoReadTool,
  task: taskTool,
  invalid: invalidTool,
}

// Get tools based on config (null = all tools)
export function getTools(enabled?: string[] | null): Record<string, Tool<any>> {
  if (!enabled) {
    return { ...allTools }
  }

  const result: Record<string, Tool<any>> = {}
  for (const name of enabled) {
    const tool = allTools[name.toLowerCase()]
    if (tool) {
      result[name.toLowerCase()] = tool
    }
  }
  return result
}

// Get tool info for API responses
export interface ToolInfo {
  name: string
  description: string
  parameters: Record<string, unknown>
}

export function getToolInfo(tools: Record<string, Tool<any>>): ToolInfo[] {
  return Object.entries(tools).map(([name, tool]) => ({
    name,
    description: tool.description,
    parameters: tool.parameters._def,
  }))
}
