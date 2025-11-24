package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/telnet2/go-practice/go-memsh/api"
)

func main() {
	port := flag.Int("port", 8080, "Port to listen on")
	flag.Parse()

	server := api.NewAPIServer()

	// Setup routes
	http.HandleFunc("/api/v1/session/create", server.HandleCreateSession)
	http.HandleFunc("/api/v1/session/list", server.HandleListSessions)
	http.HandleFunc("/api/v1/session/remove", server.HandleRemoveSession)
	http.HandleFunc("/api/v1/session/repl", server.HandleREPL)

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting API server on %s", addr)
	log.Printf("API Endpoints:")
	log.Printf("  POST /api/v1/session/create  - Create new session")
	log.Printf("  POST /api/v1/session/list    - List all sessions")
	log.Printf("  POST /api/v1/session/remove  - Remove a session")
	log.Printf("  WS   /api/v1/session/repl    - JSON-RPC WebSocket REPL")
	log.Printf("  GET  /health                  - Health check")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
