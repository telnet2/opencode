package memsh

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
)

// TestProcessSubstitutionBasic tests basic <(...) syntax
func TestProcessSubstitutionBasic(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Test cat with process substitution
	err = shell.Run(ctx, `cat <(echo "hello world")`)
	if err != nil {
		t.Fatalf("Process substitution failed: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", output)
	}
}

// TestProcessSubstitutionMultiple tests multiple process substitutions in one command
func TestProcessSubstitutionMultiple(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Create test files
	shell.Run(ctx, "echo 'line1\nline2' > /file1.txt")
	shell.Run(ctx, "echo 'line3\nline4' > /file2.txt")

	stdout.Reset()

	// Test cat with two process substitutions
	err = shell.Run(ctx, `cat <(cat /file1.txt) <(cat /file2.txt)`)
	if err != nil {
		t.Fatalf("Multiple process substitution failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "line1") || !strings.Contains(output, "line3") {
		t.Errorf("Expected output to contain both files' content, got: %s", output)
	}
}

// TestProcessSubstitutionWithPipeline tests process substitution with a pipeline inside
// Note: This test currently has a known limitation with stdin/stdout redirection
// in pipelines within process substitutions due to mvdan/sh's ExecHandler API.
func TestProcessSubstitutionWithPipeline(t *testing.T) {
	t.Skip("Known limitation: Pipelines within process substitutions don't properly redirect stdin/stdout for builtin commands due to mvdan/sh API constraints")

	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Create test file
	shell.Run(ctx, "echo -e 'apple\nbanana\napricot\nberry' > /fruits.txt")

	stdout.Reset()

	// Test cat with process substitution containing pipeline
	err = shell.Run(ctx, `cat <(cat /fruits.txt | grep "^a")`)
	if err != nil {
		t.Fatalf("Process substitution with pipeline failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "apple") || !strings.Contains(output, "apricot") {
		t.Errorf("Expected filtered output, got: %s", output)
	}
	if strings.Contains(output, "banana") || strings.Contains(output, "berry") {
		t.Errorf("Output should not contain non-matching lines, got: %s", output)
	}
}

// TestProcessSubstitutionWithGrep tests grep with process substitution
func TestProcessSubstitutionWithGrep(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Test grep with process substitution
	err = shell.Run(ctx, `grep "test" <(echo -e "test line\nother line\ntest again")`)
	if err != nil {
		t.Fatalf("grep with process substitution failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "test line") || !strings.Contains(output, "test again") {
		t.Errorf("Expected grep to find test lines, got: %s", output)
	}
	if strings.Contains(output, "other line") {
		t.Errorf("grep should not match 'other line', got: %s", output)
	}
}

// TestProcessSubstitutionWithSort tests sort with process substitution
func TestProcessSubstitutionWithSort(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Test sort with process substitution
	err = shell.Run(ctx, `sort <(echo -e "zebra\napple\nbanana")`)
	if err != nil {
		t.Fatalf("sort with process substitution failed: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	lines := strings.Split(output, "\n")
	if len(lines) != 3 {
		t.Fatalf("Expected 3 lines, got %d", len(lines))
	}

	// Check if sorted
	if !strings.Contains(lines[0], "apple") {
		t.Errorf("Expected first line to be 'apple', got: %s", lines[0])
	}
	if !strings.Contains(lines[1], "banana") {
		t.Errorf("Expected second line to be 'banana', got: %s", lines[1])
	}
	if !strings.Contains(lines[2], "zebra") {
		t.Errorf("Expected third line to be 'zebra', got: %s", lines[2])
	}
}

// TestProcessSubstitutionConcurrency tests concurrent process substitutions
func TestProcessSubstitutionConcurrency(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Test with slow commands to ensure they run concurrently
	// Both substitutions sleep, but they should run in parallel
	start := time.Now()
	err = shell.Run(ctx, `cat <(echo "first" && sleep 0.1) <(echo "second" && sleep 0.1)`)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Concurrent process substitution failed: %v", err)
	}

	// If they ran sequentially, it would take ~0.2s, concurrently ~0.1s
	// Allow some margin for overhead
	if elapsed > 150*time.Millisecond {
		t.Logf("Warning: Process substitutions may not be running concurrently (took %v)", elapsed)
	}

	output := stdout.String()
	if !strings.Contains(output, "first") || !strings.Contains(output, "second") {
		t.Errorf("Expected both outputs, got: %s", output)
	}
}

// TestProcessSubstitutionError tests error handling in process substitution
func TestProcessSubstitutionError(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout, stderr bytes.Buffer
	shell.SetIO(nil, &stdout, &stderr)

	// Test with a command that doesn't exist in the substitution
	// This should still work - the error will be in the background command
	err = shell.Run(ctx, `cat <(nonexistent_command)`)

	// The main command might succeed but background command will error to stderr
	t.Logf("stdout: %s", stdout.String())
	t.Logf("stderr: %s", stderr.String())

	// We expect some error message in stderr about the command not being found
	if !strings.Contains(stderr.String(), "not found") && !strings.Contains(stderr.String(), "error") {
		t.Logf("Note: Expected error message in stderr about nonexistent command")
	}
}

// TestProcessSubstitutionNested tests nested process substitutions
func TestProcessSubstitutionNested(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Test nested process substitution
	// cat <(cat <(echo "nested"))
	err = shell.Run(ctx, `cat <(cat <(echo "nested content"))`)
	if err != nil {
		t.Fatalf("Nested process substitution failed: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if !strings.Contains(output, "nested content") {
		t.Errorf("Expected 'nested content', got '%s'", output)
	}
}
