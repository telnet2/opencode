// Package project provides project detection and identification functionality.
// It mirrors the TypeScript OpenCode implementation to ensure session compatibility.
package project

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Info contains project metadata.
type Info struct {
	ID       string  `json:"id"`
	Worktree string  `json:"worktree"`
	VCSDir   *string `json:"vcsDir,omitempty"`
	VCS      *string `json:"vcs,omitempty"`
}

// cache stores project info by directory to avoid repeated git calls.
var (
	cacheMu sync.RWMutex
	cache   = make(map[string]*Info)
)

// FromDirectory detects project information from a directory.
// It mirrors the TypeScript implementation:
// 1. Finds the .git directory by walking up the tree
// 2. Uses the git initial commit SHA as project ID (cached in .git/opencode)
// 3. Falls back to "global" for non-git directories
func FromDirectory(directory string) (*Info, error) {
	// Normalize directory path
	directory, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}

	// Check cache first
	cacheMu.RLock()
	if info, ok := cache[directory]; ok {
		cacheMu.RUnlock()
		return info, nil
	}
	cacheMu.RUnlock()

	// Find .git directory
	gitDir := findGitDir(directory)
	if gitDir == "" {
		// Not a git repository - use "global" project
		info := &Info{
			ID:       "global",
			Worktree: "/",
		}
		cacheProject(directory, info)
		return info, nil
	}

	// Get worktree (git root directory)
	worktree := filepath.Dir(gitDir)
	worktreeCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	worktreeCmd.Dir = worktree
	if output, err := worktreeCmd.Output(); err == nil {
		worktree = strings.TrimSpace(string(output))
	}

	// Get actual git dir (handles worktrees)
	gitDirCmd := exec.Command("git", "rev-parse", "--git-dir")
	gitDirCmd.Dir = worktree
	if output, err := gitDirCmd.Output(); err == nil {
		resolvedGitDir := strings.TrimSpace(string(output))
		if !filepath.IsAbs(resolvedGitDir) {
			resolvedGitDir = filepath.Join(worktree, resolvedGitDir)
		}
		gitDir = resolvedGitDir
	}

	// Try to read cached project ID from .git/opencode
	cacheFile := filepath.Join(gitDir, "opencode")
	projectID, err := os.ReadFile(cacheFile)
	if err == nil && len(projectID) > 0 {
		id := strings.TrimSpace(string(projectID))
		vcs := "git"
		info := &Info{
			ID:       id,
			Worktree: worktree,
			VCSDir:   &gitDir,
			VCS:      &vcs,
		}
		cacheProject(directory, info)
		return info, nil
	}

	// Get project ID from git's initial commit SHA
	// This matches the TypeScript: git rev-list --max-parents=0 --all
	id := getGitProjectID(worktree)
	if id == "" {
		id = "global"
	}

	// Cache the project ID in .git/opencode for future use
	if id != "global" {
		_ = os.WriteFile(cacheFile, []byte(id), 0644)
	}

	vcs := "git"
	info := &Info{
		ID:       id,
		Worktree: worktree,
		VCSDir:   &gitDir,
		VCS:      &vcs,
	}
	cacheProject(directory, info)
	return info, nil
}

// GetProjectID returns just the project ID for a directory.
// This is a convenience function for the session service.
func GetProjectID(directory string) (string, error) {
	info, err := FromDirectory(directory)
	if err != nil {
		return "", err
	}
	return info.ID, nil
}

// HashDirectory creates a hash-based project ID from a directory path.
// This is the OLD method - kept for migration purposes only.
func HashDirectory(directory string) string {
	h := sha256.New()
	h.Write([]byte(directory))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// findGitDir walks up from the given directory looking for a .git directory.
func findGitDir(start string) string {
	current := start
	for {
		gitPath := filepath.Join(current, ".git")
		if info, err := os.Stat(gitPath); err == nil {
			if info.IsDir() {
				return gitPath
			}
			// .git might be a file (for worktrees/submodules)
			// Read the gitdir from it
			if content, err := os.ReadFile(gitPath); err == nil {
				line := strings.TrimSpace(string(content))
				if strings.HasPrefix(line, "gitdir: ") {
					gitdir := strings.TrimPrefix(line, "gitdir: ")
					if !filepath.IsAbs(gitdir) {
						gitdir = filepath.Join(current, gitdir)
					}
					return gitdir
				}
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			return ""
		}
		current = parent
	}
}

// getGitProjectID gets the project ID from git's initial commit(s).
// It matches the TypeScript implementation which uses:
// git rev-list --max-parents=0 --all
// and takes the first (alphabetically sorted) root commit SHA.
func getGitProjectID(worktree string) string {
	cmd := exec.Command("git", "rev-list", "--max-parents=0", "--all")
	cmd.Dir = worktree
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var roots []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			roots = append(roots, line)
		}
	}

	if len(roots) == 0 {
		return ""
	}

	// Sort and take the first one (matches TypeScript behavior)
	sort.Strings(roots)
	return roots[0]
}

// cacheProject adds a project to the in-memory cache.
func cacheProject(directory string, info *Info) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cache[directory] = info
}

// ClearCache clears the project cache. Useful for testing.
func ClearCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cache = make(map[string]*Info)
}
