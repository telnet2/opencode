package tool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

	input := json.RawMessage(`{"filePath": "` + testFile + `"}`)
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

	input := json.RawMessage(`{"filePath": "/nonexistent/file.txt"}`)
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

	// Read lines 3-5 (offset=3, limit=3)
	input := json.RawMessage(`{"filePath": "` + testFile + `", "offset": 3, "limit": 3}`)
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
}

func TestReadTool_EnvFileBlocked(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envFile, []byte("SECRET=value"), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	tool := NewReadTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"filePath": "` + envFile + `"}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error when reading .env file")
	}
	if !strings.Contains(err.Error(), ".env") {
		t.Errorf("Error should mention .env files, got: %v", err)
	}
}

func TestReadTool_DirectoryError(t *testing.T) {
	tmpDir := t.TempDir()

	tool := NewReadTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"filePath": "` + tmpDir + `"}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error when reading a directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("Error should mention directory, got: %v", err)
	}
}

func TestReadTool_ImageFile(t *testing.T) {
	tmpDir := t.TempDir()
	imgFile := filepath.Join(tmpDir, "test.png")

	// Create minimal PNG file (8-byte signature + minimal IHDR chunk)
	pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if err := os.WriteFile(imgFile, pngSignature, 0644); err != nil {
		t.Fatalf("Failed to create PNG file: %v", err)
	}

	tool := NewReadTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"filePath": "` + imgFile + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Image files should have attachments
	if len(result.Attachments) == 0 {
		t.Error("Image file should have attachments")
	}

	if len(result.Attachments) > 0 {
		att := result.Attachments[0]
		if att.MediaType != "image/png" {
			t.Errorf("Expected media type 'image/png', got %q", att.MediaType)
		}
		if !strings.HasPrefix(att.URL, "data:image/png;base64,") {
			t.Error("Attachment URL should be a data URL")
		}
	}
}

func TestReadTool_BinaryFile(t *testing.T) {
	tmpDir := t.TempDir()
	binFile := filepath.Join(tmpDir, "binary.dat")

	// Create file with null bytes (binary indicator)
	content := []byte{0x00, 0x01, 0x02, 0x00, 0x03, 0x04, 0x00}
	if err := os.WriteFile(binFile, content, 0644); err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}

	tool := NewReadTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"filePath": "` + binFile + `"}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error when reading binary file")
	}
	if !strings.Contains(err.Error(), "binary") {
		t.Errorf("Error should mention binary, got: %v", err)
	}
}

func TestReadTool_LongLineTruncation(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "longline.txt")

	// Create a line longer than 2000 characters
	longLine := strings.Repeat("x", 3000)
	if err := os.WriteFile(testFile, []byte(longLine), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewReadTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"filePath": "` + testFile + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// The output line should be truncated (line number prefix + truncated content + "...")
	// Line format: "    1\t" + content
	if len(result.Output) > 2100 { // Some buffer for line number and "..."
		t.Errorf("Output should be truncated, got length %d", len(result.Output))
	}
	if !strings.Contains(result.Output, "...") {
		t.Error("Truncated line should end with '...'")
	}
}

func TestReadTool_InvalidInput(t *testing.T) {
	tool := NewReadTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	// Invalid JSON
	input := json.RawMessage(`{invalid json}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error for invalid JSON input")
	}
}

func TestReadTool_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.txt")
	if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	tool := NewReadTool(tmpDir)
	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"filePath": "` + emptyFile + `"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Empty file should not cause an error
	if result.Metadata["lines"] != 0 {
		t.Errorf("Expected 0 lines for empty file, got %v", result.Metadata["lines"])
	}
}

func TestReadTool_EinoTool(t *testing.T) {
	tool := NewReadTool("/tmp")
	einoTool := tool.EinoTool()

	if einoTool == nil {
		t.Error("EinoTool should not return nil")
	}

	info, err := einoTool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if info.Name != "Read" {
		t.Errorf("Expected name 'Read', got %q", info.Name)
	}
}
