// Package project provides project management functionality.
package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	"github.com/opencode-ai/opencode/pkg/types"
)

// Service manages project information.
type Service struct {
	workDir string
}

// NewService creates a new project service.
func NewService(workDir string) *Service {
	return &Service{workDir: workDir}
}

// List returns all projects (currently just the current project).
// If directory is provided in context, it uses that instead of the default workDir.
func (s *Service) List(ctx context.Context) ([]types.Project, error) {
	current, err := s.Current(ctx)
	if err != nil {
		return nil, err
	}
	return []types.Project{*current}, nil
}

// ListForDir returns all projects for a specific directory.
func (s *Service) ListForDir(ctx context.Context, dir string) ([]types.Project, error) {
	current, err := s.CurrentForDir(ctx, dir)
	if err != nil {
		return nil, err
	}
	return []types.Project{*current}, nil
}

// Current returns the current project based on workDir.
func (s *Service) Current(ctx context.Context) (*types.Project, error) {
	return s.CurrentForDir(ctx, s.workDir)
}

// CurrentForDir returns the current project for a specific directory.
func (s *Service) CurrentForDir(ctx context.Context, dir string) (*types.Project, error) {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	// Generate ID from path hash
	hash := sha256.Sum256([]byte(absPath))
	id := hex.EncodeToString(hash[:])[:16]

	// Check for VCS
	var vcs string
	if _, err := os.Stat(filepath.Join(absPath, ".git")); err == nil {
		vcs = "git"
	}

	// Get directory creation time (or use current time as fallback)
	info, _ := os.Stat(absPath)
	created := time.Now().UnixMilli()
	if info != nil {
		created = info.ModTime().UnixMilli()
	}

	return &types.Project{
		ID:       id,
		Worktree: absPath,
		VCS:      vcs,
		Time: types.ProjectTime{
			Created: created,
		},
	}, nil
}
