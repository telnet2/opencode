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

	argsJSON := `{"filePath": "` + testFile + `"}`
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

func TestContext_IsAborted_NilChannel(t *testing.T) {
	ctx := &Context{AbortCh: nil}

	// Should not panic and return false
	if ctx.IsAborted() {
		t.Error("Should not be aborted with nil channel")
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

func TestBaseTool_EinoTool(t *testing.T) {
	baseTool := NewBaseTool(
		"test",
		"A test tool",
		json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}}}`),
		func(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error) {
			return &Result{Output: "test result"}, nil
		},
	)

	einoTool := baseTool.EinoTool()
	if einoTool == nil {
		t.Error("EinoTool should not return nil")
	}

	info, err := einoTool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}
	if info.Name != "test" {
		t.Errorf("Expected name 'test', got %q", info.Name)
	}
}

// ============================================
// parseJSONSchemaToParams Tests
// ============================================

func TestParseJSONSchemaToParams_AllTypes(t *testing.T) {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"stringProp": {"type": "string", "description": "A string"},
			"intProp": {"type": "integer", "description": "An integer"},
			"numProp": {"type": "number", "description": "A number"},
			"boolProp": {"type": "boolean", "description": "A boolean"},
			"arrayProp": {"type": "array", "description": "An array"},
			"objectProp": {"type": "object", "description": "An object"}
		},
		"required": ["stringProp", "intProp"]
	}`)

	params := parseJSONSchemaToParams(schema)
	if params == nil {
		t.Fatal("parseJSONSchemaToParams returned nil")
	}

	// Check all properties exist
	expectedProps := []string{"stringProp", "intProp", "numProp", "boolProp", "arrayProp", "objectProp"}
	for _, prop := range expectedProps {
		if _, ok := params[prop]; !ok {
			t.Errorf("Expected property %q not found", prop)
		}
	}

	// Check required fields
	if !params["stringProp"].Required {
		t.Error("stringProp should be required")
	}
	if !params["intProp"].Required {
		t.Error("intProp should be required")
	}
	if params["numProp"].Required {
		t.Error("numProp should not be required")
	}

	// Check descriptions
	if params["stringProp"].Desc != "A string" {
		t.Errorf("Expected description 'A string', got %q", params["stringProp"].Desc)
	}
}

func TestParseJSONSchemaToParams_InvalidJSON(t *testing.T) {
	schema := json.RawMessage(`{invalid json}`)
	params := parseJSONSchemaToParams(schema)
	if params != nil {
		t.Error("Expected nil for invalid JSON")
	}
}

func TestParseJSONSchemaToParams_EmptySchema(t *testing.T) {
	schema := json.RawMessage(`{}`)
	params := parseJSONSchemaToParams(schema)
	if params == nil {
		t.Error("Expected empty map, not nil")
	}
	if len(params) != 0 {
		t.Errorf("Expected 0 params, got %d", len(params))
	}
}
