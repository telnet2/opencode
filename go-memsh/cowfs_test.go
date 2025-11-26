package memsh

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

// setupTestDir creates a temporary directory with test files
func setupTestDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "cowfs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create test files
	files := map[string]string{
		"file1.txt":         "content of file1",
		"file2.txt":         "content of file2",
		"subdir/file3.txt":  "content of file3",
		"subdir/file4.txt":  "content of file4",
		"subdir/deep/a.txt": "deep file a",
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	return dir
}

func TestLocalBackend(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	backend, err := NewLocalBackend(dir)
	if err != nil {
		t.Fatalf("Failed to create local backend: %v", err)
	}

	t.Run("Name", func(t *testing.T) {
		expected := "local:" + dir
		if backend.Name() != expected {
			t.Errorf("Expected name %q, got %q", expected, backend.Name())
		}
	})

	t.Run("Open and Read", func(t *testing.T) {
		r, err := backend.Open("file1.txt")
		if err != nil {
			t.Fatalf("Failed to open file: %v", err)
		}
		defer r.Close()

		content, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(content) != "content of file1" {
			t.Errorf("Unexpected content: %q", content)
		}
	})

	t.Run("Stat", func(t *testing.T) {
		info, err := backend.Stat("file1.txt")
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}

		if info.IsDir() {
			t.Error("Expected file, got directory")
		}

		if info.Size() != int64(len("content of file1")) {
			t.Errorf("Unexpected size: %d", info.Size())
		}
	})

	t.Run("ReadDir", func(t *testing.T) {
		entries, err := backend.ReadDir("/")
		if err != nil {
			t.Fatalf("Failed to read dir: %v", err)
		}

		names := make(map[string]bool)
		for _, e := range entries {
			names[e.Name()] = true
		}

		expected := []string{"file1.txt", "file2.txt", "subdir"}
		for _, name := range expected {
			if !names[name] {
				t.Errorf("Missing expected entry: %s", name)
			}
		}
	})

	t.Run("Create and Write", func(t *testing.T) {
		w, err := backend.Create("newfile.txt")
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		_, err = w.Write([]byte("new content"))
		if err != nil {
			t.Fatalf("Failed to write: %v", err)
		}
		w.Close()

		// Verify file was created
		content, err := os.ReadFile(filepath.Join(dir, "newfile.txt"))
		if err != nil {
			t.Fatalf("Failed to read created file: %v", err)
		}

		if string(content) != "new content" {
			t.Errorf("Unexpected content: %q", content)
		}
	})

	t.Run("Exists", func(t *testing.T) {
		exists, err := backend.Exists("file1.txt")
		if err != nil {
			t.Fatalf("Failed to check existence: %v", err)
		}
		if !exists {
			t.Error("Expected file to exist")
		}

		exists, err = backend.Exists("nonexistent.txt")
		if err != nil {
			t.Fatalf("Failed to check existence: %v", err)
		}
		if exists {
			t.Error("Expected file to not exist")
		}
	})
}

func TestCopyOnWriteFs(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	backend, err := NewLocalBackend(dir)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	t.Run("Create with eager loading", func(t *testing.T) {
		cowFs, err := NewCopyOnWriteFs(backend, CopyOnWriteConfig{
			SyncDeletes: true,
			Lazy:        false,
		})
		if err != nil {
			t.Fatalf("Failed to create COW fs: %v", err)
		}

		// Should have loaded all files
		f, err := cowFs.Open("/file1.txt")
		if err != nil {
			t.Fatalf("Failed to open file: %v", err)
		}
		defer f.Close()

		content, _ := io.ReadAll(f)
		if string(content) != "content of file1" {
			t.Errorf("Unexpected content: %q", content)
		}
	})

	t.Run("Create with lazy loading", func(t *testing.T) {
		cowFs, err := NewCopyOnWriteFs(backend, CopyOnWriteConfig{
			SyncDeletes: true,
			Lazy:        true,
		})
		if err != nil {
			t.Fatalf("Failed to create COW fs: %v", err)
		}

		// File should be loaded on first access
		f, err := cowFs.Open("/file1.txt")
		if err != nil {
			t.Fatalf("Failed to open file: %v", err)
		}
		defer f.Close()

		content, _ := io.ReadAll(f)
		if string(content) != "content of file1" {
			t.Errorf("Unexpected content: %q", content)
		}
	})

	t.Run("Write creates modification", func(t *testing.T) {
		cowFs, err := NewCopyOnWriteFs(backend, CopyOnWriteConfig{
			SyncDeletes: true,
			Lazy:        false,
		})
		if err != nil {
			t.Fatalf("Failed to create COW fs: %v", err)
		}

		// Initially clean
		if cowFs.IsDirty() {
			t.Error("Expected clean state initially")
		}

		// Modify a file
		f, err := cowFs.OpenFile("/file1.txt", os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			t.Fatalf("Failed to open file for writing: %v", err)
		}
		f.Write([]byte("modified content"))
		f.Close()

		// Should be dirty now
		if !cowFs.IsDirty() {
			t.Error("Expected dirty state after modification")
		}

		// Check dirty files
		dirty := cowFs.DirtyFiles()
		if len(dirty) != 1 || dirty[0] != "/file1.txt" {
			t.Errorf("Unexpected dirty files: %v", dirty)
		}

		// Original file should be unchanged
		original, _ := os.ReadFile(filepath.Join(dir, "file1.txt"))
		if string(original) != "content of file1" {
			t.Error("Original file was modified!")
		}
	})

	t.Run("Delete marks file as deleted", func(t *testing.T) {
		cowFs, err := NewCopyOnWriteFs(backend, CopyOnWriteConfig{
			SyncDeletes: true,
			Lazy:        false,
		})
		if err != nil {
			t.Fatalf("Failed to create COW fs: %v", err)
		}

		// Delete a file
		err = cowFs.Remove("/file2.txt")
		if err != nil {
			t.Fatalf("Failed to remove file: %v", err)
		}

		// File should appear deleted
		_, err = cowFs.Open("/file2.txt")
		if !os.IsNotExist(err) {
			t.Errorf("Expected file not to exist, got: %v", err)
		}

		// Should be in deleted list
		deleted := cowFs.DeletedFiles()
		if len(deleted) != 1 || deleted[0] != "/file2.txt" {
			t.Errorf("Unexpected deleted files: %v", deleted)
		}

		// Original file should still exist
		_, err = os.Stat(filepath.Join(dir, "file2.txt"))
		if err != nil {
			t.Error("Original file was actually deleted!")
		}
	})

	t.Run("Create new file", func(t *testing.T) {
		cowFs, err := NewCopyOnWriteFs(backend, CopyOnWriteConfig{
			SyncDeletes: true,
			Lazy:        false,
		})
		if err != nil {
			t.Fatalf("Failed to create COW fs: %v", err)
		}

		// Create a new file
		f, err := cowFs.Create("/newfile.txt")
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		f.Write([]byte("new file content"))
		f.Close()

		// Should be able to read it back
		r, err := cowFs.Open("/newfile.txt")
		if err != nil {
			t.Fatalf("Failed to open new file: %v", err)
		}
		content, _ := io.ReadAll(r)
		r.Close()

		if string(content) != "new file content" {
			t.Errorf("Unexpected content: %q", content)
		}

		// Should be in modified list
		dirty := cowFs.DirtyFiles()
		found := false
		for _, f := range dirty {
			if f == "/newfile.txt" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("New file not in dirty list: %v", dirty)
		}
	})

	t.Run("Flush to same backend", func(t *testing.T) {
		cowFs, err := NewCopyOnWriteFs(backend, CopyOnWriteConfig{
			SyncDeletes: true,
			Lazy:        false,
		})
		if err != nil {
			t.Fatalf("Failed to create COW fs: %v", err)
		}

		// Modify a file
		f, err := cowFs.OpenFile("/file1.txt", os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			t.Fatalf("Failed to open file: %v", err)
		}
		f.Write([]byte("flushed content"))
		f.Close()

		// Flush
		err = cowFs.Flush()
		if err != nil {
			t.Fatalf("Failed to flush: %v", err)
		}

		// Should be clean now
		if cowFs.IsDirty() {
			t.Error("Expected clean state after flush")
		}

		// Original file should be updated
		content, _ := os.ReadFile(filepath.Join(dir, "file1.txt"))
		if string(content) != "flushed content" {
			t.Errorf("File was not flushed: %q", content)
		}
	})

	t.Run("Flush to different directory", func(t *testing.T) {
		// Reset the source file
		os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("content of file1"), 0644)

		cowFs, err := NewCopyOnWriteFs(backend, CopyOnWriteConfig{
			SyncDeletes: true,
			Lazy:        false,
		})
		if err != nil {
			t.Fatalf("Failed to create COW fs: %v", err)
		}

		// Modify a file
		f, err := cowFs.OpenFile("/file1.txt", os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			t.Fatalf("Failed to open file: %v", err)
		}
		f.Write([]byte("modified for flush"))
		f.Close()

		// Create a target directory
		targetDir, err := os.MkdirTemp("", "cowfs-flush-*")
		if err != nil {
			t.Fatalf("Failed to create target dir: %v", err)
		}
		defer os.RemoveAll(targetDir)

		// Flush to target
		err = cowFs.FlushToLocal(targetDir)
		if err != nil {
			t.Fatalf("Failed to flush to local: %v", err)
		}

		// Target should have the modified file
		content, err := os.ReadFile(filepath.Join(targetDir, "file1.txt"))
		if err != nil {
			t.Fatalf("Failed to read flushed file: %v", err)
		}
		if string(content) != "modified for flush" {
			t.Errorf("Wrong content in target: %q", content)
		}

		// Original should be unchanged
		original, _ := os.ReadFile(filepath.Join(dir, "file1.txt"))
		if string(original) != "content of file1" {
			t.Error("Original was modified during FlushToLocal")
		}
	})

	t.Run("Reset discards changes", func(t *testing.T) {
		cowFs, err := NewCopyOnWriteFs(backend, CopyOnWriteConfig{
			SyncDeletes: true,
			Lazy:        false,
		})
		if err != nil {
			t.Fatalf("Failed to create COW fs: %v", err)
		}

		// Modify a file
		f, err := cowFs.OpenFile("/file1.txt", os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			t.Fatalf("Failed to open file: %v", err)
		}
		f.Write([]byte("will be discarded"))
		f.Close()

		// Reset
		err = cowFs.Reset()
		if err != nil {
			t.Fatalf("Failed to reset: %v", err)
		}

		// Should be clean
		if cowFs.IsDirty() {
			t.Error("Expected clean state after reset")
		}

		// Should have original content
		r, _ := cowFs.Open("/file1.txt")
		content, _ := io.ReadAll(r)
		r.Close()

		if string(content) != "content of file1" {
			t.Errorf("Reset did not restore original content: %q", content)
		}
	})

	t.Run("Status returns correct info", func(t *testing.T) {
		cowFs, err := NewCopyOnWriteFs(backend, CopyOnWriteConfig{
			MountPoint:  "/project",
			SyncDeletes: true,
			Lazy:        false,
		})
		if err != nil {
			t.Fatalf("Failed to create COW fs: %v", err)
		}

		status := cowFs.Status()
		if status.MountPoint != "/project" {
			t.Errorf("Wrong mount point: %s", status.MountPoint)
		}
		if status.Backend != backend.Name() {
			t.Errorf("Wrong backend: %s", status.Backend)
		}
		if status.IsDirty {
			t.Error("Should not be dirty initially")
		}

		// Make some changes
		f, _ := cowFs.Create("/new.txt")
		f.Write([]byte("test"))
		f.Close()
		cowFs.Remove("/file2.txt")

		status = cowFs.Status()
		if !status.IsDirty {
			t.Error("Should be dirty after changes")
		}
		if status.ModifiedCount != 1 {
			t.Errorf("Wrong modified count: %d", status.ModifiedCount)
		}
		if status.DeletedCount != 1 {
			t.Errorf("Wrong deleted count: %d", status.DeletedCount)
		}
	})
}

func TestCowManager(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	backend, _ := NewLocalBackend(dir)
	cowFs, _ := NewCopyOnWriteFs(backend, CopyOnWriteConfig{Lazy: true})

	manager := NewCowManager()

	t.Run("Mount and Get", func(t *testing.T) {
		err := manager.Mount("/project", cowFs)
		if err != nil {
			t.Fatalf("Failed to mount: %v", err)
		}

		fs, ok := manager.Get("/project")
		if !ok {
			t.Fatal("Failed to get mounted fs")
		}
		if fs != cowFs {
			t.Error("Got wrong filesystem")
		}
	})

	t.Run("List", func(t *testing.T) {
		mounts := manager.List()
		if len(mounts) != 1 || mounts[0] != "/project" {
			t.Errorf("Unexpected mounts: %v", mounts)
		}
	})

	t.Run("FindMount", func(t *testing.T) {
		fs, relPath, found := manager.FindMount("/project/subdir/file.txt")
		if !found {
			t.Fatal("Failed to find mount")
		}
		if fs != cowFs {
			t.Error("Found wrong filesystem")
		}
		if relPath != "/subdir/file.txt" {
			t.Errorf("Wrong relative path: %s", relPath)
		}
	})

	t.Run("Unmount with dirty state fails", func(t *testing.T) {
		// Make it dirty
		f, _ := cowFs.Create("/dirty.txt")
		f.Write([]byte("test"))
		f.Close()

		err := manager.Unmount("/project")
		if err == nil {
			t.Error("Expected error when unmounting dirty fs")
		}
	})

	t.Run("ForceUnmount works", func(t *testing.T) {
		err := manager.ForceUnmount("/project")
		if err != nil {
			t.Fatalf("Failed to force unmount: %v", err)
		}

		_, ok := manager.Get("/project")
		if ok {
			t.Error("Mount should be removed")
		}
	})
}

func TestShellCowCommands(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	shell, err := NewShell(afero.NewMemMapFs())
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout, stderr bytes.Buffer
	shell.SetIO(nil, &stdout, &stderr)

	t.Run("cow-mount", func(t *testing.T) {
		stdout.Reset()
		stderr.Reset()

		err := shell.Run(ctx, "cow-mount "+dir+" /project")
		if err != nil {
			t.Fatalf("cow-mount failed: %v (stderr: %s)", err, stderr.String())
		}

		// Should have mounted
		_, ok := shell.GetCowManager().Get("/project")
		if !ok {
			t.Error("Mount was not registered")
		}
	})

	t.Run("cow-status", func(t *testing.T) {
		stdout.Reset()
		stderr.Reset()

		err := shell.Run(ctx, "cow-status /project")
		if err != nil {
			t.Fatalf("cow-status failed: %v", err)
		}

		output := stdout.String()
		if !bytes.Contains([]byte(output), []byte("Mount: /project")) {
			t.Errorf("Unexpected status output: %s", output)
		}
		if !bytes.Contains([]byte(output), []byte("CLEAN")) {
			t.Errorf("Expected CLEAN status: %s", output)
		}
	})

	t.Run("cow-status --json", func(t *testing.T) {
		stdout.Reset()
		stderr.Reset()

		err := shell.Run(ctx, "cow-status /project --json")
		if err != nil {
			t.Fatalf("cow-status --json failed: %v", err)
		}

		output := stdout.String()
		if !bytes.Contains([]byte(output), []byte(`"mount_point": "/project"`)) {
			t.Errorf("Unexpected JSON output: %s", output)
		}
	})

	t.Run("cow-diff with no changes", func(t *testing.T) {
		stdout.Reset()
		stderr.Reset()

		err := shell.Run(ctx, "cow-diff /project")
		if err != nil {
			t.Fatalf("cow-diff failed: %v", err)
		}

		output := stdout.String()
		if !bytes.Contains([]byte(output), []byte("No changes")) {
			t.Errorf("Expected 'No changes': %s", output)
		}
	})

	t.Run("cow-flush to different location", func(t *testing.T) {
		// First modify a file through the shell's memfs
		stdout.Reset()
		stderr.Reset()

		// The files should be synced to memfs, so we can modify them there
		// For this test, we'll just test the flush command format
		targetDir, _ := os.MkdirTemp("", "cow-flush-test-*")
		defer os.RemoveAll(targetDir)

		err := shell.Run(ctx, "cow-flush /project --to "+targetDir)
		if err != nil {
			t.Fatalf("cow-flush failed: %v", err)
		}
	})

	t.Run("cow-reset", func(t *testing.T) {
		stdout.Reset()
		stderr.Reset()

		err := shell.Run(ctx, "cow-reset /project")
		if err != nil {
			t.Fatalf("cow-reset failed: %v", err)
		}

		output := stdout.String()
		if !bytes.Contains([]byte(output), []byte("Reset /project")) {
			t.Errorf("Unexpected reset output: %s", output)
		}
	})

	t.Run("cow-unmount", func(t *testing.T) {
		stdout.Reset()
		stderr.Reset()

		// Force unmount to avoid dirty check issues
		err := shell.Run(ctx, "cow-unmount /project -f")
		if err != nil {
			t.Fatalf("cow-unmount failed: %v", err)
		}

		_, ok := shell.GetCowManager().Get("/project")
		if ok {
			t.Error("Mount should have been removed")
		}
	})
}
