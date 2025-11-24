# go-memsh

A sh-compatible shell interpreter running on afero.FS (in-memory filesystem).

## Features

- **Shell Parsing**: Uses [mvdan/sh](https://github.com/mvdan/sh) for full sh-compatible script parsing
- **In-Memory Filesystem**: Built on [afero](https://github.com/spf13/afero) for fast, isolated filesystem operations
- **Pipes**: Full support for command piping (`|`)
- **Redirections**: Supports `>`, `>>`, `<`, and `2>&1`
- **Process Substitution**: Virtual pipe layer supporting `<(command)` through /dev/fd emulation
- **Control Flow**: Supports `if/then/else`, `for` loops, `while` loops
- **Variable Expansion**: Full support for environment variable expansion (`$VAR`, `${VAR}`)
- **Test Expressions**: Full `test`/`[` command support with file, string, and numeric tests
- **Environment Management**: Set, export, and unset environment variables
- **Text Processing**: grep, head, tail, wc, sort, uniq
- **File Search**: find command with name patterns and type filters
- **HTTP & JSON**: curl for HTTP requests, jq for JSON processing
- **Import/Export**: Commands to move files/directories between local filesystem and memory filesystem
- **REST API**: HTTP endpoints for session management
- **WebSocket JSON-RPC**: Real-time command execution over WebSocket
- **Web UI**: Modern NextJS-based web application with file explorer

## Built-in Commands

### File Operations
- `pwd` - Print working directory
- `cd` - Change directory (supports `$HOME` for bare `cd`, `cd -` for previous directory)
- `ls` - List directory contents (supports `-l`, `-a`, `-R`)
- `cat` - Concatenate and display files
- `mkdir` - Create directories (supports `-p`)
- `rm` - Remove files/directories (supports `-r`, `-f`, `-i`)
- `touch` - Create empty files or update timestamps
- `cp` - Copy files/directories (supports `-r`, `-p`)
- `mv` - Move/rename files

### Text Operations
- `echo` - Print text (supports `-n` to suppress newline)
- `cat` - Concatenate and display files (also used above)
- `grep` - Search for patterns in files (supports `-i`, `-v`, `-n`, `-c`, `-q`)
- `head` - Output the first part of files (supports `-n`)
- `tail` - Output the last part of files (supports `-n`)
- `wc` - Count lines, words, and bytes (supports `-l`, `-w`, `-c`)
- `sort` - Sort lines of text (supports `-r`, `-u`, `-n`)
- `uniq` - Report or omit repeated lines (supports `-c`)

### File Search
- `find` - Search for files in a directory hierarchy (supports `-name`, `-type`)

### HTTP & JSON
- `curl` - Transfer data from or to a server (supports `-X`, `-d`, `-H`, `-o`, `-s`, `-i`, `-L`)
- `jq` - Command-line JSON processor (supports filters, `-r` for raw output, `-c` for compact)

### Environment Management
- `env` - Display or run commands with modified environment (supports `-i`, `-u`, `VAR=value command`)
- `export` - Set and export environment variables
- `set` - Set shell variables (non-exported, **Note**: Non-POSIX behavior)
- `unset` - Unset variables

### Control Flow
- `if/then/else/fi` - Conditional execution
- `for/do/done` - Loop over lists
- `while/do/done` - Loop while condition is true

### Test Expressions
- `test` or `[` - Evaluate expressions
  - File tests: `-f`, `-d`, `-e`, `-r`, `-w`, `-x`, `-s`, `-h/-L`, `-b`, `-c`, `-p`, `-S`
  - String tests: `-z`, `-n`, `=`, `!=`
  - Numeric tests: `-eq`, `-ne`, `-lt`, `-le`, `-gt`, `-ge`

### Utilities
- `help` - Display help information about commands
  - `help` - List all available commands
  - `help <command>` - Show detailed help for a specific command
- `sleep` - Sleep for specified seconds
- `true` - Return success
- `false` - Return failure
- `exit` - Exit the shell with optional status code

### Import/Export Commands
- `import-file <local-path> <memfs-path>` - Import a file from local filesystem to memory filesystem
- `import-dir <local-path> <memfs-path>` - Import a directory recursively from local filesystem
- `export-file <memfs-path> <local-path>` - Export a file from memory filesystem to local filesystem
- `export-dir <memfs-path> <local-path>` - Export a directory recursively to local filesystem

## Installation

### Build from Source

```bash
# Build CLI
cd cmd/memsh
go build -o memsh

# Build Web Shell
cd ../webshell
go build -o webshell

# Build API Server
cd ../apiserver
go build -o apiserver

# Build API Client Example
cd ../apiclient
go build -o apiclient
```

### Install Globally

```bash
# Install CLI
go install github.com/telnet2/go-practice/go-memsh/cmd/memsh@latest

# Install Web Shell
go install github.com/telnet2/go-practice/go-memsh/cmd/webshell@latest
```

## Usage

### CLI Application

#### Interactive Mode

```bash
memsh
```

This starts an interactive shell session:

```
Welcome to MemSh - Shell running on afero.FS
Type 'exit' or press Ctrl+D to exit

memsh> pwd
/
memsh> mkdir test
memsh> cd test
memsh> echo "Hello" > file.txt
memsh> cat file.txt
Hello
```

#### Demo Mode

```bash
memsh --demo
```

Runs a comprehensive demonstration of all features.

#### Execute Inline Script

```bash
memsh -c 'echo "Hello, World!" > /test.txt; cat /test.txt'
```

#### Execute Script from File

```bash
memsh -f script.sh
```

#### Help

```bash
# CLI help
memsh --help

# In-shell help (list all commands)
memsh
memsh> help

# Get help for a specific command
memsh> help grep
memsh> help find
```

### Web Shell

Start the web server:

```bash
webshell --addr :8080
```

Then open **http://localhost:8080** in your browser to access the interactive web shell.

**Features:**
- Full terminal emulation in browser
- WebSocket-based real-time communication
- Command history (Arrow Up/Down)
- Clean, VS Code-inspired UI
- Each connection gets its own isolated in-memory filesystem

**Custom Port:**

```bash
webshell --addr :3000
```

### API Server and Web Application

#### Start the API Server

```bash
cd cmd/apiserver
go run main.go -port 8080
```

The API server provides:
- REST API for session management (create, list, remove)
- WebSocket JSON-RPC endpoint for command execution
- See [API.md](API.md) for complete API documentation

#### Web Application

```bash
cd web
npm install
npm run dev
```

Then open **http://localhost:3000** in your browser.

**Features:**
- Session management with isolated filesystems
- Interactive terminal with command history
- MS Explorer-style file browser with tree view
- Import/export files and directories
- Real-time command execution via WebSocket
- Modern dark theme UI

See [web/README.md](web/README.md) for detailed documentation.

## Examples

### Pipes and Redirection

```bash
# Pipe example
echo "Hello World" | cat

# Redirect output
echo "Line 1" > file.txt
echo "Line 2" >> file.txt
cat file.txt

# Redirect stderr to stdout
some-command 2>&1 | cat
```

### Control Flow

```bash
# If statement
if [ -f file.txt ]; then
  echo "file.txt exists"
else
  echo "file.txt not found"
fi

# For loop
for i in 1 2 3; do
  echo "Number: $i"
done

# While loop
i=1
while [ $i -le 5 ]; do
  echo "Count: $i"
  i=$((i + 1))
done
```

### Environment Variables

```bash
# Export a variable
export MY_VAR="Hello World"
echo $MY_VAR

# Set a non-exported variable
set LOCAL_VAR=value

# List all exported variables
env

# List all variables (including non-exported)
set

# Unset a variable
unset MY_VAR
```

### Text Processing

```bash
# Search for pattern
grep "error" logfile.txt

# Case-insensitive search with line numbers
grep -i -n "warning" logfile.txt

# Count occurrences
grep -c "pattern" file.txt

# Show first 10 lines
head file.txt

# Show last 5 lines
tail -5 file.txt

# Count lines, words, and bytes
wc file.txt

# Sort lines
sort unsorted.txt

# Sort in reverse
sort -r file.txt

# Remove duplicate adjacent lines
uniq sorted.txt

# Count duplicates
uniq -c sorted.txt
```

### File Search

```bash
# Find all .txt files
find /path -name "*.txt"

# Find directories only
find /path -type d

# Find files only
find /path -type f
```

### HTTP & JSON Processing

```bash
# Fetch data from API
curl https://api.github.com/users/octocat

# POST request with JSON data
curl -X POST -d '{"key":"value"}' https://api.example.com/endpoint

# Save output to file
curl -o response.json https://api.example.com/data

# Silent mode (no progress bar)
curl -s https://api.example.com/data

# Process JSON with jq
echo '{"name":"John","age":30}' | jq .name

# Extract field from file
jq .name data.json

# Raw output (no quotes)
jq -r .name data.json

# Compact output
jq -c . data.json

# Complex filter
jq '.users[] | select(.age > 25)' users.json

# Combine curl and jq
curl -s https://api.github.com/users/octocat | jq .name
```

### Import/Export

```bash
# Import a file from local filesystem
import-file /etc/hosts /memfs/hosts

# Import entire directory
import-dir /home/user/project /memfs/project

# Export file to local filesystem
echo "test content" > /memfs/output.txt
export-file /memfs/output.txt /tmp/output.txt

# Export directory
export-dir /memfs/project /tmp/project-backup
```

### Complex Pipelines

```bash
# Find, filter, and count
find /var/log -name "*.log" | grep "access" | wc -l

# Sort and remove duplicates
cat file1.txt file2.txt | sort | uniq > merged.txt

# Search and display with line numbers
grep -n "error" *.log | head -20
```

## POSIX Compliance

MemSh aims for practical POSIX compatibility while maintaining simplicity. **Current compliance: ~75-80%** for implemented commands.

### Recent POSIX Improvements ✅

- **`cd` command**: Now supports `$HOME` for bare `cd` and `cd -` for previous directory
- **`echo` command**: Added `-n` flag to suppress trailing newline
- **`env` command**: Enhanced to run commands with modified environment (`env VAR=value command`, `-i`, `-u`)
- **`test` command**: Added missing file tests (`-h/-L`, `-b`, `-c`, `-p`, `-S`)

### Strengths

- ✅ Shell language features (pipes, redirections, control flow, variable expansion)
- ✅ Core file operations (pwd, cd, cat, ls, mkdir, rm, touch, cp, mv)
- ✅ Text processing (grep, head, tail, wc, sort, uniq)
- ✅ Test expressions (file, string, numeric tests)
- ✅ Utilities (sleep, true, false, exit)

### Known Limitations

- ⚠️ `set` command uses non-POSIX syntax (`set VAR=value` instead of `set -e`, `set -x`)
- ⚠️ Job control not implemented (background jobs, fg, bg, jobs)
- ⚠️ Some advanced command flags missing (see [POSIX_COMPLIANCE.md](POSIX_COMPLIANCE.md))
- ⚠️ Logical operators in `test` not implemented

### For Detailed Analysis

See [POSIX_COMPLIANCE.md](POSIX_COMPLIANCE.md) for a comprehensive command-by-command POSIX compatibility analysis including:
- Detailed compliance percentages
- Specific deviations from POSIX
- Recommendations for full compliance

## Library Usage

```go
package main

import (
    "context"
    "github.com/spf13/afero"
    "github.com/telnet2/go-practice/go-memsh"
)

func main() {
    // Create an in-memory filesystem
    fs := afero.NewMemMapFs()

    // Create shell
    shell, err := NewShell(fs)
    if err != nil {
        panic(err)
    }

    // Run commands
    ctx := context.Background()
    err = shell.Run(ctx, "echo 'Hello' > /test.txt")
    if err != nil {
        panic(err)
    }

    // Run interactive mode
    err = shell.RunInteractive(ctx)
}
```

## Architecture

### Core Library (package memsh)

- `shell.go` - Core shell interpreter with mvdan/sh integration
- `builtins.go` - Built-in command implementations (ls, cat, mkdir, etc.)
- `env.go` - Environment variable management (env, export, set, unset)
- `textutils.go` - Text processing utilities (grep, head, tail, wc, sort, uniq, find)
- `import_export.go` - Import/export functionality between filesystems

### Applications

- `cmd/memsh/` - CLI application for interactive and script execution
- `cmd/webshell/` - Web-based shell server with WebSocket support
  - `static/index.html` - Web UI with terminal emulation
- `cmd/apiserver/` - REST and WebSocket JSON-RPC API server
- `cmd/apiclient/` - Example API client in Go
- `web/` - NextJS web application with modern UI

### API Layer

- `api/` - API server implementation
  - `session.go` - Session management
  - `jsonrpc.go` - JSON-RPC 2.0 handler
  - `handlers.go` - HTTP and WebSocket handlers

## Dependencies

### Go Dependencies

- [mvdan.cc/sh/v3](https://github.com/mvdan/sh) - Shell parser and interpreter
- [github.com/spf13/afero](https://github.com/spf13/afero) - Filesystem abstraction layer
- [github.com/gorilla/websocket](https://github.com/gorilla/websocket) - WebSocket implementation
- [github.com/google/uuid](https://github.com/google/uuid) - UUID generation for sessions
- [github.com/itchyny/gojq](https://github.com/itchyny/gojq) - Pure Go implementation of jq

### Web Application Dependencies

- NextJS 14 - React framework with App Router
- TypeScript - Type-safe JavaScript
- WebSocket API - Real-time communication

## License

Part of go-practice repository.
