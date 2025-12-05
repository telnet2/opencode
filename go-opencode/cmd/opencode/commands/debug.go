package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug utilities",
	Long:  `Debug utilities for troubleshooting OpenCode configuration and setup.`,
}

var debugConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	RunE:  runDebugConfig,
}

var debugPathsCmd = &cobra.Command{
	Use:   "paths",
	Short: "Show system paths",
	RunE:  runDebugPaths,
}

func init() {
	debugCmd.AddCommand(debugConfigCmd)
	debugCmd.AddCommand(debugPathsCmd)
}

func runDebugConfig(cmd *cobra.Command, args []string) error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Load configuration
	appConfig, err := config.Load(workDir)
	if err != nil {
		return err
	}

	// Output as JSON
	data, err := json.MarshalIndent(appConfig, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}

func runDebugPaths(cmd *cobra.Command, args []string) error {
	paths := config.GetPaths()

	fmt.Println("OpenCode System Paths:")
	fmt.Println()
	fmt.Printf("  Config:   %s\n", paths.Config)
	fmt.Printf("  Data:     %s\n", paths.Data)
	fmt.Printf("  Cache:    %s\n", paths.Cache)
	fmt.Printf("  State:    %s\n", paths.State)
	fmt.Printf("  Storage:  %s\n", paths.StoragePath())
	fmt.Printf("  Auth:     %s\n", paths.AuthPath())
	fmt.Println()

	// Also show TS-compatible paths
	home := os.Getenv("HOME")
	fmt.Println("TypeScript-Compatible Paths:")
	fmt.Printf("  Config:   %s/.opencode\n", home)
	fmt.Printf("  Auth:     %s/.opencode/auth.json\n", home)

	return nil
}
