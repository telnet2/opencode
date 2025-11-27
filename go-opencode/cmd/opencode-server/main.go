// Package main provides the entry point for the OpenCode server.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/provider"
	"github.com/opencode-ai/opencode/internal/server"
	"github.com/opencode-ai/opencode/internal/storage"
	"github.com/opencode-ai/opencode/internal/tool"
)

var (
	port      = flag.Int("port", 8080, "Server port")
	directory = flag.String("directory", "", "Working directory")
	version   = flag.Bool("version", false, "Print version and exit")
	logLevel  = flag.String("log-level", "INFO", "Log level (DEBUG|INFO|WARN|ERROR)")
	printLogs = flag.Bool("print-logs", false, "Enable log output to stderr")
	logFile   = flag.Bool("log-file", false, "Write logs to /tmp/opencode-YYYYMMDD-HHMMSS.log")
)

const (
	Version   = "0.1.0"
	BuildTime = "dev"
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("opencode-server %s (%s)\n", Version, BuildTime)
		os.Exit(0)
	}

	// Initialize logging
	logCfg := logging.Config{
		Level:     logging.ParseLevel(*logLevel),
		Output:    os.Stderr,
		Pretty:    *printLogs,
		LogToFile: *logFile,
	}

	if !*printLogs && !*logFile {
		// Disable logging output by default
		logCfg.Level = logging.FatalLevel
	}

	logging.Init(logCfg)
	defer logging.Close()

	// Log startup info if file logging is enabled
	if *logFile {
		logging.Info().
			Str("logFile", logging.GetLogFilePath()).
			Msg("File logging enabled")
	}

	// Determine working directory
	workDir := *directory
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			logging.Fatal().Err(err).Msg("Failed to get working directory")
		}
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
		logging.Fatal().Err(err).Msg("Failed to create data directories")
	}

	// Load configuration
	appConfig, err := config.Load(workDir)
	if err != nil {
		logging.Fatal().Err(err).Msg("Failed to load configuration")
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
	serverConfig.Port = *port
	serverConfig.Directory = workDir

	// Create server
	srv := server.New(serverConfig, appConfig, store, providerReg, toolReg)

	// Initialize MCP servers from config
	if err := srv.InitializeMCP(ctx); err != nil {
		logging.Warn().Err(err).Msg("Failed to initialize some MCP servers")
	}

	// Start server in goroutine
	go func() {
		logging.Info().
			Int("port", *port).
			Str("url", fmt.Sprintf("http://localhost:%d", *port)).
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
}
