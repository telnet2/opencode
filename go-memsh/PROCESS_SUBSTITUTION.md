# Process Substitution in MemSh

Process substitution is an advanced shell feature that allows the output of a command to be treated as a file. The syntax `<(command)` creates a virtual file that can be read by other commands.

## Overview

MemSh implements process substitution through a **virtual pipe/device layer** that emulates `/dev/fd/N` file descriptors in the in-memory filesystem. This allows commands to read from the output of other commands as if they were files.

## Architecture

### Components

1. **VirtualPipe**: An in-memory pipe that stores command output
2. **PipeManager**: Manages lifecycle of virtual pipes
3. **VirtualFile**: File-like interface for reading from pipes
4. **/dev/fd/N emulation**: Virtual device paths that map to pipes

### How It Works

```
<(command)  →  VirtualPipe  →  /dev/fd/3  →  Reading Command
               ↓
           Background
           Execution
```

1. Command inside `<(...)` executes in background
2. Output is written to a `VirtualPipe`
3. Virtual pipe is accessible at `/dev/fd/N`
4. Other commands can read from this path

## Current Implementation Status

### ✅ Implemented

- Virtual pipe infrastructure
- /dev/fd/N path handling
- Background command execution
- Pipe lifecycle management
- Thread-safe pipe operations

### ⚠️ Manual Usage Required

Due to limitations in integrating with mvdan/sh's parser, process substitution currently requires manual setup. The infrastructure is complete, but automatic syntax parsing (`<(command)`) is not yet integrated.

### 🔜 Future Enhancement

Full automatic process substitution parsing will be added in a future release.

## Usage Examples

### Example 1: Basic Manual Process Substitution

```go
package main

import (
	"context"
	"fmt"
	"github.com/spf13/afero"
	"github.com/telnet2/go-practice/go-memsh"
)

func main() {
	fs := afero.NewMemMapFs()
	shell, _ := memsh.NewShell(fs)
	ctx := context.Background()

	// Create a virtual pipe for process substitution
	pipe := shell.PipeManager().CreatePipe()

	// Create process substitution: <(echo "Hello World")
	ps := &memsh.ProcessSubstitution{
		Command: "echo 'Hello World'",
		IsInput: true,
		Pipe:    pipe,
	}

	// Execute in background
	go ps.ExecuteInBackground(ctx, shell)

	// Wait for completion
	pipe.Wait()

	// Now you can use pipe.GetPath() (e.g., /dev/fd/3) in commands
	fmt.Printf("Virtual path: %s\n", pipe.GetPath())
	fmt.Printf("Contents: %s\n", string(pipe.GetContents()))
}
```

### Example 2: Emulating `diff <(cmd1) <(cmd2)`

```go
// Create two process substitutions
pipe1 := shell.PipeManager().CreatePipe()
ps1 := &memsh.ProcessSubstitution{
	Command: "echo 'content A'",
	IsInput: true,
	Pipe:    pipe1,
}
go ps1.ExecuteInBackground(ctx, shell)

pipe2 := shell.PipeManager().CreatePipe()
ps2 := &memsh.ProcessSubstitution{
	Command: "echo 'content B'",
	IsInput: true,
	Pipe:    pipe2,
}
go ps2.ExecuteInBackground(ctx, shell)

// Wait for both
pipe1.Wait()
pipe2.Wait()

// Now use the paths in a command
// If diff command existed:
// shell.Run(ctx, fmt.Sprintf("diff %s %s", pipe1.GetPath(), pipe2.GetPath()))

// For now, manually compare:
if string(pipe1.GetContents()) != string(pipe2.GetContents()) {
	fmt.Println("Contents differ")
}
```

### Example 3: Reading from Virtual File

```go
// Create and populate a pipe
pipe := shell.PipeManager().CreatePipe()
ps := &memsh.ProcessSubstitution{
	Command: "cat large-file.txt | grep 'important'",
	IsInput: true,
	Pipe:    pipe,
}
go ps.ExecuteInBackground(ctx, shell)
pipe.Wait()

// Open the virtual file
file, err := shell.OpenHandler(ctx, pipe.GetPath(), 0, 0)
if err != nil {
	panic(err)
}
defer file.Close()

// Read from it
buf := make([]byte, 1024)
n, err := file.Read(buf)
fmt.Printf("Read %d bytes: %s\n", n, string(buf[:n]))
```

## API Reference

### VirtualPipe

```go
type VirtualPipe struct {
	// Internal fields
}

// NewVirtualPipe creates a new virtual pipe with given ID
func NewVirtualPipe(id int) *VirtualPipe

// Write writes data to the pipe
func (vp *VirtualPipe) Write(p []byte) (n int, err error)

// Read reads data from the pipe
func (vp *VirtualPipe) Read(p []byte) (n int, err error)

// Close marks the pipe as closed
func (vp *VirtualPipe) Close() error

// Done signals command execution completion
func (vp *VirtualPipe) Done()

// Wait waits for command execution to complete
func (vp *VirtualPipe) Wait()

// GetPath returns the virtual path (/dev/fd/N)
func (vp *VirtualPipe) GetPath() string

// GetContents returns all buffered data
func (vp *VirtualPipe) GetContents() []byte
```

### PipeManager

```go
type PipeManager struct {
	// Internal fields
}

// NewPipeManager creates a new pipe manager
func NewPipeManager() *PipeManager

// CreatePipe creates a new virtual pipe
func (pm *PipeManager) CreatePipe() *VirtualPipe

// GetPipe retrieves a pipe by ID
func (pm *PipeManager) GetPipe(id int) (*VirtualPipe, bool)

// ClosePipe closes and removes a pipe
func (pm *PipeManager) ClosePipe(id int)

// CloseAll closes all pipes
func (pm *PipeManager) CloseAll()
```

### ProcessSubstitution

```go
type ProcessSubstitution struct {
	Command string       // Command to execute
	IsInput bool         // true for <(cmd), false for >(cmd)
	Pipe    *VirtualPipe // Associated virtual pipe
}

// ExecuteInBackground executes the command and writes to pipe
func (ps *ProcessSubstitution) ExecuteInBackground(ctx context.Context, shell *Shell) error
```

## Technical Details

### Thread Safety

All pipe operations are thread-safe:
- `sync.RWMutex` protects pipe buffer access
- Channel-based signaling for ready/done states
- Atomic pipe creation with unique IDs

### Memory Management

- Pipes store output in memory (`bytes.Buffer`)
- Automatic cleanup when pipes are closed
- Pipes are garbage collected after closure

### File Descriptor Range

Virtual file descriptors start at 3:
- 0: stdin (reserved)
- 1: stdout (reserved)
- 2: stderr (reserved)
- 3+: Virtual pipes

### Path Resolution

Paths like `/dev/fd/3` are intercepted in:
- `openHandler`: Returns VirtualFile for reading
- `statHandler`: Returns VirtualFileInfo for stat calls

## Limitations

### Current Limitations

1. **No Automatic Parsing**: Cannot use `<(command)` syntax directly in shell commands yet
2. **Manual Setup Required**: Must explicitly create pipes and process substitutions
3. **Read-Only**: Process substitution currently only supports input `<(...)`, not output `>(...)`

### Future Enhancements

1. **Automatic Syntax Parsing**: Detect and process `<(...)` in command arguments
2. **Output Substitution**: Support `>(command)` for writing
3. **Integration with mvdan/sh**: Deep integration with parser for seamless support
4. **Nested Substitutions**: Support `<(cmd1 <(cmd2))`

## Best Practices

1. **Always Wait**: Call `pipe.Wait()` before reading to ensure command completion
2. **Cleanup**: Use `defer pipe.Close()` or `pipeManager.CloseAll()`
3. **Error Handling**: Check errors from `ExecuteInBackground()`
4. **Context Cancellation**: Pass cancellable context for timeout control

## Comparison with Bash

| Feature | Bash | MemSh (Current) | MemSh (Future) |
|---------|------|-----------------|----------------|
| Syntax `<(cmd)` | ✅ Automatic | ❌ Manual | ✅ Planned |
| /dev/fd/N | ✅ OS-level | ✅ Emulated | ✅ Emulated |
| Background execution | ✅ Yes | ✅ Yes | ✅ Yes |
| Output substitution `>(cmd)` | ✅ Yes | ❌ Not yet | ✅ Planned |
| Nested substitution | ✅ Yes | ❌ Not yet | ✅ Planned |

## Troubleshooting

### Issue: "no such file or directory: /dev/fd/3"

**Cause**: Pipe not created or already closed

**Solution**: Ensure `CreatePipe()` was called and pipe wasn't closed

### Issue: Reading empty content

**Cause**: Reading before command execution completes

**Solution**: Call `pipe.Wait()` before reading

### Issue: Command hangs

**Cause**: Deadlock in pipe I/O

**Solution**: Ensure background command execution is started with `go`

## Example: Complete Workflow

```go
package main

import (
	"context"
	"fmt"
	"github.com/spf13/afero"
	"github.com/telnet2/go-practice/go-memsh"
)

func main() {
	// Setup
	fs := afero.NewMemMapFs()
	shell, _ := memsh.NewShell(fs)
	ctx := context.Background()

	// Create test data
	shell.Run(ctx, "echo 'line 1\nline 2\nline 3' > /data.txt")

	// Process substitution: <(cat /data.txt | grep '2')
	pipe := shell.PipeManager().CreatePipe()
	ps := &memsh.ProcessSubstitution{
		Command: "cat /data.txt | grep '2'",
		IsInput: true,
		Pipe:    pipe,
	}

	// Execute in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- ps.ExecuteInBackground(ctx, shell)
	}()

	// Wait for completion
	pipe.Wait()

	// Check for errors
	if err := <-errChan; err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Use the result
	fmt.Printf("Path: %s\n", pipe.GetPath())
	fmt.Printf("Output:\n%s\n", string(pipe.GetContents()))

	// Cleanup
	shell.PipeManager().ClosePipe(pipe.ID())
}
```

## See Also

- [POSIX_COMPLIANCE.md](POSIX_COMPLIANCE.md) - POSIX compliance details
- [README.md](README.md) - General MemSh documentation
- mvdan/sh documentation - Underlying shell parser
