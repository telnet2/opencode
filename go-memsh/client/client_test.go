package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient(ClientOptions{
		BaseURL: "http://localhost:8080",
	})

	if client == nil {
		t.Fatal("expected client to be created")
	}

	if client.State() != StateDisconnected {
		t.Errorf("expected state to be disconnected, got %s", client.State())
	}
}

func TestClientDefaultOptions(t *testing.T) {
	client := NewClient(ClientOptions{
		BaseURL: "http://localhost:8080",
	})

	if client.options.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", client.options.Timeout)
	}

	if client.options.MaxReconnectAttempts != 5 {
		t.Errorf("expected default max reconnect attempts 5, got %d", client.options.MaxReconnectAttempts)
	}

	if client.options.ReconnectDelay != time.Second {
		t.Errorf("expected default reconnect delay 1s, got %v", client.options.ReconnectDelay)
	}
}

func TestClientCustomOptions(t *testing.T) {
	client := NewClient(ClientOptions{
		BaseURL:              "http://localhost:8080",
		Timeout:              60 * time.Second,
		AutoReconnect:        true,
		MaxReconnectAttempts: 10,
		ReconnectDelay:       2 * time.Second,
	})

	if client.options.Timeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", client.options.Timeout)
	}

	if !client.options.AutoReconnect {
		t.Error("expected auto reconnect to be true")
	}

	if client.options.MaxReconnectAttempts != 10 {
		t.Errorf("expected max reconnect attempts 10, got %d", client.options.MaxReconnectAttempts)
	}

	if client.options.ReconnectDelay != 2*time.Second {
		t.Errorf("expected reconnect delay 2s, got %v", client.options.ReconnectDelay)
	}
}

func TestCreateSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/session/create" {
			t.Errorf("expected path /api/v1/session/create, got %s", r.URL.Path)
		}

		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		response := CreateSessionResponse{
			Session: SessionInfo{
				ID:        "test-session-id",
				CreatedAt: time.Now(),
				LastUsed:  time.Now(),
				Cwd:       "/",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(ClientOptions{BaseURL: server.URL})

	session, err := client.CreateSession()
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if session.ID != "test-session-id" {
		t.Errorf("expected session ID 'test-session-id', got '%s'", session.ID)
	}

	if session.Cwd != "/" {
		t.Errorf("expected cwd '/', got '%s'", session.Cwd)
	}
}

func TestListSessions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/session/list" {
			t.Errorf("expected path /api/v1/session/list, got %s", r.URL.Path)
		}

		response := ListSessionsResponse{
			Sessions: []SessionInfo{
				{
					ID:        "session-1",
					CreatedAt: time.Now(),
					LastUsed:  time.Now(),
					Cwd:       "/home",
				},
				{
					ID:        "session-2",
					CreatedAt: time.Now(),
					LastUsed:  time.Now(),
					Cwd:       "/tmp",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(ClientOptions{BaseURL: server.URL})

	sessions, err := client.ListSessions()
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}

	if sessions[0].ID != "session-1" {
		t.Errorf("expected first session ID 'session-1', got '%s'", sessions[0].ID)
	}
}

func TestRemoveSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/session/remove" {
			t.Errorf("expected path /api/v1/session/remove, got %s", r.URL.Path)
		}

		var req RemoveSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		if req.SessionID != "session-to-remove" {
			t.Errorf("expected session ID 'session-to-remove', got '%s'", req.SessionID)
		}

		response := RemoveSessionResponse{
			Success: true,
			Message: "Session removed successfully",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(ClientOptions{BaseURL: server.URL})

	err := client.RemoveSession("session-to-remove")
	if err != nil {
		t.Fatalf("failed to remove session: %v", err)
	}
}

func TestCreateSessionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(ClientOptions{BaseURL: server.URL})

	_, err := client.CreateSession()
	if err == nil {
		t.Error("expected error for failed request")
	}
}

func TestWsURL(t *testing.T) {
	tests := []struct {
		baseURL  string
		expected string
	}{
		{"http://localhost:8080", "ws://localhost:8080/api/v1/session/repl"},
		{"https://example.com", "wss://example.com/api/v1/session/repl"},
		{"http://192.168.1.1:9000", "ws://192.168.1.1:9000/api/v1/session/repl"},
	}

	for _, test := range tests {
		client := NewClient(ClientOptions{BaseURL: test.baseURL})
		wsURL, err := client.wsURL()
		if err != nil {
			t.Errorf("failed to get wsURL for %s: %v", test.baseURL, err)
			continue
		}

		if wsURL != test.expected {
			t.Errorf("expected %s, got %s", test.expected, wsURL)
		}
	}
}
