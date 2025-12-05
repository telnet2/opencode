package tool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteTool_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "output.txt")

	tool := NewWriteTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"filePath": "` + testFile + `", "content": "Hello, World!"}`)
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

	input := json.RawMessage(`{"filePath": "` + testFile + `", "content": "Nested content"}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File should have been created with parent directories")
	}

	// Verify content
	data, _ := os.ReadFile(testFile)
	if string(data) != "Nested content" {
		t.Errorf("File content = %q, want 'Nested content'", string(data))
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

	input := json.RawMessage(`{"filePath": "` + testFile + `", "content": "Updated"}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	data, _ := os.ReadFile(testFile)
	if string(data) != "Updated" {
		t.Errorf("File should be overwritten, got %q", string(data))
	}
}

func TestWriteTool_Properties(t *testing.T) {
	tool := NewWriteTool("/tmp")

	if tool.ID() != "Write" {
		t.Errorf("Expected ID 'Write', got %q", tool.ID())
	}

	desc := tool.Description()
	if !strings.Contains(desc, "file") {
		t.Error("Description should mention 'file'")
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
	if _, ok := props["filePath"]; !ok {
		t.Error("Schema should have filePath property")
	}
	if _, ok := props["content"]; !ok {
		t.Error("Schema should have content property")
	}
}

func TestWriteTool_InvalidInput(t *testing.T) {
	tool := NewWriteTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	// Invalid JSON
	input := json.RawMessage(`{invalid json}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error for invalid JSON input")
	}
}

func TestWriteTool_EmptyContent(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	tool := NewWriteTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"filePath": "` + testFile + `", "content": ""}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Should succeed with empty content
	if result.Metadata["bytes"] != 0 {
		t.Errorf("Expected 0 bytes, got %v", result.Metadata["bytes"])
	}

	// Verify file exists and is empty
	data, _ := os.ReadFile(testFile)
	if len(data) != 0 {
		t.Error("File should be empty")
	}
}

func TestWriteTool_Metadata(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "meta.txt")
	content := "Test content"

	tool := NewWriteTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"filePath": "` + testFile + `", "content": "` + content + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check metadata
	if result.Metadata["file"] != testFile {
		t.Errorf("Expected file %q in metadata, got %v", testFile, result.Metadata["file"])
	}
	if result.Metadata["bytes"] != len(content) {
		t.Errorf("Expected %d bytes in metadata, got %v", len(content), result.Metadata["bytes"])
	}
}

func TestWriteTool_EinoTool(t *testing.T) {
	tool := NewWriteTool("/tmp")
	einoTool := tool.EinoTool()

	if einoTool == nil {
		t.Error("EinoTool should not return nil")
	}

	info, err := einoTool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if info.Name != "Write" {
		t.Errorf("Expected name 'Write', got %q", info.Name)
	}
}
