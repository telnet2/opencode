// Tool exports
export { defineTool, type Tool, type ToolContext, type ToolResult } from "./tool.js"

// Individual tool exports
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
export { todoWriteTool, todoReadTool, getTodos, setTodos, type TodoInfo } from "./todo.js"
export {
  taskTool,
  setSubagentExecutor,
  getSubagentExecutor,
  getAgentConfig,
  getAvailableAgentTypes,
  type SubagentExecutor,
} from "./task.js"
export { invalidTool } from "./invalid.js"

import type { Tool } from "./tool.js"
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
import { taskTool } from "./task.js"
import { invalidTool } from "./invalid.js"

// All available tools
export const ALL_TOOLS: Record<string, Tool> = {
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

// Default tools (excludes invalid which is for error handling)
export const DEFAULT_TOOLS: string[] = [
  "bash",
  "read",
  "write",
  "edit",
  "glob",
  "grep",
  "list",
  "webfetch",
  "websearch",
  "codesearch",
  "todowrite",
  "todoread",
  "task",
]

// Tool categories for organization
export const TOOL_CATEGORIES = {
  filesystem: ["read", "write", "edit", "glob", "grep", "list"],
  execution: ["bash"],
  web: ["webfetch", "websearch", "codesearch"],
  planning: ["todowrite", "todoread"],
  subagent: ["task"],
  internal: ["invalid"],
} as const

/**
 * Get a subset of tools by name
 */
export function getTools(names: string[]): Record<string, Tool> {
  const result: Record<string, Tool> = {}
  for (const name of names) {
    const tool = ALL_TOOLS[name]
    if (tool) {
      result[name] = tool
    }
  }
  return result
}

/**
 * Get all tools except specified ones
 */
export function getToolsExcept(exclude: string[]): Record<string, Tool> {
  const result: Record<string, Tool> = {}
  for (const [name, tool] of Object.entries(ALL_TOOLS)) {
    if (!exclude.includes(name)) {
      result[name] = tool
    }
  }
  return result
}
