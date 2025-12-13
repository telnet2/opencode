package main

import (
    "bufio"
    "context"
    "fmt"
    "os"
    "strings"
)

func buildPrompt() string {
    cwd, _ := os.Getwd()
    parts := strings.Split(cwd, string(os.PathSeparator))
    last := parts[len(parts)-1]
    if last == "" {
        last = cwd
    }
    return fmt.Sprintf("%s> ", last)
}

func readMultiline(reader *bufio.Reader) (string, error) {
    var lines []string
    for {
        prompt := buildPrompt()
        if len(lines) > 0 {
            prompt = "... "
        }
        fmt.Print(prompt)
        line, err := reader.ReadString('\n')
        if err != nil {
            if len(lines) == 0 {
                return "", err
            }
            return strings.Join(lines, "\n"), nil
        }
        line = strings.TrimRight(line, "\r\n")
        if strings.HasSuffix(line, "\\") {
            lines = append(lines, strings.TrimSuffix(line, "\\"))
            continue
        }
        lines = append(lines, line)
        return strings.Join(lines, "\n"), nil
    }
}

func runRepl(cfg *ResolvedConfig, client *SimpleClient, renderer *Renderer) error {
    renderer.Banner(cfg.URL)
    reader := bufio.NewReader(os.Stdin)

    for {
        line, err := readMultiline(reader)
        if err != nil {
            return err
        }
        trimmed := strings.TrimSpace(line)
        if trimmed == "" {
            continue
        }

        if strings.HasPrefix(trimmed, "/") {
            cmd := parseCommand(trimmed)
            switch cmd.Type {
            case "exit":
                client.Close()
                return nil
            case "help":
                renderer.Help(helpText)
                continue
            case "set":
                applyCommand(cfg, cmd)
                renderer.Trace("updated", map[string]any{cmd.Key: cmd.Val})
                continue
            default:
                renderer.Help(fmt.Sprintf("Unknown command: %s\n%s", cmd.Val, helpText))
                continue
            }
        }

        renderer.User(trimmed)
        resp, err := client.SendPrompt(context.Background(), trimmed, *cfg)
        if err != nil {
            renderer.Trace("prompt failed", map[string]any{"error": err.Error()})
            fmt.Fprintf(os.Stderr, "Failed to send prompt: %v\n", err)
            continue
        }
        renderer.RenderMessage(resp.Info.ID, true, resp.Parts)
    }
}
