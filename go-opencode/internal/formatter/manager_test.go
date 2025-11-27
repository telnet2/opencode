package formatter

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/opencode-ai/opencode/pkg/types"
)

func TestNewManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir, nil)

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}
	if manager.workDir != tempDir {
		t.Errorf("expected workDir %s, got %s", tempDir, manager.workDir)
	}
	if !manager.enabled {
		t.Error("expected enabled to be true by default")
	}
}

func TestNewManagerWithConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &types.Config{
		Formatter: map[string]types.FormatterConfig{
			"custom": {
				Command:    []string{"custom-fmt", "$file"},
				Extensions: []string{".custom", ".cst"},
				Disabled:   false,
			},
		},
	}

	manager := NewManager(tempDir, cfg)

	formatter, ok := manager.GetFormatter("custom")
	if !ok {
		t.Fatal("expected custom formatter to exist")
	}
	if formatter.Name != "custom" {
		t.Errorf("unexpected name: %s", formatter.Name)
	}
	if len(formatter.Command) != 2 || formatter.Command[0] != "custom-fmt" {
		t.Errorf("unexpected command: %v", formatter.Command)
	}
	if len(formatter.Extensions) != 2 {
		t.Errorf("expected 2 extensions, got %d", len(formatter.Extensions))
	}
}

func TestDefaultFormatters(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir, nil)

	// Check default formatters exist
	expectedDefaults := []string{"prettier", "gofmt", "black", "rustfmt"}
	for _, name := range expectedDefaults {
		formatter, ok := manager.GetFormatter(name)
		if !ok {
			t.Errorf("expected default formatter %s to exist", name)
		}
		if formatter.Name != name {
			t.Errorf("expected name %s, got %s", name, formatter.Name)
		}
	}
}

func TestGetFormatterForFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir, nil)

	tests := []struct {
		file             string
		expectedName     string
		shouldExist      bool
	}{
		{"main.go", "gofmt", true},
		{"app.js", "prettier", true},
		{"app.ts", "prettier", true},
		{"style.css", "prettier", true},
		{"main.py", "black", true},
		{"lib.rs", "rustfmt", true},
		{"unknown.xyz", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			formatter, ok := manager.GetFormatterForFile(tt.file)
			if ok != tt.shouldExist {
				t.Errorf("expected exists=%v for %s", tt.shouldExist, tt.file)
			}
			if tt.shouldExist && formatter.Name != tt.expectedName {
				t.Errorf("expected formatter %s for %s, got %s", tt.expectedName, tt.file, formatter.Name)
			}
		})
	}
}

func TestSetEnabled(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir, nil)

	// Default should be enabled
	if !manager.IsEnabled() {
		t.Error("expected enabled by default")
	}

	// Disable
	manager.SetEnabled(false)
	if manager.IsEnabled() {
		t.Error("expected disabled after SetEnabled(false)")
	}

	// Re-enable
	manager.SetEnabled(true)
	if !manager.IsEnabled() {
		t.Error("expected enabled after SetEnabled(true)")
	}
}

func TestFormatWhenDisabled(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir, nil)
	manager.SetEnabled(false)

	// Create a test file
	testFile := filepath.Join(tempDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	result, err := manager.Format(context.Background(), testFile)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if !result.Success {
		t.Error("expected success when disabled")
	}
	if result.Changed {
		t.Error("expected no change when disabled")
	}
	if result.Formatter != "" {
		t.Errorf("expected empty formatter when disabled, got %s", result.Formatter)
	}
}

func TestFormatUnknownExtension(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir, nil)

	// Create a test file with unknown extension
	testFile := filepath.Join(tempDir, "test.xyz")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	result, err := manager.Format(context.Background(), testFile)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if !result.Success {
		t.Error("expected success for unknown extension")
	}
	if result.Formatter != "" {
		t.Errorf("expected empty formatter for unknown extension, got %s", result.Formatter)
	}
}

func TestFormatNonExistentFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir, nil)

	result, err := manager.Format(context.Background(), filepath.Join(tempDir, "nonexistent.go"))
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
	if result.Success {
		t.Error("expected failure for nonexistent file")
	}
	if result.Error == "" {
		t.Error("expected error message")
	}
}

func TestFormatDisabledFormatter(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &types.Config{
		Formatter: map[string]types.FormatterConfig{
			"gofmt": {
				Command:    []string{"gofmt", "-w", "$file"},
				Extensions: []string{"go"},
				Disabled:   true,
			},
		},
	}

	manager := NewManager(tempDir, cfg)

	// Create a test file
	testFile := filepath.Join(tempDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	result, err := manager.Format(context.Background(), testFile)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if !result.Success {
		t.Error("expected success for disabled formatter")
	}
	if result.Formatter != "" {
		t.Errorf("expected empty formatter for disabled formatter, got %s", result.Formatter)
	}
}

func TestAddFormatter(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir, nil)

	newFormatter := &Formatter{
		Name:       "newfmt",
		Command:    []string{"newfmt", "$file"},
		Extensions: []string{"new"},
	}

	manager.AddFormatter(newFormatter)

	// Check formatter was added
	formatter, ok := manager.GetFormatter("newfmt")
	if !ok {
		t.Fatal("expected newfmt to exist")
	}
	if formatter.Name != "newfmt" {
		t.Errorf("unexpected name: %s", formatter.Name)
	}

	// Check extension mapping
	formatter, ok = manager.GetFormatterForFile("test.new")
	if !ok {
		t.Fatal("expected formatter for .new extension")
	}
	if formatter.Name != "newfmt" {
		t.Errorf("expected newfmt for .new, got %s", formatter.Name)
	}
}

func TestRemoveFormatter(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir, nil)

	// gofmt should exist
	_, ok := manager.GetFormatter("gofmt")
	if !ok {
		t.Fatal("expected gofmt to exist before removal")
	}

	// Remove gofmt
	removed := manager.RemoveFormatter("gofmt")
	if !removed {
		t.Error("expected RemoveFormatter to return true")
	}

	// gofmt should not exist
	_, ok = manager.GetFormatter("gofmt")
	if ok {
		t.Error("expected gofmt to be removed")
	}

	// Extension mapping should be removed
	_, ok = manager.GetFormatterForFile("main.go")
	if ok {
		t.Error("expected .go extension mapping to be removed")
	}

	// Removing non-existent should return false
	removed = manager.RemoveFormatter("nonexistent")
	if removed {
		t.Error("expected RemoveFormatter to return false for nonexistent")
	}
}

func TestStatus(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir, nil)

	status := manager.Status()

	if enabled, ok := status["enabled"].(bool); !ok || !enabled {
		t.Error("expected enabled to be true in status")
	}

	formatters, ok := status["formatters"].([]map[string]any)
	if !ok {
		t.Fatal("expected formatters in status")
	}

	if len(formatters) < 4 {
		t.Errorf("expected at least 4 default formatters, got %d", len(formatters))
	}
}

func TestReload(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir, nil)

	// Add a custom formatter
	manager.AddFormatter(&Formatter{
		Name:       "custom",
		Command:    []string{"custom", "$file"},
		Extensions: []string{"cst"},
	})

	_, ok := manager.GetFormatter("custom")
	if !ok {
		t.Fatal("expected custom formatter before reload")
	}

	// Reload should reset to defaults
	manager.Reload()

	_, ok = manager.GetFormatter("custom")
	if ok {
		t.Error("expected custom formatter to be removed after reload")
	}

	// Defaults should still exist
	_, ok = manager.GetFormatter("gofmt")
	if !ok {
		t.Error("expected gofmt to exist after reload")
	}
}

func TestAddHook(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir, nil)

	hookCalled := false
	var hookPath string
	var hookResult *FormatResult

	manager.AddHook(func(ctx context.Context, path string, result *FormatResult) {
		hookCalled = true
		hookPath = path
		hookResult = result
	})

	// Format a file with unknown extension (won't actually run a formatter)
	testFile := filepath.Join(tempDir, "test.xyz")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Unfortunately hooks are only called when formatter runs
	// Let's add a formatter that uses 'cat' (should be available on most systems)
	manager.AddFormatter(&Formatter{
		Name:       "cat",
		Command:    []string{"cat", "$file"},
		Extensions: []string{"xyz"},
	})

	_, err = manager.Format(context.Background(), testFile)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if !hookCalled {
		t.Error("expected hook to be called")
	}
	if hookPath != testFile {
		t.Errorf("expected hook path %s, got %s", testFile, hookPath)
	}
	if hookResult == nil {
		t.Error("expected hook result to be set")
	}
}

func TestCheckFormatterAvailable(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir, nil)

	// Check nonexistent formatter
	_, err = manager.CheckFormatterAvailable("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent formatter")
	}

	// Add a formatter with nonexistent command
	manager.AddFormatter(&Formatter{
		Name:       "fakefmt",
		Command:    []string{"this-command-does-not-exist-12345"},
		Extensions: []string{"fake"},
	})

	available, _ := manager.CheckFormatterAvailable("fakefmt")
	if available {
		t.Error("expected unavailable for nonexistent command")
	}

	// Add a formatter with no command
	manager.AddFormatter(&Formatter{
		Name:       "nocmd",
		Command:    []string{},
		Extensions: []string{"nocmd"},
	})

	_, err = manager.CheckFormatterAvailable("nocmd")
	if err == nil {
		t.Error("expected error for formatter with no command")
	}
}

func TestFormatMultiple(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir, nil)
	manager.SetEnabled(false) // Disable to avoid running actual formatters

	// Create test files
	files := []string{
		filepath.Join(tempDir, "file1.go"),
		filepath.Join(tempDir, "file2.go"),
		filepath.Join(tempDir, "file3.go"),
	}

	for _, f := range files {
		if err := os.WriteFile(f, []byte("package main"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	results := manager.FormatMultiple(context.Background(), files)

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	for i, result := range results {
		if result.FilePath != files[i] {
			t.Errorf("expected path %s, got %s", files[i], result.FilePath)
		}
		if !result.Success {
			t.Errorf("expected success for file %s", files[i])
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir, nil)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(3)

		// Concurrent reads
		go func() {
			defer wg.Done()
			manager.GetFormatter("gofmt")
			manager.GetFormatterForFile("test.go")
			manager.IsEnabled()
			manager.Status()
		}()

		// Concurrent writes
		go func(i int) {
			defer wg.Done()
			manager.SetEnabled(i%2 == 0)
		}(i)

		// Concurrent add/remove
		go func(i int) {
			defer wg.Done()
			name := "concurrent-" + string(rune('a'+i))
			manager.AddFormatter(&Formatter{
				Name:       name,
				Command:    []string{"fmt", "$file"},
				Extensions: []string{name},
			})
			manager.RemoveFormatter(name)
		}(i)
	}

	wg.Wait()
}

func TestFormatWithEnvironment(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &types.Config{
		Formatter: map[string]types.FormatterConfig{
			"envtest": {
				Command:    []string{"cat", "$file"},
				Extensions: []string{"envtest"},
				Environment: map[string]string{
					"TEST_VAR": "test_value",
				},
			},
		},
	}

	manager := NewManager(tempDir, cfg)

	formatter, ok := manager.GetFormatter("envtest")
	if !ok {
		t.Fatal("expected envtest formatter to exist")
	}
	if formatter.Environment["TEST_VAR"] != "test_value" {
		t.Errorf("expected environment variable, got %v", formatter.Environment)
	}
}

func TestExtensionWithDot(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &types.Config{
		Formatter: map[string]types.FormatterConfig{
			"dottest": {
				Command:    []string{"cat", "$file"},
				Extensions: []string{".dot", "nodot"},
			},
		},
	}

	manager := NewManager(tempDir, cfg)

	// Both should work - with and without leading dot
	_, ok := manager.GetFormatterForFile("test.dot")
	if !ok {
		t.Error("expected formatter for .dot extension")
	}

	_, ok = manager.GetFormatterForFile("test.nodot")
	if !ok {
		t.Error("expected formatter for .nodot extension")
	}
}

func TestConfigOverridesDefaults(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "formatter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &types.Config{
		Formatter: map[string]types.FormatterConfig{
			"gofmt": {
				Command:    []string{"custom-gofmt", "$file"},
				Extensions: []string{"go"},
			},
		},
	}

	manager := NewManager(tempDir, cfg)

	formatter, ok := manager.GetFormatter("gofmt")
	if !ok {
		t.Fatal("expected gofmt formatter to exist")
	}

	// Config should override default
	if len(formatter.Command) != 2 || formatter.Command[0] != "custom-gofmt" {
		t.Errorf("expected config to override default, got %v", formatter.Command)
	}
}
