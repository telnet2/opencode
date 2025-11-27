// Package formatter provides code formatting integration for OpenCode.
package formatter

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/opencode-ai/opencode/pkg/types"
)

// Formatter represents a code formatter configuration.
type Formatter struct {
	Name        string            `json:"name"`
	Extensions  []string          `json:"extensions"`
	Command     []string          `json:"command"`
	Environment map[string]string `json:"environment,omitempty"`
	Disabled    bool              `json:"disabled"`
}

// FormatResult represents the result of formatting a file.
type FormatResult struct {
	FilePath   string `json:"filePath"`
	Success    bool   `json:"success"`
	Changed    bool   `json:"changed"`
	Error      string `json:"error,omitempty"`
	Duration   int64  `json:"duration"` // milliseconds
	Formatter  string `json:"formatter,omitempty"`
	OriginalSize int  `json:"originalSize,omitempty"`
	FormattedSize int `json:"formattedSize,omitempty"`
}

// Manager manages code formatters and their execution.
type Manager struct {
	mu         sync.RWMutex
	workDir    string
	config     *types.Config
	formatters map[string]*Formatter
	extMap     map[string]*Formatter // extension -> formatter mapping
	enabled    bool
	hooks      []FormatHook
}

// FormatHook is called before/after formatting.
type FormatHook func(ctx context.Context, path string, result *FormatResult)

// NewManager creates a new formatter manager.
func NewManager(workDir string, config *types.Config) *Manager {
	m := &Manager{
		workDir:    workDir,
		config:     config,
		formatters: make(map[string]*Formatter),
		extMap:     make(map[string]*Formatter),
		enabled:    true,
		hooks:      make([]FormatHook, 0),
	}

	m.loadFromConfig()
	m.loadDefaults()

	return m
}

// loadFromConfig loads formatters from configuration.
func (m *Manager) loadFromConfig() {
	if m.config == nil || m.config.Formatter == nil {
		return
	}

	for name, cfg := range m.config.Formatter {
		formatter := &Formatter{
			Name:        name,
			Extensions:  cfg.Extensions,
			Command:     cfg.Command,
			Environment: cfg.Environment,
			Disabled:    cfg.Disabled,
		}
		m.formatters[name] = formatter

		// Build extension mapping
		for _, ext := range cfg.Extensions {
			ext = strings.TrimPrefix(ext, ".")
			m.extMap[ext] = formatter
		}
	}
}

// loadDefaults loads default formatters for common file types.
func (m *Manager) loadDefaults() {
	defaults := map[string]*Formatter{
		"prettier": {
			Name:       "prettier",
			Extensions: []string{"js", "jsx", "ts", "tsx", "json", "css", "scss", "md", "yaml", "yml"},
			Command:    []string{"npx", "prettier", "--write", "$file"},
		},
		"gofmt": {
			Name:       "gofmt",
			Extensions: []string{"go"},
			Command:    []string{"gofmt", "-w", "$file"},
		},
		"black": {
			Name:       "black",
			Extensions: []string{"py"},
			Command:    []string{"black", "$file"},
		},
		"rustfmt": {
			Name:       "rustfmt",
			Extensions: []string{"rs"},
			Command:    []string{"rustfmt", "$file"},
		},
	}

	// Only add defaults if not already configured
	for name, formatter := range defaults {
		if _, exists := m.formatters[name]; !exists {
			m.formatters[name] = formatter
			// Add to extension map if extension not already mapped
			for _, ext := range formatter.Extensions {
				if _, exists := m.extMap[ext]; !exists {
					m.extMap[ext] = formatter
				}
			}
		}
	}
}

// Format formats a file using the appropriate formatter.
func (m *Manager) Format(ctx context.Context, filePath string) (*FormatResult, error) {
	start := time.Now()

	result := &FormatResult{
		FilePath: filePath,
	}

	if !m.enabled {
		result.Success = true
		result.Duration = time.Since(start).Milliseconds()
		return result, nil
	}

	// Get file extension
	ext := strings.TrimPrefix(filepath.Ext(filePath), ".")

	m.mu.RLock()
	formatter, ok := m.extMap[ext]
	m.mu.RUnlock()

	if !ok || formatter.Disabled {
		result.Success = true
		result.Duration = time.Since(start).Milliseconds()
		return result, nil
	}

	result.Formatter = formatter.Name

	// Read original file for comparison
	originalContent, err := os.ReadFile(filePath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to read file: %v", err)
		result.Duration = time.Since(start).Milliseconds()
		return result, err
	}
	result.OriginalSize = len(originalContent)

	// Execute formatter
	if err := m.executeFormatter(ctx, formatter, filePath); err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(start).Milliseconds()
		return result, err
	}

	// Check if file changed
	newContent, err := os.ReadFile(filePath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to read formatted file: %v", err)
		result.Duration = time.Since(start).Milliseconds()
		return result, err
	}
	result.FormattedSize = len(newContent)
	result.Changed = !bytes.Equal(originalContent, newContent)
	result.Success = true
	result.Duration = time.Since(start).Milliseconds()

	// Call hooks
	for _, hook := range m.hooks {
		hook(ctx, filePath, result)
	}

	return result, nil
}

// executeFormatter runs the formatter command.
func (m *Manager) executeFormatter(ctx context.Context, formatter *Formatter, filePath string) error {
	if len(formatter.Command) == 0 {
		return fmt.Errorf("no command configured for formatter: %s", formatter.Name)
	}

	// Build command with file substitution
	args := make([]string, len(formatter.Command))
	for i, arg := range formatter.Command {
		args[i] = strings.ReplaceAll(arg, "$file", filePath)
		args[i] = strings.ReplaceAll(args[i], "${file}", filePath)
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = m.workDir

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range formatter.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Capture output
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return fmt.Errorf("formatter %s failed: %s", formatter.Name, errMsg)
	}

	return nil
}

// FormatMultiple formats multiple files.
func (m *Manager) FormatMultiple(ctx context.Context, paths []string) []*FormatResult {
	results := make([]*FormatResult, len(paths))

	for i, path := range paths {
		result, _ := m.Format(ctx, path)
		results[i] = result
	}

	return results
}

// Status returns the status of all formatters.
func (m *Manager) Status() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	formatters := make([]map[string]any, 0, len(m.formatters))
	for name, f := range m.formatters {
		formatters = append(formatters, map[string]any{
			"name":       name,
			"extensions": f.Extensions,
			"command":    f.Command,
			"disabled":   f.Disabled,
		})
	}

	return map[string]any{
		"enabled":    m.enabled,
		"formatters": formatters,
	}
}

// GetFormatter returns a formatter by name.
func (m *Manager) GetFormatter(name string) (*Formatter, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, ok := m.formatters[name]
	return f, ok
}

// GetFormatterForFile returns the formatter for a given file.
func (m *Manager) GetFormatterForFile(filePath string) (*Formatter, bool) {
	ext := strings.TrimPrefix(filepath.Ext(filePath), ".")
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, ok := m.extMap[ext]
	return f, ok
}

// SetEnabled enables or disables all formatters.
func (m *Manager) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
}

// IsEnabled returns whether formatting is enabled.
func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// AddHook adds a format hook.
func (m *Manager) AddHook(hook FormatHook) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hooks = append(m.hooks, hook)
}

// AddFormatter adds or updates a formatter.
func (m *Manager) AddFormatter(formatter *Formatter) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.formatters[formatter.Name] = formatter

	for _, ext := range formatter.Extensions {
		ext = strings.TrimPrefix(ext, ".")
		m.extMap[ext] = formatter
	}
}

// RemoveFormatter removes a formatter by name.
func (m *Manager) RemoveFormatter(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	formatter, ok := m.formatters[name]
	if !ok {
		return false
	}

	// Remove from extension map
	for _, ext := range formatter.Extensions {
		if m.extMap[ext] == formatter {
			delete(m.extMap, ext)
		}
	}

	delete(m.formatters, name)
	return true
}

// CheckFormatterAvailable checks if a formatter's command is available.
func (m *Manager) CheckFormatterAvailable(name string) (bool, error) {
	m.mu.RLock()
	formatter, ok := m.formatters[name]
	m.mu.RUnlock()

	if !ok {
		return false, fmt.Errorf("formatter not found: %s", name)
	}

	if len(formatter.Command) == 0 {
		return false, fmt.Errorf("no command configured")
	}

	// Check if command exists
	_, err := exec.LookPath(formatter.Command[0])
	return err == nil, err
}

// Reload reloads formatters from config.
func (m *Manager) Reload() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.formatters = make(map[string]*Formatter)
	m.extMap = make(map[string]*Formatter)
	m.loadFromConfig()
	m.loadDefaults()
}
