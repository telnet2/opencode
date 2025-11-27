// Package command provides custom command execution for OpenCode.
package command

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/opencode-ai/opencode/pkg/types"
)

// Command represents a parsed command ready for execution.
type Command struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Template    string            `json:"template"`
	Agent       string            `json:"agent,omitempty"`
	Model       string            `json:"model,omitempty"`
	Subtask     bool              `json:"subtask,omitempty"`
	Source      string            `json:"source,omitempty"` // "config" or "file"
	Variables   map[string]string `json:"variables,omitempty"`
}

// ExecuteResult represents the result of command execution.
type ExecuteResult struct {
	Prompt      string `json:"prompt"`
	Agent       string `json:"agent,omitempty"`
	Model       string `json:"model,omitempty"`
	Subtask     bool   `json:"subtask,omitempty"`
	CommandName string `json:"commandName"`
}

// Executor handles command parsing and execution.
type Executor struct {
	workDir   string
	config    *types.Config
	commands  map[string]*Command
	variables map[string]string
}

// NewExecutor creates a new command executor.
func NewExecutor(workDir string, config *types.Config) *Executor {
	e := &Executor{
		workDir:   workDir,
		config:    config,
		commands:  make(map[string]*Command),
		variables: make(map[string]string),
	}

	// Load commands from config
	e.loadFromConfig()

	// Load commands from files
	e.loadFromFiles()

	// Load prompt variables
	e.loadVariables()

	return e
}

// loadFromConfig loads commands from the config file.
func (e *Executor) loadFromConfig() {
	if e.config == nil || e.config.Command == nil {
		return
	}

	for name, cfg := range e.config.Command {
		e.commands[name] = &Command{
			Name:        name,
			Description: cfg.Description,
			Template:    cfg.Template,
			Agent:       cfg.Agent,
			Model:       cfg.Model,
			Subtask:     cfg.Subtask,
			Source:      "config",
		}
	}
}

// loadFromFiles loads commands from .opencode/command/ directory.
func (e *Executor) loadFromFiles() {
	commandDir := filepath.Join(e.workDir, ".opencode", "command")
	if _, err := os.Stat(commandDir); os.IsNotExist(err) {
		return
	}

	// Walk the command directory
	err := filepath.Walk(commandDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Parse markdown command file
		cmd, parseErr := e.parseMarkdownCommand(path)
		if parseErr != nil {
			return nil // Skip parse errors
		}

		// Use relative path as command name (without .md extension)
		relPath, _ := filepath.Rel(commandDir, path)
		name := strings.TrimSuffix(relPath, ".md")
		name = strings.ReplaceAll(name, string(filepath.Separator), ":")

		cmd.Name = name
		cmd.Source = "file"
		e.commands[name] = cmd

		return nil
	})

	_ = err // Ignore walk errors
}

// parseMarkdownCommand parses a markdown file as a command definition.
func (e *Executor) parseMarkdownCommand(path string) (*Command, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cmd := &Command{}

	lines := strings.Split(string(content), "\n")
	var templateLines []string
	inFrontmatter := false
	frontmatterDone := false

	for i, line := range lines {
		// Check for frontmatter delimiter
		if i == 0 && strings.TrimSpace(line) == "---" {
			inFrontmatter = true
			continue
		}

		if inFrontmatter && strings.TrimSpace(line) == "---" {
			inFrontmatter = false
			frontmatterDone = true
			continue
		}

		if inFrontmatter {
			// Parse frontmatter (simple YAML-like)
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				value = strings.Trim(value, "\"'")

				switch key {
				case "description":
					cmd.Description = value
				case "agent":
					cmd.Agent = value
				case "model":
					cmd.Model = value
				case "subtask":
					cmd.Subtask = value == "true"
				}
			}
		} else {
			templateLines = append(templateLines, line)
		}
	}

	// If no frontmatter, use entire file as template
	if !frontmatterDone {
		cmd.Template = string(content)
	} else {
		cmd.Template = strings.TrimSpace(strings.Join(templateLines, "\n"))
	}

	return cmd, nil
}

// loadVariables loads prompt variables from config.
func (e *Executor) loadVariables() {
	if e.config == nil || e.config.PromptVariables == nil {
		return
	}

	for k, v := range e.config.PromptVariables {
		e.variables[k] = v
	}
}

// List returns all available commands.
func (e *Executor) List() []*Command {
	commands := make([]*Command, 0, len(e.commands))
	for _, cmd := range e.commands {
		commands = append(commands, cmd)
	}
	return commands
}

// Get returns a specific command by name.
func (e *Executor) Get(name string) (*Command, bool) {
	cmd, ok := e.commands[name]
	return cmd, ok
}

// Execute executes a command with the given arguments.
func (e *Executor) Execute(ctx context.Context, name string, args string) (*ExecuteResult, error) {
	cmd, ok := e.commands[name]
	if !ok {
		return nil, fmt.Errorf("command not found: %s", name)
	}

	// Parse arguments
	parsedArgs := e.parseArguments(args)

	// Build template context
	templateCtx := e.buildTemplateContext(parsedArgs)

	// Execute template
	prompt, err := e.executeTemplate(cmd.Template, templateCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return &ExecuteResult{
		Prompt:      prompt,
		Agent:       cmd.Agent,
		Model:       cmd.Model,
		Subtask:     cmd.Subtask,
		CommandName: cmd.Name,
	}, nil
}

// parseArguments parses command arguments.
func (e *Executor) parseArguments(args string) map[string]string {
	result := make(map[string]string)

	// Store the full input as $input
	result["input"] = strings.TrimSpace(args)

	// Parse numbered arguments ($1, $2, ...)
	parts := strings.Fields(args)
	for i, part := range parts {
		result[fmt.Sprintf("%d", i+1)] = part
	}

	// Parse named arguments (--name=value or --name value)
	namedRe := regexp.MustCompile(`--(\w+)(?:=(\S+)|(?:\s+(\S+))?)`)
	matches := namedRe.FindAllStringSubmatch(args, -1)
	for _, match := range matches {
		name := match[1]
		value := match[2]
		if value == "" {
			value = match[3]
		}
		if value == "" {
			value = "true"
		}
		result[name] = value
	}

	return result
}

// buildTemplateContext builds the template execution context.
func (e *Executor) buildTemplateContext(args map[string]string) map[string]any {
	ctx := make(map[string]any)

	// Add arguments
	ctx["args"] = args
	ctx["input"] = args["input"]

	// Add numbered args directly
	for k, v := range args {
		if _, err := fmt.Sscanf(k, "%d", new(int)); err == nil {
			ctx[k] = v
		}
	}

	// Add variables
	ctx["vars"] = e.variables
	for k, v := range e.variables {
		ctx["var_"+k] = v
	}

	// Add environment
	ctx["env"] = envMap()

	// Add working directory
	ctx["workDir"] = e.workDir

	return ctx
}

// executeTemplate executes a Go template with the given context.
func (e *Executor) executeTemplate(tmplStr string, ctx map[string]any) (string, error) {
	// Also support simple variable substitution for ${var} and $var syntax
	tmplStr = e.expandSimpleVariables(tmplStr, ctx)

	tmpl, err := template.New("command").Funcs(templateFuncs()).Parse(tmplStr)
	if err != nil {
		// If template parsing fails, just return the expanded string
		return tmplStr, nil
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		// If execution fails, return the original with simple expansion
		return tmplStr, nil
	}

	return buf.String(), nil
}

// expandSimpleVariables expands ${var} and $var syntax.
func (e *Executor) expandSimpleVariables(s string, ctx map[string]any) string {
	// Expand ${name} syntax
	re := regexp.MustCompile(`\$\{(\w+)\}`)
	s = re.ReplaceAllStringFunc(s, func(match string) string {
		name := match[2 : len(match)-1]
		if val, ok := ctx[name]; ok {
			return fmt.Sprint(val)
		}
		if args, ok := ctx["args"].(map[string]string); ok {
			if val, ok := args[name]; ok {
				return val
			}
		}
		return match
	})

	// Expand $name syntax (but not $$)
	re = regexp.MustCompile(`\$(\w+)`)
	s = re.ReplaceAllStringFunc(s, func(match string) string {
		name := match[1:]
		if val, ok := ctx[name]; ok {
			return fmt.Sprint(val)
		}
		if args, ok := ctx["args"].(map[string]string); ok {
			if val, ok := args[name]; ok {
				return val
			}
		}
		return match
	})

	return s
}

// templateFuncs returns custom template functions.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"env": func(name string) string {
			return os.Getenv(name)
		},
		"default": func(defaultVal, val string) string {
			if val == "" {
				return defaultVal
			}
			return val
		},
		"trim": strings.TrimSpace,
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"replace": strings.ReplaceAll,
		"split": strings.Split,
		"join":  strings.Join,
	}
}

// envMap returns environment variables as a map.
func envMap() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	return env
}

// AddCommand adds or updates a command.
func (e *Executor) AddCommand(cmd *Command) {
	e.commands[cmd.Name] = cmd
}

// RemoveCommand removes a command by name.
func (e *Executor) RemoveCommand(name string) bool {
	if _, ok := e.commands[name]; ok {
		delete(e.commands, name)
		return true
	}
	return false
}

// Reload reloads commands from config and files.
func (e *Executor) Reload() {
	e.commands = make(map[string]*Command)
	e.loadFromConfig()
	e.loadFromFiles()
	e.loadVariables()
}

// BuiltinCommands returns the list of built-in commands.
func BuiltinCommands() []*Command {
	return []*Command{
		{
			Name:        "help",
			Description: "Show available commands and help information",
			Source:      "builtin",
		},
		{
			Name:        "clear",
			Description: "Clear the current conversation",
			Source:      "builtin",
		},
		{
			Name:        "compact",
			Description: "Compact the conversation to save context",
			Source:      "builtin",
		},
		{
			Name:        "reset",
			Description: "Reset the session to its initial state",
			Source:      "builtin",
		},
		{
			Name:        "undo",
			Description: "Undo the last message",
			Source:      "builtin",
		},
		{
			Name:        "share",
			Description: "Share the current session",
			Source:      "builtin",
		},
		{
			Name:        "export",
			Description: "Export the conversation",
			Source:      "builtin",
		},
	}
}
