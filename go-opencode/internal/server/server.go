// Package server provides the HTTP server for the OpenCode API.
package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/opencode-ai/opencode/internal/command"
	"github.com/opencode-ai/opencode/internal/event"
	"github.com/opencode-ai/opencode/internal/formatter"
	"github.com/opencode-ai/opencode/internal/mcp"
	"github.com/opencode-ai/opencode/internal/provider"
	"github.com/opencode-ai/opencode/internal/session"
	"github.com/opencode-ai/opencode/internal/storage"
	"github.com/opencode-ai/opencode/internal/tool"
	"github.com/opencode-ai/opencode/pkg/types"
)

// Config holds server configuration.
type Config struct {
	Port         int
	Directory    string
	EnableCORS   bool
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DefaultConfig returns default server configuration.
func DefaultConfig() *Config {
	return &Config{
		Port:         8080,
		Directory:    "",
		EnableCORS:   true,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // No write timeout for SSE
	}
}

// Server is the HTTP server.
type Server struct {
	config           *Config
	router           *chi.Mux
	httpSrv          *http.Server
	appConfig        *types.Config
	storage          *storage.Storage
	sessionService   *session.Service
	providerReg      *provider.Registry
	toolReg          *tool.Registry
	bus              *event.Bus
	mcpClient        *mcp.Client
	commandExecutor  *command.Executor
	formatterManager *formatter.Manager
}

// New creates a new Server instance.
func New(cfg *Config, appConfig *types.Config, store *storage.Storage, providerReg *provider.Registry, toolReg *tool.Registry) *Server {
	r := chi.NewRouter()

	// Parse default provider and model from config
	// Format: "provider/model" (e.g., "ark/ep-xxx" or "anthropic/claude-sonnet-4-20250514")
	var defaultProviderID, defaultModelID string
	if appConfig != nil && appConfig.Model != "" {
		parts := strings.SplitN(appConfig.Model, "/", 2)
		if len(parts) == 2 {
			defaultProviderID = parts[0]
			defaultModelID = parts[1]
		}
	}

	// Create MCP client
	mcpClient := mcp.NewClient()

	// Create command executor
	cmdExecutor := command.NewExecutor(cfg.Directory, appConfig)

	// Create formatter manager
	fmtManager := formatter.NewManager(cfg.Directory, appConfig)

	s := &Server{
		config:           cfg,
		router:           r,
		appConfig:        appConfig,
		storage:          store,
		sessionService:   session.NewServiceWithProcessor(store, providerReg, toolReg, nil, defaultProviderID, defaultModelID),
		providerReg:      providerReg,
		toolReg:          toolReg,
		bus:              event.NewBus(),
		mcpClient:        mcpClient,
		commandExecutor:  cmdExecutor,
		formatterManager: fmtManager,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// InitializeMCP initializes MCP servers from configuration.
func (s *Server) InitializeMCP(ctx context.Context) error {
	if s.appConfig == nil || s.appConfig.MCP == nil {
		return nil
	}

	for name, cfg := range s.appConfig.MCP {
		enabled := cfg.Enabled == nil || *cfg.Enabled
		mcpCfg := &mcp.Config{
			Enabled:     enabled,
			Type:        mcp.TransportType(cfg.Type),
			URL:         cfg.URL,
			Headers:     cfg.Headers,
			Command:     cfg.Command,
			Environment: cfg.Environment,
			Timeout:     cfg.Timeout,
		}
		if err := s.mcpClient.AddServer(ctx, name, mcpCfg); err != nil {
			// Log but don't fail on individual server errors
			continue
		}
	}

	return nil
}

// CloseMCP closes all MCP server connections.
func (s *Server) CloseMCP() error {
	if s.mcpClient != nil {
		return s.mcpClient.Close()
	}
	return nil
}

// setupMiddleware configures middleware for the server.
func (s *Server) setupMiddleware() {
	// Request ID
	s.router.Use(middleware.RequestID)

	// Logging
	s.router.Use(middleware.Logger)

	// Recover from panics
	s.router.Use(middleware.Recoverer)

	// Real IP
	s.router.Use(middleware.RealIP)

	// CORS
	if s.config.EnableCORS {
		s.router.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
			ExposedHeaders:   []string{"Link", "X-Request-ID"},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	}

	// Instance context
	s.router.Use(s.instanceContext)
}

// instanceContext middleware injects directory into context.
func (s *Server) instanceContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get directory from query or use default
		dir := r.URL.Query().Get("directory")
		if dir == "" {
			dir = s.config.Directory
		}

		ctx := context.WithValue(r.Context(), contextKeyDirectory, dir)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.httpSrv = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	return s.httpSrv.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

// Router returns the Chi router for testing.
func (s *Server) Router() *chi.Mux {
	return s.router
}

// Context keys
type contextKey string

const (
	contextKeyDirectory contextKey = "directory"
)

// getDirectory returns the directory from context.
func getDirectory(ctx context.Context) string {
	if dir, ok := ctx.Value(contextKeyDirectory).(string); ok {
		return dir
	}
	return ""
}
