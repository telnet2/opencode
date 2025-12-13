// Kiana - Minimal Headless Coding Agent Library
// A standalone TypeScript package providing a headless coding agent library + CLI wrapper

// Session API
export { createSession, type Session, type SessionOptions, type SessionState, type MessageWithParts } from "./session.js"

// Config
export { loadConfig, writeConfigTemplate, type Config, type ProviderConfig } from "./config.js"

// Provider
export { createLanguageModel } from "./provider.js"

// Event system
export { EventBus, type EventTypes, type SessionEvent, type MessageEvent, type PartEvent, type TodoEvent } from "./event.js"

// Types
export type { SessionInfo } from "./types/session.js"
export type { MessageInfo } from "./types/message.js"
export type {
  Part,
  TextPart,
  ReasoningPart,
  ToolPart,
  ToolState,
  FilePart,
  StepStartPart,
  StepFinishPart,
  SnapshotPart,
  PatchPart,
  AgentPart,
  RetryPart,
  CompactionPart,
  SubtaskPart,
} from "./types/part.js"

// Tools
export {
  allTools,
  getTools,
  getToolInfo,
  defineTool,
  type Tool,
  type ToolContext,
  type ToolResult,
  type ToolInfo,
} from "./tool/index.js"

// Individual tools (for advanced usage)
export { bashTool } from "./tool/bash.js"
export { readTool } from "./tool/read.js"
export { writeTool } from "./tool/write.js"
export { editTool } from "./tool/edit.js"
export { globTool } from "./tool/glob.js"
export { grepTool } from "./tool/grep.js"
export { listTool } from "./tool/list.js"
export { webfetchTool } from "./tool/webfetch.js"
export { websearchTool } from "./tool/websearch.js"
export { codesearchTool } from "./tool/codesearch.js"
export { todoWriteTool, todoReadTool, getTodos, setTodos } from "./tool/todo.js"
export { taskTool, setSubagentExecutor } from "./tool/task.js"

// Version
export const VERSION = "0.1.0"
