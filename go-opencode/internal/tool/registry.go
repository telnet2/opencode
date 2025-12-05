package tool

import (
	"fmt"
	"sync"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/opencode-ai/opencode/internal/storage"
)

// Registry manages tool registration and lookup.
type Registry struct {
	mu      sync.RWMutex
	tools   map[string]Tool
	workDir string
	storage *storage.Storage
}

// NewRegistry creates a new tool registry.
func NewRegistry(workDir string, store *storage.Storage) *Registry {
	return &Registry{
		tools:   make(map[string]Tool),
		workDir: workDir,
		storage: store,
	}
}

// Storage returns the storage instance.
func (r *Registry) Storage() *storage.Storage {
	return r.storage
}

// Register adds a tool to the registry.
func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fmt.Printf("[registry] Registering tool: %s\n", tool.ID())
	r.tools[tool.ID()] = tool
}

// Get retrieves a tool by ID.
func (r *Registry) Get(id string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[id]
	return tool, ok
}

// List returns all registered tools.
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// IDs returns all tool IDs.
func (r *Registry) IDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.tools))
	for id := range r.tools {
		ids = append(ids, id)
	}
	return ids
}

// EinoTools returns Eino-compatible tools.
func (r *Registry) EinoTools() []einotool.BaseTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]einotool.BaseTool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t.EinoTool())
	}
	return tools
}

// ToolInfos returns Eino tool infos for all tools.
func (r *Registry) ToolInfos() ([]*schema.ToolInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]*schema.ToolInfo, 0, len(r.tools))
	for _, t := range r.tools {
		params := parseJSONSchemaToParams(t.Parameters())
		infos = append(infos, &schema.ToolInfo{
			Name:        t.ID(),
			Desc:        t.Description(),
			ParamsOneOf: schema.NewParamsOneOfByParams(params),
		})
	}
	return infos, nil
}

// DefaultRegistry creates a registry with all built-in tools.
func DefaultRegistry(workDir string, store *storage.Storage) *Registry {
	fmt.Printf("[registry] Creating DefaultRegistry with workDir=%s\n", workDir)
	r := NewRegistry(workDir, store)

	// Register core tools
	r.Register(NewReadTool(workDir))
	r.Register(NewWriteTool(workDir))
	r.Register(NewEditTool(workDir))
	r.Register(NewBashTool(workDir))
	r.Register(NewGlobTool(workDir))
	r.Register(NewGrepTool(workDir))
	r.Register(NewListTool(workDir))

	// Register todo tools
	r.Register(NewTodoWriteTool(workDir, store))
	r.Register(NewTodoReadTool(workDir, store))

	// Note: TaskTool requires agent registry, register separately if needed

	fmt.Printf("[registry] DefaultRegistry created with %d tools: %v\n", len(r.tools), r.IDs())
	return r
}
