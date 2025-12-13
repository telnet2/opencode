import readline from "readline/promises"
import { stdin as input, stdout as output } from "process"
import { Renderer } from "./renderer"
import { applyCommand, helpText, parseCommand } from "./commands"
import type { ResolvedConfig } from "./types"
import type { SimpleClient } from "./client"

function buildPrompt(): string {
  const cwd = process.cwd()
  const dirName = cwd.split(/\\|\//).pop() || cwd
  return `${dirName}> `
}

async function readMultiline(rl: readline.Interface): Promise<string | null> {
  let buffer = ""
  while (true) {
    const prompt = buffer ? "... " : buildPrompt()
    const line = await rl.question(prompt).catch(() => null)
    if (line === null) return null
    if (line.endsWith("\\")) {
      buffer += line.slice(0, -1)
      buffer += "\n"
      continue
    }
    buffer += line
    return buffer
  }
}

export async function runRepl(config: ResolvedConfig, client: SimpleClient, renderer: Renderer) {
  const rl = readline.createInterface({ input, output })
  renderer.banner(config.url)

  while (true) {
    const line = await readMultiline(rl)
    if (line === null) break
    const trimmed = line.trim()
    if (!trimmed) continue

    if (trimmed.startsWith("/")) {
      const cmd = parseCommand(trimmed)
      switch (cmd.type) {
        case "exit":
          rl.close()
          client.close()
          return
        case "help":
          renderer.help(helpText())
          continue
        case "set":
          const next = applyCommand(cmd, config)
          Object.assign(config, next)
          renderer.trace("updated", { [cmd.key]: cmd.value })
          continue
        case "unknown":
          renderer.help(`Unknown command: ${cmd.input}\n${helpText()}`)
          continue
      }
    }

    renderer.user(trimmed)
    try {
      const response = await client.sendPrompt(trimmed)
      renderer.renderMessage(response.info, response.parts)
    } catch (err) {
      renderer.trace("prompt failed", { error: String(err) })
      console.error("Failed to send prompt:", err)
    }
  }
}
