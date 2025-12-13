import type { ResolvedConfig } from "./types"

export type CommandResult =
  | { type: "exit" }
  | { type: "help" }
  | { type: "set"; key: "model" | "agent" | "provider"; value?: string }
  | { type: "unknown"; input: string }

export function parseCommand(input: string): CommandResult {
  const [command, ...rest] = input.slice(1).trim().split(/\s+/)
  const value = rest.join(" ") || undefined
  switch (command) {
    case "exit":
    case "quit":
      return { type: "exit" }
    case "help":
      return { type: "help" }
    case "model":
      return { type: "set", key: "model", value }
    case "agent":
      return { type: "set", key: "agent", value }
    case "provider":
      return { type: "set", key: "provider", value }
    default:
      return { type: "unknown", input: input.trim() }
  }
}

export function helpText(): string {
  return `Simple CLI commands:\n` +
    `  /help                 Show this message\n` +
    `  /exit                 Quit the CLI\n` +
    `  /model <name>         Select a model\n` +
    `  /provider <name>      Select a provider\n` +
    `  /agent <name>         Select an agent\n`
}

export function applyCommand(result: CommandResult, config: ResolvedConfig): ResolvedConfig {
  if (result.type !== "set") return config
  return { ...config, [result.key]: result.value }
}
