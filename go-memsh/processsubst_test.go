package memsh

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func TestVirtualPipe(t *testing.T) {
	pipe := NewVirtualPipe(3)

	// Write data to the pipe
	data := []byte("Hello, World!")
	n, err := pipe.Write(data)
	if err != nil {
		t.Fatalf("Failed to write to pipe: %v", err)
	}
	if n != len(data) {
		t.Fatalf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// Mark as done
	pipe.Done()

	// Read data from the pipe
	vf := NewVirtualFile(pipe)
	buf := make([]byte, 100)
	n, err = vf.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}

	if string(buf[:n]) != string(data) {
		t.Fatalf("Expected '%s', got '%s'", string(data), string(buf[:n]))
	}
}

func TestPipeManager(t *testing.T) {
	pm := NewPipeManager()

	// Create a pipe
	pipe1 := pm.CreatePipe()
	if pipe1.id != 3 {
		t.Fatalf("Expected first pipe ID to be 3, got %d", pipe1.id)
	}

	// Create another pipe
	pipe2 := pm.CreatePipe()
	if pipe2.id != 4 {
		t.Fatalf("Expected second pipe ID to be 4, got %d", pipe2.id)
	}

	// Retrieve pipe
	retrieved, ok := pm.GetPipe(3)
	if !ok {
		t.Fatal("Failed to retrieve pipe")
	}
	if retrieved != pipe1 {
		t.Fatal("Retrieved pipe is not the same as created pipe")
	}

	// Close pipe
	pm.ClosePipe(3)
	_, ok = pm.GetPipe(3)
	if ok {
		t.Fatal("Pipe should have been removed after close")
	}
}

func TestProcessSubstitutionExecution(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()

	// Create a pipe for process substitution
	pipe := shell.pipeManager.CreatePipe()

	// Create a process substitution
	ps := &ProcessSubstitution{
		Command: "echo 'line1\nline2\nline3'",
		IsInput: true,
		Pipe:    pipe,
	}

	// Execute in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- ps.ExecuteInBackground(ctx, shell)
	}()

	// Wait for completion
	pipe.Wait()

	// Check for errors
	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("Process substitution failed: %v", err)
		}
	default:
	}

	// Read the output
	contents := pipe.GetContents()
	output := string(contents)

	if !strings.Contains(output, "line1") {
		t.Fatalf("Expected output to contain 'line1', got: %s", output)
	}
}

func TestVirtualFileDevFd(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()

	// Create a pipe and add data
	pipe := shell.pipeManager.CreatePipe()
	pipe.Write([]byte("test data\n"))
	pipe.Done()

	// Try to open /dev/fd/N
	file, err := shell.openHandler(ctx, pipe.GetPath(), 0, 0)
	if err != nil {
		t.Fatalf("Failed to open virtual /dev/fd path: %v", err)
	}
	defer file.Close()

	// Read from it
	buf := make([]byte, 100)
	n, err := file.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from virtual file: %v", err)
	}

	output := string(buf[:n])
	if !strings.Contains(output, "test data") {
		t.Fatalf("Expected 'test data', got: %s", output)
	}
}

func TestManualProcessSubstitution(t *testing.T) {
	// This test demonstrates how to manually use process substitution
	// until we implement automatic detection

	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	var stdout bytes.Buffer
	shell.SetIO(shell.stdin, &stdout, shell.stderr)

	ctx := context.Background()

	// Simulate: diff <(echo "content1") <(echo "content2")

	// Create first process substitution
	pipe1 := shell.pipeManager.CreatePipe()
	ps1 := &ProcessSubstitution{
		Command: "echo 'content1'",
		IsInput: true,
		Pipe:    pipe1,
	}
	go ps1.ExecuteInBackground(ctx, shell)

	// Create second process substitution
	pipe2 := shell.pipeManager.CreatePipe()
	ps2 := &ProcessSubstitution{
		Command: "echo 'content2'",
		IsInput: true,
		Pipe:    pipe2,
	}
	go ps2.ExecuteInBackground(ctx, shell)

	// Wait for both to complete
	pipe1.Wait()
	pipe2.Wait()

	// Now we can use these pipes
	// For now, just verify they have content
	if len(pipe1.GetContents()) == 0 {
		t.Fatal("Pipe 1 should have content")
	}
	if len(pipe2.GetContents()) == 0 {
		t.Fatal("Pipe 2 should have content")
	}

	t.Logf("Pipe1 path: %s, content: %s", pipe1.GetPath(), string(pipe1.GetContents()))
	t.Logf("Pipe2 path: %s, content: %s", pipe2.GetPath(), string(pipe2.GetContents()))
}

func TestProcessSubstitutionInCommand(t *testing.T) {
	t.Skip("Full automatic process substitution not yet implemented - requires mvdan/sh integration")

	// This test will work once we fully integrate process substitution
	// For now, it's skipped as a placeholder for future implementation

	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	var stdout bytes.Buffer
	shell.SetIO(shell.stdin, &stdout, shell.stderr)

	ctx := context.Background()

	// This should work: diff <(echo "a") <(echo "b")
	err = shell.Run(ctx, `diff <(echo "a") <(echo "b")`)
	// Note: diff command doesn't exist yet, but the process substitution should be parsed

	if err != nil {
		// Expected to fail since diff doesn't exist
		// But should fail with "diff: command not found", not parse error
		if !strings.Contains(err.Error(), "command not found") {
			t.Logf("Error (expected): %v", err)
		}
	}
}
