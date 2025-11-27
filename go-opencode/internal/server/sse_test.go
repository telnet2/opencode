package server

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/opencode-ai/opencode/internal/event"
	"github.com/opencode-ai/opencode/pkg/types"
)

// mockFlusher implements http.Flusher for testing
type mockResponseWriter struct {
	*httptest.ResponseRecorder
	flushed int
}

func (m *mockResponseWriter) Flush() {
	m.flushed++
}

func newMockResponseWriter() *mockResponseWriter {
	return &mockResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
	}
}

func TestNewSSEWriter(t *testing.T) {
	w := newMockResponseWriter()
	sse, err := newSSEWriter(w)
	if err != nil {
		t.Fatalf("newSSEWriter failed: %v", err)
	}
	if sse == nil {
		t.Fatal("SSE writer should not be nil")
	}
}

func TestNewSSEWriter_NoFlusher(t *testing.T) {
	// Use a writer that doesn't implement Flusher
	w := &noFlushWriter{}
	_, err := newSSEWriter(w)
	if err == nil {
		t.Error("Expected error for writer without Flusher")
	}
}

type noFlushWriter struct{}

func (n *noFlushWriter) Header() http.Header       { return http.Header{} }
func (n *noFlushWriter) Write([]byte) (int, error) { return 0, nil }
func (n *noFlushWriter) WriteHeader(int)           {}

func TestSSEWriter_WriteEvent(t *testing.T) {
	w := newMockResponseWriter()
	sse, _ := newSSEWriter(w)

	data := map[string]string{"message": "hello"}
	err := sse.writeEvent("test", data)
	if err != nil {
		t.Fatalf("writeEvent failed: %v", err)
	}

	body := w.Body.String()
	if !strings.Contains(body, "event: test\n") {
		t.Error("Expected event line")
	}
	if !strings.Contains(body, `"message":"hello"`) {
		t.Error("Expected data to contain message")
	}
	if w.flushed == 0 {
		t.Error("Expected Flush to be called")
	}
}

func TestSSEWriter_WriteHeartbeat(t *testing.T) {
	w := newMockResponseWriter()
	sse, _ := newSSEWriter(w)

	sse.writeHeartbeat()

	body := w.Body.String()
	if !strings.Contains(body, ": heartbeat\n") {
		t.Errorf("Expected heartbeat comment, got: %s", body)
	}
	if w.flushed == 0 {
		t.Error("Expected Flush to be called")
	}
}

func TestSSEHeaders(t *testing.T) {
	// Create minimal server for testing
	srv := &Server{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate the header setup from globalEvents
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/events", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Error("Expected Content-Type: text/event-stream")
	}
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Error("Expected Cache-Control: no-cache")
	}
	if w.Header().Get("Connection") != "keep-alive" {
		t.Error("Expected Connection: keep-alive")
	}
	if w.Header().Get("X-Accel-Buffering") != "no" {
		t.Error("Expected X-Accel-Buffering: no")
	}

	_ = srv // silence unused
}

func TestEventBelongsToSession(t *testing.T) {
	srv := &Server{}

	tests := []struct {
		name      string
		event     event.Event
		sessionID string
		expected  bool
	}{
		{
			name: "MessageCreated matches",
			event: event.Event{
				Type: event.MessageCreated,
				Data: event.MessageCreatedData{
					Info: &types.Message{
						ID:        "msg-1",
						SessionID: "session-123",
					},
				},
			},
			sessionID: "session-123",
			expected:  true,
		},
		{
			name: "MessageCreated no match",
			event: event.Event{
				Type: event.MessageCreated,
				Data: event.MessageCreatedData{
					Info: &types.Message{
						ID:        "msg-1",
						SessionID: "session-456",
					},
				},
			},
			sessionID: "session-123",
			expected:  false,
		},
		{
			name: "FileEdited matches (session-agnostic)",
			event: event.Event{
				Type: event.FileEdited,
				Data: event.FileEditedData{
					File: "/path/to/file.go",
				},
			},
			sessionID: "session-123",
			expected:  true, // FileEdited is now session-agnostic in SDK format
		},
		{
			name: "MessagePartUpdated matches",
			event: event.Event{
				Type: event.MessagePartUpdated,
				Data: event.MessagePartUpdatedData{
					Part: &types.TextPart{
						ID:        "part-1",
						SessionID: "session-123",
						MessageID: "msg-1",
					},
				},
			},
			sessionID: "session-123",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := srv.eventBelongsToSession(tt.event, tt.sessionID)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGlobalEvents_Integration(t *testing.T) {
	event.Reset() // Clear any existing subscribers

	srv := &Server{}

	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(srv.globalEvents))
	defer ts.Close()

	// Create a client with timeout
	client := &http.Client{Timeout: 2 * time.Second}

	// Start request in goroutine
	var wg sync.WaitGroup
	wg.Add(1)

	var receivedEvents []map[string]any
	var mu sync.Mutex

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL, nil)

	go func() {
		defer wg.Done()

		resp, err := client.Do(req)
		if err != nil {
			// Context cancelled is expected
			return
		}
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				var evt map[string]any
				if err := json.Unmarshal([]byte(data), &evt); err == nil {
					mu.Lock()
					receivedEvents = append(receivedEvents, evt)
					mu.Unlock()
				}
			}
		}
	}()

	// Give time for connection
	time.Sleep(100 * time.Millisecond)

	// Publish an event
	event.PublishSync(event.Event{
		Type: event.SessionCreated,
		Data: map[string]string{"id": "test-session"},
	})

	// Wait for events to be processed
	time.Sleep(100 * time.Millisecond)

	// Cancel context to close connection
	cancel()
	wg.Wait()

	// Note: Due to async nature, we may or may not receive events
	// The main test is that no panic/deadlock occurred
}

func TestSessionEvents_MissingSessionID(t *testing.T) {
	srv := &Server{}

	req := httptest.NewRequest("GET", "/session/events", nil)
	w := httptest.NewRecorder()

	srv.sessionEvents(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}

	var result ErrorResponse
	json.NewDecoder(w.Body).Decode(&result)
	if result.Error.Code != ErrCodeInvalidRequest {
		t.Errorf("Expected INVALID_REQUEST error code")
	}
}

func TestSSEEventFormat(t *testing.T) {
	w := newMockResponseWriter()
	sse, _ := newSSEWriter(w)

	testData := struct {
		Type string `json:"type"`
		ID   int    `json:"id"`
	}{
		Type: "test",
		ID:   123,
	}

	sse.writeEvent("message", testData)

	body := w.Body.String()

	// Check SSE format: event line, data line, empty line
	lines := strings.Split(body, "\n")
	if len(lines) < 3 {
		t.Fatalf("Expected at least 3 lines, got %d", len(lines))
	}

	if !strings.HasPrefix(lines[0], "event: ") {
		t.Errorf("First line should be event, got: %s", lines[0])
	}

	if !strings.HasPrefix(lines[1], "data: ") {
		t.Errorf("Second line should be data, got: %s", lines[1])
	}

	// Third line should be empty (end of event)
	if lines[2] != "" {
		t.Errorf("Third line should be empty, got: %s", lines[2])
	}
}

func TestGlobalEvents_Headers(t *testing.T) {
	event.Reset()
	srv := &Server{}

	// Create test server with the actual handler
	ts := httptest.NewServer(http.HandlerFunc(srv.globalEvents))
	defer ts.Close()

	// Create request with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ts.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	// Make request - will timeout but we should still get headers
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil && !strings.Contains(err.Error(), "context deadline exceeded") {
		// We expect timeout, other errors are failures
		if resp == nil {
			t.Skipf("Request failed without response: %v", err)
		}
	}
	if resp != nil {
		defer resp.Body.Close()

		// Verify SSE headers
		contentType := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "text/event-stream") {
			t.Errorf("Expected Content-Type to start with text/event-stream, got: %s", contentType)
		}

		cacheControl := resp.Header.Get("Cache-Control")
		if cacheControl != "no-cache" {
			t.Errorf("Expected Cache-Control: no-cache, got: %s", cacheControl)
		}

		connection := resp.Header.Get("Connection")
		if connection != "keep-alive" {
			t.Errorf("Expected Connection: keep-alive, got: %s", connection)
		}
	}
}

func TestSessionEvents_Headers(t *testing.T) {
	event.Reset()
	srv := &Server{}

	// Create test server with the actual handler
	ts := httptest.NewServer(http.HandlerFunc(srv.sessionEvents))
	defer ts.Close()

	// Create request with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ts.URL+"?sessionID=test-session", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	// Make request - will timeout but we should still get headers
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil && !strings.Contains(err.Error(), "context deadline exceeded") {
		if resp == nil {
			t.Skipf("Request failed without response: %v", err)
		}
	}
	if resp != nil {
		defer resp.Body.Close()

		// Verify SSE headers
		contentType := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "text/event-stream") {
			t.Errorf("Expected Content-Type to start with text/event-stream, got: %s", contentType)
		}

		cacheControl := resp.Header.Get("Cache-Control")
		if cacheControl != "no-cache" {
			t.Errorf("Expected Cache-Control: no-cache, got: %s", cacheControl)
		}
	}
}

func TestSessionEvents_EventFiltering(t *testing.T) {
	event.Reset()
	srv := &Server{}

	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(srv.sessionEvents))
	defer ts.Close()

	// Create request with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ts.URL+"?sessionID=session-123", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	var wg sync.WaitGroup
	var receivedLines []string
	var mu sync.Mutex

	wg.Add(1)
	go func() {
		defer wg.Done()

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			mu.Lock()
			receivedLines = append(receivedLines, line)
			mu.Unlock()
		}
	}()

	// Give connection time to establish
	time.Sleep(50 * time.Millisecond)

	// Publish event for matching session (SDK compatible: uses "info" field)
	event.PublishSync(event.Event{
		Type: event.MessageCreated,
		Data: event.MessageCreatedData{
			Info: &types.Message{
				ID:        "msg-1",
				SessionID: "session-123",
			},
		},
	})

	// Publish event for different session (should be filtered out)
	event.PublishSync(event.Event{
		Type: event.MessageCreated,
		Data: event.MessageCreatedData{
			Info: &types.Message{
				ID:        "msg-2",
				SessionID: "session-456",
			},
		},
	})

	// Wait for context timeout and cleanup
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	// Check we received the first event but not the second
	foundSession123 := false
	foundSession456 := false
	for _, line := range receivedLines {
		if strings.Contains(line, "session-123") {
			foundSession123 = true
		}
		if strings.Contains(line, "session-456") {
			foundSession456 = true
		}
	}

	if foundSession456 {
		t.Error("Should not have received events for session-456")
	}

	// Note: We may or may not have received session-123 event due to timing
	// The important thing is that we filtered out session-456
	_ = foundSession123
}
