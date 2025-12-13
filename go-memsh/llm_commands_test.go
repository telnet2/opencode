package memsh

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
)

// TestStatCommand tests the stat command
func TestStatCommand(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(afero.Fs)
		args       string
		wantErr    bool
		checkJSON  func(t *testing.T, output string)
	}{
		{
			name: "stat file",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/test.txt", []byte("hello world"), 0644)
			},
			args:    "stat /test.txt",
			wantErr: false,
			checkJSON: func(t *testing.T, output string) {
				var result StatResult
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("failed to parse JSON: %v", err)
					return
				}
				if result.Name != "test.txt" {
					t.Errorf("expected name 'test.txt', got '%s'", result.Name)
				}
				if result.Size != 11 {
					t.Errorf("expected size 11, got %d", result.Size)
				}
				if result.IsDir {
					t.Error("expected IsDir=false")
				}
			},
		},
		{
			name: "stat directory",
			setup: func(fs afero.Fs) {
				fs.MkdirAll("/mydir", 0755)
			},
			args:    "stat /mydir",
			wantErr: false,
			checkJSON: func(t *testing.T, output string) {
				var result StatResult
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("failed to parse JSON: %v", err)
					return
				}
				if !result.IsDir {
					t.Error("expected IsDir=true")
				}
			},
		},
		{
			name:    "stat non-existent",
			args:    "stat /nonexistent",
			wantErr: true,
		},
		{
			name:    "stat missing operand",
			args:    "stat",
			wantErr: true,
		},
		{
			name: "stat multiple files",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/a.txt", []byte("a"), 0644)
				afero.WriteFile(fs, "/b.txt", []byte("bb"), 0644)
			},
			args:    "stat /a.txt /b.txt",
			wantErr: false,
			checkJSON: func(t *testing.T, output string) {
				var results []StatResult
				if err := json.Unmarshal([]byte(output), &results); err != nil {
					t.Errorf("failed to parse JSON array: %v", err)
					return
				}
				if len(results) != 2 {
					t.Errorf("expected 2 results, got %d", len(results))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.setup != nil {
				tt.setup(fs)
			}

			sh, err := NewShell(fs)
			if err != nil {
				t.Fatalf("NewShell() error = %v", err)
			}

			var stdout, stderr strings.Builder
			sh.SetIO(strings.NewReader(""), &stdout, &stderr)

			ctx := context.Background()
			err = sh.Run(ctx, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.checkJSON != nil && err == nil {
				tt.checkJSON(t, stdout.String())
			}
		})
	}
}

// TestReadfileCommand tests the readfile command
func TestReadfileCommand(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(afero.Fs)
		args     string
		wantErr  bool
		expected string
	}{
		{
			name: "read entire file",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/test.txt", []byte("hello world"), 0644)
			},
			args:     "readfile /test.txt",
			wantErr:  false,
			expected: "hello world",
		},
		{
			name: "read with offset",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/test.txt", []byte("hello world"), 0644)
			},
			args:     "readfile --offset 6 /test.txt",
			wantErr:  false,
			expected: "world",
		},
		{
			name: "read with limit",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/test.txt", []byte("hello world"), 0644)
			},
			args:     "readfile --limit 5 /test.txt",
			wantErr:  false,
			expected: "hello",
		},
		{
			name: "read with offset and limit",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/test.txt", []byte("hello world"), 0644)
			},
			args:     "readfile --offset 3 --limit 5 /test.txt",
			wantErr:  false,
			expected: "lo wo",
		},
		{
			name:    "read non-existent",
			args:    "readfile /nonexistent",
			wantErr: true,
		},
		{
			name: "read directory",
			setup: func(fs afero.Fs) {
				fs.MkdirAll("/mydir", 0755)
			},
			args:    "readfile /mydir",
			wantErr: true,
		},
		{
			name:    "readfile missing operand",
			args:    "readfile",
			wantErr: true,
		},
		{
			name: "read empty file",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/empty.txt", []byte(""), 0644)
			},
			args:     "readfile /empty.txt",
			wantErr:  false,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.setup != nil {
				tt.setup(fs)
			}

			sh, err := NewShell(fs)
			if err != nil {
				t.Fatalf("NewShell() error = %v", err)
			}

			var stdout, stderr strings.Builder
			sh.SetIO(strings.NewReader(""), &stdout, &stderr)

			ctx := context.Background()
			err = sh.Run(ctx, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v, stderr: %s", err, tt.wantErr, stderr.String())
			}

			if !tt.wantErr && stdout.String() != tt.expected {
				t.Errorf("expected output %q, got %q", tt.expected, stdout.String())
			}
		})
	}
}

// TestWritefileCommand tests the writefile command
func TestWritefileCommand(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(afero.Fs)
		script   string // Use piped input for writefile
		wantErr  bool
		verify   func(t *testing.T, fs afero.Fs)
	}{
		{
			name:    "write new file via echo pipe",
			script:  `echo -n "hello world" | writefile /test.txt`,
			wantErr: false,
			verify: func(t *testing.T, fs afero.Fs) {
				content, err := afero.ReadFile(fs, "/test.txt")
				if err != nil {
					t.Errorf("failed to read file: %v", err)
					return
				}
				if string(content) != "hello world" {
					t.Errorf("expected 'hello world', got '%s'", string(content))
				}
			},
		},
		{
			name: "overwrite existing file",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/test.txt", []byte("old content"), 0644)
			},
			script:  `echo -n "new content" | writefile /test.txt`,
			wantErr: false,
			verify: func(t *testing.T, fs afero.Fs) {
				content, err := afero.ReadFile(fs, "/test.txt")
				if err != nil {
					t.Errorf("failed to read file: %v", err)
					return
				}
				if string(content) != "new content" {
					t.Errorf("expected 'new content', got '%s'", string(content))
				}
			},
		},
		{
			name: "append to file",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/test.txt", []byte("hello "), 0644)
			},
			script:  `echo -n "world" | writefile --append /test.txt`,
			wantErr: false,
			verify: func(t *testing.T, fs afero.Fs) {
				content, err := afero.ReadFile(fs, "/test.txt")
				if err != nil {
					t.Errorf("failed to read file: %v", err)
					return
				}
				if string(content) != "hello world" {
					t.Errorf("expected 'hello world', got '%s'", string(content))
				}
			},
		},
		{
			name:    "create with parents",
			script:  `echo -n "content" | writefile --parents /a/b/c/test.txt`,
			wantErr: false,
			verify: func(t *testing.T, fs afero.Fs) {
				exists, _ := afero.Exists(fs, "/a/b/c/test.txt")
				if !exists {
					t.Error("file was not created")
				}
			},
		},
		{
			name:    "missing operand",
			script:  `echo "test" | writefile`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.setup != nil {
				tt.setup(fs)
			}

			sh, err := NewShell(fs)
			if err != nil {
				t.Fatalf("NewShell() error = %v", err)
			}

			var stdout, stderr strings.Builder
			sh.SetIO(strings.NewReader(""), &stdout, &stderr)

			ctx := context.Background()
			err = sh.Run(ctx, tt.script)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v, stderr: %s", err, tt.wantErr, stderr.String())
			}

			if tt.verify != nil && !tt.wantErr {
				tt.verify(t, fs)
			}
		})
	}
}

// TestFindExCommand tests the enhanced find command
func TestFindExCommand(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(afero.Fs)
		args     string
		wantErr  bool
		expected []string
	}{
		{
			name: "find by name",
			setup: func(fs afero.Fs) {
				fs.MkdirAll("/project/src", 0755)
				afero.WriteFile(fs, "/project/src/main.go", []byte(""), 0644)
				afero.WriteFile(fs, "/project/src/util.go", []byte(""), 0644)
				afero.WriteFile(fs, "/project/README.md", []byte(""), 0644)
			},
			args:     "findex /project -name *.go",
			wantErr:  false,
			expected: []string{"/project/src/main.go", "/project/src/util.go"},
		},
		{
			name: "find by type file",
			setup: func(fs afero.Fs) {
				fs.MkdirAll("/test/sub", 0755)
				afero.WriteFile(fs, "/test/file.txt", []byte(""), 0644)
			},
			args:     "findex /test -type f",
			wantErr:  false,
			expected: []string{"/test/file.txt"},
		},
		{
			name: "find by type directory",
			setup: func(fs afero.Fs) {
				fs.MkdirAll("/test/sub1", 0755)
				fs.MkdirAll("/test/sub2", 0755)
				afero.WriteFile(fs, "/test/file.txt", []byte(""), 0644)
			},
			args:     "findex /test -type d",
			wantErr:  false,
			expected: []string{"/test", "/test/sub1", "/test/sub2"},
		},
		{
			name: "find with maxdepth",
			setup: func(fs afero.Fs) {
				fs.MkdirAll("/a/b/c/d", 0755)
				afero.WriteFile(fs, "/a/file1.txt", []byte(""), 0644)
				afero.WriteFile(fs, "/a/b/file2.txt", []byte(""), 0644)
				afero.WriteFile(fs, "/a/b/c/file3.txt", []byte(""), 0644)
			},
			args:     "findex /a -maxdepth 2 -type f",
			wantErr:  false,
			expected: []string{"/a/file1.txt", "/a/b/file2.txt"},
		},
		{
			name: "find empty files",
			setup: func(fs afero.Fs) {
				fs.MkdirAll("/test", 0755)
				afero.WriteFile(fs, "/test/empty.txt", []byte(""), 0644)
				afero.WriteFile(fs, "/test/notempty.txt", []byte("content"), 0644)
			},
			args:     "findex /test -type f -empty",
			wantErr:  false,
			expected: []string{"/test/empty.txt"},
		},
		{
			name: "find case insensitive name",
			setup: func(fs afero.Fs) {
				fs.MkdirAll("/test", 0755)
				afero.WriteFile(fs, "/test/README.md", []byte(""), 0644)
				afero.WriteFile(fs, "/test/readme.txt", []byte(""), 0644)
			},
			args:     "findex /test -iname readme*",
			wantErr:  false,
			expected: []string{"/test/README.md", "/test/readme.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.setup != nil {
				tt.setup(fs)
			}

			sh, err := NewShell(fs)
			if err != nil {
				t.Fatalf("NewShell() error = %v", err)
			}

			var stdout, stderr strings.Builder
			sh.SetIO(strings.NewReader(""), &stdout, &stderr)

			ctx := context.Background()
			err = sh.Run(ctx, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v, stderr: %s", err, tt.wantErr, stderr.String())
			}

			if !tt.wantErr && tt.expected != nil {
				output := strings.TrimSpace(stdout.String())
				lines := strings.Split(output, "\n")
				if len(lines) == 1 && lines[0] == "" {
					lines = []string{}
				}

				// Check that all expected files are found
				found := make(map[string]bool)
				for _, line := range lines {
					found[line] = true
				}

				for _, exp := range tt.expected {
					if !found[exp] {
						t.Errorf("expected to find %s in output, got: %v", exp, lines)
					}
				}
			}
		})
	}
}

// TestGrepExCommand tests the enhanced grep command
func TestGrepExCommand(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(afero.Fs)
		args     string
		wantErr  bool
		contains []string
	}{
		{
			name: "grep basic",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/test.txt", []byte("hello\nworld\nhello world"), 0644)
			},
			args:     "grepex hello /test.txt",
			wantErr:  false,
			contains: []string{"hello", "hello world"},
		},
		{
			name: "grep with line numbers",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/test.txt", []byte("line1\nline2\nline3"), 0644)
			},
			args:     "grepex -n line2 /test.txt",
			wantErr:  false,
			contains: []string{"2:line2"},
		},
		{
			name: "grep case insensitive",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/test.txt", []byte("Hello\nHELLO\nhello"), 0644)
			},
			args:     "grepex -i hello /test.txt",
			wantErr:  false,
			contains: []string{"Hello", "HELLO", "hello"},
		},
		{
			name: "grep invert match",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/test.txt", []byte("apple\nbanana\napricot"), 0644)
			},
			args:     "grepex -v apple /test.txt",
			wantErr:  false,
			contains: []string{"banana"},
		},
		{
			name: "grep count only",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/test.txt", []byte("a\na\nb\na"), 0644)
			},
			args:     "grepex -c a /test.txt",
			wantErr:  false,
			contains: []string{"3"},
		},
		{
			name: "grep files only",
			setup: func(fs afero.Fs) {
				fs.MkdirAll("/dir", 0755)
				afero.WriteFile(fs, "/dir/a.txt", []byte("match"), 0644)
				afero.WriteFile(fs, "/dir/b.txt", []byte("no"), 0644)
			},
			args:     "grepex -l match /dir/a.txt /dir/b.txt",
			wantErr:  false,
			contains: []string{"/dir/a.txt"},
		},
		{
			name: "grep recursive",
			setup: func(fs afero.Fs) {
				fs.MkdirAll("/project/src", 0755)
				afero.WriteFile(fs, "/project/src/main.go", []byte("func main()"), 0644)
				afero.WriteFile(fs, "/project/src/util.go", []byte("func helper()"), 0644)
			},
			args:     "grepex -r func /project",
			wantErr:  false,
			contains: []string{"main", "helper"},
		},
		{
			name: "grep with context",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/test.txt", []byte("line1\nline2\nMATCH\nline4\nline5"), 0644)
			},
			args:     "grepex -B1 -A1 MATCH /test.txt",
			wantErr:  false,
			contains: []string{"line2", "MATCH", "line4"},
		},
		{
			name:    "grep no match",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/test.txt", []byte("hello"), 0644)
			},
			args:    "grepex -q notfound /test.txt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.setup != nil {
				tt.setup(fs)
			}

			sh, err := NewShell(fs)
			if err != nil {
				t.Fatalf("NewShell() error = %v", err)
			}

			var stdout, stderr strings.Builder
			sh.SetIO(strings.NewReader(""), &stdout, &stderr)

			ctx := context.Background()
			err = sh.Run(ctx, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v, stderr: %s", err, tt.wantErr, stderr.String())
			}

			if !tt.wantErr && tt.contains != nil {
				output := stdout.String()
				for _, expected := range tt.contains {
					if !strings.Contains(output, expected) {
						t.Errorf("expected output to contain %q, got: %s", expected, output)
					}
				}
			}
		})
	}
}

// TestExistsCommand tests the exists command
func TestExistsCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(afero.Fs)
		args    string
		wantErr bool
	}{
		{
			name: "file exists",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/test.txt", []byte(""), 0644)
			},
			args:    "exists /test.txt",
			wantErr: false,
		},
		{
			name:    "file not exists",
			args:    "exists /nonexistent",
			wantErr: true,
		},
		{
			name: "directory exists",
			setup: func(fs afero.Fs) {
				fs.MkdirAll("/mydir", 0755)
			},
			args:    "exists /mydir",
			wantErr: false,
		},
		{
			name: "check is directory",
			setup: func(fs afero.Fs) {
				fs.MkdirAll("/mydir", 0755)
			},
			args:    "exists -d /mydir",
			wantErr: false,
		},
		{
			name: "file is not directory",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/test.txt", []byte(""), 0644)
			},
			args:    "exists -d /test.txt",
			wantErr: true,
		},
		{
			name: "check is file",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, "/test.txt", []byte(""), 0644)
			},
			args:    "exists -f /test.txt",
			wantErr: false,
		},
		{
			name: "directory is not file",
			setup: func(fs afero.Fs) {
				fs.MkdirAll("/mydir", 0755)
			},
			args:    "exists -f /mydir",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.setup != nil {
				tt.setup(fs)
			}

			sh, err := NewShell(fs)
			if err != nil {
				t.Fatalf("NewShell() error = %v", err)
			}

			var stdout, stderr strings.Builder
			sh.SetIO(strings.NewReader(""), &stdout, &stderr)

			ctx := context.Background()
			err = sh.Run(ctx, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestStatModTime tests that stat returns correct modification time
func TestStatModTime(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/test.txt", []byte("content"), 0644)

	// Give some time for the file to be created
	time.Sleep(10 * time.Millisecond)

	sh, err := NewShell(fs)
	if err != nil {
		t.Fatalf("NewShell() error = %v", err)
	}

	var stdout, stderr strings.Builder
	sh.SetIO(strings.NewReader(""), &stdout, &stderr)

	ctx := context.Background()
	err = sh.Run(ctx, "stat /test.txt")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var result StatResult
	if err := json.Unmarshal([]byte(stdout.String()), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Parse the modification time
	modTime, err := time.Parse(time.RFC3339, result.ModTime)
	if err != nil {
		t.Fatalf("failed to parse mtime: %v", err)
	}

	// Check that the modification time is reasonable (within last minute)
	if time.Since(modTime) > time.Minute {
		t.Errorf("mtime seems too old: %v", modTime)
	}
}

// TestWritefileReadfileRoundtrip tests that writefile and readfile work together
func TestWritefileReadfileRoundtrip(t *testing.T) {
	fs := afero.NewMemMapFs()

	sh, err := NewShell(fs)
	if err != nil {
		t.Fatalf("NewShell() error = %v", err)
	}

	ctx := context.Background()

	// Write content using echo pipe (the proper way to use writefile)
	var stdout1, stderr1 strings.Builder
	sh.SetIO(strings.NewReader(""), &stdout1, &stderr1)

	// Use echo with -e to preserve newlines
	err = sh.Run(ctx, `echo -e "Hello, World!\nThis is a test." | writefile /roundtrip.txt`)
	if err != nil {
		t.Fatalf("writefile error = %v, stderr: %s", err, stderr1.String())
	}

	// Read content back
	var stdout2, stderr2 strings.Builder
	sh.SetIO(strings.NewReader(""), &stdout2, &stderr2)

	err = sh.Run(ctx, "readfile /roundtrip.txt")
	if err != nil {
		t.Fatalf("readfile error = %v, stderr: %s", err, stderr2.String())
	}

	expected := "Hello, World!\nThis is a test.\n"
	if stdout2.String() != expected {
		t.Errorf("roundtrip failed: expected %q, got %q", expected, stdout2.String())
	}
}

// TestFindExSize tests find with size filter
func TestFindExSize(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create files of different sizes
	afero.WriteFile(fs, "/small.txt", []byte("a"), 0644)           // 1 byte
	afero.WriteFile(fs, "/medium.txt", make([]byte, 1024), 0644)   // 1KB
	afero.WriteFile(fs, "/large.txt", make([]byte, 10240), 0644)   // 10KB

	sh, err := NewShell(fs)
	if err != nil {
		t.Fatalf("NewShell() error = %v", err)
	}

	// Find files larger than 1KB
	var stdout, stderr strings.Builder
	sh.SetIO(strings.NewReader(""), &stdout, &stderr)

	ctx := context.Background()
	err = sh.Run(ctx, "findex / -size +1k -type f")
	if err != nil {
		t.Fatalf("findex error = %v, stderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "large.txt") {
		t.Errorf("expected to find large.txt, got: %s", output)
	}
	if strings.Contains(output, "small.txt") {
		t.Errorf("should not find small.txt, got: %s", output)
	}
}
