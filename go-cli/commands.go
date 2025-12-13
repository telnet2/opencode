package main

import "strings"

const helpText = `Simple CLI commands:
  /help                 Show this message
  /exit                 Quit the CLI
  /model <name>         Select a model
  /provider <name>      Select a provider
  /agent <name>         Select an agent`

type commandResult struct {
    Type string
    Key  string
    Val  string
}

func parseCommand(input string) commandResult {
    parts := strings.Fields(strings.TrimPrefix(strings.TrimSpace(input), "/"))
    if len(parts) == 0 {
        return commandResult{Type: "unknown"}
    }
    switch parts[0] {
    case "exit", "quit":
        return commandResult{Type: "exit"}
    case "help":
        return commandResult{Type: "help"}
    case "model":
        return commandResult{Type: "set", Key: "model", Val: strings.Join(parts[1:], " ")}
    case "provider":
        return commandResult{Type: "set", Key: "provider", Val: strings.Join(parts[1:], " ")}
    case "agent":
        return commandResult{Type: "set", Key: "agent", Val: strings.Join(parts[1:], " ")}
    default:
        return commandResult{Type: "unknown", Val: input}
    }
}

func applyCommand(cfg *ResolvedConfig, cmd commandResult) {
    if cmd.Type != "set" {
        return
    }
    switch cmd.Key {
    case "model":
        cfg.Model = cmd.Val
    case "provider":
        cfg.Provider = cmd.Val
    case "agent":
        cfg.Agent = cmd.Val
    }
}
