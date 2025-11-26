import { parse as parseYaml } from "yaml"
import z from "zod"
import { WorkflowDefinition, WorkflowStep, WorkflowAgentConfig } from "./schema"
import { Log } from "../util/log"

const log = Log.create({ service: "workflow.parser" })

/**
 * Parse workflow definitions from various sources
 */
export namespace WorkflowParser {
  /**
   * Error thrown during workflow parsing
   */
  export class ParseError extends Error {
    constructor(
      message: string,
      public readonly source: string,
      public readonly details?: z.ZodError,
    ) {
      super(`Workflow parse error in ${source}: ${message}`)
      this.name = "WorkflowParseError"
    }
  }

  /**
   * Parse a workflow from a markdown file with YAML frontmatter
   *
   * Format:
   * ```markdown
   * ---
   * id: my-workflow
   * name: My Workflow
   * version: 1.0.0
   * inputs:
   *   files:
   *     type: string
   *     description: Files to process
   * orchestrator:
   *   mode: guided
   *   onError: pause
   * ---
   *
   * # Agent: analyzer
   *
   * You are a code analyzer...
   *
   * ```yaml
   * tools:
   *   read: true
   *   glob: true
   * model: anthropic/claude-sonnet-4-20250514
   * ```
   *
   * # Steps
   *
   * ```yaml
   * - id: analyze
   *   type: agent
   *   agent: analyzer
   *   input: "Analyze the following files: {{files}}"
   *   output: analysis
   *
   * - id: review
   *   type: pause
   *   message: "Review the analysis before proceeding"
   *   dependsOn: [analyze]
   * ```
   * ```
   */
  export function parseMarkdown(content: string, source: string): WorkflowDefinition {
    log.info("parsing markdown workflow", { source })

    // Extract frontmatter
    const frontmatterMatch = content.match(/^---\n([\s\S]*?)\n---/)
    if (!frontmatterMatch) {
      throw new ParseError("Missing YAML frontmatter", source)
    }

    const frontmatter = parseYaml(frontmatterMatch[1])
    const body = content.slice(frontmatterMatch[0].length).trim()

    // Parse agents from markdown sections
    const agents = parseAgentSections(body, source)

    // Parse steps from yaml code blocks
    const steps = parseStepsSections(body, source)

    // Combine everything
    const definition = {
      ...frontmatter,
      agents: Object.keys(agents).length > 0 ? agents : undefined,
      steps,
    }

    return validate(definition, source)
  }

  /**
   * Parse a workflow from a YAML string
   */
  export function parseYAML(content: string, source: string): WorkflowDefinition {
    log.info("parsing YAML workflow", { source })

    try {
      const parsed = parseYaml(content)
      return validate(parsed, source)
    } catch (error) {
      if (error instanceof z.ZodError) {
        throw new ParseError("Invalid workflow structure", source, error)
      }
      throw new ParseError(String(error), source)
    }
  }

  /**
   * Parse a workflow from a JSON string
   */
  export function parseJSON(content: string, source: string): WorkflowDefinition {
    log.info("parsing JSON workflow", { source })

    try {
      const parsed = JSON.parse(content)
      return validate(parsed, source)
    } catch (error) {
      if (error instanceof z.ZodError) {
        throw new ParseError("Invalid workflow structure", source, error)
      }
      throw new ParseError(String(error), source)
    }
  }

  /**
   * Parse workflow from any supported format based on file extension or content
   */
  export function parse(content: string, source: string): WorkflowDefinition {
    const trimmed = content.trim()

    // Detect format
    if (trimmed.startsWith("---")) {
      return parseMarkdown(content, source)
    } else if (trimmed.startsWith("{")) {
      return parseJSON(content, source)
    } else {
      return parseYAML(content, source)
    }
  }

  /**
   * Validate a parsed workflow definition
   */
  export function validate(data: unknown, source: string): WorkflowDefinition {
    try {
      const result = WorkflowDefinition.parse(data)

      // Additional validation
      validateStepReferences(result)
      validateAgentReferences(result)
      validateDependencies(result)

      return result
    } catch (error) {
      if (error instanceof z.ZodError) {
        throw new ParseError("Invalid workflow definition", source, error)
      }
      throw error
    }
  }

  /**
   * Parse agent sections from markdown body
   * Looks for sections like:
   * # Agent: name
   * prompt content...
   * ```yaml
   * config
   * ```
   */
  function parseAgentSections(body: string, source: string): Record<string, WorkflowAgentConfig> {
    const agents: Record<string, WorkflowAgentConfig> = {}

    // Match agent sections: # Agent: name
    const agentPattern = /^#+\s*Agent:\s*(\w+)\s*$([\s\S]*?)(?=^#+\s*(?:Agent:|Steps)|$)/gim
    let match

    while ((match = agentPattern.exec(body)) !== null) {
      const name = match[1]
      const content = match[2].trim()

      // Extract YAML config block
      const configMatch = content.match(/```ya?ml\n([\s\S]*?)\n```/)
      const config = configMatch ? parseYaml(configMatch[1]) : {}

      // Extract prompt (everything before the yaml block or everything if no block)
      let prompt = content
      if (configMatch) {
        prompt = content.slice(0, content.indexOf("```")).trim()
      }

      agents[name] = {
        name,
        prompt,
        ...config,
      }
    }

    return agents
  }

  /**
   * Parse steps from YAML code blocks in markdown
   * Looks for sections like:
   * # Steps
   * ```yaml
   * - id: step1
   *   type: agent
   *   ...
   * ```
   */
  function parseStepsSections(body: string, source: string): WorkflowStep[] {
    // Find the Steps section
    const stepsPattern = /^#+\s*Steps\s*$([\s\S]*?)(?=^#+|$)/im
    const stepsMatch = body.match(stepsPattern)

    if (!stepsMatch) {
      // Try to find steps in a yaml block anywhere
      const yamlBlockPattern = /```ya?ml\n([\s\S]*?)\n```/g
      let match
      while ((match = yamlBlockPattern.exec(body)) !== null) {
        try {
          const parsed = parseYaml(match[1])
          if (Array.isArray(parsed) && parsed.length > 0 && parsed[0].id && parsed[0].type) {
            return parsed.map((step) => WorkflowStep.parse(step))
          }
        } catch {
          // Not a steps block, continue
        }
      }
      throw new ParseError("No steps section found", source)
    }

    // Find yaml block in steps section
    const yamlMatch = stepsMatch[1].match(/```ya?ml\n([\s\S]*?)\n```/)
    if (!yamlMatch) {
      throw new ParseError("No YAML block found in steps section", source)
    }

    const steps = parseYaml(yamlMatch[1])
    if (!Array.isArray(steps)) {
      throw new ParseError("Steps must be an array", source)
    }

    return steps.map((step, index) => {
      try {
        return WorkflowStep.parse(step)
      } catch (error) {
        if (error instanceof z.ZodError) {
          throw new ParseError(`Invalid step at index ${index}`, source, error)
        }
        throw error
      }
    })
  }

  /**
   * Validate that all step references are valid
   */
  function validateStepReferences(workflow: WorkflowDefinition): void {
    const stepIds = new Set(workflow.steps.map((s) => s.id))

    // Check startStep
    if (workflow.startStep && !stepIds.has(workflow.startStep)) {
      throw new Error(`Invalid startStep: "${workflow.startStep}" does not exist`)
    }

    // Check references in steps
    for (const step of workflow.steps) {
      // Check dependsOn
      if (step.dependsOn) {
        for (const dep of step.dependsOn) {
          if (!stepIds.has(dep)) {
            throw new Error(`Step "${step.id}" depends on non-existent step "${dep}"`)
          }
        }
      }

      // Check parallel step references
      if (step.type === "parallel") {
        for (const ref of step.steps) {
          if (!stepIds.has(ref)) {
            throw new Error(`Parallel step "${step.id}" references non-existent step "${ref}"`)
          }
        }
      }

      // Check conditional step references
      if (step.type === "conditional") {
        if (!stepIds.has(step.then)) {
          throw new Error(`Conditional step "${step.id}" references non-existent 'then' step "${step.then}"`)
        }
        if (step.else && !stepIds.has(step.else)) {
          throw new Error(`Conditional step "${step.id}" references non-existent 'else' step "${step.else}"`)
        }
      }

      // Check loop step references
      if (step.type === "loop") {
        for (const ref of step.steps) {
          if (!stepIds.has(ref)) {
            throw new Error(`Loop step "${step.id}" references non-existent step "${ref}"`)
          }
        }
      }
    }
  }

  /**
   * Validate that all agent references are valid
   */
  function validateAgentReferences(workflow: WorkflowDefinition): void {
    const agentNames = new Set(Object.keys(workflow.agents ?? {}))

    for (const step of workflow.steps) {
      if (step.type === "agent") {
        // Agent references can be to workflow-local agents or global agents
        // We only validate local references here; global agents are resolved at runtime
        if (!agentNames.has(step.agent)) {
          // Mark as potentially external agent - will be validated at runtime
          log.info("agent reference may be external", {
            step: step.id,
            agent: step.agent,
          })
        }
      }
    }
  }

  /**
   * Validate that dependencies don't form cycles
   */
  function validateDependencies(workflow: WorkflowDefinition): void {
    const stepMap = new Map(workflow.steps.map((s) => [s.id, s]))
    const visited = new Set<string>()
    const visiting = new Set<string>()

    function visit(stepId: string, path: string[]): void {
      if (visiting.has(stepId)) {
        throw new Error(`Circular dependency detected: ${[...path, stepId].join(" -> ")}`)
      }
      if (visited.has(stepId)) {
        return
      }

      visiting.add(stepId)
      const step = stepMap.get(stepId)

      if (step?.dependsOn) {
        for (const dep of step.dependsOn) {
          visit(dep, [...path, stepId])
        }
      }

      visiting.delete(stepId)
      visited.add(stepId)
    }

    for (const step of workflow.steps) {
      visit(step.id, [])
    }
  }

  /**
   * Serialize a workflow definition to YAML
   */
  export function toYAML(workflow: WorkflowDefinition): string {
    const { parse: _parse, ...yaml } = require("yaml")
    return yaml.stringify(workflow)
  }

  /**
   * Serialize a workflow definition to JSON
   */
  export function toJSON(workflow: WorkflowDefinition, pretty = true): string {
    return JSON.stringify(workflow, null, pretty ? 2 : undefined)
  }

  /**
   * Serialize a workflow definition to markdown format
   */
  export function toMarkdown(workflow: WorkflowDefinition): string {
    const { stringify: yamlStringify } = require("yaml")

    const lines: string[] = []

    // Frontmatter
    const frontmatter = {
      id: workflow.id,
      name: workflow.name,
      description: workflow.description,
      version: workflow.version,
      inputs: workflow.inputs,
      orchestrator: workflow.orchestrator,
      tags: workflow.tags,
      metadata: workflow.metadata,
    }

    // Remove undefined values
    Object.keys(frontmatter).forEach((key) => {
      if ((frontmatter as any)[key] === undefined) {
        delete (frontmatter as any)[key]
      }
    })

    lines.push("---")
    lines.push(yamlStringify(frontmatter).trim())
    lines.push("---")
    lines.push("")

    // Agents
    if (workflow.agents && Object.keys(workflow.agents).length > 0) {
      for (const [name, agent] of Object.entries(workflow.agents)) {
        lines.push(`# Agent: ${name}`)
        lines.push("")
        lines.push(agent.prompt)
        lines.push("")

        const config: any = { ...agent }
        delete config.name
        delete config.prompt

        if (Object.keys(config).length > 0) {
          lines.push("```yaml")
          lines.push(yamlStringify(config).trim())
          lines.push("```")
          lines.push("")
        }
      }
    }

    // Steps
    lines.push("# Steps")
    lines.push("")
    lines.push("```yaml")
    lines.push(yamlStringify(workflow.steps).trim())
    lines.push("```")

    return lines.join("\n")
  }
}
