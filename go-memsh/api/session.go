package api

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/afero"
	"github.com/telnet2/go-practice/go-memsh"
)

// Session represents a shell session
type Session struct {
	ID        string
	Shell     *memsh.Shell
	CreatedAt time.Time
	LastUsed  time.Time
	mu        sync.Mutex
}

// SessionManager manages multiple shell sessions
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// CreateSession creates a new shell session
func (sm *SessionManager) CreateSession() (*Session, error) {
	sessionID := uuid.New().String()

	// Create new filesystem for this session
	fs := afero.NewMemMapFs()

	// Create new shell
	shell, err := memsh.NewShell(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to create shell: %w", err)
	}

	session := &Session{
		ID:        sessionID,
		Shell:     shell,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	sm.mu.Lock()
	sm.sessions[sessionID] = session
	sm.mu.Unlock()

	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	session.mu.Lock()
	session.LastUsed = time.Now()
	session.mu.Unlock()

	return session, nil
}

// ListSessions returns all active sessions
func (sm *SessionManager) ListSessions() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*Session, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// RemoveSession removes a session by ID
func (sm *SessionManager) RemoveSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.sessions[sessionID]; !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	delete(sm.sessions, sessionID)
	return nil
}

// ExecuteCommand executes a command in the session and returns output
func (s *Session) ExecuteCommand(ctx context.Context, command string, args []string) ([]string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastUsed = time.Now()

	// Build command string
	cmdStr := command
	for _, arg := range args {
		cmdStr += " " + arg
	}

	// Capture output
	var stdout, stderr strings.Builder
	s.Shell.SetIO(strings.NewReader(""), &stdout, &stderr)

	// Execute command
	err := s.Shell.Run(ctx, cmdStr)

	// Get current working directory
	cwd := s.Shell.GetCwd()

	// Split output into lines
	outputLines := []string{}
	if stdout.Len() > 0 {
		outputLines = strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	}

	// Combine stderr if there's an error
	if err != nil && stderr.Len() > 0 {
		outputLines = append(outputLines, strings.Split(strings.TrimRight(stderr.String(), "\n"), "\n")...)
	}

	return outputLines, cwd, err
}
