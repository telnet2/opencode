package memsh

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
)

// TestLsRecursive tests ls -R flag
func TestLsRecursive(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()

	// Create directory structure
	shell.Run(ctx, "mkdir -p /test/subdir1/subdir2")
	shell.Run(ctx, "mkdir -p /test/another")
	shell.Run(ctx, "echo 'test' > /test/file1.txt")
	shell.Run(ctx, "echo 'sub' > /test/subdir1/file2.txt")
	shell.Run(ctx, "echo 'deep' > /test/subdir1/subdir2/file3.txt")

	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Test ls -R
	err = shell.Run(ctx, "ls -R /test")
	if err != nil {
		t.Fatalf("ls -R failed: %v", err)
	}

	output := stdout.String()
	t.Logf("ls -R output:\n%s", output)

	// Check that recursive listing shows files from all levels
	if !strings.Contains(output, "file1.txt") {
		t.Error("Expected file1.txt in output")
	}
	if !strings.Contains(output, "file2.txt") {
		t.Error("Expected file2.txt in output")
	}
	if !strings.Contains(output, "file3.txt") {
		t.Error("Expected file3.txt in output")
	}
	if !strings.Contains(output, "/test/subdir1:") || !strings.Contains(output, "/test/subdir1/subdir2:") {
		t.Error("Expected directory headers in recursive output")
	}
}

// TestCpPreserve tests cp -p flag
func TestCpPreserve(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()

	// Create a file with specific timestamp
	shell.Run(ctx, "echo 'test content' > /test.txt")

	// Set a specific timestamp
	oldTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	fs.Chtimes("/test.txt", oldTime, oldTime)

	// Get original permissions
	srcInfo, _ := fs.Stat("/test.txt")

	// Copy with -p flag
	err = shell.Run(ctx, "cp -p /test.txt /test_copy.txt")
	if err != nil {
		t.Fatalf("cp -p failed: %v", err)
	}

	// Check that attributes were preserved
	destInfo, err := fs.Stat("/test_copy.txt")
	if err != nil {
		t.Fatalf("Failed to stat copied file: %v", err)
	}

	if !destInfo.ModTime().Equal(srcInfo.ModTime()) {
		t.Errorf("Timestamp not preserved: expected %v, got %v", srcInfo.ModTime(), destInfo.ModTime())
	}

	if destInfo.Mode() != srcInfo.Mode() {
		t.Errorf("Permissions not preserved: expected %v, got %v", srcInfo.Mode(), destInfo.Mode())
	}
}

// TestGrepQuiet tests grep -q flag
func TestGrepQuiet(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()

	// Create test file
	shell.Run(ctx, "echo 'line with pattern\nother line\nmore pattern' > /test.txt")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	shell.SetIO(nil, &stdout, &stderr)

	// Test grep -q with match (should succeed with no output)
	err = shell.Run(ctx, "grep -q pattern /test.txt")
	if err != nil {
		t.Errorf("grep -q should succeed when pattern found: %v", err)
	}

	output := stdout.String()
	if output != "" {
		t.Errorf("grep -q should produce no output, got: %s", output)
	}

	// Reset output
	stdout.Reset()

	// Test grep -q with no match (should fail with no output)
	err = shell.Run(ctx, "grep -q nonexistent /test.txt")
	if err == nil {
		t.Error("grep -q should fail when pattern not found")
	}

	output = stdout.String()
	if output != "" {
		t.Errorf("grep -q should produce no output even on failure, got: %s", output)
	}
}

// TestLsRecursiveWithLongFormat tests ls -lR combination
func TestLsRecursiveWithLongFormat(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()

	// Create directory structure
	shell.Run(ctx, "mkdir -p /test/subdir")
	shell.Run(ctx, "echo 'test' > /test/file1.txt")
	shell.Run(ctx, "echo 'sub' > /test/subdir/file2.txt")

	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Test ls -lR
	err = shell.Run(ctx, "ls -lR /test")
	if err != nil {
		t.Fatalf("ls -lR failed: %v", err)
	}

	output := stdout.String()
	t.Logf("ls -lR output:\n%s", output)

	// Check that long format shows file details
	if !strings.Contains(output, "file1.txt") || !strings.Contains(output, "file2.txt") {
		t.Error("Expected files in output")
	}
	// Long format should include permission bits
	if !strings.Contains(output, "-rw") {
		t.Error("Expected permission bits in long format")
	}
}

// TestCpRecursiveWithPreserve tests cp -rp combination
func TestCpRecursiveWithPreserve(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()

	// Create directory structure
	shell.Run(ctx, "mkdir -p /source/subdir")
	shell.Run(ctx, "echo 'test' > /source/file1.txt")
	shell.Run(ctx, "echo 'sub' > /source/subdir/file2.txt")

	// Set specific timestamp
	oldTime := time.Date(2020, 6, 15, 12, 0, 0, 0, time.UTC)
	fs.Chtimes("/source/file1.txt", oldTime, oldTime)
	fs.Chtimes("/source/subdir/file2.txt", oldTime, oldTime)

	// Copy with -rp flags
	err = shell.Run(ctx, "cp -rp /source /dest")
	if err != nil {
		t.Fatalf("cp -rp failed: %v", err)
	}

	// Check that directory was copied
	destExists, _ := afero.Exists(fs, "/dest/file1.txt")
	if !destExists {
		t.Error("Expected /dest/file1.txt to exist")
	}

	subExists, _ := afero.Exists(fs, "/dest/subdir/file2.txt")
	if !subExists {
		t.Error("Expected /dest/subdir/file2.txt to exist")
	}

	// Check timestamp preservation
	srcInfo, _ := fs.Stat("/source/file1.txt")
	destInfo, _ := fs.Stat("/dest/file1.txt")

	if !destInfo.ModTime().Equal(srcInfo.ModTime()) {
		t.Errorf("Timestamp not preserved in recursive copy: expected %v, got %v",
			srcInfo.ModTime(), destInfo.ModTime())
	}
}

// TestGrepQuietCombinations tests grep -q with other flags
func TestGrepQuietCombinations(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()

	// Create test file
	shell.Run(ctx, "echo 'Line with PATTERN\nother line' > /test.txt")

	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Test grep -qi (quiet + case insensitive)
	err = shell.Run(ctx, "grep -qi pattern /test.txt")
	if err != nil {
		t.Errorf("grep -qi should succeed: %v", err)
	}

	output := stdout.String()
	if output != "" {
		t.Errorf("grep -qi should produce no output, got: %s", output)
	}
}
