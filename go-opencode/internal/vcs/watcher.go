// Package vcs provides version control system integration.
package vcs

import (
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/opencode-ai/opencode/internal/event"
	"github.com/rs/zerolog/log"
)

// Watcher watches for git branch changes by monitoring .git/HEAD.
type Watcher struct {
	watcher       *fsnotify.Watcher
	workDir       string
	gitDir        string
	currentBranch string
	stopCh        chan struct{}
	doneCh        chan struct{}
	started       bool
	mu            sync.RWMutex
}

// NewWatcher creates a new VCS watcher for the given work directory.
// Returns nil if the directory is not a git repository.
func NewWatcher(workDir string) (*Watcher, error) {
	gitDir := findGitDir(workDir)
	if gitDir == "" {
		log.Debug().Str("workDir", workDir).Msg("not a git repository, VCS watcher disabled")
		return nil, nil
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Watch the .git directory itself (to catch HEAD changes)
	// On some systems, watching the file directly doesn't work reliably
	if err := w.Add(gitDir); err != nil {
		w.Close()
		return nil, err
	}

	branch := getCurrentBranch(workDir)
	log.Info().Str("branch", branch).Str("gitDir", gitDir).Msg("VCS watcher initialized")

	return &Watcher{
		watcher:       w,
		workDir:       workDir,
		gitDir:        gitDir,
		currentBranch: branch,
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}, nil
}

// Start begins watching for branch changes.
func (w *Watcher) Start() {
	w.mu.Lock()
	if w.started {
		w.mu.Unlock()
		return
	}
	w.started = true
	w.mu.Unlock()
	go w.run()
}

func (w *Watcher) run() {
	defer close(w.doneCh)

	for {
		select {
		case <-w.stopCh:
			return
		case ev, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			// Check if this is a write event on HEAD or a relevant file
			if ev.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				// Check if the file is HEAD
				if strings.HasSuffix(ev.Name, "HEAD") || strings.Contains(ev.Name, ".git") {
					w.checkBranchChange()
				}
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Error().Err(err).Msg("VCS watcher error")
		}
	}
}

func (w *Watcher) checkBranchChange() {
	newBranch := getCurrentBranch(w.workDir)

	w.mu.Lock()
	oldBranch := w.currentBranch
	changed := newBranch != oldBranch
	if changed {
		w.currentBranch = newBranch
	}
	w.mu.Unlock()

	if changed {
		log.Info().
			Str("from", oldBranch).
			Str("to", newBranch).
			Msg("branch changed")

		event.PublishSync(event.Event{
			Type: event.VcsBranchUpdated,
			Data: event.VcsBranchUpdatedData{Branch: newBranch},
		})
	}
}

// CurrentBranch returns the currently tracked branch name.
func (w *Watcher) CurrentBranch() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.currentBranch
}

// Stop stops the watcher.
func (w *Watcher) Stop() error {
	w.mu.Lock()
	started := w.started
	w.mu.Unlock()

	// Signal stop
	select {
	case <-w.stopCh:
		// Already stopped
	default:
		close(w.stopCh)
	}

	// Wait for run() to finish if it was started
	if started {
		<-w.doneCh
	}

	return w.watcher.Close()
}

// findGitDir finds the .git directory for a given work directory.
// Handles both regular repos (.git directory) and worktrees (.git file).
func findGitDir(workDir string) string {
	// Use git to find the actual git directory (handles worktrees too)
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	gitDir := strings.TrimSpace(string(out))
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(workDir, gitDir)
	}

	return gitDir
}

// getCurrentBranch gets the current git branch name.
func getCurrentBranch(workDir string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// GetBranch returns the current branch for a given directory (static helper).
func GetBranch(workDir string) string {
	return getCurrentBranch(workDir)
}
