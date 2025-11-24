package memsh

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

// TestGrepEdgeCases tests grep command edge cases
func TestGrepEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    string
		script   string
		expected string
		wantErr  bool
	}{
		{
			name:     "grep pattern not found",
			setup:    "echo 'hello world' > /f.txt",
			script:   "grep xyz /f.txt",
			expected: "",
		},
		{
			name:     "grep case insensitive",
			setup:    "echo 'HELLO' > /f.txt",
			script:   "grep -i hello /f.txt",
			expected: "/f.txt:HELLO",
		},
		{
			name:     "grep with line numbers",
			setup:    "echo -e 'line1\\nmatch\\nline3' > /f.txt",
			script:   "grep -n match /f.txt",
			expected: "/f.txt:2:match",
		},
		{
			name:     "grep count",
			setup:    "echo -e 'test\\ntest\\nother' > /f.txt",
			script:   "grep -c test /f.txt",
			expected: "/f.txt:2",
		},
		{
			name:     "grep invert match",
			setup:    "echo -e 'keep\\nremove\\nkeep' > /f.txt",
			script:   "grep -v remove /f.txt",
			expected: "/f.txt:keep\n/f.txt:keep",
		},
		{
			name:     "grep non-existent file",
			script:   "grep test /nonexistent",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "grep empty pattern",
			setup:    "echo 'test' > /f.txt",
			script:   "grep '' /f.txt",
			expected: "/f.txt:test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			err = sh.Run(ctx, tt.script)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				output := strings.TrimSpace(stdout.String())
				if output != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, output)
				}
			}
		})
	}
}

// TestHeadTailEdgeCases tests head and tail edge cases
func TestHeadTailEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    string
		script   string
		expected string
	}{
		{
			name:     "head default 10 lines",
			setup:    "echo '1' > /f.txt && echo '2' >> /f.txt && echo '3' >> /f.txt && echo '4' >> /f.txt && echo '5' >> /f.txt && echo '6' >> /f.txt && echo '7' >> /f.txt && echo '8' >> /f.txt && echo '9' >> /f.txt && echo '10' >> /f.txt && echo '11' >> /f.txt && echo '12' >> /f.txt",
			script:   "head /f.txt",
			expected: "1\n2\n3\n4\n5\n6\n7\n8\n9\n10",
		},
		{
			name:     "head more than available",
			setup:    "echo '1' > /f.txt && echo '2' >> /f.txt && echo '3' >> /f.txt",
			script:   "head -20 /f.txt",
			expected: "1\n2\n3",
		},
		{
			name:     "head zero lines",
			setup:    "for i in 1 2 3; do echo $i; done > /f.txt",
			script:   "head -0 /f.txt",
			expected: "",
		},
		{
			name:     "tail default 10 lines",
			setup:    "echo '1' > /f.txt && echo '2' >> /f.txt && echo '3' >> /f.txt && echo '4' >> /f.txt && echo '5' >> /f.txt && echo '6' >> /f.txt && echo '7' >> /f.txt && echo '8' >> /f.txt && echo '9' >> /f.txt && echo '10' >> /f.txt && echo '11' >> /f.txt && echo '12' >> /f.txt",
			script:   "tail /f.txt",
			expected: "3\n4\n5\n6\n7\n8\n9\n10\n11\n12",
		},
		{
			name:     "tail more than available",
			setup:    "for i in 1 2 3; do echo $i; done > /f.txt",
			script:   "tail -20 /f.txt",
			expected: "1\n2\n3",
		},
		{
			name:     "tail single line file",
			setup:    "echo 'single' > /f.txt",
			script:   "tail /f.txt",
			expected: "single",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			if err := sh.Run(ctx, tt.script); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			output := strings.TrimSpace(stdout.String())
			if output != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, output)
			}
		})
	}
}

// TestSortEdgeCases tests sort command edge cases
func TestSortEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    string
		script   string
		expected string
	}{
		{
			name:     "sort already sorted",
			setup:    "echo -e 'a\\nb\\nc' > /f.txt",
			script:   "sort /f.txt",
			expected: "a\nb\nc",
		},
		{
			name:     "sort reverse",
			setup:    "echo -e 'a\\nb\\nc' > /f.txt",
			script:   "sort -r /f.txt",
			expected: "c\nb\na",
		},
		{
			name:     "sort with duplicates",
			setup:    "echo -e 'b\\na\\nb\\na' > /f.txt",
			script:   "sort /f.txt",
			expected: "a\na\nb\nb",
		},
		{
			name:     "sort unique",
			setup:    "echo -e 'b\\na\\nb\\na' > /f.txt",
			script:   "sort -u /f.txt",
			expected: "a\nb",
		},
		{
			name:     "sort numeric",
			setup:    "echo -e '10\\n2\\n100\\n20' > /f.txt",
			script:   "sort -n /f.txt",
			expected: "2\n10\n20\n100",
		},
		{
			name:     "sort numeric reverse",
			setup:    "echo -e '10\\n2\\n100\\n20' > /f.txt",
			script:   "sort -nr /f.txt",
			expected: "100\n20\n10\n2",
		},
		{
			name:     "sort single line",
			setup:    "echo 'single' > /f.txt",
			script:   "sort /f.txt",
			expected: "single",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			if err := sh.Run(ctx, tt.script); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			output := strings.TrimSpace(stdout.String())
			if output != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, output)
			}
		})
	}
}

// TestUniqEdgeCases tests uniq command edge cases
func TestUniqEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    string
		script   string
		expected string
	}{
		{
			name:     "uniq no duplicates",
			setup:    "echo -e 'a\\nb\\nc' > /f.txt",
			script:   "uniq /f.txt",
			expected: "a\nb\nc",
		},
		{
			name:     "uniq consecutive duplicates",
			setup:    "echo -e 'a\\na\\nb\\nb\\nb\\nc' > /f.txt",
			script:   "uniq /f.txt",
			expected: "a\nb\nc",
		},
		{
			name:     "uniq with count",
			setup:    "echo -e 'a\\na\\nb\\nb\\nb\\nc' > /f.txt",
			script:   "uniq -c /f.txt",
			expected: "2 a\n      3 b\n      1 c",
		},
		{
			name:     "uniq non-consecutive duplicates",
			setup:    "echo -e 'a\\nb\\na\\nb' > /f.txt",
			script:   "uniq /f.txt",
			expected: "a\nb\na\nb",
		},
		{
			name:     "uniq all same",
			setup:    "echo -e 'a\\na\\na\\na' > /f.txt",
			script:   "uniq /f.txt",
			expected: "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			if err := sh.Run(ctx, tt.script); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			output := strings.TrimSpace(stdout.String())
			if output != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, output)
			}
		})
	}
}

// TestWcEdgeCases tests wc command edge cases
func TestWcEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    string
		script   string
		expected string
	}{
		{
			name:     "wc empty file",
			setup:    "touch /empty.txt",
			script:   "wc /empty.txt",
			expected: "0       0       0 /empty.txt",
		},
		{
			name:     "wc single word",
			setup:    "echo 'word' > /f.txt",
			script:   "wc /f.txt",
			expected: "1       1       5 /f.txt",
		},
		{
			name:     "wc lines only",
			setup:    "for i in 1 2 3; do echo line$i; done > /f.txt",
			script:   "wc -l /f.txt",
			expected: "3 /f.txt",
		},
		{
			name:     "wc words only",
			setup:    "echo 'one two three' > /f.txt",
			script:   "wc -w /f.txt",
			expected: "3 /f.txt",
		},
		{
			name:     "wc bytes only",
			setup:    "echo 'test' > /f.txt",
			script:   "wc -c /f.txt",
			expected: "5 /f.txt",
		},
		{
			name:     "wc multiple files",
			setup:    "echo 'a' > /f1.txt && echo 'b' > /f2.txt",
			script:   "wc /f1.txt /f2.txt",
			expected: "1       1       2 /f1.txt\n      1       1       2 /f2.txt\n      2       2       4 total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			if err := sh.Run(ctx, tt.script); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			output := strings.TrimSpace(stdout.String())
			if output != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, output)
			}
		})
	}
}

// TestFindEdgeCases tests find command edge cases
func TestFindEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    string
		script   string
		contains []string
		excludes []string
	}{
		{
			name:     "find empty directory",
			setup:    "mkdir /empty",
			script:   "find /empty",
			contains: []string{"/empty"},
		},
		{
			name:     "find files only",
			setup:    "mkdir /test && touch /test/f.txt && mkdir /test/subdir",
			script:   "find /test -type f",
			contains: []string{"/test/f.txt"},
			excludes: []string{"/test/subdir"},
		},
		{
			name:     "find directories only",
			setup:    "mkdir -p /test/subdir && touch /test/f.txt",
			script:   "find /test -type d",
			contains: []string{"/test", "/test/subdir"},
			excludes: []string{"/test/f.txt"},
		},
		{
			name:     "find by name pattern",
			setup:    "mkdir /test && touch /test/a.txt /test/b.log /test/c.txt",
			script:   "find /test -name '*.txt'",
			contains: []string{"/test/a.txt", "/test/c.txt"},
			excludes: []string{"/test/b.log"},
		},
		{
			name:     "find nested files",
			setup:    "mkdir -p /a/b/c && touch /a/f1.txt /a/b/f2.txt /a/b/c/f3.txt",
			script:   "find /a -name '*.txt'",
			contains: []string{"/a/f1.txt", "/a/b/f2.txt", "/a/b/c/f3.txt"},
		},
		{
			name:     "find non-existent path",
			script:   "find /nonexistent",
			contains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			// find might error on non-existent paths, that's ok
			sh.Run(ctx, tt.script)

			output := stdout.String()

			// Check that all expected items are present
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Output should contain %q, got: %q", expected, output)
				}
			}

			// Check that excluded items are not present
			for _, excluded := range tt.excludes {
				if strings.Contains(output, excluded) {
					t.Errorf("Output should not contain %q, got: %q", excluded, output)
				}
			}
		})
	}
}
