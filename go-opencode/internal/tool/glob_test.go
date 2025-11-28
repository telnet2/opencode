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

func hasRipgrep() bool {
	_, err := exec.LookPath("rg")
	return err == nil
}

func TestGlobTool_Execute(t *testing.T) {
	if !hasRipgrep() {
		t.Skip("ripgrep (rg) not installed")
	}

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
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, ".go") {
		t.Error("Output should contain .go files")
	}
}

func TestGlobTool_NoMatches(t *testing.T) {
	if !hasRipgrep() {
		t.Skip("ripgrep (rg) not installed")
	}

	tmpDir := t.TempDir()

	// Create only txt files
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte(""), 0644)

	tool := NewGlobTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	// Search for .go files (none exist)
	input := json.RawMessage(`{"pattern": "**/*.go", "path": "` + tmpDir + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Should succeed but indicate no matches
	if result.Metadata["count"] != 0 {
		t.Errorf("Expected 0 matches, got %v", result.Metadata["count"])
	}
	if !strings.Contains(result.Output, "No files matched") {
		t.Error("Output should indicate no matches")
	}
}

func TestGlobTool_Properties(t *testing.T) {
	tool := NewGlobTool("/tmp")

	if tool.ID() != "Glob" {
		t.Errorf("Expected ID 'Glob', got %q", tool.ID())
	}

	desc := tool.Description()
	if !strings.Contains(desc, "pattern") {
		t.Error("Description should mention 'pattern'")
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
}

func TestGlobTool_InvalidInput(t *testing.T) {
	tool := NewGlobTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	// Invalid JSON
	input := json.RawMessage(`{invalid json}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error for invalid JSON input")
	}
}

func TestGlobTool_RelativePath(t *testing.T) {
	if !hasRipgrep() {
		t.Skip("ripgrep (rg) not installed")
	}

	tmpDir := t.TempDir()

	// Create a subdirectory with a file
	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "test.go"), []byte(""), 0644)

	tool := NewGlobTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()
	toolCtx.WorkDir = tmpDir

	// Use relative path
	input := json.RawMessage(`{"pattern": "*.go", "path": "subdir"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "test.go") {
		t.Error("Output should contain 'test.go'")
	}
}

func TestGlobTool_DefaultPath(t *testing.T) {
	if !hasRipgrep() {
		t.Skip("ripgrep (rg) not installed")
	}

	tmpDir := t.TempDir()

	// Create a file in tmpDir
	os.WriteFile(filepath.Join(tmpDir, "default.go"), []byte(""), 0644)

	tool := NewGlobTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()
	toolCtx.WorkDir = tmpDir

	// No path specified
	input := json.RawMessage(`{"pattern": "*.go"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "default.go") {
		t.Error("Output should contain 'default.go'")
	}
}

func TestGlobTool_Metadata(t *testing.T) {
	if !hasRipgrep() {
		t.Skip("ripgrep (rg) not installed")
	}

	tmpDir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "file1.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte(""), 0644)

	tool := NewGlobTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"pattern": "*.go", "path": "` + tmpDir + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check metadata
	if result.Metadata["pattern"] != "*.go" {
		t.Errorf("Expected pattern '*.go' in metadata, got %v", result.Metadata["pattern"])
	}
	if result.Metadata["count"] != 2 {
		t.Errorf("Expected 2 files in metadata, got %v", result.Metadata["count"])
	}
}

func TestGlobTool_EinoTool(t *testing.T) {
	tool := NewGlobTool("/tmp")
	einoTool := tool.EinoTool()

	if einoTool == nil {
		t.Error("EinoTool should not return nil")
	}

	info, err := einoTool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if info.Name != "Glob" {
		t.Errorf("Expected name 'Glob', got %q", info.Name)
	}
}

func TestGlobTool_AbsolutePath(t *testing.T) {
	if !hasRipgrep() {
		t.Skip("ripgrep (rg) not installed")
	}

	tmpDir := t.TempDir()

	// Create a file
	os.WriteFile(filepath.Join(tmpDir, "abs.go"), []byte(""), 0644)

	tool := NewGlobTool("/some/other/dir") // Different default dir
	ctx := context.Background()
	toolCtx := testContext()

	// Use absolute path
	input := json.RawMessage(`{"pattern": "*.go", "path": "` + tmpDir + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "abs.go") {
		t.Error("Output should contain 'abs.go'")
	}
}
