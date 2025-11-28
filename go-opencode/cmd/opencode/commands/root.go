// Package commands provides the CLI commands for OpenCode.
package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/spf13/cobra"
)

var (
	// Version information set at build time
	Version   = "0.1.0"
	BuildTime = "dev"
)

// Global flags
var (
	printLogs   bool
	logLevel    string
	logFile     bool
	showConfig  bool
	globalModel string
)

var rootCmd = &cobra.Command{
	Use:   "opencode",
	Short: "OpenCode - AI-powered coding assistant",
	Long: `OpenCode is an AI-powered coding assistant that helps you write,
understand, and improve code through natural language interaction.

Run 'opencode run' to start an interactive session, or 'opencode serve'
to start a headless server.`,
	Version: Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize logging based on flags
		logCfg := logging.Config{
			Level:     logging.ParseLevel(logLevel),
			Output:    os.Stderr,
			Pretty:    printLogs,
			LogToFile: logFile,
		}

		if !printLogs && !logFile {
			// Disable logging output by default (only show fatal errors)
			logCfg.Level = logging.FatalLevel
		}

		logging.Init(logCfg)

		// Log startup info if file logging is enabled
		if logFile {
			logging.Info().
				Str("version", Version).
				Str("logFile", logging.GetLogFilePath()).
				Msg("OpenCode started with file logging")
		}

		// Handle --show-config flag
		if showConfig {
			dir, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
				os.Exit(1)
			}

			cfg, err := config.Load(dir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
				os.Exit(1)
			}

			jsonData, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshaling config: %v\n", err)
				os.Exit(1)
			}

			fmt.Println(string(jsonData))
			os.Exit(0)
		}
	},
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
	rootCmd.PersistentFlags().BoolVar(&logFile, "log-file", false, "Write logs to /tmp/opencode-YYYYMMDD-HHMMSS.log")
	rootCmd.PersistentFlags().BoolVar(&showConfig, "show-config", false, "Print merged configuration as JSON and exit")
	rootCmd.PersistentFlags().StringVarP(&globalModel, "model", "m", "", "Model to use (provider/model format)")

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

// GetGlobalModel returns the global model flag value.
func GetGlobalModel() string {
	return globalModel
}
