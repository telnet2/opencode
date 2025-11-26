package tool

import (
	"context"
	"encoding/json"
	"testing"

	einotool "github.com/cloudwego/eino/components/tool"
)

// mockTool implements Tool for testing
type mockTool struct {
	id          string
	description string
	params      json.RawMessage
}

func (m *mockTool) ID() string                   { return m.id }
func (m *mockTool) Description() string          { return m.description }
func (m *mockTool) Parameters() json.RawMessage  { return m.params }
func (m *mockTool) Execute(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error) {
	return &Result{Output: "mock result"}, nil
}
func (m *mockTool) EinoTool() einotool.InvokableTool {
	return &einoToolWrapper{tool: m}
}

func newMockTool(id, description string) *mockTool {
	return &mockTool{
		id:          id,
		description: description,
		params:      json.RawMessage(`{"type": "object", "properties": {}}`),
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	registry := NewRegistry("/tmp")

	tool := newMockTool("test_tool", "A test tool")
	registry.Register(tool)

	got, ok := registry.Get("test_tool")
	if !ok {
		t.Fatal("Tool not found")
	}
	if got.ID() != "test_tool" {
		t.Errorf("Got tool ID %q, want 'test_tool'", got.ID())
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	registry := NewRegistry("/tmp")

	_, ok := registry.Get("nonexistent")
	if ok {
		t.Error("Expected tool not to be found")
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry("/tmp")

	registry.Register(newMockTool("tool1", "Tool 1"))
	registry.Register(newMockTool("tool2", "Tool 2"))
	registry.Register(newMockTool("tool3", "Tool 3"))

	tools := registry.List()
	if len(tools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(tools))
	}
}

func TestRegistry_IDs(t *testing.T) {
	registry := NewRegistry("/tmp")

	registry.Register(newMockTool("alpha", "Alpha"))
	registry.Register(newMockTool("beta", "Beta"))

	ids := registry.IDs()
	if len(ids) != 2 {
		t.Errorf("Expected 2 IDs, got %d", len(ids))
	}

	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}
	if !idSet["alpha"] || !idSet["beta"] {
		t.Error("Expected 'alpha' and 'beta' in IDs")
	}
}

func TestRegistry_EinoTools(t *testing.T) {
	registry := NewRegistry("/tmp")

	registry.Register(newMockTool("tool1", "Tool 1"))
	registry.Register(newMockTool("tool2", "Tool 2"))

	einoTools := registry.EinoTools()
	if len(einoTools) != 2 {
		t.Errorf("Expected 2 Eino tools, got %d", len(einoTools))
	}
}

func TestRegistry_ToolInfos(t *testing.T) {
	registry := NewRegistry("/tmp")

	tool := &mockTool{
		id:          "read_file",
		description: "Reads a file from disk",
		params: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {"type": "string", "description": "File path"}
			},
			"required": ["path"]
		}`),
	}
	registry.Register(tool)

	infos, err := registry.ToolInfos()
	if err != nil {
		t.Fatalf("ToolInfos failed: %v", err)
	}

	if len(infos) != 1 {
		t.Fatalf("Expected 1 tool info, got %d", len(infos))
	}

	if infos[0].Name != "read_file" {
		t.Errorf("Expected name 'read_file', got %q", infos[0].Name)
	}
	if infos[0].Desc != "Reads a file from disk" {
		t.Errorf("Expected description 'Reads a file from disk', got %q", infos[0].Desc)
	}
}

func TestDefaultRegistry(t *testing.T) {
	registry := DefaultRegistry("/tmp")

	// Check that core tools are registered
	expectedTools := []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep", "List"}

	for _, name := range expectedTools {
		_, ok := registry.Get(name)
		if !ok {
			t.Errorf("Expected tool %q to be registered", name)
		}
	}

	// Verify count
	tools := registry.List()
	if len(tools) < len(expectedTools) {
		t.Errorf("Expected at least %d tools, got %d", len(expectedTools), len(tools))
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewRegistry("/tmp")

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			tool := newMockTool("tool"+string(rune('0'+n)), "Tool")
			registry.Register(tool)
			registry.List()
			registry.IDs()
			registry.Get("tool" + string(rune('0'+n)))
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	tools := registry.List()
	if len(tools) != 10 {
		t.Errorf("Expected 10 tools, got %d", len(tools))
	}
}

func TestRegistry_ReplaceExisting(t *testing.T) {
	registry := NewRegistry("/tmp")

	// Register initial tool
	tool1 := newMockTool("mytool", "Original description")
	registry.Register(tool1)

	// Register replacement with same ID
	tool2 := newMockTool("mytool", "New description")
	registry.Register(tool2)

	// Should have the new tool
	got, _ := registry.Get("mytool")
	if got.Description() != "New description" {
		t.Errorf("Expected 'New description', got %q", got.Description())
	}

	// Should still have only 1 tool
	tools := registry.List()
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool after replacement, got %d", len(tools))
	}
}
