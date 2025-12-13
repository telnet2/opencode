// Version
export const VERSION = "0.1.0"

// Main exports
export {
  CodingAgent,
  type CodingAgentConfig,
  type GenerateParams,
  type StreamParams,
  type StreamCallback,
  type StreamPart,
  formatSSE,
  parseSSE,
  createSSEHeaders,
} from "./agent.js"

// Stream types (AI SDK UI compatible)
export type {
  StartPart,
  TextStartPart,
  TextDeltaPart,
  TextEndPart,
  ReasoningStartPart,
  ReasoningDeltaPart,
  ReasoningEndPart,
  ToolInputStartPart,
  ToolInputDeltaPart,
  ToolInputAvailablePart,
  ToolOutputAvailablePart,
  ToolOutputErrorPart,
  StartStepPart,
  FinishStepPart,
  FinishPart,
  DataSessionPart,
  DataSessionIdlePart,
  DataTodoPart,
  DataSubagentContextPart,
  ErrorPart,
  MessageMetadata,
  ToolMetadata,
  TokenUsage as StreamTokenUsage,
} from "./stream.js"

// Config
export { loadConfig, writeConfigTemplate, DEFAULT_SYSTEM_PROMPT, type Config, type MCPServerConfig } from "./config.js"

// Provider
export { createLanguageModel } from "./provider.js"

// Event system (legacy - prefer StreamPart for new code)
export { EventBus, type EventTypes } from "./event.js"

// Tools
export {
  defineTool,
  type Tool,
  type ToolContext,
  type ToolResult,
  ALL_TOOLS,
  DEFAULT_TOOLS,
  TOOL_CATEGORIES,
  getTools,
  getToolsExcept,
  // Individual tools
  bashTool,
  readTool,
  writeTool,
  editTool,
  globTool,
  grepTool,
  listTool,
  webfetchTool,
  websearchTool,
  codesearchTool,
  todoWriteTool,
  todoReadTool,
  taskTool,
  invalidTool,
  // Todo helpers
  getTodos,
  setTodos,
  type TodoInfo,
  // Task helpers
  setSubagentExecutor,
  getSubagentExecutor,
  getAgentConfig,
  getAvailableAgentTypes,
  type SubagentExecutor,
} from "./tool/index.js"

// MCP Support
export {
  MCPClientManager,
  getMCPManager,
  initializeMCPServers,
  createMCPTool,
} from "./tool/mcp.js"

// Types
export type {
  SessionInfo,
  FileDiff,
  SessionSummary,
  MessageInfo,
  TokenUsage,
  MessageError,
  Part,
  TextPart,
  ReasoningPart,
  ToolPart,
  ToolState,
  StepStartPart,
  StepFinishPart,
} from "./types/index.js"
