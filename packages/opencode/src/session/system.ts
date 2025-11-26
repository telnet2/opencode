import { Ripgrep } from "../file/ripgrep"
import { Global } from "../global"
import { Filesystem } from "../util/filesystem"
import { Config } from "../config/config"
import { Session } from "."
import { Agent } from "../agent/agent"
import { Installation } from "../installation"

import { Instance } from "../project/instance"
import path from "path"
import os from "os"
import { existsSync } from "fs"

import PROMPT_ANTHROPIC from "./prompt/anthropic.txt"
import PROMPT_ANTHROPIC_WITHOUT_TODO from "./prompt/qwen.txt"
import PROMPT_POLARIS from "./prompt/polaris.txt"
import PROMPT_BEAST from "./prompt/beast.txt"
import PROMPT_GEMINI from "./prompt/gemini.txt"
import PROMPT_ANTHROPIC_SPOOF from "./prompt/anthropic_spoof.txt"
import PROMPT_COMPACTION from "./prompt/compaction.txt"
import PROMPT_SUMMARIZE from "./prompt/summarize.txt"
import PROMPT_TITLE from "./prompt/title.txt"
import PROMPT_CODEX from "./prompt/codex.txt"

export namespace SystemPrompt {
  export function header(providerID: string) {
    if (providerID.includes("anthropic")) return [PROMPT_ANTHROPIC_SPOOF.trim()]
    return []
  }

  export function provider(modelID: string) {
    if (modelID.includes("gpt-5")) return [PROMPT_CODEX]
    if (modelID.includes("gpt-") || modelID.includes("o1") || modelID.includes("o3")) return [PROMPT_BEAST]
    if (modelID.includes("gemini-")) return [PROMPT_GEMINI]
    if (modelID.includes("claude")) return [PROMPT_ANTHROPIC]
    if (modelID.includes("polaris-alpha")) return [PROMPT_POLARIS]
    return [PROMPT_ANTHROPIC_WITHOUT_TODO]
  }

  export async function environment() {
    const project = Instance.project
    return [
      [
        `Here is some useful information about the environment you are running in:`,
        `<env>`,
        `  Working directory: ${Instance.directory}`,
        `  Is directory a git repo: ${project.vcs === "git" ? "yes" : "no"}`,
        `  Platform: ${process.platform}`,
        `  Today's date: ${new Date().toDateString()}`,
        `</env>`,
        `<files>`,
        `  ${
          project.vcs === "git"
            ? await Ripgrep.tree({
                cwd: Instance.directory,
                limit: 200,
              })
            : ""
        }`,
        `</files>`,
      ].join("\n"),
    ]
  }

  const LOCAL_RULE_FILES = [
    "AGENTS.md",
    "CLAUDE.md",
    "CONTEXT.md", // deprecated
  ]
  const GLOBAL_RULE_FILES = [
    path.join(Global.Path.config, "AGENTS.md"),
    path.join(os.homedir(), ".claude", "CLAUDE.md"),
  ]

  export async function custom() {
    const config = await Config.get()
    const paths = new Set<string>()

    for (const localRuleFile of LOCAL_RULE_FILES) {
      const matches = await Filesystem.findUp(localRuleFile, Instance.directory, Instance.worktree)
      if (matches.length > 0) {
        matches.forEach((path) => paths.add(path))
        break
      }
    }

    for (const globalRuleFile of GLOBAL_RULE_FILES) {
      if (await Bun.file(globalRuleFile).exists()) {
        paths.add(globalRuleFile)
        break
      }
    }

    if (config.instructions) {
      for (let instruction of config.instructions) {
        if (instruction.startsWith("~/")) {
          instruction = path.join(os.homedir(), instruction.slice(2))
        }
        let matches: string[] = []
        if (path.isAbsolute(instruction)) {
          matches = await Array.fromAsync(
            new Bun.Glob(path.basename(instruction)).scan({
              cwd: path.dirname(instruction),
              absolute: true,
              onlyFiles: true,
            }),
          ).catch(() => [])
        } else {
          matches = await Filesystem.globUp(instruction, Instance.directory, Instance.worktree).catch(() => [])
        }
        matches.forEach((path) => paths.add(path))
      }
    }

    const found = Array.from(paths).map((p) =>
      Bun.file(p)
        .text()
        .catch(() => "")
        .then((x) => "Instructions from: " + p + "\n" + x),
    )
    return Promise.all(found).then((result) => result.filter(Boolean))
  }

  export function compaction(providerID: string) {
    switch (providerID) {
      case "anthropic":
        return [PROMPT_ANTHROPIC_SPOOF.trim(), PROMPT_COMPACTION]
      default:
        return [PROMPT_COMPACTION]
    }
  }

  export function summarize(providerID: string) {
    switch (providerID) {
      case "anthropic":
        return [PROMPT_ANTHROPIC_SPOOF.trim(), PROMPT_SUMMARIZE]
      default:
        return [PROMPT_SUMMARIZE]
    }
  }

  export function title(providerID: string) {
    switch (providerID) {
      case "anthropic":
        return [PROMPT_ANTHROPIC_SPOOF.trim(), PROMPT_TITLE]
      default:
        return [PROMPT_TITLE]
    }
  }

  function resolveTemplatePath(value: string): string {
    // Priority order for file resolution:
    // 1. Absolute path
    if (path.isAbsolute(value)) return value

    // 2. Home directory
    if (value.startsWith("~/")) return path.join(os.homedir(), value.slice(2))

    // 3. Check project-level prompts
    const projectPrompt = path.join(Instance.directory, ".opencode", "prompts", value)
    if (existsSync(projectPrompt)) return projectPrompt

    // 4. Check global prompts
    const globalPrompt = path.join(Global.Path.config, "prompts", value)
    if (existsSync(globalPrompt)) return globalPrompt

    // Fallback: treat as relative to cwd
    return path.resolve(Instance.directory, value)
  }

  async function getGitBranch(): Promise<string> {
    try {
      const result = Bun.spawn(["git", "branch", "--show-current"], {
        cwd: Instance.directory,
        stdout: "pipe",
        stderr: "pipe",
      })
      const output = await new Response(result.stdout).text()
      return output.trim() || "unknown"
    } catch {
      return "unknown"
    }
  }

  async function detectPrimaryLanguage(): Promise<string> {
    try {
      // Count file extensions in project
      const files = await Ripgrep.tree({ cwd: Instance.directory, limit: 500 })
      const extensions: Record<string, number> = {}

      for (const line of files.split("\n")) {
        const ext = path.extname(line).toLowerCase()
        if (ext) extensions[ext] = (extensions[ext] || 0) + 1
      }

      // Map extensions to languages
      const langMap: Record<string, string> = {
        ".ts": "typescript",
        ".tsx": "typescript",
        ".js": "javascript",
        ".jsx": "javascript",
        ".py": "python",
        ".go": "go",
        ".rs": "rust",
        ".java": "java",
        ".cpp": "cpp",
        ".cc": "cpp",
        ".cxx": "cpp",
        ".c": "c",
        ".rb": "ruby",
        ".php": "php",
        ".cs": "csharp",
        ".swift": "swift",
        ".kt": "kotlin",
      }

      // Find most common language
      let maxCount = 0
      let primaryLang = "unknown"
      for (const [ext, count] of Object.entries(extensions)) {
        const lang = langMap[ext]
        if (lang && count > maxCount) {
          maxCount = count
          primaryLang = lang
        }
      }

      return primaryLang
    } catch {
      return "unknown"
    }
  }

  function extractEnvVariables(): Record<string, string> {
    const vars: Record<string, string> = {}
    for (const [key, value] of Object.entries(process.env)) {
      if (key.startsWith("OPENCODE_VAR_")) {
        const varName = key.replace("OPENCODE_VAR_", "")
        vars[varName] = value || ""
      }
    }
    return vars
  }

  function applyFilter(value: string, filter: string): string {
    switch (filter) {
      case "uppercase":
        return value.toUpperCase()
      case "lowercase":
        return value.toLowerCase()
      case "capitalize":
        return value.charAt(0).toUpperCase() + value.slice(1).toLowerCase()
      default:
        return value
    }
  }

  export async function interpolateVariables(
    template: string,
    context: {
      sessionID: string
      agent?: Agent.Info
      model: { providerID: string; modelID: string }
      customVars?: Record<string, string>
    },
  ): Promise<string> {
    const session = await Session.get(context.sessionID)
    const config = await Config.get()
    const project = Instance.project

    // Build variable map
    const variables: Record<string, string> = {
      // Built-in variables
      PROJECT_NAME: path.basename(Instance.worktree),
      PROJECT_PATH: Instance.worktree,
      WORKING_DIR: Instance.directory,
      GIT_BRANCH: await getGitBranch(),
      GIT_REPO: project.vcs === "git" ? "yes" : "no",
      PRIMARY_LANGUAGE: await detectPrimaryLanguage(),
      PLATFORM: process.platform,
      DATE: new Date().toISOString().split("T")[0],
      TIME: new Date().toTimeString().split(" ")[0],
      DATETIME: new Date().toISOString().replace("T", " ").split(".")[0],
      USER: process.env.USER || process.env.USERNAME || "unknown",
      HOSTNAME: os.hostname(),
      SESSION_ID: session.id,
      SESSION_TITLE: session.title,
      AGENT_NAME: context.agent?.name || "default",
      MODEL_ID: context.model.modelID,
      OPENCODE_VERSION: Installation.VERSION,
    }

    // Merge in order of priority (later overrides earlier)
    Object.assign(
      variables,
      extractEnvVariables(), // OPENCODE_VAR_*
      config.promptVariables || {}, // Config file
      session.customPrompt?.variables || {}, // Session-specific
      context.customVars || {}, // Inline custom vars
    )

    // Interpolate: ${VAR}, ${VAR:default}, ${VAR|filter}
    return template.replace(/\$\{([A-Z_][A-Z0-9_]*)(:[^}]+)?(\|[^}]+)?\}/g, (match, varName, defaultValue, filter) => {
      let value = variables[varName]

      // Use default if variable not found
      if (value === undefined && defaultValue) {
        value = defaultValue.slice(1) // Remove leading ':'
      }

      // Return original if still not found
      if (value === undefined) {
        return match
      }

      // Apply filter if specified
      if (filter) {
        value = applyFilter(value, filter.slice(1)) // Remove leading '|'
      }

      return value
    })
  }

  export async function fromSession(
    sessionID: string,
    context: {
      agent?: Agent.Info
      model: { providerID: string; modelID: string }
    },
  ): Promise<string | null> {
    const session = await Session.get(sessionID)
    if (!session.customPrompt) return null

    let content: string

    if (session.customPrompt.type === "inline") {
      content = session.customPrompt.value
    } else if (session.customPrompt.type === "file") {
      const filePath = resolveTemplatePath(session.customPrompt.value)

      // Check file size limit (100 KB)
      try {
        const file = Bun.file(filePath)
        const size = file.size
        if (size > 100 * 1024) {
          throw new Error(`Prompt template too large: ${size} bytes (max 100 KB)`)
        }
        content = await file.text()
      } catch (error) {
        throw new Error(`Failed to load prompt template: ${filePath} - ${error}`)
      }
    } else {
      return null
    }

    // Interpolate variables
    return await interpolateVariables(content, {
      sessionID,
      agent: context.agent,
      model: context.model,
      customVars: session.customPrompt.variables,
    })
  }
}
