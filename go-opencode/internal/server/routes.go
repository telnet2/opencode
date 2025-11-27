package server

import (
	"github.com/go-chi/chi/v5"
)

// setupRoutes configures all API routes.
func (s *Server) setupRoutes() {
	r := s.router

	// Project routes
	r.Route("/project", func(r chi.Router) {
		r.Get("/", s.listProjects)
		r.Get("/current", s.getCurrentProject)
	})

	// Session routes
	r.Route("/session", func(r chi.Router) {
		r.Get("/", s.listSessions)
		r.Post("/", s.createSession)
		r.Get("/status", s.getSessionStatus)

		r.Route("/{sessionID}", func(r chi.Router) {
			r.Get("/", s.getSession)
			r.Patch("/", s.updateSession)
			r.Delete("/", s.deleteSession)

			// Messages
			r.Get("/message", s.getMessages)
			r.Post("/message", s.sendMessage) // Streaming response
			r.Get("/message/{messageID}", s.getMessage)

			// Session operations
			r.Get("/children", s.getChildren)
			r.Post("/fork", s.forkSession)
			r.Post("/abort", s.abortSession)
			r.Post("/share", s.shareSession)
			r.Delete("/share", s.unshareSession)
			r.Post("/summarize", s.summarizeSession)
			r.Post("/init", s.initSession)
			r.Get("/diff", s.getDiff)
			r.Get("/todo", s.getTodo)
			r.Post("/revert", s.revertSession)
			r.Post("/unrevert", s.unrevertSession)
			r.Post("/command", s.sendCommand)
			r.Post("/shell", s.runShell)

			// Permissions
			r.Post("/permissions/{permissionID}", s.respondPermission)
		})
	})

	// Event streaming (SSE)
	r.Get("/event", s.sessionEvents)
	r.Get("/global/event", s.globalEvents)

	// File operations
	r.Route("/file", func(r chi.Router) {
		r.Get("/", s.listFiles)
		r.Get("/content", s.readFile)
		r.Get("/status", s.gitStatus)
	})

	// Search
	r.Route("/find", func(r chi.Router) {
		r.Get("/", s.searchText)
		r.Get("/file", s.searchFiles)
		r.Get("/symbol", s.searchSymbols)
	})

	// Configuration
	r.Route("/config", func(r chi.Router) {
		r.Get("/", s.getConfig)
		r.Patch("/", s.updateConfig)
		r.Get("/providers", s.listProviders)
	})

	// Providers
	r.Route("/provider", func(r chi.Router) {
		r.Get("/", s.listAllProviders)
		r.Get("/auth", s.getAuthMethods)
		r.Post("/{providerID}/oauth/authorize", s.oauthAuthorize)
		r.Post("/{providerID}/oauth/callback", s.oauthCallback)
	})

	// Authentication
	r.Put("/auth/{providerID}", s.setAuth)

	// Advanced features
	r.Get("/lsp", s.getLSPStatus)
	r.Get("/agent", s.listAgents)

	// MCP routes
	r.Route("/mcp", func(r chi.Router) {
		r.Get("/", s.getMCPStatus)
		r.Post("/", s.addMCPServer)
		r.Delete("/{name}", s.removeMCPServer)
		r.Get("/tools", s.getMCPTools)
		r.Post("/tool/{name}", s.executeMCPTool)
		r.Get("/resources", s.getMCPResources)
		r.Get("/resource", s.readMCPResource)
	})

	// Formatter routes
	r.Route("/formatter", func(r chi.Router) {
		r.Get("/", s.getFormatterStatus)
		r.Post("/format", s.formatFile)
	})

	// Command routes
	r.Route("/command", func(r chi.Router) {
		r.Get("/", s.listCommands)
		r.Get("/{name}", s.getCommand)
		r.Post("/{name}", s.executeCommand)
	})

	// Instance management
	r.Get("/path", s.getPath)
	r.Post("/log", s.writeLog)
	r.Post("/instance/dispose", s.disposeInstance)

	// Experimental
	r.Route("/experimental", func(r chi.Router) {
		r.Get("/tool/ids", s.getToolIDs)
		r.Get("/tool", s.getToolDefinitions)
	})

	// TUI control
	r.Route("/tui", func(r chi.Router) {
		r.Post("/append-prompt", s.tuiAppendPrompt)
		r.Post("/execute-command", s.tuiExecuteCommand)
		r.Post("/show-toast", s.tuiShowToast)
		r.Post("/publish", s.tuiPublish)
		r.Post("/open-help", s.tuiOpenHelp)
		r.Post("/open-sessions", s.tuiOpenSessions)
		r.Post("/open-themes", s.tuiOpenThemes)
		r.Post("/open-models", s.tuiOpenModels)
		r.Post("/submit-prompt", s.tuiSubmitPrompt)
		r.Post("/clear-prompt", s.tuiClearPrompt)

		// TUI control queue (for remote TUI control)
		r.Route("/control", func(r chi.Router) {
			r.Get("/next", s.tuiControlNext)
			r.Post("/response", s.tuiControlResponse)
		})
	})

	// Client tools (for external tool registration)
	r.Route("/client-tools", func(r chi.Router) {
		r.Post("/register", s.registerClientTool)
		r.Delete("/unregister", s.unregisterClientTool)
		r.Post("/execute", s.executeClientTool)
		r.Post("/result", s.submitClientToolResult)

		// Query and SSE endpoints
		r.Get("/pending/{clientID}", s.clientToolsPending)
		r.Get("/tools/{clientID}", s.getClientTools)
		r.Get("/tools", s.getAllClientTools)
	})

	// OpenAPI documentation
	r.Get("/doc", s.openAPISpec)
}
