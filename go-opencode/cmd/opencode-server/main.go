// Package main provides the entry point for the OpenCode server.
package main

import (
	"context"
	"flag"
	"fmt"
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
)

var (
	port      = flag.Int("port", 8080, "Server port")
	directory = flag.String("directory", "", "Working directory")
	version   = flag.Bool("version", false, "Print version and exit")
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

	// Determine working directory
	workDir := *directory
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get working directory: %v", err)
		}
	}

	log.Printf("Starting OpenCode server v%s", Version)
	log.Printf("Working directory: %s", workDir)

	// Initialize paths
	paths := config.GetPaths()
	if err := paths.EnsurePaths(); err != nil {
		log.Fatalf("Failed to create data directories: %v", err)
	}

	// Load configuration
	appConfig, err := config.Load(workDir)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
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
	serverConfig.Port = *port
	serverConfig.Directory = workDir

	// Create server
	srv := server.New(serverConfig, appConfig, store, providerReg, toolReg)

	// Start server in goroutine
	go func() {
		log.Printf("Server listening on http://localhost:%d", *port)
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
}
