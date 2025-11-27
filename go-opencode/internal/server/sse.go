// Package server provides HTTP handlers for the opencode server.
//
// SSE Implementation Note:
// This file contains a custom Server-Sent Events (SSE) implementation rather than
// using a third-party package like r3labs/sse. This decision was made because:
//
// 1. The current implementation is simple, clean, and well-tested (~180 lines)
// 2. It integrates directly with our internal event bus architecture
// 3. It supports custom session-based filtering specific to our needs
// 4. The r3labs/sse package is a heavier framework designed for different use cases
// 5. Replacing it would add complexity without significant benefits
//
// See docs/github-packages-opportunities.md for the full analysis.
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/opencode-ai/opencode/internal/event"
)

const (
	// SSEHeartbeatInterval is the interval for SSE heartbeats.
	SSEHeartbeatInterval = 30 * time.Second
)

// sseWriter wraps http.ResponseWriter for SSE.
type sseWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// newSSEWriter creates a new SSE writer.
func newSSEWriter(w http.ResponseWriter) (*sseWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	return &sseWriter{w: w, flusher: flusher}, nil
}

// writeEvent writes an SSE event.
func (s *sseWriter) writeEvent(eventType string, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	fmt.Fprintf(s.w, "event: %s\n", eventType)
	fmt.Fprintf(s.w, "data: %s\n\n", jsonData)
	s.flusher.Flush()

	return nil
}

// writeHeartbeat writes an SSE heartbeat comment.
func (s *sseWriter) writeHeartbeat() {
	fmt.Fprintf(s.w, ": heartbeat\n\n")
	s.flusher.Flush()
}

// globalEvents handles SSE for all events.
func (srv *Server) globalEvents(w http.ResponseWriter, r *http.Request) {
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
	// This ensures client receives headers before we wait for events
	w.WriteHeader(http.StatusOK)
	sse.flusher.Flush()

	// Channel for events
	events := make(chan event.Event, 100)

	// Subscribe to all events
	unsub := event.SubscribeAll(func(e event.Event) {
		select {
		case events <- e:
		default:
			// Drop event if channel is full
		}
	})
	defer unsub()

	// Heartbeat ticker
	ticker := time.NewTicker(SSEHeartbeatInterval)
	defer ticker.Stop()

	// Wait for client disconnect or context cancellation
	for {
		select {
		case <-r.Context().Done():
			return
		case e := <-events:
			// SDK compatible format: use "properties" instead of "data"
			data := map[string]any{
				"type":       e.Type,
				"properties": e.Data,
			}
			if err := sse.writeEvent("message", data); err != nil {
				return
			}
		case <-ticker.C:
			sse.writeHeartbeat()
		}
	}
}

// sessionEvents handles SSE for session-specific events.
func (srv *Server) sessionEvents(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionID")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "sessionID required")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sse, err := newSSEWriter(w)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	// Explicitly write status and flush headers immediately
	// This ensures client receives headers before we wait for events
	w.WriteHeader(http.StatusOK)
	sse.flusher.Flush()

	// Channel for events
	events := make(chan event.Event, 100)

	// Filter for session-specific events
	unsub := event.SubscribeAll(func(e event.Event) {
		if srv.eventBelongsToSession(e, sessionID) {
			select {
			case events <- e:
			default:
				// Drop event if channel is full
			}
		}
	})
	defer unsub()

	// Heartbeat ticker
	ticker := time.NewTicker(SSEHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case e := <-events:
			// SDK compatible format: use "properties" instead of "data"
			data := map[string]any{
				"type":       e.Type,
				"properties": e.Data,
			}
			if err := sse.writeEvent("message", data); err != nil {
				return
			}
		case <-ticker.C:
			sse.writeHeartbeat()
		}
	}
}

// eventBelongsToSession checks if an event belongs to a session.
func (srv *Server) eventBelongsToSession(e event.Event, sessionID string) bool {
	switch data := e.Data.(type) {
	case event.MessageUpdatedData:
		return data.Info != nil && data.Info.SessionID == sessionID
	case event.MessageCreatedData:
		return data.Info != nil && data.Info.SessionID == sessionID
	case event.MessagePartUpdatedData:
		// SDK compatible: Part now has sessionID via PartSessionID() method
		return data.Part != nil && data.Part.PartSessionID() == sessionID
	case event.SessionUpdatedData:
		return data.Info != nil && data.Info.ID == sessionID
	case event.SessionCreatedData:
		return data.Info != nil && data.Info.ID == sessionID
	case event.SessionDeletedData:
		return data.Info != nil && data.Info.ID == sessionID
	case event.PermissionUpdatedData:
		return data.SessionID == sessionID
	case event.PermissionRepliedData:
		return data.SessionID == sessionID
	case event.FileEditedData:
		return true // File events are session-agnostic in SDK format
	case event.SessionIdleData:
		return data.SessionID == sessionID
	case event.SessionErrorData:
		return data.SessionID == sessionID
	}
	return false
}
