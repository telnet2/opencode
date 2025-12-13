package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashDirectory(t *testing.T) {
	// Test that hash is deterministic
	hash1 := HashDirectory("/home/user/test")
	hash2 := HashDirectory("/home/user/test")
	if hash1 != hash2 {
		t.Errorf("HashDirectory not deterministic: %s != %s", hash1, hash2)
	}

	// Test that different paths produce different hashes
	hash3 := HashDirectory("/home/user/other")
	if hash1 == hash3 {
		t.Errorf("Different paths should produce different hashes")
	}

	// Test hash length
	if len(hash1) != 16 {
		t.Errorf("Hash should be 16 characters, got %d", len(hash1))
	}
}

func TestFindGitDir(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Test directory without .git
	result := findGitDir(tmpDir)
	if result != "" {
		t.Errorf("Expected empty string for non-git dir, got %s", result)
	}

	// Create .git directory
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Test from root
	result = findGitDir(tmpDir)
	if result != gitDir {
		t.Errorf("Expected %s, got %s", gitDir, result)
	}

	// Test from subdirectory
	subDir := filepath.Join(tmpDir, "sub", "dir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	result = findGitDir(subDir)
	if result != gitDir {
		t.Errorf("Expected %s, got %s", gitDir, result)
	}
}

func TestFromDirectoryNonGit(t *testing.T) {
	ClearCache()
	tmpDir := t.TempDir()

	info, err := FromDirectory(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if info.ID != "global" {
		t.Errorf("Expected 'global' project ID for non-git dir, got %s", info.ID)
	}

	if info.Worktree != "/" {
		t.Errorf("Expected '/' worktree for non-git dir, got %s", info.Worktree)
	}
}

func TestFromDirectoryGit(t *testing.T) {
	ClearCache()
	tmpDir := t.TempDir()

	// Initialize a git repo
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create initial commit (simulated - just create the necessary structure)
	// In a real git repo, we'd have commit objects, but for testing without
	// running git commands, we'll just test the caching mechanism

	// Write a cached project ID
	cacheFile := filepath.Join(gitDir, "opencode")
	expectedID := "testprojectid123"
	if err := os.WriteFile(cacheFile, []byte(expectedID), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := FromDirectory(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if info.ID != expectedID {
		t.Errorf("Expected cached project ID %s, got %s", expectedID, info.ID)
	}

	if info.VCS == nil || *info.VCS != "git" {
		t.Error("Expected VCS to be 'git'")
	}
}

func TestGetProjectID(t *testing.T) {
	ClearCache()
	tmpDir := t.TempDir()

	id, err := GetProjectID(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if id != "global" {
		t.Errorf("Expected 'global' for non-git dir, got %s", id)
	}
}

func TestCaching(t *testing.T) {
	ClearCache()
	tmpDir := t.TempDir()

	// First call
	info1, err := FromDirectory(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Second call should return cached result
	info2, err := FromDirectory(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if info1 != info2 {
		t.Error("Expected cached result to be same pointer")
	}

	// Clear cache and call again
	ClearCache()
	info3, err := FromDirectory(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if info1 == info3 {
		t.Error("Expected new result after cache clear")
	}
}
