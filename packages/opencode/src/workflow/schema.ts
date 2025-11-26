import z from "zod"
import { Identifier } from "../id/id"
import { Config } from "../config/config"

/**
 * Agentic Workflow DSL Schema
 *
 * This module defines the schema for declarative agentic workflows that can
 * orchestrate multiple subagents with their own prompts, tools, and execution flows.
 *
 * Key concepts:
 * - **Workflow**: A named collection of steps that execute subagents
 * - **Step**: A unit of work executed by a specific agent
 * - **Transition**: Rules for moving between steps (sequential, parallel, conditional)
 * - **Orchestrator**: Controls workflow execution with pause/resume capabilities
 * - **Variables**: Data passed between steps
 */

// =============================================================================
// Agent Configuration
// =============================================================================

/**
 * Agent configuration within a workflow.
 * Defines the subagent's behavior, available tools, and prompt.
 */
export const WorkflowAgentConfig = z
  .object({
    /** Unique identifier for this agent within the workflow */
    name: z.string(),

    /** Human-readable description of the agent's purpose */
    description: z.string().optional(),

    /** The system prompt that defines the agent's behavior */
    prompt: z.string(),

    /** Model to use for this agent (format: provider/model-id) */
    model: z.string().optional(),

    /** Tools enabled for this agent */
    tools: z.record(z.string(), z.boolean()).optional(),

    /** Permission overrides for this agent */
    permission: z
      .object({
        edit: Config.Permission.optional(),
        bash: z.union([Config.Permission, z.record(z.string(), Config.Permission)]).optional(),
        webfetch: Config.Permission.optional(),
      })
      .optional(),

    /** Temperature setting for the model */
    temperature: z.number().min(0).max(2).optional(),

    /** Top-p sampling parameter */
    topP: z.number().min(0).max(1).optional(),

    /** Maximum tokens for response */
    maxTokens: z.number().positive().optional(),
  })
  .meta({
    ref: "WorkflowAgentConfig",
  })

export type WorkflowAgentConfig = z.infer<typeof WorkflowAgentConfig>

// =============================================================================
// Step Types
// =============================================================================

/**
 * Base step properties shared by all step types
 */
const StepBase = z.object({
  /** Unique identifier for this step */
  id: z.string(),

  /** Human-readable name for this step */
  name: z.string().optional(),

  /** Description of what this step does */
  description: z.string().optional(),

  /** Step IDs that must complete before this step can run */
  dependsOn: z.array(z.string()).optional(),

  /** Condition expression that must be true for this step to run */
  condition: z.string().optional(),

  /** Timeout in milliseconds for this step */
  timeout: z.number().positive().optional(),

  /** Number of retries on failure */
  retries: z.number().min(0).max(10).default(0),

  /** Custom metadata for this step */
  metadata: z.record(z.string(), z.any()).optional(),
})

/**
 * Agent step - executes a subagent with a prompt
 */
export const AgentStep = StepBase.extend({
  type: z.literal("agent"),

  /** The agent to use (references workflow.agents or global agents) */
  agent: z.string(),

  /** The prompt to send to the agent. Supports variable interpolation: {{varName}} */
  input: z.string(),

  /** Variable name to store the agent's output */
  output: z.string().optional(),

  /** Additional context to pass to the agent */
  context: z.record(z.string(), z.string()).optional(),
}).meta({
  ref: "AgentStep",
})

export type AgentStep = z.infer<typeof AgentStep>

/**
 * Pause step - waits for human review/approval
 */
export const PauseStep = StepBase.extend({
  type: z.literal("pause"),

  /** Message to display to the user */
  message: z.string(),

  /** Variable name to store the approval result */
  approvalVariable: z.string().optional(),

  /** Options for the pause action */
  options: z
    .object({
      /** Whether to allow editing of previous step outputs */
      allowEdit: z.boolean().default(false),

      /** Whether to allow rejecting and going back */
      allowReject: z.boolean().default(true),

      /** Custom approval labels */
      approveLabel: z.string().default("Approve"),
      rejectLabel: z.string().default("Reject"),

      /** Auto-approve after timeout (in milliseconds) */
      autoApproveAfter: z.number().positive().optional(),
    })
    .optional(),
}).meta({
  ref: "PauseStep",
})

export type PauseStep = z.infer<typeof PauseStep>

/**
 * Parallel step - executes multiple steps concurrently
 */
export const ParallelStep = StepBase.extend({
  type: z.literal("parallel"),

  /** Step IDs to execute in parallel */
  steps: z.array(z.string()),

  /** How to handle failures: "fail-fast" stops on first failure, "continue" runs all */
  onFailure: z.enum(["fail-fast", "continue"]).default("fail-fast"),

  /** Maximum concurrent executions (0 = unlimited) */
  maxConcurrency: z.number().min(0).default(0),
}).meta({
  ref: "ParallelStep",
})

export type ParallelStep = z.infer<typeof ParallelStep>

/**
 * Conditional step - branches based on a condition
 */
export const ConditionalStep = StepBase.extend({
  type: z.literal("conditional"),

  /** Condition expression to evaluate */
  condition: z.string(),

  /** Step ID to execute if condition is true */
  then: z.string(),

  /** Step ID to execute if condition is false */
  else: z.string().optional(),
}).meta({
  ref: "ConditionalStep",
})

export type ConditionalStep = z.infer<typeof ConditionalStep>

/**
 * Loop step - repeats steps until a condition is met
 */
export const LoopStep = StepBase.extend({
  type: z.literal("loop"),

  /** Steps to execute in each iteration */
  steps: z.array(z.string()),

  /** Condition to check before each iteration (continues while true) */
  while: z.string().optional(),

  /** Condition to check after each iteration (continues until true) */
  until: z.string().optional(),

  /** Maximum number of iterations */
  maxIterations: z.number().min(1).default(10),

  /** Variable name for the current iteration index */
  indexVariable: z.string().default("_loopIndex"),
}).meta({
  ref: "LoopStep",
})

export type LoopStep = z.infer<typeof LoopStep>

/**
 * Transform step - transforms data between steps
 */
export const TransformStep = StepBase.extend({
  type: z.literal("transform"),

  /** Input variable or expression */
  input: z.string(),

  /** Output variable name */
  output: z.string(),

  /** Transformation type */
  transform: z.enum([
    "json-parse",
    "json-stringify",
    "extract-code",
    "extract-json",
    "template",
    "split",
    "join",
    "trim",
    "uppercase",
    "lowercase",
  ]),

  /** Additional options for the transformation */
  options: z.record(z.string(), z.any()).optional(),
}).meta({
  ref: "TransformStep",
})

export type TransformStep = z.infer<typeof TransformStep>

/**
 * Union of all step types
 */
export const WorkflowStep = z
  .discriminatedUnion("type", [AgentStep, PauseStep, ParallelStep, ConditionalStep, LoopStep, TransformStep])
  .meta({
    ref: "WorkflowStep",
  })

export type WorkflowStep = z.infer<typeof WorkflowStep>

// =============================================================================
// Orchestrator Configuration
// =============================================================================

/**
 * Orchestrator configuration controls how the workflow is executed
 */
export const OrchestratorConfig = z
  .object({
    /** Execution mode */
    mode: z
      .enum([
        "auto", // Execute all steps automatically
        "guided", // Pause at pause steps only
        "manual", // Pause after each step for review
      ])
      .default("guided"),

    /** Error handling strategy */
    onError: z
      .enum([
        "pause", // Pause and wait for human input
        "retry", // Retry the failed step
        "fail", // Fail the entire workflow
        "skip", // Skip the failed step and continue
      ])
      .default("pause"),

    /** Maximum retries for failed steps */
    maxRetries: z.number().min(0).max(10).default(3),

    /** Default timeout per step (in milliseconds) */
    defaultTimeout: z.number().positive().default(300000), // 5 minutes

    /** Global workflow timeout (in milliseconds) */
    workflowTimeout: z.number().positive().optional(),

    /** Callback events */
    hooks: z
      .object({
        /** Called before workflow starts */
        onStart: z.string().optional(),

        /** Called after workflow completes */
        onComplete: z.string().optional(),

        /** Called on workflow error */
        onError: z.string().optional(),

        /** Called before each step */
        beforeStep: z.string().optional(),

        /** Called after each step */
        afterStep: z.string().optional(),
      })
      .optional(),

    /** Enable detailed logging */
    verbose: z.boolean().default(false),
  })
  .meta({
    ref: "OrchestratorConfig",
  })

export type OrchestratorConfig = z.infer<typeof OrchestratorConfig>

// =============================================================================
// Workflow Definition
// =============================================================================

/**
 * Complete workflow definition
 */
export const WorkflowDefinition = z
  .object({
    /** Workflow identifier */
    id: z.string(),

    /** Workflow name */
    name: z.string(),

    /** Workflow description */
    description: z.string().optional(),

    /** Workflow version */
    version: z.string().default("1.0.0"),

    /** Input variables required to start the workflow */
    inputs: z
      .record(
        z.string(),
        z.object({
          type: z.enum(["string", "number", "boolean", "array", "object"]),
          description: z.string().optional(),
          required: z.boolean().default(true),
          default: z.any().optional(),
        }),
      )
      .optional(),

    /** Agents defined within this workflow */
    agents: z.record(z.string(), WorkflowAgentConfig).optional(),

    /** Workflow steps */
    steps: z.array(WorkflowStep),

    /** Initial step ID (defaults to first step) */
    startStep: z.string().optional(),

    /** Orchestrator configuration */
    orchestrator: OrchestratorConfig.optional(),

    /** Tags for categorization */
    tags: z.array(z.string()).optional(),

    /** Custom metadata */
    metadata: z.record(z.string(), z.any()).optional(),
  })
  .meta({
    ref: "WorkflowDefinition",
  })

export type WorkflowDefinition = z.infer<typeof WorkflowDefinition>

// =============================================================================
// Workflow Instance (Runtime State)
// =============================================================================

/**
 * Status of a workflow step
 */
export const StepStatus = z.enum([
  "pending", // Not started
  "running", // Currently executing
  "paused", // Waiting for human input
  "completed", // Successfully completed
  "failed", // Failed with error
  "skipped", // Skipped (condition not met)
  "cancelled", // Cancelled by user
])

export type StepStatus = z.infer<typeof StepStatus>

/**
 * Runtime state of a step
 */
export const StepState = z
  .object({
    stepId: z.string(),
    status: StepStatus,
    startedAt: z.number().optional(),
    completedAt: z.number().optional(),
    output: z.any().optional(),
    error: z.string().optional(),
    retryCount: z.number().default(0),
    sessionId: z.string().optional(), // Session ID for agent steps
    metadata: z.record(z.string(), z.any()).optional(),
  })
  .meta({
    ref: "StepState",
  })

export type StepState = z.infer<typeof StepState>

/**
 * Workflow instance status
 */
export const WorkflowStatus = z.enum([
  "pending", // Not started
  "running", // In progress
  "paused", // Waiting for human input
  "completed", // Successfully completed
  "failed", // Failed with error
  "cancelled", // Cancelled by user
])

export type WorkflowStatus = z.infer<typeof WorkflowStatus>

/**
 * Runtime state of a workflow instance
 */
export const WorkflowInstance = z
  .object({
    /** Unique instance ID */
    id: Identifier.schema("workflow"),

    /** Reference to the workflow definition */
    workflowId: z.string(),

    /** Workflow definition snapshot (for reproducibility) */
    definition: WorkflowDefinition,

    /** Current status */
    status: WorkflowStatus,

    /** Current step ID */
    currentStepId: z.string().optional(),

    /** Input variables */
    inputs: z.record(z.string(), z.any()),

    /** Workflow variables (accumulated outputs) */
    variables: z.record(z.string(), z.any()),

    /** State of each step */
    stepStates: z.record(z.string(), StepState),

    /** Parent session ID */
    parentSessionId: z.string().optional(),

    /** Timestamps */
    time: z.object({
      created: z.number(),
      started: z.number().optional(),
      completed: z.number().optional(),
      updated: z.number(),
    }),

    /** Error information if failed */
    error: z
      .object({
        message: z.string(),
        stepId: z.string().optional(),
        stack: z.string().optional(),
      })
      .optional(),

    /** Execution log */
    log: z
      .array(
        z.object({
          timestamp: z.number(),
          level: z.enum(["info", "warn", "error", "debug"]),
          message: z.string(),
          stepId: z.string().optional(),
          metadata: z.record(z.string(), z.any()).optional(),
        }),
      )
      .optional(),
  })
  .meta({
    ref: "WorkflowInstance",
  })

export type WorkflowInstance = z.infer<typeof WorkflowInstance>

// =============================================================================
// Workflow Events
// =============================================================================

export const WorkflowEventPayloads = {
  Started: z.object({
    instanceId: z.string(),
    workflowId: z.string(),
    inputs: z.record(z.string(), z.any()),
  }),

  StepStarted: z.object({
    instanceId: z.string(),
    stepId: z.string(),
    stepType: z.string(),
  }),

  StepCompleted: z.object({
    instanceId: z.string(),
    stepId: z.string(),
    output: z.any().optional(),
    duration: z.number(),
  }),

  StepFailed: z.object({
    instanceId: z.string(),
    stepId: z.string(),
    error: z.string(),
    retryCount: z.number(),
  }),

  Paused: z.object({
    instanceId: z.string(),
    stepId: z.string(),
    message: z.string(),
    options: z.any().optional(),
  }),

  Resumed: z.object({
    instanceId: z.string(),
    stepId: z.string(),
    approved: z.boolean(),
    feedback: z.string().optional(),
  }),

  Completed: z.object({
    instanceId: z.string(),
    workflowId: z.string(),
    outputs: z.record(z.string(), z.any()),
    duration: z.number(),
  }),

  Failed: z.object({
    instanceId: z.string(),
    workflowId: z.string(),
    error: z.string(),
    stepId: z.string().optional(),
  }),

  Cancelled: z.object({
    instanceId: z.string(),
    workflowId: z.string(),
    reason: z.string().optional(),
  }),
}
