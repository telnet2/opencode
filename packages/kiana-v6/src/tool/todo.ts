import { z } from "zod"
import { defineTool } from "./tool.js"

const DESCRIPTION_WRITE = `Use this tool to create and manage a structured task list for your current coding session. This helps you track progress, organize complex tasks, and demonstrate thoroughness to the user.
It also helps the user understand the progress of the task and overall progress of their requests.

## When to Use This Tool
Use this tool proactively in these scenarios:

1. Complex multi-step tasks - When a task requires 3 or more distinct steps or actions
2. Non-trivial and complex tasks - Tasks that require careful planning or multiple operations
3. User explicitly requests todo list - When the user directly asks you to use the todo list
4. User provides multiple tasks - When users provide a list of things to be done (numbered or comma-separated)
5. After receiving new instructions - Immediately capture user requirements as todos. Feel free to edit the todo list based on new information.
6. After completing a task - Mark it complete and add any new follow-up tasks
7. When you start working on a new task, mark the todo as in_progress. Ideally you should only have one todo as in_progress at a time. Complete existing tasks before starting new ones.

## When NOT to Use This Tool

Skip using this tool when:
1. There is only a single, straightforward task
2. The task is trivial and tracking it provides no organizational benefit
3. The task can be completed in less than 3 trivial steps
4. The task is purely conversational or informational

NOTE that you should not use this tool if there is only one trivial task to do. In this case you are better off just doing the task directly.

## Task States and Management

1. **Task States**: Use these states to track progress:
   - pending: Task not yet started
   - in_progress: Currently working on (limit to ONE task at a time)
   - completed: Task finished successfully

2. **Task Management**:
   - Update task status in real-time as you work
   - Mark tasks complete IMMEDIATELY after finishing (don't batch completions)
   - Only have ONE task in_progress at any time
   - Complete current tasks before starting new ones

3. **Task Breakdown**:
   - Create specific, actionable items
   - Break complex tasks into smaller, manageable steps
   - Use clear, descriptive task names

When in doubt, use this tool. Being proactive with task management demonstrates attentiveness and ensures you complete all requirements successfully.`

const DESCRIPTION_READ = `Use this tool to read your current todo list`

// Schema for todo items
const TodoInfoSchema = z.object({
  content: z.string().describe("The task description"),
  status: z
    .enum(["pending", "in_progress", "completed"])
    .describe("The task status"),
  activeForm: z
    .string()
    .describe("The present continuous form of the task (e.g., 'Implementing feature')"),
})

export type TodoInfo = z.infer<typeof TodoInfoSchema>

// In-memory todo storage per session
const todoStorage = new Map<string, TodoInfo[]>()

export function getTodos(sessionID: string): TodoInfo[] {
  return todoStorage.get(sessionID) || []
}

export function setTodos(sessionID: string, todos: TodoInfo[]): void {
  todoStorage.set(sessionID, todos)
}

export const todoWriteTool = defineTool("todowrite", {
  description: DESCRIPTION_WRITE,
  parameters: z.object({
    todos: z.array(TodoInfoSchema).describe("The updated todo list"),
  }),
  async execute(params, ctx) {
    setTodos(ctx.sessionID, params.todos)

    const pendingCount = params.todos.filter((x) => x.status !== "completed").length

    return {
      title: `${pendingCount} todos`,
      output: JSON.stringify(params.todos, null, 2),
      metadata: {
        todos: params.todos,
      },
    }
  },
})

export const todoReadTool = defineTool("todoread", {
  description: DESCRIPTION_READ,
  parameters: z.object({}),
  async execute(_params, ctx) {
    const todos = getTodos(ctx.sessionID)
    const pendingCount = todos.filter((x) => x.status !== "completed").length

    return {
      title: `${pendingCount} todos`,
      metadata: {
        todos,
      },
      output: JSON.stringify(todos, null, 2),
    }
  },
})
