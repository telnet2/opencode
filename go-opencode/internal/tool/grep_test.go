package tool

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func hasRg() bool {
	_, err := exec.LookPath("rg")
	return err == nil
}

func TestGrepTool_Execute(t *testing.T) {
	if !hasRg() {
		t.Skip("ripgrep (rg) not installed")
	}

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
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Output == "" {
		t.Error("Output should not be empty for matching pattern")
	}
	if !strings.Contains(result.Output, "Hello") {
		t.Error("Output should contain matches")
	}
}

func TestGrepTool_NoMatches(t *testing.T) {
	if !hasRg() {
		t.Skip("ripgrep (rg) not installed")
	}

	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "search.txt")
	content := "Hello World\nFoo Bar\n"
	os.WriteFile(testFile, []byte(content), 0644)

	tool := NewGrepTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	// Search for non-existent pattern
	input := json.RawMessage(`{"pattern": "NonExistent", "path": "` + tmpDir + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Should succeed but indicate no matches
	if result.Metadata["count"] != 0 {
		t.Errorf("Expected 0 matches, got %v", result.Metadata["count"])
	}
	if !strings.Contains(result.Output, "No matches") {
		t.Error("Output should indicate no matches")
	}
}

func TestGrepTool_WithGlobFilter(t *testing.T) {
	if !hasRg() {
		t.Skip("ripgrep (rg) not installed")
	}

	tmpDir := t.TempDir()

	// Create files with different extensions
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte("Hello from Go"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("Hello from TXT"), 0644)

	tool := NewGrepTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	// Search only in .go files
	input := json.RawMessage(`{"pattern": "Hello", "path": "` + tmpDir + `", "glob": "*.go"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "Go") {
		t.Error("Output should contain match from .go file")
	}
	if strings.Contains(result.Output, "TXT") {
		t.Error("Output should not contain match from .txt file")
	}
}

func TestGrepTool_Properties(t *testing.T) {
	tool := NewGrepTool("/tmp")

	if tool.ID() != "Grep" {
		t.Errorf("Expected ID 'Grep', got %q", tool.ID())
	}

	desc := tool.Description()
	if !strings.Contains(desc, "search") {
		t.Error("Description should mention 'search'")
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
	if _, ok := props["pattern"]; !ok {
		t.Error("Schema should have pattern property")
	}
	if _, ok := props["path"]; !ok {
		t.Error("Schema should have path property")
	}
	if _, ok := props["glob"]; !ok {
		t.Error("Schema should have glob property")
	}
}

func TestGrepTool_InvalidInput(t *testing.T) {
	tool := NewGrepTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	// Invalid JSON
	input := json.RawMessage(`{invalid json}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error for invalid JSON input")
	}
}

func TestGrepTool_DefaultPath(t *testing.T) {
	if !hasRg() {
		t.Skip("ripgrep (rg) not installed")
	}

	tmpDir := t.TempDir()

	// Create a file in tmpDir
	os.WriteFile(filepath.Join(tmpDir, "default.txt"), []byte("searchable content"), 0644)

	tool := NewGrepTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()
	toolCtx.WorkDir = tmpDir

	// No path specified - should use workDir
	input := json.RawMessage(`{"pattern": "searchable"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "searchable") {
		t.Error("Output should contain 'searchable'")
	}
}

func TestGrepTool_LineNumbers(t *testing.T) {
	if !hasRg() {
		t.Skip("ripgrep (rg) not installed")
	}

	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "lines.txt")
	content := "Line 1\nSearchable Line 2\nLine 3\n"
	os.WriteFile(testFile, []byte(content), 0644)

	tool := NewGrepTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"pattern": "Searchable", "path": "` + tmpDir + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Output should include line number
	if !strings.Contains(result.Output, ":2:") {
		t.Error("Output should include line number 2")
	}
}

func TestGrepTool_Metadata(t *testing.T) {
	if !hasRg() {
		t.Skip("ripgrep (rg) not installed")
	}

	tmpDir := t.TempDir()

	// Create test file with multiple matches
	testFile := filepath.Join(tmpDir, "multi.txt")
	content := "Hello\nHello\nHello\n"
	os.WriteFile(testFile, []byte(content), 0644)

	tool := NewGrepTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"pattern": "Hello", "path": "` + tmpDir + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check metadata
	if result.Metadata["pattern"] != "Hello" {
		t.Errorf("Expected pattern 'Hello' in metadata, got %v", result.Metadata["pattern"])
	}
	if result.Metadata["count"] != 3 {
		t.Errorf("Expected 3 matches in metadata, got %v", result.Metadata["count"])
	}
}

func TestGrepTool_RegexPattern(t *testing.T) {
	if !hasRg() {
		t.Skip("ripgrep (rg) not installed")
	}

	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "regex.txt")
	content := "log.Error\nlog.Warning\nlog.Info\n"
	os.WriteFile(testFile, []byte(content), 0644)

	tool := NewGrepTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	// Use regex pattern
	input := json.RawMessage(`{"pattern": "log\\.(Error|Warning)", "path": "` + tmpDir + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "Error") {
		t.Error("Output should contain 'Error'")
	}
	if !strings.Contains(result.Output, "Warning") {
		t.Error("Output should contain 'Warning'")
	}
	if strings.Contains(result.Output, "Info") {
		t.Error("Output should not contain 'Info'")
	}
}

func TestGrepTool_EinoTool(t *testing.T) {
	tool := NewGrepTool("/tmp")
	einoTool := tool.EinoTool()

	if einoTool == nil {
		t.Error("EinoTool should not return nil")
	}

	info, err := einoTool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if info.Name != "Grep" {
		t.Errorf("Expected name 'Grep', got %q", info.Name)
	}
}

func TestGrepTool_FileWithPath(t *testing.T) {
	if !hasRg() {
		t.Skip("ripgrep (rg) not installed")
	}

	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.go")
	content := "func main() {\n\treturn\n}\n"
	os.WriteFile(testFile, []byte(content), 0644)

	tool := NewGrepTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	// Search in specific file
	input := json.RawMessage(`{"pattern": "func", "path": "` + testFile + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "func") {
		t.Error("Output should contain 'func'")
	}
}
