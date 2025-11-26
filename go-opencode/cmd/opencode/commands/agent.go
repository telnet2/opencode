package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage custom agents",
	Long: `Manage custom agent configurations.

Agents are defined in the .opencode/agent/ directory as markdown files
or in the configuration file under the "agent" key.`,
}

var agentListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all agents",
	RunE:    runAgentList,
}

var agentCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new agent",
	RunE:  runAgentCreate,
}

var agentDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete an agent",
	RunE:  runAgentDelete,
}

func init() {
	agentCmd.AddCommand(agentListCmd)
	agentCmd.AddCommand(agentCreateCmd)
	agentCmd.AddCommand(agentDeleteCmd)
}

func runAgentList(cmd *cobra.Command, args []string) error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Load configuration
	appConfig, err := config.Load(workDir)
	if err != nil {
		return err
	}

	// List agents from config
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSOURCE\tTOOLS\t")

	// Built-in agents
	builtinAgents := []string{"coder", "plan", "explorer"}
	for _, name := range builtinAgents {
		fmt.Fprintf(w, "%s\tbuilt-in\tall\t\n", name)
	}

	// Config agents
	for name, agent := range appConfig.Agent {
		tools := "all"
		if len(agent.Tools) > 0 {
			var enabled []string
			for t, v := range agent.Tools {
				if v {
					enabled = append(enabled, t)
				}
			}
			if len(enabled) > 0 {
				tools = strings.Join(enabled, ", ")
			}
		}
		fmt.Fprintf(w, "%s\tconfig\t%s\t\n", name, tools)
	}

	// File-based agents
	agentDir := filepath.Join(workDir, ".opencode", "agent")
	entries, _ := os.ReadDir(agentDir)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			name := strings.TrimSuffix(entry.Name(), ".md")
			fmt.Fprintf(w, "%s\tfile\tcustom\t\n", name)
		}
	}

	return w.Flush()
}

func runAgentCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("agent name required")
	}

	name := args[0]
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Create .opencode/agent directory
	agentDir := filepath.Join(workDir, ".opencode", "agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		return err
	}

	// Create agent file
	agentFile := filepath.Join(agentDir, name+".md")
	if _, err := os.Stat(agentFile); err == nil {
		return fmt.Errorf("agent %s already exists", name)
	}

	template := fmt.Sprintf(`---
name: %s
description: Custom agent for %s
mode: all
tools:
  bash: true
  edit: true
  read: true
  write: true
  glob: true
  grep: true
permission:
  edit: ask
  bash: ask
---

# %s Agent

You are a specialized agent for %s tasks.

## Capabilities

- Describe what this agent can do
- List specific behaviors

## Guidelines

- Add specific instructions for this agent
`, name, name, name, name)

	if err := os.WriteFile(agentFile, []byte(template), 0644); err != nil {
		return err
	}

	fmt.Printf("Created agent: %s\n", agentFile)
	return nil
}

func runAgentDelete(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("agent name required")
	}

	name := args[0]
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Check if it's a file-based agent
	agentFile := filepath.Join(workDir, ".opencode", "agent", name+".md")
	if _, err := os.Stat(agentFile); err != nil {
		return fmt.Errorf("agent %s not found (file-based agents only can be deleted)", name)
	}

	if err := os.Remove(agentFile); err != nil {
		return err
	}

	fmt.Printf("Deleted agent: %s\n", name)
	return nil
}
