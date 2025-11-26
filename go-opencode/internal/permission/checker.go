package permission

import (
	"context"
	"sync"

	"github.com/oklog/ulid/v2"
	"github.com/opencode-ai/opencode/internal/event"
)

// Checker handles permission checks and approvals.
type Checker struct {
	mu       sync.RWMutex
	approved map[string]map[PermissionType]bool   // sessionID -> type -> approved
	patterns map[string]map[string]bool           // sessionID -> pattern -> approved (for bash patterns)
	pending  map[string]chan Response             // requestID -> response channel
}

// NewChecker creates a new permission checker.
func NewChecker() *Checker {
	return &Checker{
		approved: make(map[string]map[PermissionType]bool),
		patterns: make(map[string]map[string]bool),
		pending:  make(map[string]chan Response),
	}
}

// Check performs a permission check based on action configuration.
func (c *Checker) Check(ctx context.Context, req Request, action PermissionAction) error {
	switch action {
	case ActionAllow:
		return nil
	case ActionDeny:
		return &RejectedError{
			SessionID: req.SessionID,
			Type:      req.Type,
			CallID:    req.CallID,
			Metadata:  req.Metadata,
			Message:   "Permission denied by configuration",
		}
	case ActionAsk:
		return c.Ask(ctx, req)
	}
	return nil
}

// Ask prompts the user for permission.
func (c *Checker) Ask(ctx context.Context, req Request) error {
	// Check if already approved for this session and type
	c.mu.RLock()
	if sessionApprovals, ok := c.approved[req.SessionID]; ok {
		if sessionApprovals[req.Type] {
			c.mu.RUnlock()
			return nil
		}
	}

	// Check if any pattern is approved
	if len(req.Pattern) > 0 {
		if sessionPatterns, ok := c.patterns[req.SessionID]; ok {
			allApproved := true
			for _, p := range req.Pattern {
				if !sessionPatterns[p] {
					allApproved = false
					break
				}
			}
			if allApproved {
				c.mu.RUnlock()
				return nil
			}
		}
	}
	c.mu.RUnlock()

	// Generate request ID if not set
	if req.ID == "" {
		req.ID = ulid.Make().String()
	}

	// Create response channel
	respChan := make(chan Response, 1)
	c.mu.Lock()
	c.pending[req.ID] = respChan
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, req.ID)
		c.mu.Unlock()
	}()

	// Publish permission request event
	event.Publish(event.Event{
		Type: event.PermissionRequired,
		Data: event.PermissionRequiredData{
			ID:             req.ID,
			SessionID:      req.SessionID,
			PermissionType: string(req.Type),
			Pattern:        req.Pattern,
			Title:          req.Title,
		},
	})

	// Wait for response
	select {
	case <-ctx.Done():
		return ctx.Err()
	case resp := <-respChan:
		switch resp.Action {
		case "once":
			return nil
		case "always":
			c.approve(req.SessionID, req.Type, req.Pattern)
			return nil
		case "reject":
			return &RejectedError{
				SessionID: req.SessionID,
				Type:      req.Type,
				CallID:    req.CallID,
				Metadata:  req.Metadata,
				Message:   "Permission rejected by user",
			}
		}
	}
	return nil
}

// Respond handles a user's response to a permission request.
func (c *Checker) Respond(requestID string, action string) {
	c.mu.RLock()
	ch, ok := c.pending[requestID]
	c.mu.RUnlock()

	if ok {
		ch <- Response{
			RequestID: requestID,
			Action:    action,
		}
	}

	// Publish resolved event
	event.Publish(event.Event{
		Type: event.PermissionResolved,
		Data: event.PermissionResolvedData{
			ID:      requestID,
			Granted: action != "reject",
		},
	})
}

// approve marks a permission type and patterns as approved for a session.
func (c *Checker) approve(sessionID string, permType PermissionType, patterns []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Approve the permission type
	if c.approved[sessionID] == nil {
		c.approved[sessionID] = make(map[PermissionType]bool)
	}
	c.approved[sessionID][permType] = true

	// Approve individual patterns
	if len(patterns) > 0 {
		if c.patterns[sessionID] == nil {
			c.patterns[sessionID] = make(map[string]bool)
		}
		for _, p := range patterns {
			c.patterns[sessionID][p] = true
		}
	}
}

// IsApproved checks if a permission type is already approved.
func (c *Checker) IsApproved(sessionID string, permType PermissionType) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if sessionApprovals, ok := c.approved[sessionID]; ok {
		return sessionApprovals[permType]
	}
	return false
}

// IsPatternApproved checks if a specific pattern is approved.
func (c *Checker) IsPatternApproved(sessionID string, pattern string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if sessionPatterns, ok := c.patterns[sessionID]; ok {
		return sessionPatterns[pattern]
	}
	return false
}

// ClearSession clears all approvals for a session.
func (c *Checker) ClearSession(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.approved, sessionID)
	delete(c.patterns, sessionID)
}

// ApprovePattern explicitly approves a pattern for a session.
func (c *Checker) ApprovePattern(sessionID string, pattern string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.patterns[sessionID] == nil {
		c.patterns[sessionID] = make(map[string]bool)
	}
	c.patterns[sessionID][pattern] = true
}
