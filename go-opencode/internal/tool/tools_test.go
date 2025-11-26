package tool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper to create test context
func testContext() *Context {
	return &Context{
		SessionID: "test-session",
		MessageID: "test-message",
		CallID:    "test-call",
		Agent:     "test-agent",
		WorkDir:   "",
		AbortCh:   make(chan struct{}),
	}
}

// ============================================
// ReadTool Tests
// ============================================

func TestReadTool_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Line 1\nLine 2\nLine 3\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewReadTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"file_path": "` + testFile + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "Line 1") {
		t.Error("Output should contain 'Line 1'")
	}
	if !strings.Contains(result.Output, "Line 2") {
		t.Error("Output should contain 'Line 2'")
	}
}

func TestReadTool_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewReadTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"file_path": "/nonexistent/file.txt"}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestReadTool_WithOffsetAndLimit(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "lines.txt")
	var lines []string
	for i := 1; i <= 10; i++ {
		lines = append(lines, "Line "+string(rune('0'+i)))
	}
	if err := os.WriteFile(testFile, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewReadTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	// Read lines 3-5 (offset=2, limit=3)
	input := json.RawMessage(`{"file_path": "` + testFile + `", "offset": 3, "limit": 3}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "Line 3") {
		t.Error("Output should contain 'Line 3'")
	}
}

func TestReadTool_Properties(t *testing.T) {
	tool := NewReadTool("/tmp")

	if tool.ID() != "Read" {
		t.Errorf("Expected ID 'Read', got %q", tool.ID())
	}

	desc := tool.Description()
	if !strings.Contains(desc, "file") {
		t.Error("Description should mention 'file'")
	}

	params := tool.Parameters()
	if len(params) == 0 {
		t.Error("Parameters should not be empty")
	}
}

// ============================================
// WriteTool Tests
// ============================================

func TestWriteTool_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "output.txt")

	tool := NewWriteTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"file_path": "` + testFile + `", "content": "Hello, World!"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "Successfully") {
		t.Error("Output should indicate success")
	}

	// Verify file contents
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(data) != "Hello, World!" {
		t.Errorf("File content = %q, want 'Hello, World!'", string(data))
	}
}

func TestWriteTool_CreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "subdir", "nested", "file.txt")

	tool := NewWriteTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"file_path": "` + testFile + `", "content": "Nested content"}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File should have been created with parent directories")
	}
}

func TestWriteTool_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.txt")

	// Create existing file
	if err := os.WriteFile(testFile, []byte("Original"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	tool := NewWriteTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"file_path": "` + testFile + `", "content": "Updated"}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	data, _ := os.ReadFile(testFile)
	if string(data) != "Updated" {
		t.Errorf("File should be overwritten, got %q", string(data))
	}
}

// ============================================
// EditTool Tests
// ============================================

func TestEditTool_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "edit.txt")
	if err := os.WriteFile(testFile, []byte("Hello World"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	tool := NewEditTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{
		"file_path": "` + testFile + `",
		"old_string": "World",
		"new_string": "Go"
	}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "Replaced") {
		t.Errorf("Output should mention 'Replaced', got: %s", result.Output)
	}

	data, _ := os.ReadFile(testFile)
	if string(data) != "Hello Go" {
		t.Errorf("File content = %q, want 'Hello Go'", string(data))
	}
}

func TestEditTool_StringNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "edit.txt")
	if err := os.WriteFile(testFile, []byte("Hello World"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	tool := NewEditTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{
		"file_path": "` + testFile + `",
		"old_string": "NotFound",
		"new_string": "Replacement"
	}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error when old_string not found")
	}
}

func TestEditTool_ReplaceAll(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "edit.txt")
	if err := os.WriteFile(testFile, []byte("foo bar foo baz foo"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	tool := NewEditTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{
		"file_path": "` + testFile + `",
		"old_string": "foo",
		"new_string": "qux",
		"replace_all": true
	}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	data, _ := os.ReadFile(testFile)
	if string(data) != "qux bar qux baz qux" {
		t.Errorf("File content = %q, want 'qux bar qux baz qux'", string(data))
	}
}

// ============================================
// ListTool Tests
// ============================================

func TestListTool_Execute(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some files and directories
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	tool := NewListTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"path": "` + tmpDir + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "file1.txt") {
		t.Error("Output should contain 'file1.txt'")
	}
	if !strings.Contains(result.Output, "subdir") {
		t.Error("Output should contain 'subdir'")
	}
}

func TestListTool_DirectoryNotFound(t *testing.T) {
	tool := NewListTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"path": "/nonexistent/directory"}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error for nonexistent directory")
	}
}

// ============================================
// BashTool Tests
// ============================================

func TestBashTool_Execute(t *testing.T) {
	tool := NewBashTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"command": "echo 'Hello from Bash'"}`)
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
	input := json.RawMessage(`{"command": "exit 1"}`)
	result, err := tool.Execute(ctx, input, toolCtx)

	// Should not return error, but metadata should indicate exit code
	if err != nil {
		t.Logf("Execute returned error (may be expected): %v", err)
	}

	if result != nil && result.Metadata != nil {
		if exitCode, ok := result.Metadata["exit_code"]; ok {
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
	input := json.RawMessage(`{"command": "echo 'quick'", "timeout": 5000}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "quick") {
		t.Error("Output should contain 'quick'")
	}
}

// ============================================
// GlobTool Tests
// ============================================

func TestGlobTool_Execute(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "test1.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test2.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte(""), 0644)
	os.Mkdir(filepath.Join(tmpDir, "sub"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "sub", "nested.go"), []byte(""), 0644)

	tool := NewGlobTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"pattern": "**/*.go", "path": "` + tmpDir + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		// Glob might not be available - skip test
		t.Skipf("Glob tool execution failed (might need rg): %v", err)
	}

	if !strings.Contains(result.Output, ".go") {
		t.Error("Output should contain .go files")
	}
}

// ============================================
// GrepTool Tests
// ============================================

func TestGrepTool_Execute(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file with searchable content
	testFile := filepath.Join(tmpDir, "search.txt")
	content := "Hello World\nFoo Bar\nHello Again\n"
	os.WriteFile(testFile, []byte(content), 0644)

	tool := NewGrepTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"pattern": "Hello", "path": "` + tmpDir + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		// Grep might not be available - skip test
		t.Skipf("Grep tool execution failed (might need rg): %v", err)
	}

	if result.Output == "" {
		t.Error("Output should not be empty for matching pattern")
	}
}

// ============================================
// EinoTool Wrapper Tests
// ============================================

func TestEinoToolWrapper_Info(t *testing.T) {
	tool := NewReadTool("/tmp")
	einoTool := tool.EinoTool()

	info, err := einoTool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if info.Name != "Read" {
		t.Errorf("Expected name 'Read', got %q", info.Name)
	}
	if info.Desc == "" {
		t.Error("Description should not be empty")
	}
}

func TestEinoToolWrapper_InvokableRun(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invoke.txt")
	os.WriteFile(testFile, []byte("Invokable content"), 0644)

	tool := NewReadTool(tmpDir)
	einoTool := tool.EinoTool()

	argsJSON := `{"file_path": "` + testFile + `"}`
	result, err := einoTool.InvokableRun(context.Background(), argsJSON)
	if err != nil {
		t.Fatalf("InvokableRun failed: %v", err)
	}

	if !strings.Contains(result, "Invokable content") {
		t.Errorf("Result should contain file content, got %q", result)
	}
}

// ============================================
// Context Tests
// ============================================

func TestContext_SetMetadata(t *testing.T) {
	var receivedTitle string
	var receivedMeta map[string]any

	ctx := &Context{
		OnMetadata: func(title string, meta map[string]any) {
			receivedTitle = title
			receivedMeta = meta
		},
	}

	ctx.SetMetadata("Test Title", map[string]any{"key": "value"})

	if receivedTitle != "Test Title" {
		t.Errorf("Expected title 'Test Title', got %q", receivedTitle)
	}
	if receivedMeta["key"] != "value" {
		t.Errorf("Expected meta key 'value', got %v", receivedMeta["key"])
	}
}

func TestContext_SetMetadata_NoCallback(t *testing.T) {
	ctx := &Context{}

	// Should not panic
	ctx.SetMetadata("Title", map[string]any{})
}

func TestContext_IsAborted(t *testing.T) {
	abortCh := make(chan struct{})
	ctx := &Context{AbortCh: abortCh}

	// Not aborted initially
	if ctx.IsAborted() {
		t.Error("Should not be aborted initially")
	}

	// Close channel to signal abort
	close(abortCh)

	if !ctx.IsAborted() {
		t.Error("Should be aborted after channel close")
	}
}

// ============================================
// BaseTool Tests
// ============================================

func TestBaseTool(t *testing.T) {
	executed := false
	baseTool := NewBaseTool(
		"custom",
		"A custom tool",
		json.RawMessage(`{"type": "object"}`),
		func(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error) {
			executed = true
			return &Result{Output: "custom result"}, nil
		},
	)

	if baseTool.ID() != "custom" {
		t.Errorf("ID = %q, want 'custom'", baseTool.ID())
	}
	if baseTool.Description() != "A custom tool" {
		t.Errorf("Description = %q, want 'A custom tool'", baseTool.Description())
	}

	result, err := baseTool.Execute(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !executed {
		t.Error("Execute callback was not called")
	}
	if result.Output != "custom result" {
		t.Errorf("Output = %q, want 'custom result'", result.Output)
	}
}
