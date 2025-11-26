package commands

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opencode-ai/opencode/internal/config"
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

	log.Printf("Starting OpenCode server v%s", Version)
	log.Printf("Working directory: %s", workDir)

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

	// Initialize storage
	store := storage.New(paths.StoragePath())

	// Initialize providers
	ctx := context.Background()
	providerReg, err := provider.InitializeProviders(ctx, appConfig)
	if err != nil {
		log.Printf("Warning: Failed to initialize some providers: %v", err)
	}

	// Initialize tool registry
	toolReg := tool.DefaultRegistry(workDir)

	// Configure server
	serverConfig := server.DefaultConfig()
	serverConfig.Port = servePort
	serverConfig.Directory = workDir

	// Create server
	srv := server.New(serverConfig, appConfig, store, providerReg, toolReg)

	// Start server in goroutine
	go func() {
		log.Printf("Server listening on http://%s:%d", serveHostname, servePort)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
	return nil
}
