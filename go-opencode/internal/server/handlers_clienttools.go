package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/opencode-ai/opencode/internal/clienttool"
	"github.com/opencode-ai/opencode/internal/event"
)

// clientToolsPending streams tool execution requests via SSE.
// GET /client-tools/pending/{clientID}
func (s *Server) clientToolsPending(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "clientID")
	if clientID == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "clientID required")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	sse, err := newSSEWriter(w)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	// Explicitly write status and flush headers immediately
	w.WriteHeader(http.StatusOK)
	sse.flusher.Flush()

	ctx := r.Context()

	// Channel for events (buffer to prevent blocking)
	events := make(chan event.Event, 100)

	// Subscribe to tool request events
	unsub := event.Subscribe(event.ClientToolRequest, func(e event.Event) {
		data, ok := e.Data.(event.ClientToolRequestData)
		if !ok {
			return
		}

		// Filter by clientID
		if data.ClientID != clientID {
			return
		}

		select {
		case events <- e:
		default:
			// Drop event if channel is full
		}
	})
	defer unsub()

	// Keepalive ticker (30 seconds)
	ticker := time.NewTicker(SSEHeartbeatInterval)
	defer ticker.Stop()

	// Event loop
	for {
		select {
		case <-ctx.Done():
			// Client disconnected - cleanup
			clienttool.Cleanup(clientID)
			return
		case e := <-events:
			data := e.Data.(event.ClientToolRequestData)
			// Write SSE event
			jsonData, err := json.Marshal(data.Request)
			if err != nil {
				continue
			}
			fmt.Fprintf(sse.w, "event: tool-request\ndata: %s\n\n", jsonData)
			sse.flusher.Flush()
		case <-ticker.C:
			// Send ping to keep connection alive
			fmt.Fprintf(sse.w, "event: ping\ndata: \n\n")
			sse.flusher.Flush()
		}
	}
}

// getClientTools returns tools for a specific client.
// GET /client-tools/tools/{clientID}
func (s *Server) getClientTools(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "clientID")
	if clientID == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "clientID required")
		return
	}

	tools := clienttool.GetTools(clientID)
	if tools == nil {
		tools = []clienttool.ToolDefinition{}
	}

	writeJSON(w, http.StatusOK, tools)
}

// getAllClientTools returns all registered client tools.
// GET /client-tools/tools
func (s *Server) getAllClientTools(w http.ResponseWriter, r *http.Request) {
	tools := clienttool.GetAllTools()
	if tools == nil {
		tools = make(map[string]clienttool.ToolDefinition)
	}

	writeJSON(w, http.StatusOK, tools)
}
