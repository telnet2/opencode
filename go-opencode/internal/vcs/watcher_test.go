package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/opencode-ai/opencode/internal/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBranch(t *testing.T) {
	// Test in current directory (should be a git repo)
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Go up to find the repo root
	repoRoot := findRepoRoot(cwd)
	if repoRoot == "" {
		t.Skip("Not running in a git repository")
	}

	branch := GetBranch(repoRoot)
	assert.NotEmpty(t, branch, "should return a branch name in a git repo")
}

func TestGetBranch_NonGitDir(t *testing.T) {
	// Create a temporary directory that's not a git repo
	tmpDir, err := os.MkdirTemp("", "vcs-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	branch := GetBranch(tmpDir)
	assert.Empty(t, branch, "should return empty string for non-git directory")
}

func TestNewWatcher_NonGitDir(t *testing.T) {
	// Create a temporary directory that's not a git repo
	tmpDir, err := os.MkdirTemp("", "vcs-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	watcher, err := NewWatcher(tmpDir)
	assert.NoError(t, err, "should not error for non-git directory")
	assert.Nil(t, watcher, "should return nil watcher for non-git directory")
}

func TestNewWatcher_GitRepo(t *testing.T) {
	// Create a temporary git repository
	tmpDir := createTempGitRepo(t)
	defer os.RemoveAll(tmpDir)

	watcher, err := NewWatcher(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, watcher, "should create watcher for git repository")

	// Clean up
	err = watcher.Stop()
	assert.NoError(t, err)
}

func TestWatcher_CurrentBranch(t *testing.T) {
	// Create a temporary git repository
	tmpDir := createTempGitRepo(t)
	defer os.RemoveAll(tmpDir)

	watcher, err := NewWatcher(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, watcher)
	defer watcher.Stop()

	branch := watcher.CurrentBranch()
	assert.Equal(t, "main", branch, "should return the current branch")
}

func TestWatcher_StartStop(t *testing.T) {
	// Create a temporary git repository
	tmpDir := createTempGitRepo(t)
	defer os.RemoveAll(tmpDir)

	watcher, err := NewWatcher(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, watcher)

	// Start and stop should work cleanly
	watcher.Start()
	err = watcher.Stop()
	assert.NoError(t, err)
}

func TestWatcher_CheckBranchChange(t *testing.T) {
	// Create a temporary git repository
	tmpDir := createTempGitRepo(t)
	defer os.RemoveAll(tmpDir)

	// Reset event bus for clean test
	event.Reset()

	watcher, err := NewWatcher(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, watcher)
	defer watcher.Stop()

	// Subscribe to branch update events
	eventReceived := make(chan event.VcsBranchUpdatedData, 1)
	unsubscribe := event.Subscribe(event.VcsBranchUpdated, func(e event.Event) {
		if data, ok := e.Data.(event.VcsBranchUpdatedData); ok {
			select {
			case eventReceived <- data:
			default:
			}
		}
	})
	defer unsubscribe()

	// Manually update the branch in the watcher and trigger check
	runGit(t, tmpDir, "checkout", "-b", "feature-branch")

	// Manually call checkBranchChange (simulates what happens when file event is received)
	watcher.checkBranchChange()

	// Should have received the event
	select {
	case data := <-eventReceived:
		assert.Equal(t, "feature-branch", data.Branch, "should detect branch change")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("should have received branch change event")
	}

	// Verify watcher's cached branch is updated
	assert.Equal(t, "feature-branch", watcher.CurrentBranch())
}

func TestWatcher_CheckBranchChange_NoChange(t *testing.T) {
	// Create a temporary git repository
	tmpDir := createTempGitRepo(t)
	defer os.RemoveAll(tmpDir)

	// Reset event bus for clean test
	event.Reset()

	watcher, err := NewWatcher(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, watcher)
	defer watcher.Stop()

	// Subscribe to branch update events
	eventReceived := make(chan event.VcsBranchUpdatedData, 1)
	unsubscribe := event.Subscribe(event.VcsBranchUpdated, func(e event.Event) {
		if data, ok := e.Data.(event.VcsBranchUpdatedData); ok {
			select {
			case eventReceived <- data:
			default:
			}
		}
	})
	defer unsubscribe()

	// Call checkBranchChange without actually changing the branch
	watcher.checkBranchChange()

	// Should NOT receive an event
	select {
	case <-eventReceived:
		t.Fatal("should not receive event when branch hasn't changed")
	case <-time.After(50 * time.Millisecond):
		// Expected - no event
	}

	// Branch should still be main
	assert.Equal(t, "main", watcher.CurrentBranch())
}

func TestFindGitDir(t *testing.T) {
	// Create a temporary git repository
	tmpDir := createTempGitRepo(t)
	defer os.RemoveAll(tmpDir)

	gitDir := findGitDir(tmpDir)
	assert.NotEmpty(t, gitDir, "should find .git directory")
	assert.True(t, filepath.IsAbs(gitDir), "should return absolute path")

	// Verify it ends with .git
	assert.Equal(t, ".git", filepath.Base(gitDir))
}

func TestFindGitDir_NonGitDir(t *testing.T) {
	// Create a temporary directory that's not a git repo
	tmpDir, err := os.MkdirTemp("", "vcs-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	gitDir := findGitDir(tmpDir)
	assert.Empty(t, gitDir, "should return empty string for non-git directory")
}

func TestGetCurrentBranch(t *testing.T) {
	// Create a temporary git repository
	tmpDir := createTempGitRepo(t)
	defer os.RemoveAll(tmpDir)

	branch := getCurrentBranch(tmpDir)
	assert.Equal(t, "main", branch, "should return 'main' for new repo")

	// Create and switch to a new branch
	runGit(t, tmpDir, "checkout", "-b", "test-branch")

	branch = getCurrentBranch(tmpDir)
	assert.Equal(t, "test-branch", branch, "should return new branch name")
}

func TestWatcher_ConcurrentAccess(t *testing.T) {
	// Create a temporary git repository
	tmpDir := createTempGitRepo(t)
	defer os.RemoveAll(tmpDir)

	watcher, err := NewWatcher(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, watcher)
	defer watcher.Stop()

	watcher.Start()

	// Concurrent reads should be safe
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = watcher.CurrentBranch()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Helper functions

func createTempGitRepo(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "vcs-test-repo-*")
	require.NoError(t, err)

	// Initialize git repo
	runGit(t, tmpDir, "init", "-b", "main")

	// Configure git user (required for commits)
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test User")

	// Create initial commit (required for branch operations)
	testFile := filepath.Join(tmpDir, "README.md")
	err = os.WriteFile(testFile, []byte("# Test\n"), 0644)
	require.NoError(t, err)

	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "Initial commit")

	return tmpDir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, string(output))
}

func findRepoRoot(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return filepath.Clean(string(out[:len(out)-1])) // Remove trailing newline
}
