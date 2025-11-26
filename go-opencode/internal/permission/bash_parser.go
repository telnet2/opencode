package permission

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// BashCommand represents a parsed command with its arguments.
type BashCommand struct {
	Name       string   // Command name (e.g., "rm", "git")
	Args       []string // Command arguments
	Subcommand string   // First non-flag argument (e.g., "commit" in "git commit")
}

// ParseBashCommand parses a bash command string into structured commands.
func ParseBashCommand(command string) ([]BashCommand, error) {
	parser := syntax.NewParser(
		syntax.Variant(syntax.LangBash),
		syntax.KeepComments(false),
	)

	file, err := parser.Parse(strings.NewReader(command), "")
	if err != nil {
		return nil, fmt.Errorf("failed to parse command: %w", err)
	}

	var commands []BashCommand
	syntax.Walk(file, func(node syntax.Node) bool {
		switch n := node.(type) {
		case *syntax.CallExpr:
			cmd := extractCommand(n)
			if cmd != nil {
				commands = append(commands, *cmd)
			}
		}
		return true
	})

	return commands, nil
}

// extractCommand extracts command name and arguments from a CallExpr.
func extractCommand(call *syntax.CallExpr) *BashCommand {
	if len(call.Args) == 0 {
		return nil
	}

	cmd := &BashCommand{}

	// Extract command name from first word
	cmd.Name = wordToString(call.Args[0])
	if cmd.Name == "" {
		return nil
	}

	// Extract arguments
	for _, arg := range call.Args[1:] {
		argStr := wordToString(arg)
		cmd.Args = append(cmd.Args, argStr)

		// Find first non-flag argument as subcommand
		if cmd.Subcommand == "" && !strings.HasPrefix(argStr, "-") {
			cmd.Subcommand = argStr
		}
	}

	return cmd
}

// wordToString converts a syntax.Word to a string.
func wordToString(word *syntax.Word) string {
	var sb strings.Builder
	for _, part := range word.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			sb.WriteString(p.Value)
		case *syntax.SglQuoted:
			sb.WriteString(p.Value)
		case *syntax.DblQuoted:
			for _, qp := range p.Parts {
				if lit, ok := qp.(*syntax.Lit); ok {
					sb.WriteString(lit.Value)
				}
			}
		case *syntax.ParamExp:
			// Variable expansion - return placeholder
			sb.WriteString("$" + p.Param.Value)
		case *syntax.CmdSubst:
			// Command substitution - ignore the content, mark as dynamic
			sb.WriteString("$()")
		}
	}
	return sb.String()
}

// DangerousCommands are commands that modify files and need path validation.
var DangerousCommands = map[string]bool{
	"cd":    true,
	"rm":    true,
	"cp":    true,
	"mv":    true,
	"mkdir": true,
	"touch": true,
	"chmod": true,
	"chown": true,
	"rmdir": true,
	"dd":    true,
}

// IsDangerousCommand checks if a command is in the dangerous list.
func IsDangerousCommand(name string) bool {
	return DangerousCommands[name]
}

// ExtractPaths extracts file paths from command arguments.
func ExtractPaths(cmd BashCommand) []string {
	var paths []string
	for _, arg := range cmd.Args {
		// Skip flags
		if strings.HasPrefix(arg, "-") {
			continue
		}
		// Skip chmod mode arguments (numeric or symbolic like u+x)
		if cmd.Name == "chmod" {
			if len(arg) > 0 && (arg[0] >= '0' && arg[0] <= '9' ||
				arg[0] == 'u' || arg[0] == 'g' || arg[0] == 'o' || arg[0] == 'a' ||
				arg[0] == '+' || arg[0] == '=') {
				continue
			}
		}
		paths = append(paths, arg)
	}
	return paths
}

// ResolvePath resolves a path to absolute, handling relative paths.
func ResolvePath(ctx context.Context, path, workDir string) (string, error) {
	// Handle absolute paths
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	// Handle home directory
	if strings.HasPrefix(path, "~") {
		// Can't safely expand ~ without knowing the user
		return path, nil
	}

	// Use realpath for relative paths if available
	cmd := exec.CommandContext(ctx, "realpath", "-m", path)
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		// Fallback to manual resolution
		return filepath.Clean(filepath.Join(workDir, path)), nil
	}
	return strings.TrimSpace(string(output)), nil
}

// IsWithinDir checks if path is within or under directory.
func IsWithinDir(path, dir string) bool {
	// Clean both paths
	path = filepath.Clean(path)
	dir = filepath.Clean(dir)

	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}
