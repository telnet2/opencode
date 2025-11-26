// Package commands provides the CLI commands for OpenCode.
package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version information set at build time
	Version   = "0.1.0"
	BuildTime = "dev"
)

// Global flags
var (
	printLogs bool
	logLevel  string
)

var rootCmd = &cobra.Command{
	Use:   "opencode",
	Short: "OpenCode - AI-powered coding assistant",
	Long: `OpenCode is an AI-powered coding assistant that helps you write,
understand, and improve code through natural language interaction.

Run 'opencode run' to start an interactive session, or 'opencode serve'
to start a headless server.`,
	Version: Version,
	// Run serve by default if no subcommand specified
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand, show help
		cmd.Help()
	},
}

func init() {
	// Global flags available to all commands
	rootCmd.PersistentFlags().BoolVar(&printLogs, "print-logs", false, "Print logs to stderr")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "INFO", "Log level (DEBUG|INFO|WARN|ERROR)")

	// Version template
	rootCmd.SetVersionTemplate(fmt.Sprintf("opencode %s (%s)\n", Version, BuildTime))

	// Add subcommands
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(modelsCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(debugCmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// GetWorkDir returns the working directory from flag or current directory.
func GetWorkDir(dir string) (string, error) {
	if dir != "" {
		return dir, nil
	}
	return os.Getwd()
}
