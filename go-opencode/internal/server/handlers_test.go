package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/opencode-ai/opencode/internal/session"
	"github.com/opencode-ai/opencode/internal/storage"
	"github.com/opencode-ai/opencode/pkg/types"
)

func setupTestServer(t *testing.T) *Server {
	tmpDir := t.TempDir()
	store := storage.New(tmpDir)
	sessionSvc := session.NewService(store)

	srv := &Server{
		sessionService: sessionSvc,
		storage:        store,
		appConfig:      &types.Config{},
	}
	return srv
}

func TestListSessions_Empty(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest("GET", "/session", nil)
	w := httptest.NewRecorder()

	srv.listSessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var sessions []types.Session
	if err := json.NewDecoder(w.Body).Decode(&sessions); err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("Expected empty list, got %d sessions", len(sessions))
	}
}

func TestCreateSession(t *testing.T) {
	srv := setupTestServer(t)

	body := CreateSessionRequest{Directory: "/tmp/test"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/session", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.createSession(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var session types.Session
	if err := json.NewDecoder(w.Body).Decode(&session); err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if session.ID == "" {
		t.Error("Session ID should not be empty")
	}
	if session.Directory != "/tmp/test" {
		t.Errorf("Directory mismatch: got %s", session.Directory)
	}
}

func TestCreateSession_InvalidJSON(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest("POST", "/session", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.createSession(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestGetSession(t *testing.T) {
	srv := setupTestServer(t)
	ctx := context.Background()

	// Create a session first
	session, err := srv.sessionService.Create(ctx, "/tmp/test")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Set up chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionID", session.ID)

	req := httptest.NewRequest("GET", "/session/"+session.ID, nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	srv.getSession(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var retrieved types.Session
	if err := json.NewDecoder(w.Body).Decode(&retrieved); err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("Session ID mismatch: got %s, want %s", retrieved.ID, session.ID)
	}
}

func TestGetSession_NotFound(t *testing.T) {
	srv := setupTestServer(t)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionID", "nonexistent")

	req := httptest.NewRequest("GET", "/session/nonexistent", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	srv.getSession(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

func TestDeleteSession(t *testing.T) {
	srv := setupTestServer(t)
	ctx := context.Background()

	// Create a session first
	session, err := srv.sessionService.Create(ctx, "/tmp/test")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionID", session.ID)

	req := httptest.NewRequest("DELETE", "/session/"+session.ID, nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	srv.deleteSession(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deleted
	_, err = srv.sessionService.Get(ctx, session.ID)
	if err == nil {
		t.Error("Session should be deleted")
	}
}

func TestUpdateSession(t *testing.T) {
	srv := setupTestServer(t)
	ctx := context.Background()

	// Create a session first
	session, err := srv.sessionService.Create(ctx, "/tmp/test")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionID", session.ID)

	updates := map[string]any{
		"title": "Updated Title",
	}
	jsonBody, _ := json.Marshal(updates)

	req := httptest.NewRequest("PATCH", "/session/"+session.ID, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	srv.updateSession(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated types.Session
	if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if updated.Title != "Updated Title" {
		t.Errorf("Title not updated: got %s", updated.Title)
	}
}

func TestGetConfig(t *testing.T) {
	srv := setupTestServer(t)
	srv.appConfig = &types.Config{
		Model: "anthropic/claude-3-opus",
	}

	req := httptest.NewRequest("GET", "/config", nil)
	w := httptest.NewRecorder()

	srv.getConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var config types.Config
	if err := json.NewDecoder(w.Body).Decode(&config); err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if config.Model != "anthropic/claude-3-opus" {
		t.Errorf("Model mismatch: got %s", config.Model)
	}
}

func TestReadFile_NotFound(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest("GET", "/file?path=/nonexistent/file.txt", nil)
	w := httptest.NewRecorder()

	srv.readFile(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

func TestReadFile_MissingPath(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest("GET", "/file", nil)
	w := httptest.NewRecorder()

	srv.readFile(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}
