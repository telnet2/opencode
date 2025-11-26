// Package main provides the entry point for the OpenCode CLI.
package main

import (
	"fmt"
	"os"

	"github.com/opencode-ai/opencode/cmd/opencode/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
