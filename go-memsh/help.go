package memsh

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// CommandHelp stores help information for a command
type CommandHelp struct {
	Name        string
	Usage       string
	Description string
	Examples    []string
}

// AllCommands returns help information for all commands
var commandHelp = map[string]CommandHelp{
	"help": {
		Name:        "help",
		Usage:       "help [command]",
		Description: "Display help information about commands",
		Examples: []string{
			"help          # List all available commands",
			"help grep     # Show detailed help for grep command",
		},
	},
	"pwd": {
		Name:        "pwd",
		Usage:       "pwd",
		Description: "Print the current working directory",
		Examples: []string{
			"pwd",
		},
	},
	"cd": {
		Name:        "cd",
		Usage:       "cd [directory]",
		Description: "Change the current working directory",
		Examples: []string{
			"cd /home/user",
			"cd ..          # Go to parent directory",
			"cd             # Go to root directory",
		},
	},
	"ls": {
		Name:        "ls",
		Usage:       "ls [-la] [path...]",
		Description: "List directory contents",
		Examples: []string{
			"ls",
			"ls -l          # Long format",
			"ls -a          # Include hidden files",
			"ls -la /home   # Long format with hidden files",
		},
	},
	"cat": {
		Name:        "cat",
		Usage:       "cat [file...]",
		Description: "Concatenate and display file contents",
		Examples: []string{
			"cat file.txt",
			"cat file1.txt file2.txt",
			"cat            # Read from stdin",
		},
	},
	"echo": {
		Name:        "echo",
		Usage:       "echo [text...]",
		Description: "Display a line of text",
		Examples: []string{
			"echo Hello World",
			"echo \"Hello $USER\"",
			"echo test > file.txt",
		},
	},
	"mkdir": {
		Name:        "mkdir",
		Usage:       "mkdir [-p] directory...",
		Description: "Create directories",
		Examples: []string{
			"mkdir testdir",
			"mkdir -p /path/to/nested/dir",
			"mkdir dir1 dir2 dir3",
		},
	},
	"rm": {
		Name:        "rm",
		Usage:       "rm [-rf] file...",
		Description: "Remove files or directories",
		Examples: []string{
			"rm file.txt",
			"rm -r directory",
			"rm -rf /tmp/*",
			"rm -f file.txt  # Force, ignore errors",
		},
	},
	"touch": {
		Name:        "touch",
		Usage:       "touch file...",
		Description: "Create empty files or update timestamps",
		Examples: []string{
			"touch newfile.txt",
			"touch file1.txt file2.txt",
		},
	},
	"cp": {
		Name:        "cp",
		Usage:       "cp [-r] source... destination",
		Description: "Copy files or directories",
		Examples: []string{
			"cp file1.txt file2.txt",
			"cp file.txt /dest/dir/",
			"cp -r srcdir destdir",
		},
	},
	"mv": {
		Name:        "mv",
		Usage:       "mv source destination",
		Description: "Move or rename files",
		Examples: []string{
			"mv oldname.txt newname.txt",
			"mv file.txt /dest/dir/",
		},
	},
	"grep": {
		Name:        "grep",
		Usage:       "grep [-ivnc] pattern [file...]",
		Description: "Search for patterns in files",
		Examples: []string{
			"grep error logfile.txt",
			"grep -i ERROR file.txt     # Case-insensitive",
			"grep -n pattern file.txt   # Show line numbers",
			"grep -c pattern file.txt   # Count matches",
			"grep -v pattern file.txt   # Invert match",
			"cat file.txt | grep error",
		},
	},
	"head": {
		Name:        "head",
		Usage:       "head [-n count] [file...]",
		Description: "Output the first part of files",
		Examples: []string{
			"head file.txt",
			"head -n 5 file.txt",
			"head -20 file.txt",
		},
	},
	"tail": {
		Name:        "tail",
		Usage:       "tail [-n count] [file...]",
		Description: "Output the last part of files",
		Examples: []string{
			"tail file.txt",
			"tail -n 5 file.txt",
			"tail -20 file.txt",
		},
	},
	"wc": {
		Name:        "wc",
		Usage:       "wc [-lwc] [file...]",
		Description: "Count lines, words, and bytes",
		Examples: []string{
			"wc file.txt",
			"wc -l file.txt      # Lines only",
			"wc -w file.txt      # Words only",
			"wc -c file.txt      # Bytes only",
			"cat file.txt | wc -l",
		},
	},
	"sort": {
		Name:        "sort",
		Usage:       "sort [-run] [file...]",
		Description: "Sort lines of text",
		Examples: []string{
			"sort file.txt",
			"sort -r file.txt    # Reverse order",
			"sort -u file.txt    # Unique lines only",
			"sort -n numbers.txt # Numeric sort",
		},
	},
	"uniq": {
		Name:        "uniq",
		Usage:       "uniq [-c] [file]",
		Description: "Report or omit repeated lines",
		Examples: []string{
			"uniq file.txt",
			"uniq -c file.txt    # Count occurrences",
			"sort file.txt | uniq",
		},
	},
	"find": {
		Name:        "find",
		Usage:       "find [path] [-name pattern] [-type type]",
		Description: "Search for files in a directory hierarchy",
		Examples: []string{
			"find /home",
			"find . -name '*.txt'",
			"find /var -type f     # Files only",
			"find /var -type d     # Directories only",
			"find . -name '*.log' -type f",
		},
	},
	"env": {
		Name:        "env",
		Usage:       "env",
		Description: "Display all exported environment variables",
		Examples: []string{
			"env",
			"env | grep PATH",
		},
	},
	"export": {
		Name:        "export",
		Usage:       "export [VAR=value...]",
		Description: "Set and export environment variables",
		Examples: []string{
			"export PATH=/usr/bin",
			"export MY_VAR=\"Hello World\"",
			"export VAR1=val1 VAR2=val2",
			"export              # List exported variables",
		},
	},
	"set": {
		Name:        "set",
		Usage:       "set [VAR=value...]",
		Description: "Set shell variables (non-exported)",
		Examples: []string{
			"set LOCAL_VAR=value",
			"set VAR1=val1 VAR2=val2",
			"set                 # List all variables",
		},
	},
	"unset": {
		Name:        "unset",
		Usage:       "unset variable...",
		Description: "Unset variables",
		Examples: []string{
			"unset MY_VAR",
			"unset VAR1 VAR2 VAR3",
		},
	},
	"test": {
		Name:        "test / [",
		Usage:       "test expression  or  [ expression ]",
		Description: "Evaluate conditional expressions",
		Examples: []string{
			"test -f file.txt",
			"[ -d /home ]",
			"[ -f file.txt ] && echo exists",
			"[ \"$VAR\" = \"value\" ]",
			"[ $NUM -eq 10 ]",
			"",
			"File tests: -f (file) -d (dir) -e (exists) -s (non-empty)",
			"String tests: = != -z (empty) -n (non-empty)",
			"Numeric tests: -eq -ne -lt -le -gt -ge",
		},
	},
	"sleep": {
		Name:        "sleep",
		Usage:       "sleep seconds",
		Description: "Sleep for specified seconds",
		Examples: []string{
			"sleep 5",
			"sleep 1",
		},
	},
	"true": {
		Name:        "true",
		Usage:       "true",
		Description: "Return success (exit status 0)",
		Examples: []string{
			"true && echo success",
		},
	},
	"false": {
		Name:        "false",
		Usage:       "false",
		Description: "Return failure (exit status 1)",
		Examples: []string{
			"false || echo failed",
		},
	},
	"exit": {
		Name:        "exit",
		Usage:       "exit [code]",
		Description: "Exit the shell with optional status code",
		Examples: []string{
			"exit",
			"exit 0",
			"exit 1",
		},
	},
	"import-file": {
		Name:        "import-file",
		Usage:       "import-file local-path memfs-path",
		Description: "Import a file from local filesystem to memory filesystem",
		Examples: []string{
			"import-file /etc/hosts /hosts",
			"import-file ~/data.txt /imported/data.txt",
		},
	},
	"import-dir": {
		Name:        "import-dir",
		Usage:       "import-dir local-path memfs-path",
		Description: "Import a directory recursively from local filesystem",
		Examples: []string{
			"import-dir /home/user/project /project",
			"import-dir ~/data /imported/data",
		},
	},
	"export-file": {
		Name:        "export-file",
		Usage:       "export-file memfs-path local-path",
		Description: "Export a file from memory filesystem to local filesystem",
		Examples: []string{
			"export-file /output.txt ~/output.txt",
			"export-file /data.log /tmp/data.log",
		},
	},
	"export-dir": {
		Name:        "export-dir",
		Usage:       "export-dir memfs-path local-path",
		Description: "Export a directory recursively to local filesystem",
		Examples: []string{
			"export-dir /project ~/backup/project",
			"export-dir /output /tmp/output",
		},
	},
	"jq": {
		Name:        "jq",
		Usage:       "jq [options] filter [file...]",
		Description: "JSON processor - query and manipulate JSON data",
		Examples: []string{
			"echo '{\"name\":\"John\",\"age\":30}' | jq .name",
			"jq '.users[] | select(.age > 25)' data.json",
			"jq -r '.name' data.json              # Raw output (no quotes)",
			"jq -c '.[]' data.json                # Compact output",
			"jq '.[] | {name, email}' users.json  # Select fields",
			"cat api.json | jq '.results[0]'      # Extract first result",
		},
	},
	"curl": {
		Name:        "curl",
		Usage:       "curl [options] URL",
		Description: "Transfer data from or to a server using HTTP/HTTPS",
		Examples: []string{
			"curl https://api.example.com/data",
			"curl -X POST -d '{\"key\":\"value\"}' https://api.example.com",
			"curl -H 'Authorization: Bearer token' https://api.example.com",
			"curl -o output.json https://api.example.com/data",
			"curl -s https://api.example.com        # Silent mode",
			"curl -i https://api.example.com        # Include headers",
			"curl -L https://shortened.url          # Follow redirects",
		},
	},
}

// cmdHelp implements the help command
func (s *Shell) cmdHelp(ctx context.Context, args []string) error {
	if len(args) == 1 {
		// List all commands
		return s.listAllCommands(ctx)
	}

	// Show help for specific command
	cmdName := args[1]
	return s.showCommandHelp(ctx, cmdName)
}

// listAllCommands prints a list of all available commands
func (s *Shell) listAllCommands(ctx context.Context) error {
	_, stdout, _ := s.stdio(ctx)
	fmt.Fprintln(stdout, "MemSh - Available Commands")
	fmt.Fprintln(stdout, "===========================")
	fmt.Fprintln(stdout)

	// Group commands by category
	categories := map[string][]string{
		"File Operations": {
			"pwd", "cd", "ls", "cat", "mkdir", "rm", "touch", "cp", "mv",
		},
		"Text Processing": {
			"echo", "grep", "head", "tail", "wc", "sort", "uniq",
		},
		"File Search": {
			"find",
		},
		"Environment": {
			"env", "export", "set", "unset",
		},
		"Control Flow & Testing": {
			"test", "true", "false",
		},
		"Utilities": {
			"sleep", "exit",
		},
		"HTTP & JSON": {
			"curl", "jq",
		},
		"Import/Export": {
			"import-file", "import-dir", "export-file", "export-dir",
		},
		"Help": {
			"help",
		},
	}

	categoryOrder := []string{
		"File Operations",
		"Text Processing",
		"File Search",
		"Environment",
		"Control Flow & Testing",
		"Utilities",
		"HTTP & JSON",
		"Import/Export",
		"Help",
	}

	for _, category := range categoryOrder {
		commands := categories[category]
		fmt.Fprintf(stdout, "%s:\n", category)

		for _, cmd := range commands {
			if help, ok := commandHelp[cmd]; ok {
				fmt.Fprintf(stdout, "  %-15s %s\n", help.Name, help.Description)
			}
		}
		fmt.Fprintln(stdout)
	}

	fmt.Fprintln(stdout, "For detailed help on a specific command, use: help <command>")
	fmt.Fprintln(stdout, "Example: help grep")
	fmt.Fprintln(stdout)

	return nil
}

// showCommandHelp prints detailed help for a specific command
func (s *Shell) showCommandHelp(ctx context.Context, cmdName string) error {
	_, stdout, _ := s.stdio(ctx)
	// Handle aliases
	if cmdName == "[" {
		cmdName = "test"
	}

	help, ok := commandHelp[cmdName]
	if !ok {
		return fmt.Errorf("help: no help available for '%s'", cmdName)
	}

	fmt.Fprintf(stdout, "Command: %s\n", help.Name)
	fmt.Fprintln(stdout, strings.Repeat("=", len("Command: "+help.Name)))
	fmt.Fprintln(stdout)

	fmt.Fprintf(stdout, "Usage: %s\n", help.Usage)
	fmt.Fprintln(stdout)

	fmt.Fprintf(stdout, "Description:\n  %s\n", help.Description)
	fmt.Fprintln(stdout)

	if len(help.Examples) > 0 {
		fmt.Fprintln(stdout, "Examples:")
		for _, example := range help.Examples {
			if example == "" {
				fmt.Fprintln(stdout)
			} else {
				fmt.Fprintf(stdout, "  %s\n", example)
			}
		}
		fmt.Fprintln(stdout)
	}

	return nil
}

// GetCommandNames returns a sorted list of all command names
func GetCommandNames() []string {
	names := make([]string, 0, len(commandHelp))
	for name := range commandHelp {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
