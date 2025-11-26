import z from "zod"
import { Tool } from "../tool/tool"
import { Workflow } from "./index"
import { WorkflowExecutor } from "./executor"
import { WorkflowInstance } from "./schema"
import { Bus } from "../bus"
import { Log } from "../util/log"

const log = Log.create({ service: "workflow.tool" })

/**
 * Tool for executing agentic workflows
 */
export const WorkflowTool = Tool.define("workflow", async () => {
  const workflows = await Workflow.list()
  const workflowList =
    workflows.length > 0
      ? workflows.map((w) => `- ${w.id}: ${w.description ?? w.name}`).join("\n")
      : "No workflows defined. Create workflows in .opencode/workflow/*.md"

  return {
    description: `Execute an agentic workflow that orchestrates multiple subagents.

Available workflows:
${workflowList}

Workflows can:
- Execute multiple subagents with their own prompts and tools
- Run steps sequentially, in parallel, or conditionally
- Pause for human review at key decision points
- Pass data between steps using variables

Use this tool when you need to coordinate complex multi-agent tasks.`,

    parameters: z.object({
      workflow: z.string().describe("The workflow ID to execute"),
      inputs: z.record(z.string(), z.any()).describe("Input variables for the workflow").optional(),
      action: z
        .enum(["start", "resume", "cancel", "status"])
        .describe("Action to perform: start a new workflow, resume a paused one, cancel, or check status")
        .default("start"),
      instanceId: z.string().describe("Instance ID for resume/cancel/status actions").optional(),
      approved: z.boolean().describe("Whether to approve a paused step (for resume action)").optional(),
      feedback: z.string().describe("Feedback for the paused step (for resume action)").optional(),
    }),

    async execute(params, ctx) {
      log.info("workflow tool called", { params })

      switch (params.action) {
        case "start":
          return handleStart(params, ctx)
        case "resume":
          return handleResume(params, ctx)
        case "cancel":
          return handleCancel(params, ctx)
        case "status":
          return handleStatus(params, ctx)
        default:
          throw new Error(`Unknown action: ${params.action}`)
      }
    },
  }
})

async function handleStart(
  params: {
    workflow: string
    inputs?: Record<string, any>
    instanceId?: string
  },
  ctx: Tool.Context,
): Promise<{ title: string; metadata: any; output: string }> {
  const definition = await Workflow.get(params.workflow)
  if (!definition) {
    return {
      title: "Workflow not found",
      metadata: { error: true },
      output: `Workflow "${params.workflow}" not found. Available workflows:\n${(await Workflow.list()).map((w) => `- ${w.id}`).join("\n")}`,
    }
  }

  // Create workflow instance
  const instance = await WorkflowExecutor.create({
    definition,
    inputs: params.inputs ?? {},
    parentSessionId: ctx.sessionID,
  })

  log.info("starting workflow", { instanceId: instance.id, workflowId: definition.id })

  // Set up pause handler for interactive mode
  let pausedStep: { stepId: string; message: string; options: any } | undefined

  const executionCtx: WorkflowExecutor.ExecutionContext = {
    instance,
    abort: ctx.abort,
    onPause: async (stepId, message, options) => {
      pausedStep = { stepId, message, options }
      // In tool context, we pause and return - the user will need to resume
      return { approved: false }
    },
  }

  // Subscribe to events for real-time updates
  const events: any[] = []
  const unsubscribers = [
    Bus.subscribe(WorkflowExecutor.Event.StepStarted, (evt) => {
      if (evt.properties.instanceId === instance.id) {
        events.push({ type: "step_started", ...evt.properties })
        ctx.metadata({
          title: `Running: ${evt.properties.stepId}`,
          metadata: { instanceId: instance.id, events },
        })
      }
    }),
    Bus.subscribe(WorkflowExecutor.Event.StepCompleted, (evt) => {
      if (evt.properties.instanceId === instance.id) {
        events.push({ type: "step_completed", ...evt.properties })
        ctx.metadata({
          title: `Completed: ${evt.properties.stepId}`,
          metadata: { instanceId: instance.id, events },
        })
      }
    }),
    Bus.subscribe(WorkflowExecutor.Event.Paused, (evt) => {
      if (evt.properties.instanceId === instance.id) {
        events.push({ type: "paused", ...evt.properties })
        ctx.metadata({
          title: `Paused: ${evt.properties.message}`,
          metadata: { instanceId: instance.id, events, paused: true },
        })
      }
    }),
  ]

  try {
    // Execute workflow
    const result = await WorkflowExecutor.execute(executionCtx)

    // Build output
    const output = formatWorkflowResult(result, pausedStep)

    return {
      title: result.status === "paused" ? `Workflow paused: ${pausedStep?.message}` : `Workflow ${result.status}`,
      metadata: {
        instanceId: result.id,
        status: result.status,
        variables: result.variables,
        events,
        paused: result.status === "paused",
        pausedStep,
      },
      output,
    }
  } finally {
    unsubscribers.forEach((unsub) => unsub())
  }
}

async function handleResume(
  params: {
    workflow: string
    inputs?: Record<string, any>
    instanceId?: string
    approved?: boolean
    feedback?: string
  },
  ctx: Tool.Context,
): Promise<{ title: string; metadata: any; output: string }> {
  if (!params.instanceId) {
    return {
      title: "Missing instance ID",
      metadata: { error: true },
      output: "Instance ID is required for resume action",
    }
  }

  const instance = await WorkflowExecutor.get(params.instanceId)
  if (!instance) {
    return {
      title: "Instance not found",
      metadata: { error: true },
      output: `Workflow instance "${params.instanceId}" not found`,
    }
  }

  if (instance.status !== "paused") {
    return {
      title: "Not paused",
      metadata: { error: true, status: instance.status },
      output: `Workflow instance is not paused (current status: ${instance.status})`,
    }
  }

  log.info("resuming workflow", { instanceId: params.instanceId, approved: params.approved })

  // Set up pause handler for next pause
  let pausedStep: { stepId: string; message: string; options: any } | undefined

  const executionCtx: WorkflowExecutor.ExecutionContext = {
    instance,
    abort: ctx.abort,
    onPause: async (stepId, message, options) => {
      pausedStep = { stepId, message, options }
      return { approved: false }
    },
  }

  // Resume execution
  const result = await WorkflowExecutor.resume(executionCtx, {
    approved: params.approved ?? true,
    feedback: params.feedback,
  })

  const output = formatWorkflowResult(result, pausedStep)

  return {
    title: result.status === "paused" ? `Workflow paused: ${pausedStep?.message}` : `Workflow ${result.status}`,
    metadata: {
      instanceId: result.id,
      status: result.status,
      variables: result.variables,
      paused: result.status === "paused",
      pausedStep,
    },
    output,
  }
}

async function handleCancel(
  params: {
    workflow: string
    instanceId?: string
  },
  ctx: Tool.Context,
): Promise<{ title: string; metadata: any; output: string }> {
  if (!params.instanceId) {
    return {
      title: "Missing instance ID",
      metadata: { error: true },
      output: "Instance ID is required for cancel action",
    }
  }

  const instance = await WorkflowExecutor.cancel(params.instanceId, "Cancelled by user")

  return {
    title: "Workflow cancelled",
    metadata: {
      instanceId: instance.id,
      status: instance.status,
    },
    output: `Workflow "${instance.workflowId}" has been cancelled.`,
  }
}

async function handleStatus(
  params: {
    workflow: string
    instanceId?: string
  },
  ctx: Tool.Context,
): Promise<{ title: string; metadata: any; output: string }> {
  if (!params.instanceId) {
    // List recent instances for this workflow
    const instances: WorkflowInstance[] = []
    for await (const inst of WorkflowExecutor.list()) {
      if (inst.workflowId === params.workflow) {
        instances.push(inst)
      }
    }

    if (instances.length === 0) {
      return {
        title: "No instances found",
        metadata: { workflow: params.workflow },
        output: `No instances found for workflow "${params.workflow}"`,
      }
    }

    const lines = instances.slice(0, 10).map((inst) => {
      const duration = inst.time.completed
        ? `${((inst.time.completed - (inst.time.started ?? inst.time.created)) / 1000).toFixed(1)}s`
        : "in progress"
      return `- ${inst.id}: ${inst.status} (${duration})`
    })

    return {
      title: `Workflow instances (${instances.length})`,
      metadata: { workflow: params.workflow, instances: instances.slice(0, 10) },
      output: `Recent instances for "${params.workflow}":\n${lines.join("\n")}`,
    }
  }

  const instance = await WorkflowExecutor.get(params.instanceId)
  if (!instance) {
    return {
      title: "Instance not found",
      metadata: { error: true },
      output: `Workflow instance "${params.instanceId}" not found`,
    }
  }

  return {
    title: `Workflow ${instance.status}`,
    metadata: {
      instanceId: instance.id,
      status: instance.status,
      variables: instance.variables,
      stepStates: instance.stepStates,
    },
    output: formatWorkflowResult(instance, undefined),
  }
}

function formatWorkflowResult(
  instance: WorkflowInstance,
  pausedStep: { stepId: string; message: string; options: any } | undefined,
): string {
  const lines: string[] = []

  lines.push(`# Workflow: ${instance.definition.name}`)
  lines.push(`**Status**: ${instance.status}`)
  lines.push(`**Instance ID**: ${instance.id}`)
  lines.push("")

  // Step states
  lines.push("## Steps")
  for (const step of instance.definition.steps) {
    const state = instance.stepStates[step.id]
    const statusIcon =
      {
        pending: "⏳",
        running: "🔄",
        paused: "⏸️",
        completed: "✅",
        failed: "❌",
        skipped: "⏭️",
        cancelled: "🚫",
      }[state?.status ?? "pending"] ?? "?"

    lines.push(`${statusIcon} **${step.id}** (${step.type}): ${state?.status ?? "pending"}`)
    if (state?.error) {
      lines.push(`   Error: ${state.error}`)
    }
  }
  lines.push("")

  // Paused info
  if (instance.status === "paused" && pausedStep) {
    lines.push("## ⏸️ Paused for Review")
    lines.push(`**Step**: ${pausedStep.stepId}`)
    lines.push(`**Message**: ${pausedStep.message}`)
    lines.push("")
    lines.push("To continue, use the workflow tool with:")
    lines.push("```")
    lines.push(`action: "resume"`)
    lines.push(`instanceId: "${instance.id}"`)
    lines.push(`approved: true  # or false to reject`)
    lines.push("```")
    lines.push("")
  }

  // Output variables
  if (Object.keys(instance.variables).length > 0) {
    lines.push("## Variables")
    for (const [key, value] of Object.entries(instance.variables)) {
      const displayValue = typeof value === "string" && value.length > 100 ? value.slice(0, 100) + "..." : value
      lines.push(`- **${key}**: ${JSON.stringify(displayValue)}`)
    }
    lines.push("")
  }

  // Error info
  if (instance.error) {
    lines.push("## ❌ Error")
    lines.push(`**Message**: ${instance.error.message}`)
    if (instance.error.stepId) {
      lines.push(`**Step**: ${instance.error.stepId}`)
    }
  }

  return lines.join("\n")
}
