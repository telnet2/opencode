package memsh

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

// TestImportFileEdgeCases tests import-file edge cases
func TestImportFileEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(string) error
		cleanup func(string)
		script  func(string) string
		wantErr bool
	}{
		{
			name: "import non-existent file",
			script: func(tmpDir string) string {
				return "import-file " + filepath.Join(tmpDir, "nonexistent.txt") + " /dest.txt"
			},
			wantErr: true,
		},
		{
			name: "import valid file",
			setup: func(tmpDir string) error {
				return os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test content"), 0644)
			},
			script: func(tmpDir string) string {
				return "import-file " + filepath.Join(tmpDir, "test.txt") + " /imported.txt && cat /imported.txt"
			},
			wantErr: false,
		},
		{
			name: "import to nested path",
			setup: func(tmpDir string) error {
				return os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("content"), 0644)
			},
			script: func(tmpDir string) string {
				return "import-file " + filepath.Join(tmpDir, "test.txt") + " /a/b/c/imported.txt && cat /a/b/c/imported.txt"
			},
			wantErr: false,
		},
		{
			name: "import directory as file",
			setup: func(tmpDir string) error {
				return os.Mkdir(filepath.Join(tmpDir, "dir"), 0755)
			},
			script: func(tmpDir string) string {
				return "import-file " + filepath.Join(tmpDir, "dir") + " /dest.txt"
			},
			wantErr: true,
		},
		{
			name: "import empty file",
			setup: func(tmpDir string) error {
				return os.WriteFile(filepath.Join(tmpDir, "empty.txt"), []byte(""), 0644)
			},
			script: func(tmpDir string) string {
				return "import-file " + filepath.Join(tmpDir, "empty.txt") + " /empty.txt && wc /empty.txt"
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "memsh-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if tt.setup != nil {
				if err := tt.setup(tmpDir); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			fs := afero.NewMemMapFs()
			sh, err := NewShell(fs)
			if err != nil {
				t.Fatalf("NewShell() error = %v", err)
			}

			var stdout strings.Builder
			sh.SetIO(strings.NewReader(""), &stdout, &stdout)

			ctx := context.Background()
			err = sh.Run(ctx, tt.script(tmpDir))
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.cleanup != nil {
				tt.cleanup(tmpDir)
			}
		})
	}
}

// TestImportDirEdgeCases tests import-dir edge cases
func TestImportDirEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(string) error
		script  func(string) string
		verify  string
		wantErr bool
	}{
		{
			name: "import non-existent directory",
			script: func(tmpDir string) string {
				return "import-dir " + filepath.Join(tmpDir, "nonexistent") + " /dest"
			},
			wantErr: true,
		},
		{
			name: "import empty directory",
			setup: func(tmpDir string) error {
				return os.Mkdir(filepath.Join(tmpDir, "empty"), 0755)
			},
			script: func(tmpDir string) string {
				return "import-dir " + filepath.Join(tmpDir, "empty") + " /imported"
			},
			verify:  "[ -d /imported ]",
			wantErr: false,
		},
		{
			name: "import directory with files",
			setup: func(tmpDir string) error {
				dirPath := filepath.Join(tmpDir, "testdir")
				if err := os.Mkdir(dirPath, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(dirPath, "file1.txt"), []byte("content1"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dirPath, "file2.txt"), []byte("content2"), 0644)
			},
			script: func(tmpDir string) string {
				return "import-dir " + filepath.Join(tmpDir, "testdir") + " /imported"
			},
			verify:  "ls /imported | wc -l",
			wantErr: false,
		},
		{
			name: "import nested directory",
			setup: func(tmpDir string) error {
				dirPath := filepath.Join(tmpDir, "parent", "child")
				if err := os.MkdirAll(dirPath, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dirPath, "nested.txt"), []byte("nested"), 0644)
			},
			script: func(tmpDir string) string {
				return "import-dir " + filepath.Join(tmpDir, "parent") + " /imported"
			},
			verify:  "cat /imported/child/nested.txt",
			wantErr: false,
		},
		{
			name: "import file as directory",
			setup: func(tmpDir string) error {
				return os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("content"), 0644)
			},
			script: func(tmpDir string) string {
				return "import-dir " + filepath.Join(tmpDir, "file.txt") + " /dest"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "memsh-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if tt.setup != nil {
				if err := tt.setup(tmpDir); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			fs := afero.NewMemMapFs()
			sh, err := NewShell(fs)
			if err != nil {
				t.Fatalf("NewShell() error = %v", err)
			}

			var stdout strings.Builder
			sh.SetIO(strings.NewReader(""), &stdout, &stdout)

			ctx := context.Background()
			err = sh.Run(ctx, tt.script(tmpDir))
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify if provided
			if !tt.wantErr && tt.verify != "" {
				stdout.Reset()
				if err := sh.Run(ctx, tt.verify); err != nil {
					t.Errorf("Verify command failed: %v", err)
				}
			}
		})
	}
}

// TestExportFileEdgeCases tests export-file edge cases
func TestExportFileEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		setup   string
		script  func(string) string
		verify  func(string) error
		wantErr bool
	}{
		{
			name:  "export non-existent file",
			script: func(tmpDir string) string {
				return "export-file /nonexistent.txt " + filepath.Join(tmpDir, "out.txt")
			},
			wantErr: true,
		},
		{
			name:  "export valid file",
			setup: "echo 'test content' > /test.txt",
			script: func(tmpDir string) string {
				return "export-file /test.txt " + filepath.Join(tmpDir, "exported.txt")
			},
			verify: func(tmpDir string) error {
				content, err := os.ReadFile(filepath.Join(tmpDir, "exported.txt"))
				if err != nil {
					return err
				}
				if strings.TrimSpace(string(content)) != "test content" {
					return os.ErrInvalid
				}
				return nil
			},
			wantErr: false,
		},
		{
			name:  "export to nested path",
			setup: "echo 'content' > /test.txt",
			script: func(tmpDir string) string {
				return "export-file /test.txt " + filepath.Join(tmpDir, "a", "b", "c", "exported.txt")
			},
			verify: func(tmpDir string) error {
				_, err := os.Stat(filepath.Join(tmpDir, "a", "b", "c", "exported.txt"))
				return err
			},
			wantErr: false,
		},
		{
			name:  "export directory as file",
			setup: "mkdir /testdir",
			script: func(tmpDir string) string {
				return "export-file /testdir " + filepath.Join(tmpDir, "out.txt")
			},
			wantErr: true,
		},
		{
			name:  "export empty file",
			setup: "touch /empty.txt",
			script: func(tmpDir string) string {
				return "export-file /empty.txt " + filepath.Join(tmpDir, "empty.txt")
			},
			verify: func(tmpDir string) error {
				info, err := os.Stat(filepath.Join(tmpDir, "empty.txt"))
				if err != nil {
					return err
				}
				if info.Size() != 0 {
					return os.ErrInvalid
				}
				return nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "memsh-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			fs := afero.NewMemMapFs()
			sh, err := NewShell(fs)
			if err != nil {
				t.Fatalf("NewShell() error = %v", err)
			}

			ctx := context.Background()
			if tt.setup != "" {
				if err := sh.Run(ctx, tt.setup); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			var stdout strings.Builder
			sh.SetIO(strings.NewReader(""), &stdout, &stdout)

			err = sh.Run(ctx, tt.script(tmpDir))
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify if provided
			if !tt.wantErr && tt.verify != nil {
				if err := tt.verify(tmpDir); err != nil {
					t.Errorf("Verification failed: %v", err)
				}
			}
		})
	}
}

// TestExportDirEdgeCases tests export-dir edge cases
func TestExportDirEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		setup   string
		script  func(string) string
		verify  func(string) error
		wantErr bool
	}{
		{
			name: "export non-existent directory",
			script: func(tmpDir string) string {
				return "export-dir /nonexistent " + filepath.Join(tmpDir, "out")
			},
			wantErr: true,
		},
		{
			name:  "export empty directory",
			setup: "mkdir /empty",
			script: func(tmpDir string) string {
				return "export-dir /empty " + filepath.Join(tmpDir, "exported")
			},
			verify: func(tmpDir string) error {
				info, err := os.Stat(filepath.Join(tmpDir, "exported"))
				if err != nil {
					return err
				}
				if !info.IsDir() {
					return os.ErrInvalid
				}
				return nil
			},
			wantErr: false,
		},
		{
			name:  "export directory with files",
			setup: "mkdir /test && echo 'file1' > /test/f1.txt && echo 'file2' > /test/f2.txt",
			script: func(tmpDir string) string {
				return "export-dir /test " + filepath.Join(tmpDir, "exported")
			},
			verify: func(tmpDir string) error {
				// Check if both files exist
				if _, err := os.Stat(filepath.Join(tmpDir, "exported", "f1.txt")); err != nil {
					return err
				}
				if _, err := os.Stat(filepath.Join(tmpDir, "exported", "f2.txt")); err != nil {
					return err
				}
				return nil
			},
			wantErr: false,
		},
		{
			name:  "export nested directory",
			setup: "mkdir -p /parent/child && echo 'nested' > /parent/child/file.txt",
			script: func(tmpDir string) string {
				return "export-dir /parent " + filepath.Join(tmpDir, "exported")
			},
			verify: func(tmpDir string) error {
				content, err := os.ReadFile(filepath.Join(tmpDir, "exported", "child", "file.txt"))
				if err != nil {
					return err
				}
				if strings.TrimSpace(string(content)) != "nested" {
					return os.ErrInvalid
				}
				return nil
			},
			wantErr: false,
		},
		{
			name:  "export file as directory",
			setup: "echo 'content' > /file.txt",
			script: func(tmpDir string) string {
				return "export-dir /file.txt " + filepath.Join(tmpDir, "out")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "memsh-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			fs := afero.NewMemMapFs()
			sh, err := NewShell(fs)
			if err != nil {
				t.Fatalf("NewShell() error = %v", err)
			}

			ctx := context.Background()
			if tt.setup != "" {
				if err := sh.Run(ctx, tt.setup); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			var stdout strings.Builder
			sh.SetIO(strings.NewReader(""), &stdout, &stdout)

			err = sh.Run(ctx, tt.script(tmpDir))
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify if provided
			if !tt.wantErr && tt.verify != nil {
				if err := tt.verify(tmpDir); err != nil {
					t.Errorf("Verification failed: %v", err)
				}
			}
		})
	}
}
