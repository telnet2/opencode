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
        choices: ["list", "show"],
        demandOption: true,
      })
      .option("name", {
        describe: "template name (for 'show' action)",
        type: "string",
      })
      .option("verbose", {
        alias: "v",
        describe: "show full template content",
        type: "boolean",
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
