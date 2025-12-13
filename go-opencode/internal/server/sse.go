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
	"github.com/opencode-ai/opencode/internal/logging"
)

// SDKEvent represents an SDK-compatible event with proper JSON field ordering.
// TypeScript expects: {"type": "...", "properties": {...}}
type SDKEvent struct {
	Type       event.EventType `json:"type"`
	Properties any             `json:"properties"`
}

const (
	// SSEHeartbeatInterval is the interval for SSE heartbeats.
	SSEHeartbeatInterval = 30 * time.Second
)

// sseWriter wraps http.ResponseWriter for SSE.
type sseWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	rc      *http.ResponseController
}

// newSSEWriter creates a new SSE writer.
func newSSEWriter(w http.ResponseWriter) (*sseWriter, error) {
	// Use ResponseController for more reliable flushing (Go 1.20+)
	rc := http.NewResponseController(w)

	// Try to get flusher interface as well
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	return &sseWriter{w: w, flusher: flusher, rc: rc}, nil
}

// writeEvent writes an SSE event with optional throttling.
func (s *sseWriter) writeEvent(eventType string, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Write SSE format: event type, data, and blank line
	_, err = fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", eventType, jsonData)
	if err != nil {
		return err
	}

	// Flush immediately using ResponseController (more reliable than Flusher interface)
	// This ensures data is sent even through middleware wrappers
	if flushErr := s.rc.Flush(); flushErr != nil {
		// Fallback to traditional flusher
		s.flusher.Flush()
	}

	return nil
}

// writeHeartbeat writes an SSE heartbeat comment.
func (s *sseWriter) writeHeartbeat() {
	fmt.Fprintf(s.w, ": heartbeat\n\n")
	s.flusher.Flush()
}

// allEvents handles SSE for all events (used by /event endpoint).
// This is the main event endpoint that the TUI connects to.
func (srv *Server) allEvents(w http.ResponseWriter, r *http.Request) {
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

	// Send server.connected event first (SDK compatible)
	connectedEvent := SDKEvent{
		Type:       "server.connected",
		Properties: map[string]any{},
	}
	if err := sse.writeEvent("message", connectedEvent); err != nil {
		return
	}

	// Channel for events - use small buffer for low-latency streaming
	events := make(chan event.Event, 10)

	// Subscribe to all events
	unsub := event.SubscribeAll(func(e event.Event) {
		select {
		case events <- e:
		default:
			logging.Warn().
				Str("eventType", string(e.Type)).
				Msg("SSE event dropped: channel full")
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
			// SDK compatible format: use struct for proper field ordering
			data := SDKEvent{
				Type:       e.Type,
				Properties: e.Data,
			}
			if err := sse.writeEvent("message", data); err != nil {
				return
			}
		case <-ticker.C:
			sse.writeHeartbeat()
		}
	}
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

	// Channel for events - use small buffer for low-latency streaming
	events := make(chan event.Event, 10)

	// Subscribe to all events
	unsub := event.SubscribeAll(func(e event.Event) {
		select {
		case events <- e:
		default:
			logging.Warn().
				Str("eventType", string(e.Type)).
				Msg("SSE global event dropped: channel full")
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
			// SDK compatible format: use struct for proper field ordering
			data := SDKEvent{
				Type:       e.Type,
				Properties: e.Data,
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

	// Channel for events - use small buffer for low-latency streaming
	events := make(chan event.Event, 10)

	// Filter for session-specific events
	unsub := event.SubscribeAll(func(e event.Event) {
		if srv.eventBelongsToSession(e, sessionID) {
			select {
			case events <- e:
			default:
				logging.Warn().
					Str("eventType", string(e.Type)).
					Str("sessionID", sessionID).
					Msg("SSE session event dropped: channel full")
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
			// SDK compatible format: use struct for proper field ordering
			data := SDKEvent{
				Type:       e.Type,
				Properties: e.Data,
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
	case event.SessionDiffData:
		return data.SessionID == sessionID
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
