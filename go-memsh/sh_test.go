package memsh

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"mvdan.cc/sh/v3/interp"
)

// TestShBasic tests basic sh command execution
func TestShBasic(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Create a simple script
	script := `#!/bin/sh
echo "Hello from script"
`
	afero.WriteFile(fs, "/test.sh", []byte(script), 0644)

	// Execute the script
	stdout.Reset()
	err = shell.Run(ctx, "sh /test.sh")
	if err != nil {
		t.Fatalf("sh command failed: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output != "Hello from script" {
		t.Errorf("Expected 'Hello from script', got '%s'", output)
	}
}

// TestShArguments tests script with positional arguments
func TestShArguments(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Create a script that uses positional parameters
	script := `#!/bin/sh
echo "Script name: $0"
echo "First arg: $1"
echo "Second arg: $2"
echo "Third arg: $3"
`
	afero.WriteFile(fs, "/args.sh", []byte(script), 0644)

	// Execute the script with arguments
	stdout.Reset()
	err = shell.Run(ctx, "sh /args.sh apple banana cherry")
	if err != nil {
		t.Fatalf("sh command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Script name: /args.sh") {
		t.Errorf("Expected script name in output, got: %s", output)
	}
	if !strings.Contains(output, "First arg: apple") {
		t.Errorf("Expected first arg 'apple', got: %s", output)
	}
	if !strings.Contains(output, "Second arg: banana") {
		t.Errorf("Expected second arg 'banana', got: %s", output)
	}
	if !strings.Contains(output, "Third arg: cherry") {
		t.Errorf("Expected third arg 'cherry', got: %s", output)
	}
}

// TestShMultiLine tests scripts with multi-line commands
func TestShMultiLine(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Create a script with multi-line command
	script := `#!/bin/sh
if [ "$1" = "hello" ]; then
    echo "Greeting received"
    echo "Responding with hello"
fi
`
	afero.WriteFile(fs, "/multiline.sh", []byte(script), 0644)

	// Execute the script
	stdout.Reset()
	err = shell.Run(ctx, "sh /multiline.sh hello")
	if err != nil {
		t.Fatalf("sh command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Greeting received") {
		t.Errorf("Expected 'Greeting received' in output, got: %s", output)
	}
	if !strings.Contains(output, "Responding with hello") {
		t.Errorf("Expected 'Responding with hello' in output, got: %s", output)
	}
}

// TestShExitStatus tests exit status handling
func TestShExitStatus(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Create a script that exits with a specific code
	script := `#!/bin/sh
echo "Before exit"
exit 42
echo "After exit"
`
	afero.WriteFile(fs, "/exit.sh", []byte(script), 0644)

	// Execute the script
	stdout.Reset()
	err = shell.Run(ctx, "sh /exit.sh")

	// Check exit status
	if exitErr, ok := err.(interp.ExitStatus); ok {
		if exitErr != 42 {
			t.Errorf("Expected exit status 42, got %d", exitErr)
		}
	} else {
		t.Errorf("Expected ExitStatus error, got: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Before exit") {
		t.Errorf("Expected 'Before exit' in output, got: %s", output)
	}
	if strings.Contains(output, "After exit") {
		t.Errorf("Should not contain 'After exit', got: %s", output)
	}
}

// TestShEnvironment tests script accessing environment variables
func TestShEnvironment(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Set an environment variable
	shell.Run(ctx, "export MY_VAR=test_value")

	// Create a script that uses the environment variable
	script := `#!/bin/sh
echo "MY_VAR is: $MY_VAR"
`
	afero.WriteFile(fs, "/env.sh", []byte(script), 0644)

	// Execute the script
	stdout.Reset()
	err = shell.Run(ctx, "sh /env.sh")
	if err != nil {
		t.Fatalf("sh command failed: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if !strings.Contains(output, "MY_VAR is: test_value") {
		t.Errorf("Expected 'MY_VAR is: test_value', got '%s'", output)
	}
}

// TestShEnvironmentIsolation tests that script modifications don't affect parent
func TestShEnvironmentIsolation(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Set an environment variable in parent
	shell.Run(ctx, "export PARENT_VAR=parent_value")

	// Create a script that modifies environment
	script := `#!/bin/sh
export PARENT_VAR=modified_value
export SCRIPT_VAR=script_value
`
	afero.WriteFile(fs, "/modify_env.sh", []byte(script), 0644)

	// Execute the script
	err = shell.Run(ctx, "sh /modify_env.sh")
	if err != nil {
		t.Fatalf("sh command failed: %v", err)
	}

	// Check that parent environment is not modified
	stdout.Reset()
	shell.Run(ctx, "echo $PARENT_VAR")
	output := strings.TrimSpace(stdout.String())
	if output != "parent_value" {
		t.Errorf("Parent variable was modified! Expected 'parent_value', got '%s'", output)
	}

	// Check that script variable doesn't leak to parent
	stdout.Reset()
	shell.Run(ctx, "echo $SCRIPT_VAR")
	output = strings.TrimSpace(stdout.String())
	if output != "" {
		t.Errorf("Script variable leaked to parent! Expected empty, got '%s'", output)
	}
}

// TestShEnvironmentInheritance tests that when configured, script modifications propagate to parent
func TestShEnvironmentInheritance(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShellWithConfig(fs, ShellConfig{MergeScriptEnv: true})
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	script := `#!/bin/sh
cd /tmp
export CHILD_VAR=child
`
	if err := afero.WriteFile(fs, "/inherit_env.sh", []byte(script), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	stdout.Reset()
	if err := shell.Run(ctx, "mkdir -p /tmp && sh /inherit_env.sh"); err != nil {
		t.Fatalf("sh command failed: %v", err)
	}

	stdout.Reset()
	if err := shell.Run(ctx, "echo $CHILD_VAR"); err != nil {
		t.Fatalf("echo failed: %v", err)
	}
	if strings.TrimSpace(stdout.String()) != "child" {
		t.Fatalf("expected inherited variable, got %q", strings.TrimSpace(stdout.String()))
	}

	if shell.GetCwd() != "/tmp" {
		t.Fatalf("expected cwd to be updated to /tmp, got %s", shell.GetCwd())
	}
}

// TestShPipeline tests script with pipelines
func TestShPipeline(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Create a script with a pipeline
	script := `#!/bin/sh
echo -e "apple\nbanana\napricot\nberry" | grep "^a"
`
	afero.WriteFile(fs, "/pipeline.sh", []byte(script), 0644)

	// Execute the script
	stdout.Reset()
	err = shell.Run(ctx, "sh /pipeline.sh")
	if err != nil {
		t.Fatalf("sh command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "apple") || !strings.Contains(output, "apricot") {
		t.Errorf("Expected pipeline output, got: %s", output)
	}
	if strings.Contains(output, "banana") || strings.Contains(output, "berry") {
		t.Errorf("Pipeline didn't filter correctly, got: %s", output)
	}
}

// TestShFileOperations tests script performing file operations
func TestShFileOperations(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Create a script that creates and manipulates files
	script := `#!/bin/sh
echo "Creating files..."
echo "content1" > /tmp/file1.txt
echo "content2" > /tmp/file2.txt
cat /tmp/file1.txt /tmp/file2.txt
`
	afero.WriteFile(fs, "/fileops.sh", []byte(script), 0644)

	// Execute the script
	stdout.Reset()
	err = shell.Run(ctx, "sh /fileops.sh")
	if err != nil {
		t.Fatalf("sh command failed: %v", err)
	}

	// Check output
	output := stdout.String()
	if !strings.Contains(output, "Creating files...") {
		t.Errorf("Expected 'Creating files...' in output")
	}
	if !strings.Contains(output, "content1") {
		t.Errorf("Expected 'content1' in output")
	}
	if !strings.Contains(output, "content2") {
		t.Errorf("Expected 'content2' in output")
	}

	// Verify files were created
	content1, _ := afero.ReadFile(fs, "/tmp/file1.txt")
	if strings.TrimSpace(string(content1)) != "content1" {
		t.Errorf("File1 content incorrect: %s", string(content1))
	}
}

// TestShMissingFile tests error handling for missing script file
func TestShMissingFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Try to execute non-existent script
	err = shell.Run(ctx, "sh /nonexistent.sh")
	if err == nil {
		t.Error("Expected error for missing script file")
	}
	if !strings.Contains(err.Error(), "cannot open") {
		t.Errorf("Expected 'cannot open' error, got: %v", err)
	}
}

// TestShNoArgument tests error handling when no script file is provided
func TestShNoArgument(t *testing.T) {
	fs := afero.NewMemMapFs()
	shell, err := NewShell(fs)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()
	var stdout bytes.Buffer
	shell.SetIO(nil, &stdout, &stdout)

	// Try to execute sh without arguments
	err = shell.Run(ctx, "sh")
	if err == nil {
		t.Error("Expected error when no script file provided")
	}
	if !strings.Contains(err.Error(), "missing script file") {
		t.Errorf("Expected 'missing script file' error, got: %v", err)
	}
}
