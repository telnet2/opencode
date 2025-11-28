package commands

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/mcp"
	"github.com/opencode-ai/opencode/internal/provider"
	"github.com/opencode-ai/opencode/internal/server"
	"github.com/opencode-ai/opencode/internal/storage"
	"github.com/opencode-ai/opencode/internal/tool"
	"github.com/spf13/cobra"
)

var (
	servePort     int
	serveHostname string
	serveDir      string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start headless OpenCode server",
	Long: `Start OpenCode as a headless server that exposes an HTTP API.

This is useful for integrating OpenCode with other tools or running
it in a server environment.`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "Port to listen on")
	serveCmd.Flags().StringVar(&serveHostname, "hostname", "127.0.0.1", "Hostname to listen on")
	serveCmd.Flags().StringVar(&serveDir, "directory", "", "Working directory")
}

func runServe(cmd *cobra.Command, args []string) error {
	// Determine working directory
	workDir, err := GetWorkDir(serveDir)
	if err != nil {
		return err
	}

	logging.Info().
		Str("version", Version).
		Msg("Starting OpenCode server")
	logging.Info().
		Str("directory", workDir).
		Msg("Working directory")

	// Initialize paths
	paths := config.GetPaths()
	if err := paths.EnsurePaths(); err != nil {
		return err
	}

	// Load configuration
	appConfig, err := config.Load(workDir)
	if err != nil {
		return err
	}

	// Override model if specified via global flag
	if model := GetGlobalModel(); model != "" {
		appConfig.Model = model
	}

	// Initialize storage
	store := storage.New(paths.StoragePath())

	// Initialize providers
	ctx := context.Background()
	providerReg, err := provider.InitializeProviders(ctx, appConfig)
	if err != nil {
		logging.Warn().Err(err).Msg("Failed to initialize some providers")
	}

	// Initialize tool registry
	toolReg := tool.DefaultRegistry(workDir)

	// Configure server
	serverConfig := server.DefaultConfig()
	serverConfig.Port = servePort
	serverConfig.Directory = workDir

	// Create server
	srv := server.New(serverConfig, appConfig, store, providerReg, toolReg)

	// Initialize MCP servers from config
	if err := srv.InitializeMCP(ctx); err != nil {
		logging.Warn().Err(err).Msg("Failed to initialize some MCP servers")
	}

	// Register MCP tools in tool registry for session processor
	if srv.MCPClient() != nil && srv.ToolRegistry() != nil {
		mcp.RegisterMCPTools(srv.MCPClient(), srv.ToolRegistry())
		logging.Info().
			Int("mcpToolCount", len(srv.MCPClient().Tools())).
			Msg("Registered MCP tools in tool registry")
	}

	// Start server in goroutine
	go func() {
		logging.Info().
			Str("hostname", serveHostname).
			Int("port", servePort).
			Str("url", fmt.Sprintf("http://%s:%d", serveHostname, servePort)).
			Msg("Server listening")
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			logging.Fatal().Err(err).Msg("Server error")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logging.Info().Msg("Shutting down server...")

	// Close MCP servers
	if err := srv.CloseMCP(); err != nil {
		logging.Warn().Err(err).Msg("Error closing MCP servers")
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logging.Error().Err(err).Msg("Server shutdown error")
	}

	logging.Info().Msg("Server stopped")
	return nil
}
