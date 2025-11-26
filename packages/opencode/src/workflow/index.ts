import { readdir, readFile } from "node:fs/promises"
import { join, extname, basename } from "node:path"
import { homedir } from "node:os"
import z from "zod"
import { Instance } from "../project/instance"
import { Config } from "../config/config"
import { Log } from "../util/log"
import {
  WorkflowDefinition,
  WorkflowInstance,
  WorkflowStatus,
  WorkflowStep,
  WorkflowAgentConfig,
  StepState,
  StepStatus,
} from "./schema"
import { WorkflowParser } from "./parser"
import { WorkflowExecutor } from "./executor"

const log = Log.create({ service: "workflow" })

// Re-export types and modules
export * from "./schema"
export { WorkflowParser } from "./parser"
export { WorkflowExecutor } from "./executor"
export { WorkflowTool } from "./tool"

/**
 * Workflow module - manages agentic workflow definitions and execution
 */
export namespace Workflow {
  /**
   * Configuration for workflows in opencode.jsonc
   */
  export const ConfigSchema = z.record(
    z.string(),
    z
      .object({
        /** Disable this workflow */
        disable: z.boolean().optional(),
        /** Path to workflow file (relative to .opencode/ or absolute) */
        path: z.string().optional(),
        /** Inline workflow definition */
        definition: WorkflowDefinition.partial().optional(),
      })
      .or(WorkflowDefinition),
  )

  export type ConfigType = z.infer<typeof ConfigSchema>

  /**
   * Cached workflow definitions
   */
  const state = Instance.state(async () => {
    const workflows = new Map<string, WorkflowDefinition>()

    // Load workflows from configuration
    const cfg = await Config.get()
    if ((cfg as any).workflow) {
      for (const [id, workflowCfg] of Object.entries((cfg as any).workflow as ConfigType)) {
        try {
          if ("disable" in workflowCfg && workflowCfg.disable) {
            continue
          }

          if ("path" in workflowCfg && workflowCfg.path) {
            const definition = await loadFromPath(workflowCfg.path, id)
            if (definition) {
              workflows.set(id, definition)
            }
          } else if ("definition" in workflowCfg && workflowCfg.definition) {
            const definition = WorkflowParser.validate({ id, ...workflowCfg.definition }, `config:${id}`)
            workflows.set(id, definition)
          } else if ("steps" in workflowCfg) {
            // Direct workflow definition
            const definition = WorkflowParser.validate({ id, ...workflowCfg }, `config:${id}`)
            workflows.set(id, definition)
          }
        } catch (error) {
          log.error("failed to load workflow from config", { id, error })
        }
      }
    }

    // Load workflows from .opencode/workflow/*.md or .opencode/workflow/*.yaml
    const workflowDirs = [
      join(Instance.directory, ".opencode", "workflow"),
      join(homedir(), ".opencode", "workflow"),
    ]

    for (const dir of workflowDirs) {
      try {
        const files = await readdir(dir).catch(() => [])
        for (const file of files) {
          const ext = extname(file)
          if (![".md", ".yaml", ".yml", ".json"].includes(ext)) continue

          try {
            const filePath = join(dir, file)
            const content = await readFile(filePath, "utf-8")
            const definition = WorkflowParser.parse(content, filePath)

            // Use filename as ID if not specified
            if (!definition.id) {
              (definition as any).id = basename(file, ext)
            }

            workflows.set(definition.id, definition)
            log.info("loaded workflow", { id: definition.id, path: filePath })
          } catch (error) {
            log.error("failed to load workflow file", { file: join(dir, file), error })
          }
        }
      } catch {
        // Directory doesn't exist, skip
      }
    }

    return { workflows }
  })

  /**
   * Load a workflow from a file path
   */
  async function loadFromPath(path: string, defaultId: string): Promise<WorkflowDefinition | null> {
    let fullPath: string

    if (path.startsWith("/")) {
      fullPath = path
    } else if (path.startsWith("~/")) {
      fullPath = join(homedir(), path.slice(2))
    } else {
      fullPath = join(Instance.directory, ".opencode", path)
    }

    try {
      const content = await readFile(fullPath, "utf-8")
      const definition = WorkflowParser.parse(content, fullPath)
      if (!definition.id) {
        (definition as any).id = defaultId
      }
      return definition
    } catch (error) {
      log.error("failed to load workflow from path", { path: fullPath, error })
      return null
    }
  }

  /**
   * Get a workflow definition by ID
   */
  export async function get(id: string): Promise<WorkflowDefinition | undefined> {
    const { workflows } = await state()
    return workflows.get(id)
  }

  /**
   * List all workflow definitions
   */
  export async function list(): Promise<WorkflowDefinition[]> {
    const { workflows } = await state()
    return Array.from(workflows.values())
  }

  /**
   * Check if a workflow exists
   */
  export async function exists(id: string): Promise<boolean> {
    const { workflows } = await state()
    return workflows.has(id)
  }

  /**
   * Create a new workflow instance and execute it
   */
  export async function run(input: {
    workflowId: string
    inputs: Record<string, any>
    parentSessionId?: string
    onPause?: WorkflowExecutor.ExecutionContext["onPause"]
    abort?: AbortSignal
  }): Promise<WorkflowInstance> {
    const definition = await get(input.workflowId)
    if (!definition) {
      throw new Error(`Workflow not found: ${input.workflowId}`)
    }

    const instance = await WorkflowExecutor.create({
      definition,
      inputs: input.inputs,
      parentSessionId: input.parentSessionId,
    })

    const ctx: WorkflowExecutor.ExecutionContext = {
      instance,
      abort: input.abort ?? new AbortController().signal,
      onPause: input.onPause,
    }

    return WorkflowExecutor.execute(ctx)
  }

  /**
   * Resume a paused workflow instance
   */
  export async function resume(input: {
    instanceId: string
    approved: boolean
    feedback?: string
    editedVariables?: Record<string, any>
    abort?: AbortSignal
  }): Promise<WorkflowInstance> {
    const instance = await WorkflowExecutor.get(input.instanceId)
    if (!instance) {
      throw new Error(`Workflow instance not found: ${input.instanceId}`)
    }

    const ctx: WorkflowExecutor.ExecutionContext = {
      instance,
      abort: input.abort ?? new AbortController().signal,
    }

    return WorkflowExecutor.resume(ctx, {
      approved: input.approved,
      feedback: input.feedback,
      editedVariables: input.editedVariables,
    })
  }

  /**
   * Cancel a running or paused workflow instance
   */
  export async function cancel(instanceId: string, reason?: string): Promise<WorkflowInstance> {
    return WorkflowExecutor.cancel(instanceId, reason)
  }

  /**
   * Get a workflow instance by ID
   */
  export async function getInstance(instanceId: string): Promise<WorkflowInstance | null> {
    return WorkflowExecutor.get(instanceId)
  }

  /**
   * List workflow instances
   */
  export async function* listInstances(workflowId?: string): AsyncGenerator<WorkflowInstance> {
    for await (const instance of WorkflowExecutor.list()) {
      if (!workflowId || instance.workflowId === workflowId) {
        yield instance
      }
    }
  }

  /**
   * Delete a workflow instance
   */
  export async function deleteInstance(instanceId: string): Promise<void> {
    return WorkflowExecutor.remove(instanceId)
  }

  /**
   * Reload workflow definitions (clear cache)
   */
  export function reload(): void {
    Instance.reset()
  }
}
