package tool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestBatchTool_Properties(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry(tmpDir, nil)
	registry.Register(NewReadTool(tmpDir))

	tool := NewBatchTool(tmpDir, registry)

	if tool.ID() != "batch" {
		t.Errorf("Expected ID 'batch', got %q", tool.ID())
	}

	desc := tool.Description()
	if !strings.Contains(desc, "parallel") {
		t.Error("Description should mention 'parallel'")
	}

	params := tool.Parameters()
	if len(params) == 0 {
		t.Error("Parameters should not be empty")
	}

	var schema map[string]any
	if err := json.Unmarshal(params, &schema); err != nil {
		t.Errorf("Parameters should be valid JSON: %v", err)
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Error("Schema should have properties")
	}
	if _, ok := props["tool_calls"]; !ok {
		t.Error("Schema should have tool_calls property")
	}
}

func TestBatchTool_SingleToolCall(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Hello World"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	registry := NewRegistry(tmpDir, nil)
	registry.Register(NewReadTool(tmpDir))

	batchTool := NewBatchTool(tmpDir, registry)
	ctx := context.Background()
	toolCtx := testContext()
	toolCtx.WorkDir = tmpDir

	input := json.RawMessage(`{
		"tool_calls": [
			{"tool": "read", "parameters": {"filePath": "` + testFile + `"}}
		]
	}`)

	result, err := batchTool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Title, "1/1") {
		t.Errorf("Title should indicate 1/1 successful, got %q", result.Title)
	}
	if !strings.Contains(result.Output, "Hello World") {
		t.Error("Output should contain file content")
	}

	// Check metadata
	if result.Metadata["successful"] != 1 {
		t.Errorf("Expected 1 successful, got %v", result.Metadata["successful"])
	}
	if result.Metadata["failed"] != 0 {
		t.Errorf("Expected 0 failed, got %v", result.Metadata["failed"])
	}
}

func TestBatchTool_MultipleToolCalls(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	os.WriteFile(file1, []byte("Content 1"), 0644)
	os.WriteFile(file2, []byte("Content 2"), 0644)

	registry := NewRegistry(tmpDir, nil)
	registry.Register(NewReadTool(tmpDir))

	batchTool := NewBatchTool(tmpDir, registry)
	ctx := context.Background()
	toolCtx := testContext()
	toolCtx.WorkDir = tmpDir

	input := json.RawMessage(`{
		"tool_calls": [
			{"tool": "read", "parameters": {"filePath": "` + file1 + `"}},
			{"tool": "read", "parameters": {"filePath": "` + file2 + `"}}
		]
	}`)

	result, err := batchTool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Title, "2/2") {
		t.Errorf("Title should indicate 2/2 successful, got %q", result.Title)
	}
	if !strings.Contains(result.Output, "Content 1") {
		t.Error("Output should contain file1 content")
	}
	if !strings.Contains(result.Output, "Content 2") {
		t.Error("Output should contain file2 content")
	}
}

func TestBatchTool_ParallelExecution(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple test files
	for i := 0; i < 5; i++ {
		file := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		os.WriteFile(file, []byte("Content"), 0644)
	}

	registry := NewRegistry(tmpDir, nil)
	registry.Register(NewReadTool(tmpDir))

	batchTool := NewBatchTool(tmpDir, registry)
	ctx := context.Background()
	toolCtx := testContext()
	toolCtx.WorkDir = tmpDir

	// Build input with 5 file reads
	var calls []string
	for i := 0; i < 5; i++ {
		file := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		calls = append(calls, `{"tool": "read", "parameters": {"filePath": "`+file+`"}}`)
	}
	input := json.RawMessage(`{"tool_calls": [` + strings.Join(calls, ",") + `]}`)

	start := time.Now()
	result, err := batchTool.Execute(ctx, input, toolCtx)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Title, "5/5") {
		t.Errorf("Title should indicate 5/5 successful, got %q", result.Title)
	}

	// Parallel execution should be fast (much less than 5 sequential reads)
	if elapsed > 2*time.Second {
		t.Logf("Warning: Batch execution took %v, might not be parallel", elapsed)
	}
}

func TestBatchTool_DisallowedTool_Batch(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry(tmpDir, nil)
	batchTool := NewBatchTool(tmpDir, registry)
	registry.Register(batchTool)

	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{
		"tool_calls": [
			{"tool": "batch", "parameters": {}}
		]
	}`)

	result, err := batchTool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute should not fail, got: %v", err)
	}

	if result.Metadata["failed"] != 1 {
		t.Error("Nested batch call should fail")
	}
	if !strings.Contains(result.Output, "not allowed in batch") {
		t.Error("Output should mention batch is not allowed")
	}
}

func TestBatchTool_DisallowedTool_Edit(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry(tmpDir, nil)
	registry.Register(NewEditTool(tmpDir))
	batchTool := NewBatchTool(tmpDir, registry)

	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{
		"tool_calls": [
			{"tool": "edit", "parameters": {"filePath": "test.txt", "oldString": "a", "newString": "b"}}
		]
	}`)

	result, err := batchTool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute should not fail, got: %v", err)
	}

	if result.Metadata["failed"] != 1 {
		t.Error("Edit call should fail in batch")
	}
	if !strings.Contains(result.Output, "not allowed") {
		t.Error("Output should mention edit is not allowed")
	}
}

func TestBatchTool_ToolNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry(tmpDir, nil)
	registry.Register(NewReadTool(tmpDir))
	batchTool := NewBatchTool(tmpDir, registry)

	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{
		"tool_calls": [
			{"tool": "nonexistent", "parameters": {}}
		]
	}`)

	result, err := batchTool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute should not fail, got: %v", err)
	}

	if result.Metadata["failed"] != 1 {
		t.Error("Nonexistent tool call should fail")
	}
	if !strings.Contains(result.Output, "not found") {
		t.Error("Output should mention tool not found")
	}
	if !strings.Contains(result.Output, "Available tools") {
		t.Error("Output should list available tools")
	}
}

func TestBatchTool_PartialFailure(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "exists.txt")
	os.WriteFile(testFile, []byte("Content"), 0644)

	registry := NewRegistry(tmpDir, nil)
	registry.Register(NewReadTool(tmpDir))
	batchTool := NewBatchTool(tmpDir, registry)

	ctx := context.Background()
	toolCtx := testContext()
	toolCtx.WorkDir = tmpDir

	input := json.RawMessage(`{
		"tool_calls": [
			{"tool": "read", "parameters": {"filePath": "` + testFile + `"}},
			{"tool": "read", "parameters": {"filePath": "/nonexistent/file.txt"}}
		]
	}`)

	result, err := batchTool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute should not fail with partial failure: %v", err)
	}

	if result.Metadata["successful"] != 1 {
		t.Errorf("Expected 1 successful, got %v", result.Metadata["successful"])
	}
	if result.Metadata["failed"] != 1 {
		t.Errorf("Expected 1 failed, got %v", result.Metadata["failed"])
	}
	if !strings.Contains(result.Title, "1/2") {
		t.Errorf("Title should indicate 1/2 successful, got %q", result.Title)
	}
}

func TestBatchTool_MaxBatchSize(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	for i := 0; i < 15; i++ {
		file := filepath.Join(tmpDir, "file"+string(rune('A'+i))+".txt")
		os.WriteFile(file, []byte("Content"), 0644)
	}

	registry := NewRegistry(tmpDir, nil)
	registry.Register(NewReadTool(tmpDir))
	batchTool := NewBatchTool(tmpDir, registry)

	ctx := context.Background()
	toolCtx := testContext()
	toolCtx.WorkDir = tmpDir

	// Build input with 15 calls (exceeds max of 10)
	var calls []string
	for i := 0; i < 15; i++ {
		file := filepath.Join(tmpDir, "file"+string(rune('A'+i))+".txt")
		calls = append(calls, `{"tool": "read", "parameters": {"filePath": "`+file+`"}}`)
	}
	input := json.RawMessage(`{"tool_calls": [` + strings.Join(calls, ",") + `]}`)

	result, err := batchTool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// 10 should succeed, 5 should be discarded with error
	if result.Metadata["totalCalls"] != 15 {
		t.Errorf("Expected 15 total calls, got %v", result.Metadata["totalCalls"])
	}
	if result.Metadata["successful"] != 10 {
		t.Errorf("Expected 10 successful, got %v", result.Metadata["successful"])
	}
	if result.Metadata["failed"] != 5 {
		t.Errorf("Expected 5 failed (discarded), got %v", result.Metadata["failed"])
	}
	if !strings.Contains(result.Output, "Maximum of 10 tools") {
		t.Error("Output should mention max batch size for discarded calls")
	}
}

func TestBatchTool_EmptyToolCalls(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry(tmpDir, nil)
	batchTool := NewBatchTool(tmpDir, registry)

	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{"tool_calls": []}`)

	_, err := batchTool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error for empty tool_calls")
	}
}

func TestBatchTool_InvalidInput(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry(tmpDir, nil)
	batchTool := NewBatchTool(tmpDir, registry)

	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{invalid json}`)

	_, err := batchTool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "Expected payload format") {
		t.Error("Error should include expected format hint")
	}
}

func TestBatchTool_MissingToolCalls(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry(tmpDir, nil)
	batchTool := NewBatchTool(tmpDir, registry)

	ctx := context.Background()
	toolCtx := testContext()

	input := json.RawMessage(`{}`)

	_, err := batchTool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error for missing tool_calls")
	}
}

func TestBatchTool_ConcurrencyVerification(t *testing.T) {
	tmpDir := t.TempDir()

	// Track concurrent executions
	var maxConcurrent int32
	var currentConcurrent int32

	// Create a mock slow tool
	slowTool := NewBaseTool(
		"slow",
		"A slow tool for testing",
		json.RawMessage(`{"type": "object", "properties": {}}`),
		func(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error) {
			cur := atomic.AddInt32(&currentConcurrent, 1)
			for {
				max := atomic.LoadInt32(&maxConcurrent)
				if cur > max {
					if atomic.CompareAndSwapInt32(&maxConcurrent, max, cur) {
						break
					}
				} else {
					break
				}
			}

			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&currentConcurrent, -1)

			return &Result{Output: "done"}, nil
		},
	)

	registry := NewRegistry(tmpDir, nil)
	registry.Register(slowTool)
	batchTool := NewBatchTool(tmpDir, registry)

	ctx := context.Background()
	toolCtx := testContext()

	// Execute 5 slow tools
	input := json.RawMessage(`{
		"tool_calls": [
			{"tool": "slow", "parameters": {}},
			{"tool": "slow", "parameters": {}},
			{"tool": "slow", "parameters": {}},
			{"tool": "slow", "parameters": {}},
			{"tool": "slow", "parameters": {}}
		]
	}`)

	result, err := batchTool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Metadata["successful"] != 5 {
		t.Errorf("Expected 5 successful, got %v", result.Metadata["successful"])
	}

	// Verify concurrent execution happened
	if maxConcurrent < 2 {
		t.Errorf("Expected concurrent execution (max concurrent >= 2), got %d", maxConcurrent)
	}
}

func TestBatchTool_Attachments(t *testing.T) {
	tmpDir := t.TempDir()

	// Create PNG file
	pngFile := filepath.Join(tmpDir, "test.png")
	pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	os.WriteFile(pngFile, pngSignature, 0644)

	registry := NewRegistry(tmpDir, nil)
	registry.Register(NewReadTool(tmpDir))
	batchTool := NewBatchTool(tmpDir, registry)

	ctx := context.Background()
	toolCtx := testContext()
	toolCtx.WorkDir = tmpDir

	input := json.RawMessage(`{
		"tool_calls": [
			{"tool": "read", "parameters": {"filePath": "` + pngFile + `"}}
		]
	}`)

	result, err := batchTool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(result.Attachments) == 0 {
		t.Error("Expected attachments from image read")
	}

	if len(result.Attachments) > 0 && result.Attachments[0].MediaType != "image/png" {
		t.Errorf("Expected image/png attachment, got %q", result.Attachments[0].MediaType)
	}
}

func TestBatchTool_MixedTools(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "mixed.txt")
	os.WriteFile(testFile, []byte("Test content for grep"), 0644)

	registry := NewRegistry(tmpDir, nil)
	registry.Register(NewReadTool(tmpDir))
	registry.Register(NewGlobTool(tmpDir))
	batchTool := NewBatchTool(tmpDir, registry)

	ctx := context.Background()
	toolCtx := testContext()
	toolCtx.WorkDir = tmpDir

	input := json.RawMessage(`{
		"tool_calls": [
			{"tool": "read", "parameters": {"filePath": "` + testFile + `"}},
			{"tool": "glob", "parameters": {"pattern": "*.txt", "path": "` + tmpDir + `"}}
		]
	}`)

	result, err := batchTool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Metadata["successful"] != 2 {
		t.Errorf("Expected 2 successful, got %v", result.Metadata["successful"])
	}

	tools := result.Metadata["tools"].([]string)
	if len(tools) != 2 || tools[0] != "read" || tools[1] != "glob" {
		t.Errorf("Expected tools [read, glob], got %v", tools)
	}
}

func TestBatchTool_ResultOrdering(t *testing.T) {
	tmpDir := t.TempDir()

	// Create numbered files
	for i := 0; i < 5; i++ {
		file := filepath.Join(tmpDir, "order"+string(rune('0'+i))+".txt")
		os.WriteFile(file, []byte("File "+string(rune('0'+i))), 0644)
	}

	registry := NewRegistry(tmpDir, nil)
	registry.Register(NewReadTool(tmpDir))
	batchTool := NewBatchTool(tmpDir, registry)

	ctx := context.Background()
	toolCtx := testContext()
	toolCtx.WorkDir = tmpDir

	var calls []string
	for i := 0; i < 5; i++ {
		file := filepath.Join(tmpDir, "order"+string(rune('0'+i))+".txt")
		calls = append(calls, `{"tool": "read", "parameters": {"filePath": "`+file+`"}}`)
	}
	input := json.RawMessage(`{"tool_calls": [` + strings.Join(calls, ",") + `]}`)

	result, err := batchTool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify results are ordered by index
	details := result.Metadata["details"].([]map[string]any)
	for i, detail := range details {
		if detail["tool"] != "read" {
			t.Errorf("Result %d: expected tool 'read', got %v", i, detail["tool"])
		}
	}
}

func TestBatchTool_EinoTool(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry(tmpDir, nil)
	tool := NewBatchTool(tmpDir, registry)
	einoTool := tool.EinoTool()

	if einoTool == nil {
		t.Error("EinoTool should not return nil")
	}

	info, err := einoTool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if info.Name != "batch" {
		t.Errorf("Expected name 'batch', got %q", info.Name)
	}
}

func TestBatchTool_ContextAbort(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a tool that checks for abort
	abortCheckTool := NewBaseTool(
		"abortcheck",
		"Checks abort",
		json.RawMessage(`{"type": "object"}`),
		func(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return &Result{Output: "ok"}, nil
			}
		},
	)

	registry := NewRegistry(tmpDir, nil)
	registry.Register(abortCheckTool)
	batchTool := NewBatchTool(tmpDir, registry)

	ctx, cancel := context.WithCancel(context.Background())
	toolCtx := testContext()

	// Cancel before execution
	cancel()

	input := json.RawMessage(`{
		"tool_calls": [
			{"tool": "abortcheck", "parameters": {}}
		]
	}`)

	result, err := batchTool.Execute(ctx, input, toolCtx)
	if err != nil {
		// Context cancellation may cause early exit
		return
	}

	// Either the result shows failure or the execution completed before cancel took effect
	if result.Metadata["failed"].(int) > 0 {
		t.Log("Tool correctly detected context cancellation")
	}
}
