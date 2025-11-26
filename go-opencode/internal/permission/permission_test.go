package permission

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/opencode-ai/opencode/internal/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchBashPermission(t *testing.T) {
	permissions := map[string]PermissionAction{
		"git *":         ActionAllow,
		"rm *":          ActionDeny,
		"npm install *": ActionAsk,
		"*":             ActionAsk,
	}

	tests := []struct {
		name     string
		cmd      BashCommand
		expected PermissionAction
	}{
		{
			name:     "git allowed",
			cmd:      BashCommand{Name: "git", Subcommand: "commit"},
			expected: ActionAllow,
		},
		{
			name:     "git push allowed",
			cmd:      BashCommand{Name: "git", Subcommand: "push", Args: []string{"push", "origin", "main"}},
			expected: ActionAllow,
		},
		{
			name:     "rm denied",
			cmd:      BashCommand{Name: "rm", Args: []string{"-rf", "dir"}},
			expected: ActionDeny,
		},
		{
			name:     "npm install ask",
			cmd:      BashCommand{Name: "npm", Subcommand: "install", Args: []string{"install", "express"}},
			expected: ActionAsk,
		},
		{
			name:     "unknown command defaults to global wildcard",
			cmd:      BashCommand{Name: "unknown"},
			expected: ActionAsk,
		},
		{
			name:     "ls defaults to global wildcard",
			cmd:      BashCommand{Name: "ls", Args: []string{"-la"}},
			expected: ActionAsk,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchBashPermission(tt.cmd, permissions)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchBashPermission_SpecificSubcommand(t *testing.T) {
	permissions := map[string]PermissionAction{
		"git commit *": ActionAllow,
		"git push *":   ActionDeny,
		"git *":        ActionAsk,
	}

	tests := []struct {
		name     string
		cmd      BashCommand
		expected PermissionAction
	}{
		{
			name:     "git commit matches specific",
			cmd:      BashCommand{Name: "git", Subcommand: "commit", Args: []string{"commit", "-m", "msg"}},
			expected: ActionAllow,
		},
		{
			name:     "git push matches specific deny",
			cmd:      BashCommand{Name: "git", Subcommand: "push", Args: []string{"push", "origin"}},
			expected: ActionDeny,
		},
		{
			name:     "git status falls back to git *",
			cmd:      BashCommand{Name: "git", Subcommand: "status", Args: []string{"status"}},
			expected: ActionAsk,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchBashPermission(tt.cmd, permissions)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchBashPermission_NoGlobalWildcard(t *testing.T) {
	permissions := map[string]PermissionAction{
		"git *": ActionAllow,
	}

	// Unknown command with no global wildcard should default to ask
	cmd := BashCommand{Name: "unknown"}
	result := MatchBashPermission(cmd, permissions)
	assert.Equal(t, ActionAsk, result)
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		cmd     BashCommand
		matches bool
	}{
		{
			name:    "global wildcard",
			pattern: "*",
			cmd:     BashCommand{Name: "anything"},
			matches: true,
		},
		{
			name:    "command wildcard",
			pattern: "git *",
			cmd:     BashCommand{Name: "git", Subcommand: "commit"},
			matches: true,
		},
		{
			name:    "command wildcard mismatch",
			pattern: "git *",
			cmd:     BashCommand{Name: "npm"},
			matches: false,
		},
		{
			name:    "subcommand wildcard",
			pattern: "git commit *",
			cmd:     BashCommand{Name: "git", Args: []string{"commit", "-m", "msg"}},
			matches: true,
		},
		{
			name:    "subcommand mismatch",
			pattern: "git commit *",
			cmd:     BashCommand{Name: "git", Args: []string{"push"}},
			matches: false,
		},
		{
			name:    "exact command match",
			pattern: "pwd",
			cmd:     BashCommand{Name: "pwd"},
			matches: true,
		},
		{
			name:    "exact command with args mismatch",
			pattern: "pwd",
			cmd:     BashCommand{Name: "pwd", Args: []string{"-L"}},
			matches: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchPattern(tt.pattern, tt.cmd)
			assert.Equal(t, tt.matches, result)
		})
	}
}

func TestBuildPattern(t *testing.T) {
	tests := []struct {
		name     string
		cmd      BashCommand
		expected string
	}{
		{
			name:     "simple command",
			cmd:      BashCommand{Name: "ls", Args: []string{"-la"}},
			expected: "ls *",
		},
		{
			name:     "command with subcommand",
			cmd:      BashCommand{Name: "git", Subcommand: "commit", Args: []string{"commit", "-m", "msg"}},
			expected: "git commit *",
		},
		{
			name:     "npm install",
			cmd:      BashCommand{Name: "npm", Subcommand: "install", Args: []string{"install", "express"}},
			expected: "npm install *",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildPattern(tt.cmd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildPatterns(t *testing.T) {
	commands := []BashCommand{
		{Name: "git", Subcommand: "add", Args: []string{"add", "."}},
		{Name: "git", Subcommand: "commit", Args: []string{"commit", "-m", "msg"}},
		{Name: "cd", Args: []string{"/tmp"}}, // Should be skipped
		{Name: "npm", Subcommand: "install", Args: []string{"install"}},
		{Name: "git", Subcommand: "add", Args: []string{"add", "file.txt"}}, // Duplicate pattern
	}

	patterns := BuildPatterns(commands)

	// Should have 3 unique patterns (cd is skipped, duplicate git add is deduplicated)
	assert.Len(t, patterns, 3)
	assert.Contains(t, patterns, "git add *")
	assert.Contains(t, patterns, "git commit *")
	assert.Contains(t, patterns, "npm install *")
}

func TestDoomLoopDetector(t *testing.T) {
	detector := NewDoomLoopDetector()
	sessionID := "test-session"

	// First call - not a loop
	assert.False(t, detector.Check(sessionID, "read", map[string]string{"file": "test.txt"}))

	// Second identical call - still not a loop
	assert.False(t, detector.Check(sessionID, "read", map[string]string{"file": "test.txt"}))

	// Third identical call - THIS is a doom loop
	assert.True(t, detector.Check(sessionID, "read", map[string]string{"file": "test.txt"}))

	// Fourth call with same input - still a loop
	assert.True(t, detector.Check(sessionID, "read", map[string]string{"file": "test.txt"}))
}

func TestDoomLoopDetector_DifferentInput(t *testing.T) {
	detector := NewDoomLoopDetector()
	sessionID := "test-session"

	// Two identical calls
	assert.False(t, detector.Check(sessionID, "read", map[string]string{"file": "a.txt"}))
	assert.False(t, detector.Check(sessionID, "read", map[string]string{"file": "a.txt"}))

	// Different input breaks the pattern
	assert.False(t, detector.Check(sessionID, "read", map[string]string{"file": "b.txt"}))

	// New sequence starts
	assert.False(t, detector.Check(sessionID, "read", map[string]string{"file": "c.txt"}))
	assert.False(t, detector.Check(sessionID, "read", map[string]string{"file": "c.txt"}))
	assert.True(t, detector.Check(sessionID, "read", map[string]string{"file": "c.txt"}))
}

func TestDoomLoopDetector_DifferentTool(t *testing.T) {
	detector := NewDoomLoopDetector()
	sessionID := "test-session"

	// Two read calls
	assert.False(t, detector.Check(sessionID, "read", map[string]string{"file": "test.txt"}))
	assert.False(t, detector.Check(sessionID, "read", map[string]string{"file": "test.txt"}))

	// Different tool breaks pattern
	assert.False(t, detector.Check(sessionID, "write", map[string]string{"file": "test.txt"}))

	// Can still detect loops for new pattern
	assert.False(t, detector.Check(sessionID, "read", map[string]string{"file": "test.txt"}))
	assert.False(t, detector.Check(sessionID, "read", map[string]string{"file": "test.txt"}))
	assert.True(t, detector.Check(sessionID, "read", map[string]string{"file": "test.txt"}))
}

func TestDoomLoopDetector_DifferentSessions(t *testing.T) {
	detector := NewDoomLoopDetector()

	// Session 1
	assert.False(t, detector.Check("session1", "read", map[string]string{"file": "test.txt"}))
	assert.False(t, detector.Check("session1", "read", map[string]string{"file": "test.txt"}))

	// Session 2 starts fresh
	assert.False(t, detector.Check("session2", "read", map[string]string{"file": "test.txt"}))
	assert.False(t, detector.Check("session2", "read", map[string]string{"file": "test.txt"}))

	// Session 1 continues
	assert.True(t, detector.Check("session1", "read", map[string]string{"file": "test.txt"}))

	// Session 2 also loops
	assert.True(t, detector.Check("session2", "read", map[string]string{"file": "test.txt"}))
}

func TestDoomLoopDetector_Clear(t *testing.T) {
	detector := NewDoomLoopDetector()
	sessionID := "test-session"

	assert.False(t, detector.Check(sessionID, "read", map[string]string{"file": "test.txt"}))
	assert.False(t, detector.Check(sessionID, "read", map[string]string{"file": "test.txt"}))

	// Clear resets the session
	detector.Clear(sessionID)

	// Starts fresh
	assert.False(t, detector.Check(sessionID, "read", map[string]string{"file": "test.txt"}))
	assert.False(t, detector.Check(sessionID, "read", map[string]string{"file": "test.txt"}))
	assert.True(t, detector.Check(sessionID, "read", map[string]string{"file": "test.txt"}))
}

func TestChecker_Check(t *testing.T) {
	checker := NewChecker()
	ctx := context.Background()

	// Allow action should pass immediately
	err := checker.Check(ctx, Request{SessionID: "test"}, ActionAllow)
	assert.NoError(t, err)

	// Deny action should return RejectedError
	err = checker.Check(ctx, Request{SessionID: "test", Type: PermBash}, ActionDeny)
	assert.Error(t, err)
	assert.True(t, IsRejectedError(err))
}

func TestChecker_AlreadyApproved(t *testing.T) {
	// Reset event bus for clean test
	event.Reset()

	checker := NewChecker()
	ctx := context.Background()
	sessionID := "test-session"

	// Manually approve a permission
	checker.approve(sessionID, PermBash, nil)

	// Ask should return immediately for approved permission
	done := make(chan error)
	go func() {
		done <- checker.Ask(ctx, Request{
			SessionID: sessionID,
			Type:      PermBash,
		})
	}()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Ask should return immediately for approved permission")
	}
}

func TestChecker_PatternApproved(t *testing.T) {
	// Reset event bus for clean test
	event.Reset()

	checker := NewChecker()
	ctx := context.Background()
	sessionID := "test-session"

	// Approve specific patterns
	checker.ApprovePattern(sessionID, "git *")
	checker.ApprovePattern(sessionID, "npm install *")

	// Ask with approved patterns should return immediately
	done := make(chan error)
	go func() {
		done <- checker.Ask(ctx, Request{
			SessionID: sessionID,
			Type:      PermBash,
			Pattern:   []string{"git *"},
		})
	}()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Ask should return immediately for approved pattern")
	}
}

func TestChecker_AskAndRespond(t *testing.T) {
	// Reset event bus for clean test
	event.Reset()

	checker := NewChecker()
	ctx := context.Background()
	sessionID := "test-session"

	var receivedEvent event.Event
	var wg sync.WaitGroup
	wg.Add(1)

	// Subscribe to permission events
	unsub := event.Subscribe(event.PermissionRequired, func(e event.Event) {
		receivedEvent = e
		wg.Done()
	})
	defer unsub()

	// Start Ask in background
	errChan := make(chan error)
	go func() {
		errChan <- checker.Ask(ctx, Request{
			ID:        "test-request-id",
			SessionID: sessionID,
			Type:      PermBash,
			Title:     "git commit -m 'test'",
			Pattern:   []string{"git *"},
		})
	}()

	// Wait for event
	wg.Wait()

	// Verify event was published
	data, ok := receivedEvent.Data.(event.PermissionRequiredData)
	require.True(t, ok)
	assert.Equal(t, "test-request-id", data.ID)
	assert.Equal(t, sessionID, data.SessionID)
	assert.Equal(t, "bash", data.PermissionType)

	// Respond with "once"
	checker.Respond("test-request-id", "once")

	// Check result
	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Ask should complete after Respond")
	}
}

func TestChecker_AskAndReject(t *testing.T) {
	// Reset event bus for clean test
	event.Reset()

	checker := NewChecker()
	ctx := context.Background()
	sessionID := "test-session"

	var wg sync.WaitGroup
	wg.Add(1)

	// Subscribe to permission events
	unsub := event.Subscribe(event.PermissionRequired, func(e event.Event) {
		wg.Done()
	})
	defer unsub()

	// Start Ask in background
	errChan := make(chan error)
	go func() {
		errChan <- checker.Ask(ctx, Request{
			ID:        "reject-request-id",
			SessionID: sessionID,
			Type:      PermBash,
			Title:     "rm -rf /",
		})
	}()

	// Wait for event
	wg.Wait()

	// Respond with reject
	checker.Respond("reject-request-id", "reject")

	// Check result
	select {
	case err := <-errChan:
		assert.Error(t, err)
		assert.True(t, IsRejectedError(err))
	case <-time.After(time.Second):
		t.Fatal("Ask should complete after Respond")
	}
}

func TestChecker_AskContextCanceled(t *testing.T) {
	// Reset event bus for clean test
	event.Reset()

	checker := NewChecker()
	ctx, cancel := context.WithCancel(context.Background())
	sessionID := "test-session"

	// Start Ask in background
	errChan := make(chan error)
	go func() {
		errChan <- checker.Ask(ctx, Request{
			SessionID: sessionID,
			Type:      PermBash,
		})
	}()

	// Cancel context
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Check result
	select {
	case err := <-errChan:
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	case <-time.After(time.Second):
		t.Fatal("Ask should complete when context is canceled")
	}
}

func TestChecker_ClearSession(t *testing.T) {
	checker := NewChecker()
	sessionID := "test-session"

	// Add some approvals
	checker.approve(sessionID, PermBash, []string{"git *"})
	checker.ApprovePattern(sessionID, "npm *")

	assert.True(t, checker.IsApproved(sessionID, PermBash))
	assert.True(t, checker.IsPatternApproved(sessionID, "npm *"))

	// Clear session
	checker.ClearSession(sessionID)

	// Should no longer be approved
	assert.False(t, checker.IsApproved(sessionID, PermBash))
	assert.False(t, checker.IsPatternApproved(sessionID, "npm *"))
}

func TestRejectedError(t *testing.T) {
	err := &RejectedError{
		SessionID: "test-session",
		Type:      PermBash,
		CallID:    "call-123",
		Message:   "Permission denied",
		Metadata:  map[string]any{"command": "rm -rf /"},
	}

	assert.Equal(t, "Permission denied", err.Error())
	assert.True(t, IsRejectedError(err))
	assert.False(t, IsRejectedError(context.Canceled))
}

func TestDefaultAgentPermissions(t *testing.T) {
	perms := DefaultAgentPermissions()

	assert.Equal(t, ActionAsk, perms.Edit)
	assert.Equal(t, ActionAsk, perms.WebFetch)
	assert.Equal(t, ActionAsk, perms.ExternalDir)
	assert.Equal(t, ActionAsk, perms.DoomLoop)
	assert.NotNil(t, perms.Bash)
}
