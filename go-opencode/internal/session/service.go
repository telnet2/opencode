// Package session provides session management functionality.
package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/opencode-ai/opencode/internal/permission"
	"github.com/opencode-ai/opencode/internal/provider"
	"github.com/opencode-ai/opencode/internal/storage"
	"github.com/opencode-ai/opencode/internal/tool"
	"github.com/opencode-ai/opencode/pkg/types"
)

// Service manages session operations.
type Service struct {
	storage *storage.Storage

	// Active session processing
	mu       sync.RWMutex
	active   map[string]*ActiveSession
	abortChs map[string]chan struct{}

	// Processor for agentic loop
	processor *Processor
}

// ActiveSession tracks an active processing session.
type ActiveSession struct {
	SessionID string
	AbortCh   chan struct{}
	StartTime time.Time
}

// NewService creates a new session service.
func NewService(store *storage.Storage) *Service {
	return &Service{
		storage:  store,
		active:   make(map[string]*ActiveSession),
		abortChs: make(map[string]chan struct{}),
	}
}

// NewServiceWithProcessor creates a new session service with processor dependencies.
func NewServiceWithProcessor(
	store *storage.Storage,
	providerReg *provider.Registry,
	toolReg *tool.Registry,
	permChecker *permission.Checker,
	defaultProviderID string,
	defaultModelID string,
) *Service {
	s := &Service{
		storage:  store,
		active:   make(map[string]*ActiveSession),
		abortChs: make(map[string]chan struct{}),
	}
	s.processor = NewProcessor(providerReg, toolReg, store, permChecker, defaultProviderID, defaultModelID)
	return s
}

// GetProcessor returns the session processor.
func (s *Service) GetProcessor() *Processor {
	return s.processor
}

// Create creates a new session.
func (s *Service) Create(ctx context.Context, directory string, title string) (*types.Session, error) {
	now := time.Now().UnixMilli()
	projectID := hashDirectory(directory)

	// Use default title if not provided
	if title == "" {
		title = "New Session"
	}

	session := &types.Session{
		ID:        generateID(),
		ProjectID: projectID,
		Directory: directory,
		Title:     title,
		Version:   "1",
		Summary: types.SessionSummary{
			Additions: 0,
			Deletions: 0,
			Files:     0,
		},
		Time: types.SessionTime{
			Created: now,
			Updated: now,
		},
	}

	if err := s.storage.Put(ctx, []string{"session", projectID, session.ID}, session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

// Get retrieves a session by ID.
func (s *Service) Get(ctx context.Context, sessionID string) (*types.Session, error) {
	// Try to find in any project
	projects, err := s.storage.List(ctx, []string{"session"})
	if err != nil {
		return nil, err
	}

	for _, projectID := range projects {
		var session types.Session
		if err := s.storage.Get(ctx, []string{"session", projectID, sessionID}, &session); err == nil {
			return &session, nil
		}
	}

	return nil, storage.ErrNotFound
}

// Update updates a session with the given updates.
func (s *Service) Update(ctx context.Context, sessionID string, updates map[string]any) (*types.Session, error) {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if title, ok := updates["title"].(string); ok {
		session.Title = title
	}

	session.Time.Updated = time.Now().UnixMilli()

	if err := s.storage.Put(ctx, []string{"session", session.ProjectID, session.ID}, session); err != nil {
		return nil, err
	}

	return session, nil
}

// Delete deletes a session.
func (s *Service) Delete(ctx context.Context, sessionID string) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	// Delete session file
	if err := s.storage.Delete(ctx, []string{"session", session.ProjectID, sessionID}); err != nil {
		return err
	}

	// Delete associated messages
	messages, _ := s.GetMessages(ctx, sessionID)
	for _, msg := range messages {
		s.storage.Delete(ctx, []string{"message", sessionID, msg.ID})
	}

	return nil
}

// List lists sessions for a directory.
// If directory is empty, lists all sessions across all projects.
func (s *Service) List(ctx context.Context, directory string) ([]*types.Session, error) {
	var sessions []*types.Session

	if directory == "" {
		// List ALL sessions across all projects
		projects, err := s.storage.List(ctx, []string{"session"})
		if err != nil {
			return nil, err
		}

		for _, projectID := range projects {
			err := s.storage.Scan(ctx, []string{"session", projectID}, func(key string, data json.RawMessage) error {
				var session types.Session
				if err := json.Unmarshal(data, &session); err != nil {
					return err
				}
				sessions = append(sessions, &session)
				return nil
			})
			if err != nil {
				return nil, err
			}
		}

		return sessions, nil
	}

	// List sessions for a specific directory/project
	projectID := hashDirectory(directory)
	err := s.storage.Scan(ctx, []string{"session", projectID}, func(key string, data json.RawMessage) error {
		var session types.Session
		if err := json.Unmarshal(data, &session); err != nil {
			return err
		}
		sessions = append(sessions, &session)
		return nil
	})

	return sessions, err
}

// GetChildren returns child sessions (forks).
func (s *Service) GetChildren(ctx context.Context, sessionID string) ([]*types.Session, error) {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	all, err := s.List(ctx, session.Directory)
	if err != nil {
		return nil, err
	}

	var children []*types.Session
	for _, sess := range all {
		if sess.ParentID != nil && *sess.ParentID == sessionID {
			children = append(children, sess)
		}
	}

	return children, nil
}

// Fork creates a fork of a session at a specific message.
func (s *Service) Fork(ctx context.Context, sessionID, messageID string) (*types.Session, error) {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Create new session with fork title
	newSession, err := s.Create(ctx, session.Directory, session.Title+" (fork)")
	if err != nil {
		return nil, err
	}

	// Set parent
	newSession.ParentID = &sessionID

	// Copy messages up to the fork point
	messages, err := s.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	for _, msg := range messages {
		// Copy message
		newMsg := *msg
		newMsg.SessionID = newSession.ID
		s.AddMessage(ctx, newSession.ID, &newMsg)

		if msg.ID == messageID {
			break
		}
	}

	// Save updated session
	if err := s.storage.Put(ctx, []string{"session", newSession.ProjectID, newSession.ID}, newSession); err != nil {
		return nil, err
	}

	return newSession, nil
}

// Abort aborts an active session.
func (s *Service) Abort(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ch, ok := s.abortChs[sessionID]; ok {
		close(ch)
		delete(s.abortChs, sessionID)
	}

	return nil
}

// Share shares a session and returns a share URL.
func (s *Service) Share(ctx context.Context, sessionID string) (string, error) {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return "", err
	}

	// Generate a share URL (placeholder)
	shareURL := fmt.Sprintf("https://opencode.ai/share/%s", sessionID)

	session.Share = &types.SessionShare{URL: shareURL}
	session.Time.Updated = time.Now().UnixMilli()

	if err := s.storage.Put(ctx, []string{"session", session.ProjectID, session.ID}, session); err != nil {
		return "", err
	}

	return shareURL, nil
}

// Unshare removes sharing from a session.
func (s *Service) Unshare(ctx context.Context, sessionID string) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	session.Share = nil
	session.Time.Updated = time.Now().UnixMilli()

	return s.storage.Put(ctx, []string{"session", session.ProjectID, session.ID}, session)
}

// Summarize generates a summary of the session.
func (s *Service) Summarize(ctx context.Context, sessionID string) (*types.SessionSummary, error) {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	return &session.Summary, nil
}

// GetDiffs returns diffs for a session.
func (s *Service) GetDiffs(ctx context.Context, sessionID string) ([]types.FileDiff, error) {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	return session.Summary.Diffs, nil
}

// GetTodos returns todos for a session.
func (s *Service) GetTodos(ctx context.Context, sessionID string) ([]map[string]any, error) {
	// TODO: Implement todo tracking
	return []map[string]any{}, nil
}

// Revert reverts a session to a specific message.
func (s *Service) Revert(ctx context.Context, sessionID, messageID string, partID *string) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	session.Revert = &types.SessionRevert{
		MessageID: messageID,
		PartID:    partID,
	}
	session.Time.Updated = time.Now().UnixMilli()

	return s.storage.Put(ctx, []string{"session", session.ProjectID, session.ID}, session)
}

// Unrevert removes the revert state from a session.
func (s *Service) Unrevert(ctx context.Context, sessionID string) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	session.Revert = nil
	session.Time.Updated = time.Now().UnixMilli()

	return s.storage.Put(ctx, []string{"session", session.ProjectID, session.ID}, session)
}

// ExecuteCommand executes a slash command.
func (s *Service) ExecuteCommand(ctx context.Context, sessionID, command string) (map[string]any, error) {
	// TODO: Implement command execution
	return map[string]any{"result": "command executed"}, nil
}

// RunShell runs a shell command in the session context.
func (s *Service) RunShell(ctx context.Context, sessionID, command string, timeout int) (map[string]any, error) {
	// TODO: Implement shell execution
	return map[string]any{"output": ""}, nil
}

// RespondPermission responds to a permission request.
func (s *Service) RespondPermission(ctx context.Context, sessionID, permissionID string, granted bool) error {
	// TODO: Implement permission response handling
	return nil
}

// AddMessage adds a message to a session.
func (s *Service) AddMessage(ctx context.Context, sessionID string, msg *types.Message) error {
	return s.storage.Put(ctx, []string{"message", sessionID, msg.ID}, msg)
}

// GetMessages returns all messages for a session.
func (s *Service) GetMessages(ctx context.Context, sessionID string) ([]*types.Message, error) {
	var messages []*types.Message
	err := s.storage.Scan(ctx, []string{"message", sessionID}, func(key string, data json.RawMessage) error {
		var msg types.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			return err
		}
		messages = append(messages, &msg)
		return nil
	})
	return messages, err
}

// GetParts returns all parts for a message.
func (s *Service) GetParts(ctx context.Context, messageID string) ([]types.Part, error) {
	var parts []types.Part
	err := s.storage.Scan(ctx, []string{"part", messageID}, func(key string, data json.RawMessage) error {
		part, err := types.UnmarshalPart(data)
		if err != nil {
			return err
		}
		parts = append(parts, part)
		return nil
	})
	return parts, err
}

// ProcessMessage processes a user message and generates an assistant response.
// This is the main agentic loop.
func (s *Service) ProcessMessage(
	ctx context.Context,
	session *types.Session,
	content string,
	model *types.ModelRef,
	onUpdate func(msg *types.Message, parts []types.Part),
) (*types.Message, []types.Part, error) {
	// First, save the user message
	userMsg := &types.Message{
		ID:        generateID(),
		SessionID: session.ID,
		Role:      "user",
		Time: types.MessageTime{
			Created: time.Now().UnixMilli(),
		},
	}
	if model != nil {
		userMsg.Model = model
	}

	if err := s.AddMessage(ctx, session.ID, userMsg); err != nil {
		return nil, nil, err
	}

	// Save user's text content as a part
	userPart := &types.TextPart{
		ID:   generateID(),
		Type: "text",
		Text: content,
	}
	if err := s.storage.Put(ctx, []string{"part", userMsg.ID, userPart.ID}, userPart); err != nil {
		return nil, nil, err
	}

	// Use processor if available
	if s.processor != nil {
		var finalMsg *types.Message
		var finalParts []types.Part

		err := s.processor.Process(ctx, session.ID, DefaultAgent(), func(msg *types.Message, parts []types.Part) {
			finalMsg = msg
			finalParts = parts
			if onUpdate != nil {
				onUpdate(msg, parts)
			}
		})

		if err != nil {
			return finalMsg, finalParts, err
		}

		return finalMsg, finalParts, nil
	}

	// Fallback: Create placeholder assistant message if no processor
	assistantMsg := &types.Message{
		ID:        generateID(),
		SessionID: session.ID,
		Role:      "assistant",
		Time: types.MessageTime{
			Created: time.Now().UnixMilli(),
		},
	}

	if model != nil {
		assistantMsg.ProviderID = model.ProviderID
		assistantMsg.ModelID = model.ModelID
	}

	parts := []types.Part{
		&types.TextPart{
			ID:   generateID(),
			Type: "text",
			Text: "Processor not initialized. Please configure providers.",
		},
	}

	// Save message
	if err := s.AddMessage(ctx, session.ID, assistantMsg); err != nil {
		return nil, nil, err
	}

	// Notify of update
	if onUpdate != nil {
		onUpdate(assistantMsg, parts)
	}

	return assistantMsg, parts, nil
}

// generateID generates a new ULID.
func generateID() string {
	return ulid.Make().String()
}

// hashDirectory creates a project ID from a directory path.
func hashDirectory(directory string) string {
	h := sha256.New()
	h.Write([]byte(directory))
	return hex.EncodeToString(h.Sum(nil))[:16]
}
