import { z } from "zod"
import { defineTool, ToolContext } from "./tool.js"

const DESCRIPTION = `Launch a new agent to handle complex, multi-step tasks autonomously.

Available agent types:
- general-purpose: General-purpose agent for researching complex questions, searching for code, and executing multi-step tasks.
- explore: Fast agent specialized for exploring codebases. Use this when you need to quickly find files by patterns, search code for keywords, or answer questions about the codebase.

When using the Task tool, you must specify a subagent_type parameter to select which agent type to use.

When to use the Task tool:
- When you are instructed to execute custom slash commands
- When exploring large codebases
- When tasks require multiple rounds of searching and analysis

When NOT to use the Task tool:
- If you want to read a specific file path, use the Read or Glob tool instead
- If you are searching for a specific class definition, use the Glob tool instead
- If you are searching for code within a specific file or set of 2-3 files, use the Read tool instead

Usage notes:
1. Launch multiple agents concurrently whenever possible for maximum performance
2. When the agent is done, it will return a single message back to you
3. Each agent invocation is stateless - provide a highly detailed task description
4. The agent's outputs should generally be trusted
5. Clearly tell the agent whether you expect it to write code or just to do research`

// Agent configuration for subagents
interface AgentConfig {
  name: string
  description: string
  systemPrompt: string
}

const AGENTS: Record<string, AgentConfig> = {
  "general-purpose": {
    name: "general-purpose",
    description: "General-purpose agent for researching complex questions and executing multi-step tasks",
    systemPrompt: `You are a helpful coding assistant running as a subagent. Complete the task autonomously and return a concise summary of your findings or actions.

When searching for code or files:
- Use Glob to find files by patterns
- Use Grep to search file contents
- Use Read to examine specific files

Be thorough but efficient. Return only the essential information needed to answer the question or complete the task.`,
  },
  explore: {
    name: "explore",
    description: "Fast agent specialized for exploring codebases",
    systemPrompt: `You are a fast code exploration agent. Your job is to quickly find files and code patterns in the codebase.

Use these tools efficiently:
- Glob: Find files by pattern (e.g., "**/*.ts", "src/**/*.tsx")
- Grep: Search file contents for keywords or patterns
- Read: Examine specific files when needed

Be quick and focused. Return a concise summary of what you found.`,
  },
}

// Subagent session executor type - will be injected by the session manager
export type SubagentExecutor = (params: {
  sessionID: string
  parentSessionID: string
  prompt: string
  agentConfig: AgentConfig
  agentType: string
  tools: Record<string, boolean>
  depth: number // nesting level (1 = direct subagent, 2 = subagent of subagent)
}) => Promise<{
  output: string
  sessionID: string
}>

// This will be set by the session manager
let subagentExecutor: SubagentExecutor | null = null

export function setSubagentExecutor(executor: SubagentExecutor) {
  subagentExecutor = executor
}

export function getSubagentExecutor(): SubagentExecutor | null {
  return subagentExecutor
}

export function getAgentConfig(agentType: string): AgentConfig | undefined {
  return AGENTS[agentType]
}

export function getAvailableAgentTypes(): string[] {
  return Object.keys(AGENTS)
}

export const taskTool = defineTool("task", {
  description: DESCRIPTION,
  parameters: z.object({
    description: z.string().describe("A short (3-5 words) description of the task"),
    prompt: z.string().describe("The task for the agent to perform"),
    subagent_type: z.string().describe("The type of specialized agent to use for this task"),
    session_id: z.string().describe("Existing Task session to continue").optional(),
  }),
  async execute(params, ctx) {
    const agent = AGENTS[params.subagent_type]
    if (!agent) {
      const availableAgents = Object.keys(AGENTS).join(", ")
      throw new Error(
        `Unknown agent type: ${params.subagent_type}. Available agents: ${availableAgents}`
      )
    }

    // Update metadata to show task in progress
    ctx.metadata({
      title: params.description,
      metadata: {
        agentType: params.subagent_type,
        status: "running",
      },
    })

    if (!subagentExecutor) {
      // Fallback: Return a message indicating subagent execution is not available
      // This happens when the task tool is used without a proper session manager
      return {
        title: params.description,
        metadata: {
          agentType: params.subagent_type,
          status: "skipped",
        },
        output: `Subagent execution is not available in this context. Task: ${params.prompt}`,
      }
    }

    try {
      const result = await subagentExecutor({
        sessionID: params.session_id || `task-${Date.now()}`,
        parentSessionID: ctx.sessionID,
        prompt: params.prompt,
        agentConfig: agent,
        agentType: params.subagent_type,
        tools: {
          todowrite: false,
          todoread: false,
          task: false, // Prevent recursive task calls
        },
        depth: 1, // Direct subagent call from parent
      })

      const output =
        result.output +
        "\n\n" +
        ["<task_metadata>", `session_id: ${result.sessionID}`, "</task_metadata>"].join("\n")

      return {
        title: params.description,
        metadata: {
          sessionId: result.sessionID,
          status: "completed",
        },
        output,
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error)
      return {
        title: params.description,
        metadata: {
          agentType: params.subagent_type,
          status: "error",
          error: errorMessage,
        },
        output: `Task failed: ${errorMessage}`,
      }
    }
  },
})
