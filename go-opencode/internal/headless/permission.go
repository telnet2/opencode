package headless

import (
	"context"

	"github.com/opencode-ai/opencode/internal/event"
	"github.com/opencode-ai/opencode/internal/permission"
)

// AutoApproveChecker is a permission checker that automatically approves all requests.
// This is used in headless mode when --auto-approve is enabled.
type AutoApproveChecker struct {
	// inner is the underlying checker (used for tracking/logging only)
	inner *permission.Checker
	// verbose enables logging of auto-approved permissions
	verbose bool
}

// NewAutoApproveChecker creates a new auto-approve permission checker.
func NewAutoApproveChecker(verbose bool) *AutoApproveChecker {
	return &AutoApproveChecker{
		inner:   permission.NewChecker(),
		verbose: verbose,
	}
}

// Check always approves the permission request.
// It publishes events for tracking but never blocks.
func (c *AutoApproveChecker) Check(ctx context.Context, req permission.Request, action permission.PermissionAction) error {
	// In auto-approve mode, we always approve regardless of the action
	// But we still publish events for tracking/logging purposes

	if c.verbose {
		// Publish that we received a permission request
		event.Publish(event.Event{
			Type: event.PermissionUpdated,
			Data: event.PermissionUpdatedData{
				ID:             req.ID,
				SessionID:      req.SessionID,
				PermissionType: string(req.Type),
				Pattern:        req.Pattern,
				Title:          req.Title,
			},
		})

		// Immediately publish that we auto-approved it
		event.Publish(event.Event{
			Type: event.PermissionReplied,
			Data: event.PermissionRepliedData{
				PermissionID: req.ID,
				SessionID:    req.SessionID,
				Response:     "always", // Auto-approve with "always" action
			},
		})
	}

	// Always return nil (approved)
	return nil
}

// Ask is called when explicit user approval would normally be required.
// In auto-approve mode, it always returns nil (approved).
func (c *AutoApproveChecker) Ask(ctx context.Context, req permission.Request) error {
	return c.Check(ctx, req, permission.ActionAsk)
}

// Respond handles a user's response to a permission request.
// This is a no-op in auto-approve mode since we don't wait for responses.
func (c *AutoApproveChecker) Respond(requestID string, action string) {
	// No-op in auto-approve mode
}

// IsApproved always returns true in auto-approve mode.
func (c *AutoApproveChecker) IsApproved(sessionID string, permType permission.PermissionType) bool {
	return true
}

// IsPatternApproved always returns true in auto-approve mode.
func (c *AutoApproveChecker) IsPatternApproved(sessionID string, pattern string) bool {
	return true
}

// ClearSession is a no-op in auto-approve mode.
func (c *AutoApproveChecker) ClearSession(sessionID string) {
	// No-op
}

// ApprovePattern is a no-op in auto-approve mode (everything is already approved).
func (c *AutoApproveChecker) ApprovePattern(sessionID string, pattern string) {
	// No-op
}

// PermissionCheckerInterface defines the interface for permission checking.
// This allows us to swap between regular and auto-approve checkers.
type PermissionCheckerInterface interface {
	Check(ctx context.Context, req permission.Request, action permission.PermissionAction) error
	Ask(ctx context.Context, req permission.Request) error
	Respond(requestID string, action string)
	IsApproved(sessionID string, permType permission.PermissionType) bool
	IsPatternApproved(sessionID string, pattern string) bool
	ClearSession(sessionID string)
	ApprovePattern(sessionID string, pattern string)
}

// Ensure both checkers implement the interface
var _ PermissionCheckerInterface = (*AutoApproveChecker)(nil)
