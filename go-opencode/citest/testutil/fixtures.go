package testutil

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// RandomString generates a random string of n characters
func RandomString(n int) string {
	bytes := make([]byte, n/2+1)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:n]
}

// TempFile creates a temporary file with content
type TempFile struct {
	Path string
}

// NewTempFile creates a temp file with content
func NewTempFile(content string) (*TempFile, error) {
	dir := os.TempDir()
	name := fmt.Sprintf("opencode-test-%s.txt", RandomString(8))
	path := filepath.Join(dir, name)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, err
	}

	return &TempFile{Path: path}, nil
}

// NewTempFileInDir creates a temp file in specific directory
func NewTempFileInDir(dir, content string) (*TempFile, error) {
	name := fmt.Sprintf("test-%s.txt", RandomString(8))
	path := filepath.Join(dir, name)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, err
	}

	return &TempFile{Path: path}, nil
}

// Read reads the file content
func (f *TempFile) Read() (string, error) {
	content, err := os.ReadFile(f.Path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// Exists checks if the file exists
func (f *TempFile) Exists() bool {
	_, err := os.Stat(f.Path)
	return err == nil
}

// Cleanup removes the temp file
func (f *TempFile) Cleanup() {
	os.Remove(f.Path)
}

// TempDir creates a temporary directory
type TempDir struct {
	Path string
}

// NewTempDir creates a temp directory
func NewTempDir() (*TempDir, error) {
	path, err := os.MkdirTemp("", "opencode-test-*")
	if err != nil {
		return nil, err
	}
	return &TempDir{Path: path}, nil
}

// CreateFile creates a file in the temp directory
func (d *TempDir) CreateFile(name, content string) (*TempFile, error) {
	path := filepath.Join(d.Path, name)

	// Create parent directories if needed
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, err
	}

	return &TempFile{Path: path}, nil
}

// CreateSubDir creates a subdirectory
func (d *TempDir) CreateSubDir(name string) (string, error) {
	path := filepath.Join(d.Path, name)
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", err
	}
	return path, nil
}

// Cleanup removes the temp directory and all contents
func (d *TempDir) Cleanup() {
	os.RemoveAll(d.Path)
}

// ---- Test Session Manager ----

// SessionManager manages test sessions for cleanup
type SessionManager struct {
	client   *TestClient
	sessions []string
}

// NewSessionManager creates a session manager
func NewSessionManager(client *TestClient) *SessionManager {
	return &SessionManager{
		client:   client,
		sessions: make([]string, 0),
	}
}

// Create creates a session and tracks it for cleanup
func (m *SessionManager) Create(dir string) (*Session, error) {
	session, err := m.client.CreateSession(nil, dir)
	if err != nil {
		return nil, err
	}
	m.sessions = append(m.sessions, session.ID)
	return session, nil
}

// Cleanup deletes all tracked sessions
func (m *SessionManager) Cleanup() {
	for _, id := range m.sessions {
		m.client.DeleteSession(nil, id)
	}
	m.sessions = m.sessions[:0]
}

// ---- Assertion Matchers ----

// EventMatcher helps match SSE events
type EventMatcher struct {
	events []SSEEvent
}

// NewEventMatcher creates an event matcher
func NewEventMatcher(events []SSEEvent) *EventMatcher {
	return &EventMatcher{events: events}
}

// HasType checks if any event has the given type
func (m *EventMatcher) HasType(eventType string) bool {
	for _, evt := range m.events {
		if evt.Type == eventType {
			return true
		}
	}
	return false
}

// CountType counts events of given type
func (m *EventMatcher) CountType(eventType string) int {
	count := 0
	for _, evt := range m.events {
		if evt.Type == eventType {
			count++
		}
	}
	return count
}

// FilterType returns events of given type
func (m *EventMatcher) FilterType(eventType string) []SSEEvent {
	var filtered []SSEEvent
	for _, evt := range m.events {
		if evt.Type == eventType {
			filtered = append(filtered, evt)
		}
	}
	return filtered
}

// ---- Environment Helpers ----

// RequireEnv checks if required env vars are set
func RequireEnv(vars ...string) error {
	var missing []string
	for _, v := range vars {
		if os.Getenv(v) == "" {
			missing = append(missing, v)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missing)
	}
	return nil
}

// SkipIfMissingEnv returns true if any env var is missing
func SkipIfMissingEnv(vars ...string) bool {
	return RequireEnv(vars...) != nil
}
