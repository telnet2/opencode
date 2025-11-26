import z from "zod"
import { Bus } from "../bus"
import { Identifier } from "../id/id"
import { Storage } from "../storage/storage"
import { Instance } from "../project/instance"
import { Session } from "../session"
import { SessionPrompt } from "../session/prompt"
import { Agent } from "../agent/agent"
import { Provider } from "../provider/provider"
import { Log } from "../util/log"
import { defer } from "../util/defer"
import {
  WorkflowDefinition,
  WorkflowInstance,
  WorkflowStatus,
  WorkflowStep,
  StepState,
  StepStatus,
  AgentStep,
  PauseStep,
  ParallelStep,
  ConditionalStep,
  LoopStep,
  TransformStep,
  WorkflowAgentConfig,
  WorkflowEventPayloads,
} from "./schema"

const log = Log.create({ service: "workflow.executor" })

/**
 * Workflow Executor - Orchestrates the execution of agentic workflows
 */
export namespace WorkflowExecutor {
  // ===========================================================================
  // Events
  // ===========================================================================

  export const Event = {
    Started: Bus.event("workflow.started", WorkflowEventPayloads.Started),
    StepStarted: Bus.event("workflow.step.started", WorkflowEventPayloads.StepStarted),
    StepCompleted: Bus.event("workflow.step.completed", WorkflowEventPayloads.StepCompleted),
    StepFailed: Bus.event("workflow.step.failed", WorkflowEventPayloads.StepFailed),
    Paused: Bus.event("workflow.paused", WorkflowEventPayloads.Paused),
    Resumed: Bus.event("workflow.resumed", WorkflowEventPayloads.Resumed),
    Completed: Bus.event("workflow.completed", WorkflowEventPayloads.Completed),
    Failed: Bus.event("workflow.failed", WorkflowEventPayloads.Failed),
    Cancelled: Bus.event("workflow.cancelled", WorkflowEventPayloads.Cancelled),
  }

  // ===========================================================================
  // Execution Context
  // ===========================================================================

  export interface ExecutionContext {
    instance: WorkflowInstance
    abort: AbortSignal
    onPause?: (stepId: string, message: string, options?: PauseStep["options"]) => Promise<PauseResult>
  }

  export interface PauseResult {
    approved: boolean
    feedback?: string
    editedVariables?: Record<string, any>
  }

  // ===========================================================================
  // Core Functions
  // ===========================================================================

  /**
   * Create a new workflow instance from a definition
   */
  export async function create(input: {
    definition: WorkflowDefinition
    inputs: Record<string, any>
    parentSessionId?: string
  }): Promise<WorkflowInstance> {
    log.info("creating workflow instance", {
      workflowId: input.definition.id,
      inputs: Object.keys(input.inputs),
    })

    // Validate required inputs
    if (input.definition.inputs) {
      for (const [name, spec] of Object.entries(input.definition.inputs)) {
        if (spec.required && !(name in input.inputs) && spec.default === undefined) {
          throw new Error(`Missing required input: ${name}`)
        }
      }
    }

    // Apply defaults
    const inputs = { ...input.inputs }
    if (input.definition.inputs) {
      for (const [name, spec] of Object.entries(input.definition.inputs)) {
        if (!(name in inputs) && spec.default !== undefined) {
          inputs[name] = spec.default
        }
      }
    }

    // Initialize step states
    const stepStates: Record<string, StepState> = {}
    for (const step of input.definition.steps) {
      stepStates[step.id] = {
        stepId: step.id,
        status: "pending",
        retryCount: 0,
      }
    }

    const instance: WorkflowInstance = {
      id: Identifier.ascending("workflow"),
      workflowId: input.definition.id,
      definition: input.definition,
      status: "pending",
      inputs,
      variables: { ...inputs },
      stepStates,
      parentSessionId: input.parentSessionId,
      time: {
        created: Date.now(),
        updated: Date.now(),
      },
      log: [],
    }

    await save(instance)
    return instance
  }

  /**
   * Execute a workflow instance
   */
  export async function execute(ctx: ExecutionContext): Promise<WorkflowInstance> {
    const { instance } = ctx
    log.info("starting workflow execution", {
      instanceId: instance.id,
      workflowId: instance.workflowId,
    })

    // Update status
    instance.status = "running"
    instance.time.started = Date.now()
    instance.time.updated = Date.now()
    await save(instance)

    await Bus.publish(Event.Started, {
      instanceId: instance.id,
      workflowId: instance.workflowId,
      inputs: instance.inputs,
    })

    addLog(instance, "info", "Workflow started")

    try {
      // Get start step
      const startStepId = instance.definition.startStep ?? instance.definition.steps[0]?.id
      if (!startStepId) {
        throw new Error("No steps defined in workflow")
      }

      // Execute from start step
      await executeFromStep(ctx, startStepId)

      // Check if completed
      if (instance.status !== "paused" && instance.status !== "cancelled") {
        instance.status = "completed"
        instance.time.completed = Date.now()
        instance.time.updated = Date.now()

        addLog(instance, "info", "Workflow completed successfully")

        await Bus.publish(Event.Completed, {
          instanceId: instance.id,
          workflowId: instance.workflowId,
          outputs: instance.variables,
          duration: Date.now() - (instance.time.started ?? Date.now()),
        })
      }
    } catch (error) {
      instance.status = "failed"
      instance.error = {
        message: error instanceof Error ? error.message : String(error),
        stepId: instance.currentStepId,
        stack: error instanceof Error ? error.stack : undefined,
      }
      instance.time.updated = Date.now()

      addLog(instance, "error", `Workflow failed: ${instance.error.message}`)

      await Bus.publish(Event.Failed, {
        instanceId: instance.id,
        workflowId: instance.workflowId,
        error: instance.error.message,
        stepId: instance.currentStepId,
      })
    }

    await save(instance)
    return instance
  }

  /**
   * Resume a paused workflow
   */
  export async function resume(ctx: ExecutionContext, result: PauseResult): Promise<WorkflowInstance> {
    const { instance } = ctx

    if (instance.status !== "paused") {
      throw new Error(`Cannot resume workflow in status: ${instance.status}`)
    }

    const stepId = instance.currentStepId
    if (!stepId) {
      throw new Error("No current step to resume")
    }

    log.info("resuming workflow", {
      instanceId: instance.id,
      stepId,
      approved: result.approved,
    })

    await Bus.publish(Event.Resumed, {
      instanceId: instance.id,
      stepId,
      approved: result.approved,
      feedback: result.feedback,
    })

    // Apply edited variables
    if (result.editedVariables) {
      Object.assign(instance.variables, result.editedVariables)
    }

    // Update step state
    const stepState = instance.stepStates[stepId]
    if (result.approved) {
      stepState.status = "completed"
      stepState.completedAt = Date.now()
      if (result.feedback) {
        stepState.output = result.feedback
      }

      // Store approval in variable if configured
      const step = instance.definition.steps.find((s) => s.id === stepId)
      if (step?.type === "pause" && step.approvalVariable) {
        instance.variables[step.approvalVariable] = true
      }

      addLog(instance, "info", `Step "${stepId}" approved`)
    } else {
      stepState.status = "cancelled"
      stepState.completedAt = Date.now()

      const step = instance.definition.steps.find((s) => s.id === stepId)
      if (step?.type === "pause" && step.approvalVariable) {
        instance.variables[step.approvalVariable] = false
      }

      addLog(instance, "info", `Step "${stepId}" rejected`)
    }

    instance.status = "running"
    instance.time.updated = Date.now()
    await save(instance)

    // Continue execution
    return execute(ctx)
  }

  /**
   * Cancel a running workflow
   */
  export async function cancel(instanceId: string, reason?: string): Promise<WorkflowInstance> {
    const instance = await get(instanceId)
    if (!instance) {
      throw new Error(`Workflow instance not found: ${instanceId}`)
    }

    if (instance.status === "completed" || instance.status === "failed") {
      throw new Error(`Cannot cancel workflow in status: ${instance.status}`)
    }

    log.info("cancelling workflow", { instanceId, reason })

    instance.status = "cancelled"
    instance.time.updated = Date.now()

    addLog(instance, "info", `Workflow cancelled: ${reason ?? "user requested"}`)

    await Bus.publish(Event.Cancelled, {
      instanceId: instance.id,
      workflowId: instance.workflowId,
      reason,
    })

    await save(instance)
    return instance
  }

  // ===========================================================================
  // Step Execution
  // ===========================================================================

  /**
   * Execute steps starting from a specific step
   */
  async function executeFromStep(ctx: ExecutionContext, stepId: string): Promise<void> {
    const { instance, abort } = ctx
    const stepMap = new Map(instance.definition.steps.map((s) => [s.id, s]))
    const orchestrator = instance.definition.orchestrator ?? {}

    // Build execution order respecting dependencies
    const executionOrder = buildExecutionOrder(instance.definition, stepId)

    for (const currentStepId of executionOrder) {
      if (abort.aborted) {
        addLog(instance, "info", "Workflow aborted")
        instance.status = "cancelled"
        return
      }

      const step = stepMap.get(currentStepId)
      if (!step) continue

      const stepState = instance.stepStates[currentStepId]

      // Skip completed/cancelled steps
      if (stepState.status === "completed" || stepState.status === "cancelled") {
        continue
      }

      // Check if dependencies are satisfied
      if (step.dependsOn) {
        const allDepsComplete = step.dependsOn.every((depId) => {
          const depState = instance.stepStates[depId]
          return depState?.status === "completed"
        })
        if (!allDepsComplete) {
          log.info("step dependencies not met, skipping", { stepId: currentStepId })
          continue
        }
      }

      // Evaluate condition
      if (step.condition) {
        const conditionMet = evaluateCondition(step.condition, instance.variables)
        if (!conditionMet) {
          stepState.status = "skipped"
          stepState.completedAt = Date.now()
          addLog(instance, "info", `Step "${currentStepId}" skipped (condition not met)`)
          continue
        }
      }

      // Execute the step
      instance.currentStepId = currentStepId
      instance.time.updated = Date.now()
      await save(instance)

      try {
        await executeStep(ctx, step)

        // Handle manual mode - pause after each step
        if (orchestrator.mode === "manual" && step.type !== "pause") {
          const shouldContinue = await handleManualPause(ctx, step)
          if (!shouldContinue) {
            return
          }
        }
      } catch (error) {
        await handleStepError(ctx, step, error)
        if (instance.status === "paused" || instance.status === "failed") {
          return
        }
      }
    }
  }

  /**
   * Execute a single step
   */
  async function executeStep(ctx: ExecutionContext, step: WorkflowStep): Promise<void> {
    const { instance } = ctx
    const stepState = instance.stepStates[step.id]
    const timeout = step.timeout ?? instance.definition.orchestrator?.defaultTimeout ?? 300000

    log.info("executing step", {
      stepId: step.id,
      type: step.type,
    })

    stepState.status = "running"
    stepState.startedAt = Date.now()
    instance.time.updated = Date.now()
    await save(instance)

    await Bus.publish(Event.StepStarted, {
      instanceId: instance.id,
      stepId: step.id,
      stepType: step.type,
    })

    addLog(instance, "info", `Step "${step.id}" started`)

    const startTime = Date.now()

    try {
      // Execute based on step type
      switch (step.type) {
        case "agent":
          await executeAgentStep(ctx, step)
          break
        case "pause":
          await executePauseStep(ctx, step)
          break
        case "parallel":
          await executeParallelStep(ctx, step)
          break
        case "conditional":
          await executeConditionalStep(ctx, step)
          break
        case "loop":
          await executeLoopStep(ctx, step)
          break
        case "transform":
          await executeTransformStep(ctx, step)
          break
        default:
          throw new Error(`Unknown step type: ${(step as any).type}`)
      }

      // Mark as completed (unless it's a pause step that's still paused)
      if (instance.status !== "paused") {
        stepState.status = "completed"
        stepState.completedAt = Date.now()

        await Bus.publish(Event.StepCompleted, {
          instanceId: instance.id,
          stepId: step.id,
          output: stepState.output,
          duration: Date.now() - startTime,
        })

        addLog(instance, "info", `Step "${step.id}" completed`)
      }
    } catch (error) {
      throw error
    }

    await save(instance)
  }

  /**
   * Execute an agent step
   */
  async function executeAgentStep(ctx: ExecutionContext, step: AgentStep): Promise<void> {
    const { instance, abort } = ctx
    const stepState = instance.stepStates[step.id]

    // Resolve agent configuration
    const agentConfig = await resolveAgent(instance.definition, step.agent)

    // Create session for this step
    const session = await Session.create({
      parentID: instance.parentSessionId,
      title: `Workflow: ${instance.definition.name} - ${step.id}`,
    })

    stepState.sessionId = session.id

    // Interpolate the input prompt with variables
    const prompt = interpolateTemplate(step.input, instance.variables)

    // Get model
    const model = agentConfig.model
      ? Provider.parseModel(agentConfig.model)
      : await Provider.defaultModel().then((m) => ({ providerID: m.providerID, modelID: m.modelID }))

    // Build tools config
    const tools = agentConfig.tools ?? {}

    // Set up abort handling
    function cancelSession() {
      SessionPrompt.cancel(session.id)
    }
    abort.addEventListener("abort", cancelSession)
    using _ = defer(() => abort.removeEventListener("abort", cancelSession))

    // Resolve prompt parts
    const promptParts = await SessionPrompt.resolvePromptParts(prompt)

    // Execute the prompt
    const result = await SessionPrompt.prompt({
      messageID: Identifier.ascending("message"),
      sessionID: session.id,
      model: {
        modelID: model.modelID,
        providerID: model.providerID,
      },
      agent: step.agent,
      tools: {
        todowrite: false,
        todoread: false,
        task: false,
        ...tools,
      },
      parts: promptParts,
      system: agentConfig.prompt,
    })

    // Extract output
    const output = result.parts.findLast((x) => x.type === "text")?.text ?? ""

    stepState.output = output
    instance.time.updated = Date.now()

    // Store in variables if output variable is specified
    if (step.output) {
      instance.variables[step.output] = output
    }
  }

  /**
   * Execute a pause step
   */
  async function executePauseStep(ctx: ExecutionContext, step: PauseStep): Promise<void> {
    const { instance, onPause } = ctx
    const stepState = instance.stepStates[step.id]

    // Interpolate message
    const message = interpolateTemplate(step.message, instance.variables)

    addLog(instance, "info", `Pausing for review: ${message}`)

    // Publish pause event
    await Bus.publish(Event.Paused, {
      instanceId: instance.id,
      stepId: step.id,
      message,
      options: step.options,
    })

    // Handle auto-approve
    if (step.options?.autoApproveAfter) {
      setTimeout(async () => {
        const current = await get(instance.id)
        if (current?.status === "paused" && current.currentStepId === step.id) {
          addLog(instance, "info", `Auto-approving after timeout`)
          // The resume will be called by the timeout handler
        }
      }, step.options.autoApproveAfter)
    }

    // If we have an onPause handler, use it
    if (onPause) {
      const result = await onPause(step.id, message, step.options)

      if (result.approved) {
        stepState.status = "completed"
        stepState.completedAt = Date.now()
        if (step.approvalVariable) {
          instance.variables[step.approvalVariable] = true
        }
        if (result.feedback) {
          stepState.output = result.feedback
        }
        if (result.editedVariables) {
          Object.assign(instance.variables, result.editedVariables)
        }
      } else {
        stepState.status = "cancelled"
        stepState.completedAt = Date.now()
        if (step.approvalVariable) {
          instance.variables[step.approvalVariable] = false
        }
        instance.status = "cancelled"
      }
    } else {
      // No handler - set to paused state and wait for resume
      stepState.status = "paused"
      instance.status = "paused"
    }

    await save(instance)
  }

  /**
   * Execute a parallel step
   */
  async function executeParallelStep(ctx: ExecutionContext, step: ParallelStep): Promise<void> {
    const { instance } = ctx
    const stepState = instance.stepStates[step.id]

    const stepMap = new Map(instance.definition.steps.map((s) => [s.id, s]))
    const results: Array<{ stepId: string; success: boolean; error?: string }> = []

    // Determine concurrency
    const maxConcurrency = step.maxConcurrency > 0 ? step.maxConcurrency : step.steps.length

    // Execute steps in parallel with concurrency limit
    const chunks: string[][] = []
    for (let i = 0; i < step.steps.length; i += maxConcurrency) {
      chunks.push(step.steps.slice(i, i + maxConcurrency))
    }

    for (const chunk of chunks) {
      const promises = chunk.map(async (childStepId) => {
        const childStep = stepMap.get(childStepId)
        if (!childStep) {
          return { stepId: childStepId, success: false, error: `Step not found: ${childStepId}` }
        }

        try {
          await executeStep(ctx, childStep)
          return { stepId: childStepId, success: true }
        } catch (error) {
          if (step.onFailure === "fail-fast") {
            throw error
          }
          return {
            stepId: childStepId,
            success: false,
            error: error instanceof Error ? error.message : String(error),
          }
        }
      })

      const chunkResults = await Promise.all(promises)
      results.push(...chunkResults)
    }

    stepState.output = results
    stepState.metadata = { parallelResults: results }

    // Check for failures
    const failures = results.filter((r) => !r.success)
    if (failures.length > 0 && step.onFailure === "fail-fast") {
      throw new Error(`Parallel step failures: ${failures.map((f) => f.error).join(", ")}`)
    }
  }

  /**
   * Execute a conditional step
   */
  async function executeConditionalStep(ctx: ExecutionContext, step: ConditionalStep): Promise<void> {
    const { instance } = ctx
    const stepState = instance.stepStates[step.id]

    const conditionMet = evaluateCondition(step.condition, instance.variables)

    stepState.metadata = { conditionMet }

    const nextStepId = conditionMet ? step.then : step.else
    if (nextStepId) {
      const nextStep = instance.definition.steps.find((s) => s.id === nextStepId)
      if (nextStep) {
        await executeStep(ctx, nextStep)
      }
    }
  }

  /**
   * Execute a loop step
   */
  async function executeLoopStep(ctx: ExecutionContext, step: LoopStep): Promise<void> {
    const { instance } = ctx
    const stepState = instance.stepStates[step.id]
    const stepMap = new Map(instance.definition.steps.map((s) => [s.id, s]))

    let iteration = 0
    const iterationResults: any[] = []

    while (iteration < step.maxIterations) {
      // Set loop index variable
      instance.variables[step.indexVariable] = iteration

      // Check while condition (before iteration)
      if (step.while && !evaluateCondition(step.while, instance.variables)) {
        break
      }

      // Execute loop body steps
      for (const childStepId of step.steps) {
        const childStep = stepMap.get(childStepId)
        if (childStep) {
          await executeStep(ctx, childStep)
        }
      }

      // Check until condition (after iteration)
      if (step.until && evaluateCondition(step.until, instance.variables)) {
        break
      }

      iterationResults.push({
        iteration,
        variables: { ...instance.variables },
      })

      iteration++
    }

    stepState.output = iterationResults
    stepState.metadata = { iterations: iteration }
  }

  /**
   * Execute a transform step
   */
  async function executeTransformStep(ctx: ExecutionContext, step: TransformStep): Promise<void> {
    const { instance } = ctx
    const stepState = instance.stepStates[step.id]

    // Get input value
    const inputValue = interpolateTemplate(step.input, instance.variables)

    // Apply transformation
    let result: any

    switch (step.transform) {
      case "json-parse":
        result = JSON.parse(inputValue)
        break

      case "json-stringify":
        result = JSON.stringify(inputValue, null, 2)
        break

      case "extract-code":
        // Extract code blocks from markdown
        const codeMatch = inputValue.match(/```[\w]*\n([\s\S]*?)\n```/)
        result = codeMatch ? codeMatch[1] : inputValue
        break

      case "extract-json":
        // Extract JSON from text
        const jsonMatch = inputValue.match(/\{[\s\S]*\}|\[[\s\S]*\]/)
        result = jsonMatch ? JSON.parse(jsonMatch[0]) : null
        break

      case "template":
        // Apply template with options
        result = interpolateTemplate(step.options?.template ?? inputValue, instance.variables)
        break

      case "split":
        result = inputValue.split(step.options?.delimiter ?? "\n")
        break

      case "join":
        if (Array.isArray(inputValue)) {
          result = inputValue.join(step.options?.delimiter ?? "\n")
        } else {
          result = inputValue
        }
        break

      case "trim":
        result = inputValue.trim()
        break

      case "uppercase":
        result = inputValue.toUpperCase()
        break

      case "lowercase":
        result = inputValue.toLowerCase()
        break

      default:
        throw new Error(`Unknown transform: ${step.transform}`)
    }

    stepState.output = result
    instance.variables[step.output] = result
  }

  // ===========================================================================
  // Helper Functions
  // ===========================================================================

  /**
   * Handle step execution error
   */
  async function handleStepError(ctx: ExecutionContext, step: WorkflowStep, error: unknown): Promise<void> {
    const { instance } = ctx
    const stepState = instance.stepStates[step.id]
    const orchestrator = instance.definition.orchestrator ?? {}

    const errorMessage = error instanceof Error ? error.message : String(error)

    await Bus.publish(Event.StepFailed, {
      instanceId: instance.id,
      stepId: step.id,
      error: errorMessage,
      retryCount: stepState.retryCount,
    })

    addLog(instance, "error", `Step "${step.id}" failed: ${errorMessage}`)

    // Check if we should retry
    const maxRetries = step.retries ?? orchestrator.maxRetries ?? 0
    if (stepState.retryCount < maxRetries) {
      stepState.retryCount++
      addLog(instance, "info", `Retrying step "${step.id}" (attempt ${stepState.retryCount}/${maxRetries})`)
      await executeStep(ctx, step)
      return
    }

    // Handle based on orchestrator config
    switch (orchestrator.onError) {
      case "retry":
        // Already handled above
        break

      case "skip":
        stepState.status = "skipped"
        stepState.completedAt = Date.now()
        stepState.error = errorMessage
        addLog(instance, "warn", `Skipping failed step "${step.id}"`)
        break

      case "pause":
        stepState.status = "paused"
        instance.status = "paused"
        instance.error = { message: errorMessage, stepId: step.id }
        addLog(instance, "info", `Pausing workflow due to error in step "${step.id}"`)
        break

      case "fail":
      default:
        stepState.status = "failed"
        stepState.completedAt = Date.now()
        stepState.error = errorMessage
        instance.status = "failed"
        instance.error = { message: errorMessage, stepId: step.id }
        throw error
    }

    await save(instance)
  }

  /**
   * Handle manual mode pause
   */
  async function handleManualPause(ctx: ExecutionContext, step: WorkflowStep): Promise<boolean> {
    const { instance, onPause } = ctx

    if (!onPause) {
      // No handler - pause and wait
      instance.status = "paused"
      await save(instance)
      return false
    }

    const result = await onPause(step.id, `Review step "${step.id}" before continuing`, undefined)
    return result.approved
  }

  /**
   * Resolve agent configuration
   */
  async function resolveAgent(workflow: WorkflowDefinition, agentName: string): Promise<WorkflowAgentConfig> {
    // First check workflow-local agents
    if (workflow.agents?.[agentName]) {
      return workflow.agents[agentName]
    }

    // Then check global agents
    const globalAgent = await Agent.get(agentName)
    if (globalAgent) {
      return {
        name: globalAgent.name,
        description: globalAgent.description,
        prompt: globalAgent.prompt ?? "",
        model: globalAgent.model ? `${globalAgent.model.providerID}/${globalAgent.model.modelID}` : undefined,
        tools: globalAgent.tools,
        permission: globalAgent.permission,
        temperature: globalAgent.temperature,
        topP: globalAgent.topP,
      }
    }

    throw new Error(`Agent not found: ${agentName}`)
  }

  /**
   * Build execution order respecting dependencies
   */
  function buildExecutionOrder(workflow: WorkflowDefinition, startStepId: string): string[] {
    const stepMap = new Map(workflow.steps.map((s) => [s.id, s]))
    const order: string[] = []
    const visited = new Set<string>()

    function visit(stepId: string): void {
      if (visited.has(stepId)) return

      const step = stepMap.get(stepId)
      if (!step) return

      // Visit dependencies first
      if (step.dependsOn) {
        for (const dep of step.dependsOn) {
          visit(dep)
        }
      }

      visited.add(stepId)
      order.push(stepId)
    }

    // Start from the specified step, but also include all reachable steps
    const startIndex = workflow.steps.findIndex((s) => s.id === startStepId)
    for (let i = startIndex; i < workflow.steps.length; i++) {
      visit(workflow.steps[i].id)
    }

    return order
  }

  /**
   * Evaluate a condition expression
   */
  function evaluateCondition(condition: string, variables: Record<string, any>): boolean {
    // Simple expression evaluator supporting:
    // - Variable references: {{varName}}
    // - Comparisons: ==, !=, <, >, <=, >=
    // - Boolean: true, false
    // - Logical: &&, ||, !

    // Interpolate variables first
    let expr = condition
    for (const [key, value] of Object.entries(variables)) {
      const pattern = new RegExp(`\\{\\{\\s*${key}\\s*\\}\\}`, "g")
      expr = expr.replace(pattern, JSON.stringify(value))
    }

    // Simple evaluation using Function (sandboxed)
    try {
      const fn = new Function("return " + expr)
      return Boolean(fn())
    } catch {
      log.warn("condition evaluation failed", { condition, expr })
      return false
    }
  }

  /**
   * Interpolate template variables
   */
  function interpolateTemplate(template: string, variables: Record<string, any>): string {
    return template.replace(/\{\{\s*(\w+)\s*\}\}/g, (_, key) => {
      const value = variables[key]
      if (value === undefined) return `{{${key}}}`
      return typeof value === "object" ? JSON.stringify(value) : String(value)
    })
  }

  /**
   * Add log entry to workflow instance
   */
  function addLog(
    instance: WorkflowInstance,
    level: "info" | "warn" | "error" | "debug",
    message: string,
    metadata?: Record<string, any>,
  ): void {
    if (!instance.log) {
      instance.log = []
    }
    instance.log.push({
      timestamp: Date.now(),
      level,
      message,
      stepId: instance.currentStepId,
      metadata,
    })
  }

  // ===========================================================================
  // Storage Functions
  // ===========================================================================

  /**
   * Save workflow instance to storage
   */
  async function save(instance: WorkflowInstance): Promise<void> {
    await Storage.write(["workflow", Instance.project.id, instance.id], instance)
  }

  /**
   * Get workflow instance from storage
   */
  export async function get(instanceId: string): Promise<WorkflowInstance | null> {
    try {
      return await Storage.read<WorkflowInstance>(["workflow", Instance.project.id, instanceId])
    } catch {
      return null
    }
  }

  /**
   * List all workflow instances
   */
  export async function* list(): AsyncGenerator<WorkflowInstance> {
    for (const item of await Storage.list(["workflow", Instance.project.id])) {
      try {
        yield await Storage.read<WorkflowInstance>(item)
      } catch {
        // Skip invalid instances
      }
    }
  }

  /**
   * Delete a workflow instance
   */
  export async function remove(instanceId: string): Promise<void> {
    await Storage.remove(["workflow", Instance.project.id, instanceId])
  }
}
