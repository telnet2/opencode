package main

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/fatih/color"
    opencode "github.com/sst/opencode-sdk-go"
)

// Renderer mirrors the TS renderer to show conversation progress.
type Renderer struct {
    opts rendererOptions
    c    *color.Color
}

type rendererOptions struct {
    NoColor bool
    Quiet   bool
    JSON    bool
    Verbose bool
}

func NewRenderer(cfg ResolvedConfig) *Renderer {
    color.NoColor = cfg.NoColor
    return &Renderer{
        opts: rendererOptions{
            NoColor: cfg.NoColor,
            Quiet:   cfg.Quiet,
            JSON:    cfg.JSON,
            Verbose: cfg.Verbose,
        },
        c: color.New(color.FgCyan),
    }
}

func (r *Renderer) Banner(url string) {
    if r.opts.Quiet {
        return
    }
    fmt.Fprintln(os.Stderr, color.New(color.FgHiBlack).Sprintf("Connected to %s", url))
}

func (r *Renderer) Help(text string) {
    if r.opts.Quiet {
        return
    }
    fmt.Println(text)
}

func (r *Renderer) User(input string) {
    if r.opts.JSON {
        b, _ := json.Marshal(map[string]string{"type": "user", "text": input})
        fmt.Println(string(b))
        return
    }
    fmt.Printf("%s %s\n", color.New(color.FgCyan, color.Bold).Sprint("you ›"), input)
}

func (r *Renderer) Assistant(message string) {
    if r.opts.JSON {
        b, _ := json.Marshal(map[string]string{"type": "assistant", "text": message})
        fmt.Println(string(b))
        return
    }
    fmt.Printf("%s %s\n", color.New(color.FgGreen, color.Bold).Sprint("assistant ›"), message)
}

func (r *Renderer) Trace(msg string, details map[string]any) {
    if !r.opts.Verbose {
        return
    }
    if details != nil {
        fmt.Fprintln(os.Stderr, color.New(color.FgHiBlack).Sprintf("[trace] %s %v", msg, details))
    } else {
        fmt.Fprintln(os.Stderr, color.New(color.FgHiBlack).Sprintf("[trace] %s", msg))
    }
}

func (r *Renderer) Tool(part opencode.ToolPart) {
    summary := describeToolState(part.State)
    if r.opts.JSON {
        b, _ := json.Marshal(map[string]any{
            "type":   "tool",
            "tool":   part.Tool,
            "callID": part.CallID,
            "state":  part.State,
        })
        fmt.Println(string(b))
        return
    }
    fmt.Printf("%s\n", color.New(color.FgYellow).Sprintf("→ tool %s (%s)", part.Tool, summary))
    state := part.State.AsUnion()
    switch s := state.(type) {
    case opencode.ToolStateCompleted:
        if s.Output != "" {
            fmt.Println(color.New(color.FgHiBlack).Sprint(s.Output))
        }
    case opencode.ToolStateError:
        if s.Error != "" {
            fmt.Fprintln(os.Stderr, color.New(color.FgRed).Sprintf("  error: %s", s.Error))
        }
    }
}

func (r *Renderer) RenderMessage(messageID string, isAssistant bool, parts []opencode.Part) {
    var textParts []string
    for _, p := range parts {
        switch v := p.AsUnion().(type) {
        case opencode.TextPart:
            textParts = append(textParts, v.Text)
        case opencode.ToolPart:
            r.Tool(v)
        }
    }
    if len(textParts) > 0 {
        r.Assistant(joinLines(textParts))
    } else if isAssistant {
        r.Assistant(fmt.Sprintf("message %s updated", messageID))
    }
}

func (r *Renderer) RenderPart(part opencode.Part) {
    switch v := part.AsUnion().(type) {
    case opencode.TextPart:
        r.Assistant(v.Text)
    case opencode.ToolPart:
        r.Tool(v)
    }
}

func describeToolState(state opencode.ToolPartState) string {
    switch state.Status {
    case opencode.ToolPartStateStatusPending:
        return "pending"
    case opencode.ToolPartStateStatusRunning:
        return "running"
    case opencode.ToolPartStateStatusCompleted:
        return "done"
    case opencode.ToolPartStateStatusError:
        return "error"
    default:
        return "unknown"
    }
}

func joinLines(parts []string) string {
    if len(parts) == 0 {
        return ""
    }
    result := parts[0]
    for _, p := range parts[1:] {
        result += "\n" + p
    }
    return result
}
