// Command calculator-mcp runs the calculator MCP server over stdio.
// This is used for testing the MCP client integration.
package main

import (
	"log"

	"github.com/mark3labs/mcp-go/server"
	"github.com/opencode-ai/opencode/pkg/mcpserver/calculator"
)

func main() {
	s := calculator.NewServer()
	if err := server.ServeStdio(s); err != nil {
		log.Fatal(err)
	}
}
