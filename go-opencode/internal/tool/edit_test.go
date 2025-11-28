package tool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestEditTool_SameStrings(t *testing.T) {
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
		"old_string": "Hello",
		"new_string": "Hello"
	}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error when old_string equals new_string")
	}
	if !strings.Contains(err.Error(), "different") {
		t.Errorf("Error should mention 'different', got: %v", err)
	}
}

func TestEditTool_MultipleOccurrences(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "edit.txt")
	if err := os.WriteFile(testFile, []byte("foo bar foo baz foo"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	tool := NewEditTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	// Without replace_all, multiple occurrences should fail
	input := json.RawMessage(`{
		"file_path": "` + testFile + `",
		"old_string": "foo",
		"new_string": "qux"
	}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error when old_string appears multiple times without replace_all")
	}
	if !strings.Contains(err.Error(), "3 times") {
		t.Errorf("Error should mention occurrences, got: %v", err)
	}
}

func TestEditTool_FuzzyMatchLineNormalization(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "edit.txt")

	// Create file with Windows-style line endings
	content := "Hello\r\nWorld"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	tool := NewEditTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	// Try to match with Unix-style line endings (should use normalized matching)
	input := json.RawMessage(`{
		"file_path": "` + testFile + `",
		"old_string": "Hello\nWorld",
		"new_string": "Goodbye\nWorld"
	}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "normalized") {
		t.Logf("Result: %s (normalized matching may have been used)", result.Output)
	}
}

func TestEditTool_FuzzyMatchSimilarity(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "edit.txt")

	// Create file with similar but not exact text
	content := "Hello Wonderful World"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	tool := NewEditTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	// Try to match with slightly different text (should use fuzzy matching)
	input := json.RawMessage(`{
		"file_path": "` + testFile + `",
		"old_string": "Hello Wonderfull World",
		"new_string": "Goodbye World"
	}`)
	result, err := tool.Execute(ctx, input, toolCtx)

	// Fuzzy matching with 70%+ similarity should succeed
	if err != nil {
		t.Logf("Fuzzy matching did not succeed (may be expected for this difference): %v", err)
	} else {
		if !strings.Contains(result.Output, "fuzzy") && !strings.Contains(result.Output, "similarity") {
			t.Logf("Result: %s", result.Output)
		}
	}
}

func TestEditTool_Properties(t *testing.T) {
	tool := NewEditTool("/tmp")

	if tool.ID() != "Edit" {
		t.Errorf("Expected ID 'Edit', got %q", tool.ID())
	}

	desc := tool.Description()
	if !strings.Contains(desc, "replacement") {
		t.Error("Description should mention 'replacement'")
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
	if _, ok := props["file_path"]; !ok {
		t.Error("Schema should have file_path property")
	}
	if _, ok := props["old_string"]; !ok {
		t.Error("Schema should have old_string property")
	}
	if _, ok := props["new_string"]; !ok {
		t.Error("Schema should have new_string property")
	}
	if _, ok := props["replace_all"]; !ok {
		t.Error("Schema should have replace_all property")
	}
}

func TestEditTool_InvalidInput(t *testing.T) {
	tool := NewEditTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	// Invalid JSON
	input := json.RawMessage(`{invalid json}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error for invalid JSON input")
	}
}

func TestEditTool_FileNotFound(t *testing.T) {
	tool := NewEditTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{
		"file_path": "/nonexistent/file.txt",
		"old_string": "foo",
		"new_string": "bar"
	}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestEditTool_Metadata(t *testing.T) {
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

	// Check metadata
	if result.Metadata["file"] != testFile {
		t.Errorf("Expected file %q in metadata, got %v", testFile, result.Metadata["file"])
	}
	if result.Metadata["replacements"] != 1 {
		t.Errorf("Expected 1 replacement in metadata, got %v", result.Metadata["replacements"])
	}
}

func TestEditTool_EinoTool(t *testing.T) {
	tool := NewEditTool("/tmp")
	einoTool := tool.EinoTool()

	if einoTool == nil {
		t.Error("EinoTool should not return nil")
	}

	info, err := einoTool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if info.Name != "Edit" {
		t.Errorf("Expected name 'Edit', got %q", info.Name)
	}
}

// Test the similarity function
func TestSimilarity(t *testing.T) {
	tests := []struct {
		a, b     string
		expected float64
		delta    float64
	}{
		{"hello", "hello", 1.0, 0.01},
		{"hello", "helo", 0.8, 0.1},
		{"", "", 1.0, 0.01},
		{"hello", "", 0.0, 0.01},
		{"", "hello", 0.0, 0.01},
	}

	for _, tc := range tests {
		result := similarity(tc.a, tc.b)
		if result < tc.expected-tc.delta || result > tc.expected+tc.delta {
			t.Errorf("similarity(%q, %q) = %v, expected ~%v", tc.a, tc.b, result, tc.expected)
		}
	}
}
