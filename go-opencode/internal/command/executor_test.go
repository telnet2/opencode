package command

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/opencode-ai/opencode/pkg/types"
)

func TestNewExecutor(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &types.Config{}
	executor := NewExecutor(tempDir, cfg)

	if executor == nil {
		t.Fatal("expected non-nil executor")
	}
	if executor.workDir != tempDir {
		t.Errorf("expected workDir %s, got %s", tempDir, executor.workDir)
	}
}

func TestNewExecutorWithConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &types.Config{
		Command: map[string]types.CommandConfig{
			"greet": {
				Template:    "Hello, $1!",
				Description: "Greet someone",
				Agent:       "default",
				Model:       "gpt-4",
				Subtask:     true,
			},
		},
	}

	executor := NewExecutor(tempDir, cfg)

	cmd, ok := executor.Get("greet")
	if !ok {
		t.Fatal("expected greet command to exist")
	}
	if cmd.Template != "Hello, $1!" {
		t.Errorf("unexpected template: %s", cmd.Template)
	}
	if cmd.Description != "Greet someone" {
		t.Errorf("unexpected description: %s", cmd.Description)
	}
	if cmd.Agent != "default" {
		t.Errorf("unexpected agent: %s", cmd.Agent)
	}
	if cmd.Model != "gpt-4" {
		t.Errorf("unexpected model: %s", cmd.Model)
	}
	if !cmd.Subtask {
		t.Error("expected subtask to be true")
	}
	if cmd.Source != "config" {
		t.Errorf("expected source 'config', got %s", cmd.Source)
	}
}

func TestLoadFromFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create command directory
	commandDir := filepath.Join(tempDir, ".opencode", "command")
	if err := os.MkdirAll(commandDir, 0755); err != nil {
		t.Fatalf("failed to create command dir: %v", err)
	}

	// Create a command file with frontmatter
	commandContent := `---
description: Run tests
agent: test-agent
model: claude-3
subtask: true
---
Run tests for $1 package`

	if err := os.WriteFile(filepath.Join(commandDir, "test.md"), []byte(commandContent), 0644); err != nil {
		t.Fatalf("failed to write command file: %v", err)
	}

	executor := NewExecutor(tempDir, nil)

	cmd, ok := executor.Get("test")
	if !ok {
		t.Fatal("expected test command to exist")
	}
	if cmd.Description != "Run tests" {
		t.Errorf("unexpected description: %s", cmd.Description)
	}
	if cmd.Agent != "test-agent" {
		t.Errorf("unexpected agent: %s", cmd.Agent)
	}
	if cmd.Model != "claude-3" {
		t.Errorf("unexpected model: %s", cmd.Model)
	}
	if !cmd.Subtask {
		t.Error("expected subtask to be true")
	}
	if cmd.Source != "file" {
		t.Errorf("expected source 'file', got %s", cmd.Source)
	}
	if cmd.Template != "Run tests for $1 package" {
		t.Errorf("unexpected template: %s", cmd.Template)
	}
}

func TestLoadFromFilesNested(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create nested command directory
	nestedDir := filepath.Join(tempDir, ".opencode", "command", "sub")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	// Create a nested command file
	if err := os.WriteFile(filepath.Join(nestedDir, "nested.md"), []byte("Nested command"), 0644); err != nil {
		t.Fatalf("failed to write nested command file: %v", err)
	}

	executor := NewExecutor(tempDir, nil)

	// Nested files should use : separator
	cmd, ok := executor.Get("sub:nested")
	if !ok {
		t.Fatal("expected sub:nested command to exist")
	}
	if cmd.Template != "Nested command" {
		t.Errorf("unexpected template: %s", cmd.Template)
	}
}

func TestParseMarkdownWithoutFrontmatter(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	commandDir := filepath.Join(tempDir, ".opencode", "command")
	if err := os.MkdirAll(commandDir, 0755); err != nil {
		t.Fatalf("failed to create command dir: %v", err)
	}

	// Create a command file without frontmatter
	commandContent := `This is a simple command without frontmatter.
It has multiple lines.`

	if err := os.WriteFile(filepath.Join(commandDir, "simple.md"), []byte(commandContent), 0644); err != nil {
		t.Fatalf("failed to write command file: %v", err)
	}

	executor := NewExecutor(tempDir, nil)

	cmd, ok := executor.Get("simple")
	if !ok {
		t.Fatal("expected simple command to exist")
	}
	if cmd.Template != commandContent {
		t.Errorf("expected entire content as template, got: %s", cmd.Template)
	}
}

func TestList(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &types.Config{
		Command: map[string]types.CommandConfig{
			"cmd1": {Template: "Command 1"},
			"cmd2": {Template: "Command 2"},
		},
	}

	executor := NewExecutor(tempDir, cfg)
	commands := executor.List()

	if len(commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(commands))
	}

	// Check both commands exist
	names := make(map[string]bool)
	for _, cmd := range commands {
		names[cmd.Name] = true
	}
	if !names["cmd1"] || !names["cmd2"] {
		t.Error("expected both cmd1 and cmd2 to be in list")
	}
}

func TestGet(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &types.Config{
		Command: map[string]types.CommandConfig{
			"exists": {Template: "I exist"},
		},
	}

	executor := NewExecutor(tempDir, cfg)

	// Test existing command
	cmd, ok := executor.Get("exists")
	if !ok {
		t.Error("expected exists command to be found")
	}
	if cmd.Template != "I exist" {
		t.Errorf("unexpected template: %s", cmd.Template)
	}

	// Test non-existing command
	_, ok = executor.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent command to not be found")
	}
}

func TestExecuteSimple(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &types.Config{
		Command: map[string]types.CommandConfig{
			"greet": {
				Template:    "Hello, $1!",
				Agent:       "greeter",
				Model:       "gpt-4",
				Subtask:     true,
				Description: "Greet",
			},
		},
	}

	executor := NewExecutor(tempDir, cfg)

	result, err := executor.Execute(context.Background(), "greet", "World")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Prompt != "Hello, World!" {
		t.Errorf("unexpected prompt: %s", result.Prompt)
	}
	if result.Agent != "greeter" {
		t.Errorf("unexpected agent: %s", result.Agent)
	}
	if result.Model != "gpt-4" {
		t.Errorf("unexpected model: %s", result.Model)
	}
	if !result.Subtask {
		t.Error("expected subtask to be true")
	}
	if result.CommandName != "greet" {
		t.Errorf("unexpected command name: %s", result.CommandName)
	}
}

func TestExecuteMultipleArgs(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &types.Config{
		Command: map[string]types.CommandConfig{
			"concat": {Template: "$1 and $2"},
		},
	}

	executor := NewExecutor(tempDir, cfg)

	result, err := executor.Execute(context.Background(), "concat", "first second")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Prompt != "first and second" {
		t.Errorf("unexpected prompt: %s", result.Prompt)
	}
}

func TestExecuteWithInput(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &types.Config{
		Command: map[string]types.CommandConfig{
			"echo": {Template: "You said: $input"},
		},
	}

	executor := NewExecutor(tempDir, cfg)

	result, err := executor.Execute(context.Background(), "echo", "hello world")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Prompt != "You said: hello world" {
		t.Errorf("unexpected prompt: %s", result.Prompt)
	}
}

func TestExecuteBracketSyntax(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &types.Config{
		Command: map[string]types.CommandConfig{
			"brackets": {Template: "Value: ${1}"},
		},
	}

	executor := NewExecutor(tempDir, cfg)

	result, err := executor.Execute(context.Background(), "brackets", "test")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Prompt != "Value: test" {
		t.Errorf("unexpected prompt: %s", result.Prompt)
	}
}

func TestExecuteNotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	executor := NewExecutor(tempDir, nil)

	_, err = executor.Execute(context.Background(), "nonexistent", "")
	if err == nil {
		t.Error("expected error for nonexistent command")
	}
}

func TestExecuteWithGoTemplate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &types.Config{
		Command: map[string]types.CommandConfig{
			"template": {Template: "{{ upper .input }}"},
		},
	}

	executor := NewExecutor(tempDir, cfg)

	result, err := executor.Execute(context.Background(), "template", "hello")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Prompt != "HELLO" {
		t.Errorf("unexpected prompt: %s", result.Prompt)
	}
}

func TestExecuteWithVariables(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &types.Config{
		Command: map[string]types.CommandConfig{
			"withvar": {Template: "Project: {{ .var_project }}"},
		},
		PromptVariables: map[string]string{
			"project": "OpenCode",
		},
	}

	executor := NewExecutor(tempDir, cfg)

	result, err := executor.Execute(context.Background(), "withvar", "")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Prompt != "Project: OpenCode" {
		t.Errorf("unexpected prompt: %s", result.Prompt)
	}
}

func TestParseArguments(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	executor := NewExecutor(tempDir, nil)

	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:  "positional args",
			input: "arg1 arg2 arg3",
			expected: map[string]string{
				"input": "arg1 arg2 arg3",
				"1":     "arg1",
				"2":     "arg2",
				"3":     "arg3",
			},
		},
		{
			name:  "named args with equals",
			input: "--name=value",
			expected: map[string]string{
				"input": "--name=value",
				"1":     "--name=value",
				"name":  "value",
			},
		},
		{
			name:  "empty input",
			input: "",
			expected: map[string]string{
				"input": "",
			},
		},
		{
			name:  "whitespace only",
			input: "   ",
			expected: map[string]string{
				"input": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.parseArguments(tt.input)

			for key, expected := range tt.expected {
				if result[key] != expected {
					t.Errorf("for key %s: expected %q, got %q", key, expected, result[key])
				}
			}
		})
	}
}

func TestAddCommand(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	executor := NewExecutor(tempDir, nil)

	cmd := &Command{
		Name:     "new",
		Template: "New command",
	}
	executor.AddCommand(cmd)

	retrieved, ok := executor.Get("new")
	if !ok {
		t.Fatal("expected command to be added")
	}
	if retrieved.Template != "New command" {
		t.Errorf("unexpected template: %s", retrieved.Template)
	}
}

func TestRemoveCommand(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &types.Config{
		Command: map[string]types.CommandConfig{
			"toremove": {Template: "Remove me"},
		},
	}

	executor := NewExecutor(tempDir, cfg)

	// Command should exist initially
	_, ok := executor.Get("toremove")
	if !ok {
		t.Fatal("expected command to exist before removal")
	}

	// Remove command
	removed := executor.RemoveCommand("toremove")
	if !removed {
		t.Error("expected RemoveCommand to return true")
	}

	// Command should not exist after removal
	_, ok = executor.Get("toremove")
	if ok {
		t.Error("expected command to be removed")
	}

	// Removing non-existent command should return false
	removed = executor.RemoveCommand("nonexistent")
	if removed {
		t.Error("expected RemoveCommand to return false for nonexistent command")
	}
}

func TestReload(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &types.Config{
		Command: map[string]types.CommandConfig{
			"original": {Template: "Original"},
		},
	}

	executor := NewExecutor(tempDir, cfg)

	// Add a dynamic command
	executor.AddCommand(&Command{Name: "dynamic", Template: "Dynamic"})

	if len(executor.List()) != 2 {
		t.Errorf("expected 2 commands before reload, got %d", len(executor.List()))
	}

	// Reload should reset to config only
	executor.Reload()

	if len(executor.List()) != 1 {
		t.Errorf("expected 1 command after reload, got %d", len(executor.List()))
	}

	_, ok := executor.Get("dynamic")
	if ok {
		t.Error("dynamic command should be removed after reload")
	}
}

func TestBuiltinCommands(t *testing.T) {
	builtins := BuiltinCommands()

	expectedNames := []string{"help", "clear", "compact", "reset", "undo", "share", "export"}

	if len(builtins) != len(expectedNames) {
		t.Errorf("expected %d builtin commands, got %d", len(expectedNames), len(builtins))
	}

	nameSet := make(map[string]bool)
	for _, cmd := range builtins {
		nameSet[cmd.Name] = true
		if cmd.Source != "builtin" {
			t.Errorf("expected source 'builtin' for %s, got %s", cmd.Name, cmd.Source)
		}
		if cmd.Description == "" {
			t.Errorf("expected description for builtin command %s", cmd.Name)
		}
	}

	for _, name := range expectedNames {
		if !nameSet[name] {
			t.Errorf("expected builtin command %s to exist", name)
		}
	}
}

func TestTemplateFunctions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name     string
		template string
		args     string
		expected string
	}{
		{
			name:     "trim",
			template: "{{ trim .input }}",
			args:     "  hello  ",
			expected: "hello",
		},
		{
			name:     "upper",
			template: "{{ upper .input }}",
			args:     "hello",
			expected: "HELLO",
		},
		{
			name:     "lower",
			template: "{{ lower .input }}",
			args:     "HELLO",
			expected: "hello",
		},
		{
			name:     "default with empty",
			template: `{{ default "fallback" "" }}`,
			args:     "",
			expected: "fallback",
		},
		{
			name:     "default with value",
			template: `{{ default "fallback" "actual" }}`,
			args:     "",
			expected: "actual",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &types.Config{
				Command: map[string]types.CommandConfig{
					"test": {Template: tt.template},
				},
			}
			executor := NewExecutor(tempDir, cfg)

			result, err := executor.Execute(context.Background(), "test", tt.args)
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			if result.Prompt != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Prompt)
			}
		})
	}
}

func TestEnvFunction(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set test environment variable
	os.Setenv("TEST_COMMAND_VAR", "test_value")
	defer os.Unsetenv("TEST_COMMAND_VAR")

	cfg := &types.Config{
		Command: map[string]types.CommandConfig{
			"envtest": {Template: `{{ env "TEST_COMMAND_VAR" }}`},
		},
	}

	executor := NewExecutor(tempDir, cfg)

	result, err := executor.Execute(context.Background(), "envtest", "")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Prompt != "test_value" {
		t.Errorf("expected 'test_value', got %q", result.Prompt)
	}
}

func TestWorkDirInContext(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &types.Config{
		Command: map[string]types.CommandConfig{
			"workdir": {Template: "{{ .workDir }}"},
		},
	}

	executor := NewExecutor(tempDir, cfg)

	result, err := executor.Execute(context.Background(), "workdir", "")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Prompt != tempDir {
		t.Errorf("expected %q, got %q", tempDir, result.Prompt)
	}
}

func TestNilConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Should not panic with nil config
	executor := NewExecutor(tempDir, nil)

	if executor == nil {
		t.Fatal("expected non-nil executor")
	}

	commands := executor.List()
	if len(commands) != 0 {
		t.Errorf("expected 0 commands with nil config, got %d", len(commands))
	}
}
