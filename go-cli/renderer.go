package main

import (
    "encoding/json"
    "fmt"
    "os"
    "strings"

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

// RenderStreamingPart renders a part during streaming, using delta if available.
func (r *Renderer) RenderStreamingPart(part opencode.Part, delta string, toolStates map[string]string) {
    switch part.Type {
    case opencode.PartTypeText:
        // Use the delta if provided for efficient streaming, otherwise use full text
        text := delta
        if text == "" {
            text = part.Text
        }
        if text != "" {
            if r.opts.JSON {
                b, _ := json.Marshal(map[string]string{"type": "delta", "text": text})
                fmt.Println(string(b))
            } else {
                fmt.Print(text)
            }
        }
    case opencode.PartTypeTool:
        // Only render tool if state changed
        state, _ := part.State.(opencode.ToolPartState)
        currentState := string(state.Status)
        lastState := toolStates[part.CallID]
        if currentState != lastState {
            toolStates[part.CallID] = currentState
            r.ToolFromPart(part)
        }
    }
}

// ToolFromPart renders tool information from a Part struct.
func (r *Renderer) ToolFromPart(part opencode.Part) {
    state, ok := part.State.(opencode.ToolPartState)
    summary := "unknown"
    if ok {
        summary = describeToolState(state)
    }

    // Get tool name - prefer Tool field, fallback to Name
    toolName := part.Tool
    if toolName == "" {
        toolName = part.Name
    }

    // Format input args
    var argsStr string
    if ok && state.Input != nil {
        switch input := state.Input.(type) {
        case map[string]interface{}:
            argsStr = formatToolArgs(input)
        case string:
            argsStr = truncateArg(input, 50)
        }
    }

    if r.opts.JSON {
        b, _ := json.Marshal(map[string]any{
            "type":   "tool",
            "tool":   toolName,
            "callID": part.CallID,
            "state":  summary,
            "input":  state.Input,
        })
        fmt.Println(string(b))
        return
    }

    // Format: → tool_name(arg1=val1, arg2=val2) [status]
    if argsStr != "" {
        fmt.Printf("%s\n", color.New(color.FgYellow).Sprintf("→ %s(%s) [%s]", toolName, argsStr, summary))
    } else {
        fmt.Printf("%s\n", color.New(color.FgYellow).Sprintf("→ %s [%s]", toolName, summary))
    }
}

// formatToolArgs formats tool arguments for display.
func formatToolArgs(args map[string]interface{}) string {
    if len(args) == 0 {
        return ""
    }
    var parts []string
    for k, v := range args {
        valStr := fmt.Sprintf("%v", v)
        valStr = truncateArg(valStr, 40)
        parts = append(parts, fmt.Sprintf("%s=%s", k, valStr))
    }
    result := strings.Join(parts, ", ")
    if len(result) > 80 {
        return result[:77] + "..."
    }
    return result
}

// truncateArg truncates a string argument for display.
func truncateArg(s string, maxLen int) string {
    // Remove newlines for cleaner display
    s = strings.ReplaceAll(s, "\n", " ")
    s = strings.ReplaceAll(s, "\r", "")
    if len(s) > maxLen {
        return s[:maxLen-3] + "..."
    }
    return s
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
