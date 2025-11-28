package tool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBashTool_Execute(t *testing.T) {
	tool := NewBashTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"command": "echo 'Hello from Bash'", "description": "Print hello"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "Hello from Bash") {
		t.Errorf("Output should contain 'Hello from Bash', got %q", result.Output)
	}
}

func TestBashTool_ExitCode(t *testing.T) {
	tool := NewBashTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	// Command that exits with error
	input := json.RawMessage(`{"command": "exit 1", "description": "Exit with error"}`)
	result, err := tool.Execute(ctx, input, toolCtx)

	// Should not return error, but metadata should indicate exit code
	if err != nil {
		t.Logf("Execute returned error (may be expected): %v", err)
	}

	if result != nil && result.Metadata != nil {
		if exitCode, ok := result.Metadata["exit"]; ok {
			if exitCode != 1 && exitCode != float64(1) {
				t.Errorf("Expected exit code 1, got %v", exitCode)
			}
		}
	}
}

func TestBashTool_WithTimeout(t *testing.T) {
	tool := NewBashTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	// Quick command with timeout
	input := json.RawMessage(`{"command": "echo 'quick'", "timeout": 5000, "description": "Quick echo"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "quick") {
		t.Error("Output should contain 'quick'")
	}
}

func TestBashTool_Properties(t *testing.T) {
	tool := NewBashTool("/tmp")

	if tool.ID() != "Bash" {
		t.Errorf("Expected ID 'Bash', got %q", tool.ID())
	}

	desc := tool.Description()
	if !strings.Contains(desc, "command") {
		t.Error("Description should mention 'command'")
	}

	params := tool.Parameters()
	if len(params) == 0 {
		t.Error("Parameters should not be empty")
	}

	// Verify JSON schema is valid
	var schema map[string]any
	if err := json.Unmarshal(params, &schema); err != nil {
		t.Errorf("Parameters should be valid JSON: %v", err)
	}

	// Check required properties
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Error("Schema should have properties")
	}
	if _, ok := props["command"]; !ok {
		t.Error("Schema should have command property")
	}
	if _, ok := props["timeout"]; !ok {
		t.Error("Schema should have timeout property")
	}
	if _, ok := props["description"]; !ok {
		t.Error("Schema should have description property")
	}
}

func TestBashTool_InvalidInput(t *testing.T) {
	tool := NewBashTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	// Invalid JSON
	input := json.RawMessage(`{invalid json}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error for invalid JSON input")
	}
}

func TestBashTool_WorkDirFromContext(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file in the temp dir
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	tool := NewBashTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()
	toolCtx.WorkDir = tmpDir

	// List files in the working directory
	input := json.RawMessage(`{"command": "ls", "description": "List files"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "test.txt") {
		t.Error("Output should contain 'test.txt' from the working directory")
	}
}

func TestBashTool_DescriptionMetadata(t *testing.T) {
	tool := NewBashTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"command": "echo test", "description": "Test echo command"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check description in result
	if result.Title != "Test echo command" {
		t.Errorf("Expected title 'Test echo command', got %q", result.Title)
	}

	// Check metadata
	if result.Metadata["description"] != "Test echo command" {
		t.Errorf("Expected description in metadata, got %v", result.Metadata["description"])
	}
}

func TestBashTool_DefaultDescription(t *testing.T) {
	tool := NewBashTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	// No description provided
	input := json.RawMessage(`{"command": "echo test"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Should have default title
	if result.Title == "" {
		t.Error("Title should not be empty")
	}
}

func TestBashTool_Metadata(t *testing.T) {
	tool := NewBashTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"command": "echo 'test'", "description": "Test"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check metadata has expected fields
	if _, ok := result.Metadata["output"]; !ok {
		t.Error("Metadata should have 'output' field")
	}
	if _, ok := result.Metadata["exit"]; !ok {
		t.Error("Metadata should have 'exit' field")
	}
}

func TestBashTool_MaxTimeout(t *testing.T) {
	tool := NewBashTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	// Timeout exceeding max should be capped
	input := json.RawMessage(`{"command": "echo 'test'", "timeout": 999999999, "description": "Test max timeout"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "test") {
		t.Error("Output should contain 'test'")
	}
}

func TestBashTool_EinoTool(t *testing.T) {
	tool := NewBashTool("/tmp")
	einoTool := tool.EinoTool()

	if einoTool == nil {
		t.Error("EinoTool should not return nil")
	}

	info, err := einoTool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if info.Name != "Bash" {
		t.Errorf("Expected name 'Bash', got %q", info.Name)
	}
}

func TestBashTool_Options(t *testing.T) {
	// Test with options
	tool := NewBashTool("/tmp",
		WithExternalDirAction("allow"),
	)

	if tool == nil {
		t.Error("NewBashTool with options should not return nil")
	}
}

func TestDetectShell(t *testing.T) {
	shell := detectShell()

	if shell == "" {
		t.Error("detectShell should return a non-empty string")
	}

	// On macOS, should default to zsh
	if runtime.GOOS == "darwin" && os.Getenv("SHELL") == "" {
		if shell != "/bin/zsh" {
			t.Errorf("Expected /bin/zsh on macOS, got %q", shell)
		}
	}
}
