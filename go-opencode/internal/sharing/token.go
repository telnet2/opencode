// Package sharing provides session sharing functionality.
package sharing

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// ShareInfo represents sharing metadata for a session.
type ShareInfo struct {
	Token     string    `json:"token"`
	SessionID string    `json:"sessionID"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
	Views     int       `json:"views"`
	MaxViews  int       `json:"maxViews,omitempty"` // 0 = unlimited
	Public    bool      `json:"public"`
}

// Manager manages session sharing.
type Manager struct {
	mu     sync.RWMutex
	shares map[string]*ShareInfo // token -> share info
	bySession map[string]string  // sessionID -> token
	baseURL string
}

// NewManager creates a new sharing manager.
func NewManager(baseURL string) *Manager {
	if baseURL == "" {
		baseURL = "https://opencode.ai/share"
	}
	return &Manager{
		shares:    make(map[string]*ShareInfo),
		bySession: make(map[string]string),
		baseURL:   baseURL,
	}
}

// Share creates or updates a share for a session.
func (m *Manager) Share(sessionID string, opts *ShareOptions) (*ShareInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already shared
	if token, exists := m.bySession[sessionID]; exists {
		if info, ok := m.shares[token]; ok {
			// Update existing share
			if opts != nil {
				if opts.ExpiresIn > 0 {
					info.ExpiresAt = time.Now().Add(opts.ExpiresIn)
				}
				if opts.MaxViews > 0 {
					info.MaxViews = opts.MaxViews
				}
				info.Public = opts.Public
			}
			return info, nil
		}
	}

	// Generate new token
	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	info := &ShareInfo{
		Token:     token,
		SessionID: sessionID,
		URL:       fmt.Sprintf("%s/%s", m.baseURL, token),
		CreatedAt: time.Now(),
		Public:    true,
	}

	if opts != nil {
		if opts.ExpiresIn > 0 {
			info.ExpiresAt = time.Now().Add(opts.ExpiresIn)
		}
		info.MaxViews = opts.MaxViews
		info.Public = opts.Public
	}

	m.shares[token] = info
	m.bySession[sessionID] = token

	return info, nil
}

// ShareOptions configures sharing behavior.
type ShareOptions struct {
	ExpiresIn time.Duration
	MaxViews  int
	Public    bool
}

// Unshare removes sharing from a session.
func (m *Manager) Unshare(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	token, exists := m.bySession[sessionID]
	if !exists {
		return fmt.Errorf("session not shared")
	}

	delete(m.shares, token)
	delete(m.bySession, sessionID)

	return nil
}

// GetByToken retrieves share info by token.
func (m *Manager) GetByToken(token string) (*ShareInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.shares[token]
	if !ok {
		return nil, fmt.Errorf("share not found")
	}

	// Check expiration
	if !info.ExpiresAt.IsZero() && time.Now().After(info.ExpiresAt) {
		return nil, fmt.Errorf("share expired")
	}

	// Check view limit
	if info.MaxViews > 0 && info.Views >= info.MaxViews {
		return nil, fmt.Errorf("share view limit exceeded")
	}

	return info, nil
}

// GetBySession retrieves share info by session ID.
func (m *Manager) GetBySession(sessionID string) (*ShareInfo, error) {
	m.mu.RLock()
	token, exists := m.bySession[sessionID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not shared")
	}

	return m.GetByToken(token)
}

// RecordView increments the view count.
func (m *Manager) RecordView(token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.shares[token]
	if !ok {
		return fmt.Errorf("share not found")
	}

	info.Views++
	return nil
}

// IsShared checks if a session is shared.
func (m *Manager) IsShared(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.bySession[sessionID]
	return exists
}

// ListShares returns all active shares.
func (m *Manager) ListShares() []*ShareInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	shares := make([]*ShareInfo, 0, len(m.shares))
	for _, info := range m.shares {
		shares = append(shares, info)
	}
	return shares
}

// CleanExpired removes expired shares.
func (m *Manager) CleanExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	count := 0

	for token, info := range m.shares {
		expired := !info.ExpiresAt.IsZero() && now.After(info.ExpiresAt)
		viewLimitExceeded := info.MaxViews > 0 && info.Views >= info.MaxViews

		if expired || viewLimitExceeded {
			delete(m.shares, token)
			delete(m.bySession, info.SessionID)
			count++
		}
	}

	return count
}

// generateToken generates a secure random token.
func generateToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:22], nil
}

// GenerateShortCode generates a short shareable code.
func GenerateShortCode() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:8], nil
}
