package tool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestListTool_DefaultPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	os.WriteFile(filepath.Join(tmpDir, "default.txt"), []byte(""), 0644)

	tool := NewListTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()
	toolCtx.WorkDir = tmpDir

	// Empty path should use default
	input := json.RawMessage(`{}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "default.txt") {
		t.Error("Output should contain 'default.txt'")
	}
}

func TestListTool_RelativePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a subdirectory with a file
	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte(""), 0644)

	tool := NewListTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()
	toolCtx.WorkDir = tmpDir

	// Use relative path
	input := json.RawMessage(`{"path": "subdir"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "nested.txt") {
		t.Error("Output should contain 'nested.txt'")
	}
}

func TestListTool_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	tool := NewListTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"path": "` + tmpDir + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Should succeed with empty output
	if result.Metadata["count"] != 0 {
		t.Errorf("Expected 0 items, got %v", result.Metadata["count"])
	}
}

func TestListTool_Properties(t *testing.T) {
	tool := NewListTool("/tmp")

	if tool.ID() != "List" {
		t.Errorf("Expected ID 'List', got %q", tool.ID())
	}

	desc := tool.Description()
	if !strings.Contains(desc, "files") || !strings.Contains(desc, "directories") {
		t.Error("Description should mention 'files' and 'directories'")
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

	// Check properties
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Error("Schema should have properties")
	}
	if _, ok := props["path"]; !ok {
		t.Error("Schema should have path property")
	}
}

func TestListTool_InvalidInput(t *testing.T) {
	tool := NewListTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	// Invalid JSON
	input := json.RawMessage(`{invalid json}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error for invalid JSON input")
	}
}

func TestListTool_FileTypes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file and a directory
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("content"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "directory"), 0755)

	tool := NewListTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"path": "` + tmpDir + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check output format indicates file vs directory
	if !strings.Contains(result.Output, "[file]") {
		t.Error("Output should indicate file type")
	}
	if !strings.Contains(result.Output, "[dir") {
		t.Error("Output should indicate directory type")
	}
}

func TestListTool_FileSize(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with known content
	content := "Hello, World!"
	os.WriteFile(filepath.Join(tmpDir, "sized.txt"), []byte(content), 0644)

	tool := NewListTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"path": "` + tmpDir + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check output shows file size
	if !strings.Contains(result.Output, "bytes") {
		t.Error("Output should show file size in bytes")
	}
}

func TestListTool_Metadata(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte(""), 0644)

	tool := NewListTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"path": "` + tmpDir + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check metadata
	if result.Metadata["path"] != tmpDir {
		t.Errorf("Expected path %q in metadata, got %v", tmpDir, result.Metadata["path"])
	}
	if result.Metadata["count"] != 2 {
		t.Errorf("Expected 2 items in metadata, got %v", result.Metadata["count"])
	}
}

func TestListTool_EinoTool(t *testing.T) {
	tool := NewListTool("/tmp")
	einoTool := tool.EinoTool()

	if einoTool == nil {
		t.Error("EinoTool should not return nil")
	}

	info, err := einoTool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if info.Name != "List" {
		t.Errorf("Expected name 'List', got %q", info.Name)
	}
}
