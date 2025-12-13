import type { Argv } from "yargs"
import { Instance } from "../../project/instance"
import { cmd } from "./cmd"
import { UI } from "../ui"
import { EOL } from "os"
import path from "path"
import os from "os"
import { Global } from "../../global"

export const PromptsCommand = cmd({
  command: "prompts <action>",
  describe: "manage custom prompt templates",
  builder: (yargs: Argv) => {
    return yargs
      .positional("action", {
        describe: "action to perform",
        type: "string",
        choices: ["list", "show", "create", "edit", "delete"],
        demandOption: true,
      })
      .option("name", {
        describe: "template name (for 'show', 'create', 'edit', 'delete' actions)",
        type: "string",
      })
      .option("global", {
        alias: "g",
        describe: "use global templates directory (~/.opencode/prompts/)",
        type: "boolean",
      })
      .option("verbose", {
        alias: "v",
        describe: "show full template content (for 'show' action)",
        type: "boolean",
      })
      .option("base", {
        describe: "base template to copy from (for 'create' action)",
        type: "string",
        choices: ["anthropic", "beast", "gemini", "codex", "qwen", "polaris"],
      })
  },
  handler: async (args) => {
    await Instance.provide({
      directory: process.cwd(),
      async fn() {
        if (args.action === "list") {
          await listTemplates()
        } else if (args.action === "show") {
          if (!args.name) {
            UI.error("Template name is required for 'show' action. Use --name <template>")
            process.exit(1)
          }
          await showTemplate(args.name, args.verbose)
        } else if (args.action === "create") {
          if (!args.name) {
            UI.error("Template name is required for 'create' action. Use --name <template>")
            process.exit(1)
          }
          await createTemplate(args.name, args.global, args.base)
        } else if (args.action === "edit") {
          if (!args.name) {
            UI.error("Template name is required for 'edit' action. Use --name <template>")
            process.exit(1)
          }
          await editTemplate(args.name)
        } else if (args.action === "delete") {
          if (!args.name) {
            UI.error("Template name is required for 'delete' action. Use --name <template>")
            process.exit(1)
          }
          await deleteTemplate(args.name, args.global)
        }
      },
    })
  },
})

async function listTemplates() {
  const templates: { location: string; name: string; path: string; size: number }[] = []

  // Check project-level prompts
  const projectDir = path.join(Instance.directory, ".opencode", "prompts")
  if (await Bun.file(projectDir).exists()) {
    const projectFiles = await Array.fromAsync(
      new Bun.Glob("*.{txt,md}").scan({
        cwd: projectDir,
        onlyFiles: true,
      }),
    )
    for (const file of projectFiles) {
      const filePath = path.join(projectDir, file)
      const stat = await Bun.file(filePath).stat()
      templates.push({
        location: "project",
        name: file,
        path: filePath,
        size: stat.size,
      })
    }
  }

  // Check global prompts
  const globalDir = path.join(Global.Path.config, "prompts")
  if (await Bun.file(globalDir).exists()) {
    const globalFiles = await Array.fromAsync(
      new Bun.Glob("*.{txt,md}").scan({
        cwd: globalDir,
        onlyFiles: true,
      }),
    )
    for (const file of globalFiles) {
      const filePath = path.join(globalDir, file)
      const stat = await Bun.file(filePath).stat()
      templates.push({
        location: "global",
        name: file,
        path: filePath,
        size: stat.size,
      })
    }
  }

  if (templates.length === 0) {
    UI.println("No prompt templates found.")
    UI.println()
    UI.println("Create templates in:")
    UI.println(`  Project: ${projectDir}`)
    UI.println(`  Global:  ${globalDir}`)
    return
  }

  // Group by location
  const byLocation = templates.reduce(
    (acc, t) => {
      if (!acc[t.location]) acc[t.location] = []
      acc[t.location].push(t)
      return acc
    },
    {} as Record<string, typeof templates>,
  )

  for (const [location, items] of Object.entries(byLocation)) {
    const title = location === "project" ? "Project templates" : "Global templates"
    const dir = location === "project" ? projectDir : globalDir
    UI.println(UI.Style.TEXT_INFO_BOLD + title + UI.Style.TEXT_NORMAL + ` (${dir})`)

    for (const template of items) {
      const sizeKB = (template.size / 1024).toFixed(1)
      UI.println(`  ${template.name}  ${UI.Style.TEXT_DIM}(${sizeKB} KB)`)
    }

    UI.println()
  }

  UI.println("Usage:")
  UI.println(`  opencode run --prompt <template-name> "your message"`)
  UI.println(`  opencode prompts show --name <template-name>`)
}

async function showTemplate(name: string, verbose?: boolean) {
  // Try to find the template
  const projectPath = path.join(Instance.directory, ".opencode", "prompts", name)
  const globalPath = path.join(Global.Path.config, "prompts", name)

  let filePath: string | null = null
  let location: string | null = null

  if (await Bun.file(projectPath).exists()) {
    filePath = projectPath
    location = "project"
  } else if (await Bun.file(globalPath).exists()) {
    filePath = globalPath
    location = "global"
  }

  if (!filePath) {
    UI.error(`Template not found: ${name}`)
    UI.println()
    UI.println("Available templates:")
    await listTemplates()
    process.exit(1)
  }

  const content = await Bun.file(filePath).text()
  const stat = await Bun.file(filePath).stat()
  const sizeKB = (stat.size / 1024).toFixed(1)

  UI.println(UI.Style.TEXT_INFO_BOLD + `Template: ${name}`)
  UI.println(UI.Style.TEXT_DIM + `Location: ${location} (${filePath})`)
  UI.println(UI.Style.TEXT_DIM + `Size: ${sizeKB} KB`)
  UI.println()

  if (verbose) {
    UI.println(UI.Style.TEXT_INFO_BOLD + "Content:")
    UI.println(UI.Style.TEXT_DIM + "─".repeat(80))
    process.stdout.write(content)
    if (!content.endsWith("\n")) process.stdout.write(EOL)
    UI.println(UI.Style.TEXT_DIM + "─".repeat(80))
  } else {
    const lines = content.split("\n")
    const preview = lines.slice(0, 10).join("\n")
    UI.println(UI.Style.TEXT_INFO_BOLD + "Preview (first 10 lines):")
    UI.println(UI.Style.TEXT_DIM + "─".repeat(80))
    process.stdout.write(preview)
    if (!preview.endsWith("\n")) process.stdout.write(EOL)
    UI.println(UI.Style.TEXT_DIM + "─".repeat(80))

    if (lines.length > 10) {
      UI.println()
      UI.println(UI.Style.TEXT_DIM + `... ${lines.length - 10} more lines`)
      UI.println(UI.Style.TEXT_DIM + "Use --verbose to see full content")
    }
  }
}

async function createTemplate(name: string, isGlobal?: boolean, base?: string) {
  // Determine target directory
  const targetDir = isGlobal ? path.join(Global.Path.config, "prompts") : path.join(Instance.directory, ".opencode", "prompts")

  // Ensure directory exists
  await Bun.$`mkdir -p ${targetDir}`.quiet()

  // Add extension if not present
  const filename = name.endsWith(".txt") || name.endsWith(".md") ? name : `${name}.txt`
  const targetPath = path.join(targetDir, filename)

  // Check if file already exists
  if (await Bun.file(targetPath).exists()) {
    UI.error(`Template already exists: ${targetPath}`)
    UI.println()
    UI.println("Use 'opencode prompts edit' to modify existing templates")
    process.exit(1)
  }

  let content = ""

  // Load base template if specified
  if (base) {
    const baseTemplatePath = path.join(
      path.dirname(path.dirname(path.dirname(__dirname))),
      "src",
      "session",
      "prompt",
      `${base}.txt`,
    )

    if (await Bun.file(baseTemplatePath).exists()) {
      content = await Bun.file(baseTemplatePath).text()
      UI.println(UI.Style.TEXT_INFO_BOLD + `Copying base template: ${base}`)
    } else {
      UI.println(UI.Style.TEXT_WARNING_BOLD + `Base template not found: ${base}, starting with blank template`)
    }
  }

  // Create placeholder content if no base template
  if (!content) {
    content = `# Custom Prompt Template: ${name}

## Instructions
Write your custom prompt template here. You can use variables like:

- \${PROJECT_NAME} - Name of the current project
- \${GIT_BRANCH} - Current git branch
- \${PRIMARY_LANGUAGE} - Detected primary programming language
- \${DATE} - Current date (YYYY-MM-DD)
- \${AGENT_NAME} - Name of the agent being used

See documentation for full list of available variables and syntax.

## Your Prompt
[Write your custom instructions here]
`
  }

  // Write template file
  await Bun.write(targetPath, content)

  const location = isGlobal ? "global" : "project"
  UI.println(UI.Style.TEXT_SUCCESS_BOLD + `✓ Created ${location} template: ${filename}`)
  UI.println(UI.Style.TEXT_DIM + `  Path: ${targetPath}`)
  UI.println()

  // Open in editor if available
  const editor = process.env.EDITOR || process.env.VISUAL
  if (editor) {
    UI.println(UI.Style.TEXT_INFO_BOLD + `Opening in editor: ${editor}`)
    try {
      await Bun.$`${editor} ${targetPath}`.quiet()
    } catch (error) {
      UI.println(UI.Style.TEXT_WARNING_BOLD + `Failed to open editor: ${error}`)
    }
  } else {
    UI.println(UI.Style.TEXT_DIM + "Set $EDITOR environment variable to auto-open templates in your preferred editor")
  }

  UI.println()
  UI.println("Usage:")
  UI.println(`  opencode run --prompt ${filename} "your message"`)
}

async function editTemplate(name: string) {
  // Try to find the template
  const projectPath = path.join(Instance.directory, ".opencode", "prompts", name)
  const globalPath = path.join(Global.Path.config, "prompts", name)

  let filePath: string | null = null
  let location: string | null = null

  if (await Bun.file(projectPath).exists()) {
    filePath = projectPath
    location = "project"
  } else if (await Bun.file(globalPath).exists()) {
    filePath = globalPath
    location = "global"
  }

  if (!filePath) {
    UI.error(`Template not found: ${name}`)
    UI.println()
    UI.println("Available templates:")
    await listTemplates()
    process.exit(1)
  }

  const editor = process.env.EDITOR || process.env.VISUAL
  if (!editor) {
    UI.error("No editor configured")
    UI.println()
    UI.println("Set the $EDITOR or $VISUAL environment variable to use this command")
    UI.println("Example: export EDITOR=vim")
    UI.println()
    UI.println(`Alternatively, edit the file directly: ${filePath}`)
    process.exit(1)
  }

  UI.println(UI.Style.TEXT_INFO_BOLD + `Editing ${location} template: ${name}`)
  UI.println(UI.Style.TEXT_DIM + `Path: ${filePath}`)
  UI.println()

  try {
    await Bun.$`${editor} ${filePath}`.quiet()
    UI.println(UI.Style.TEXT_SUCCESS_BOLD + "✓ Template saved")
  } catch (error) {
    UI.error(`Failed to open editor: ${error}`)
    process.exit(1)
  }
}

async function deleteTemplate(name: string, isGlobal?: boolean) {
  let filePath: string | null = null
  let location: string | null = null

  if (isGlobal) {
    // Only check global directory
    const globalPath = path.join(Global.Path.config, "prompts", name)
    if (await Bun.file(globalPath).exists()) {
      filePath = globalPath
      location = "global"
    }
  } else {
    // Check project first, then global
    const projectPath = path.join(Instance.directory, ".opencode", "prompts", name)
    const globalPath = path.join(Global.Path.config, "prompts", name)

    if (await Bun.file(projectPath).exists()) {
      filePath = projectPath
      location = "project"
    } else if (await Bun.file(globalPath).exists()) {
      filePath = globalPath
      location = "global"
    }
  }

  if (!filePath) {
    UI.error(`Template not found: ${name}`)
    UI.println()
    UI.println("Available templates:")
    await listTemplates()
    process.exit(1)
  }

  UI.println(UI.Style.TEXT_WARNING_BOLD + `Delete ${location} template: ${name}`)
  UI.println(UI.Style.TEXT_DIM + `Path: ${filePath}`)
  UI.println()

  // Prompt for confirmation
  const readline = require("readline")
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  })

  const confirmed = await new Promise<boolean>((resolve) => {
    rl.question("Are you sure? (y/N): ", (answer: string) => {
      rl.close()
      resolve(answer.toLowerCase() === "y" || answer.toLowerCase() === "yes")
    })
  })

  if (!confirmed) {
    UI.println("Cancelled")
    process.exit(0)
  }

  try {
    await Bun.$`rm ${filePath}`.quiet()
    UI.println(UI.Style.TEXT_SUCCESS_BOLD + "✓ Template deleted")
  } catch (error) {
    UI.error(`Failed to delete template: ${error}`)
    process.exit(1)
  }
}
