package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/spf13/afero"
	"github.com/telnet2/go-practice/go-memsh"
)

func main() {
	// Command line flags
	scriptFile := flag.String("c", "", "Execute script from string")
	fileArg := flag.String("f", "", "Execute script from file")
	demo := flag.Bool("demo", false, "Run demo mode")

	flag.Parse()

	// Create an in-memory filesystem
	fs := afero.NewMemMapFs()

	// Create shell
	sh, err := memsh.NewShell(fs)
	if err != nil {
		log.Fatalf("Failed to create shell: %v", err)
	}

	ctx := context.Background()

	// Determine mode
	if *demo {
		runDemo(sh, ctx)
		return
	}

	if *scriptFile != "" {
		// Execute inline script
		if err := sh.Run(ctx, *scriptFile); err != nil {
			log.Fatalf("Script execution failed: %v", err)
		}
		return
	}

	if *fileArg != "" {
		// Execute script from file
		content, err := os.ReadFile(*fileArg)
		if err != nil {
			log.Fatalf("Failed to read script: %v", err)
		}
		if err := sh.Run(ctx, string(content)); err != nil {
			log.Fatalf("Script execution failed: %v", err)
		}
		return
	}

	// Check for positional arguments (backward compatibility)
	args := flag.Args()
	if len(args) > 0 {
		if args[0] == "demo" {
			runDemo(sh, ctx)
			return
		} else if args[0] == "script" && len(args) > 1 {
			content, err := os.ReadFile(args[1])
			if err != nil {
				log.Fatalf("Failed to read script: %v", err)
			}
			if err := sh.Run(ctx, string(content)); err != nil {
				log.Fatalf("Script execution failed: %v", err)
			}
			return
		}
	}

	// Default: interactive mode
	runInteractive(sh, ctx)
}

func runDemo(sh *memsh.Shell, ctx context.Context) {
	fmt.Println("=== MemSh Demo ===")
	fmt.Println()

	fmt.Println("=== Basic File Operations ===")
	runCommand(sh, ctx, "pwd")
	runCommand(sh, ctx, "mkdir -p /home/user/test")
	runCommand(sh, ctx, "cd /home/user/test")
	runCommand(sh, ctx, "pwd")
	runCommand(sh, ctx, "echo 'Hello, World!' > hello.txt")
	runCommand(sh, ctx, "cat hello.txt")
	runCommand(sh, ctx, "ls -la")

	fmt.Println("\n=== Environment Variables ===")
	runCommand(sh, ctx, "export MY_VAR=Hello")
	runCommand(sh, ctx, "echo \"Variable: $MY_VAR\"")

	fmt.Println("\n=== Pipes ===")
	runCommand(sh, ctx, "echo 'Line 1' > lines.txt")
	runCommand(sh, ctx, "echo 'Line 2' >> lines.txt")
	runCommand(sh, ctx, "echo 'Line 3' >> lines.txt")
	runCommand(sh, ctx, "cat lines.txt | wc -l")

	fmt.Println("\n=== Text Processing ===")
	runCommand(sh, ctx, "echo 'apple' > fruits.txt")
	runCommand(sh, ctx, "echo 'banana' >> fruits.txt")
	runCommand(sh, ctx, "echo 'cherry' >> fruits.txt")
	runCommand(sh, ctx, "echo 'apricot' >> fruits.txt")
	runCommand(sh, ctx, "grep 'ap' fruits.txt")
	runCommand(sh, ctx, "sort fruits.txt")

	fmt.Println("\n=== Control Flow - If Statement ===")
	script := `
if [ -f hello.txt ]; then
  echo "hello.txt exists"
else
  echo "hello.txt does not exist"
fi
`
	runCommand(sh, ctx, script)

	fmt.Println("\n=== Control Flow - For Loop ===")
	script = `
for i in 1 2 3 4 5; do
  echo "Number: $i"
done
`
	runCommand(sh, ctx, script)

	fmt.Println("\n=== File Operations ===")
	runCommand(sh, ctx, "mkdir dir1 dir2")
	runCommand(sh, ctx, "touch dir1/file1.txt dir1/file2.txt")
	runCommand(sh, ctx, "ls dir1")
	runCommand(sh, ctx, "cp -r dir1 dir3")
	runCommand(sh, ctx, "ls dir3")

	fmt.Println("\n=== Finding Files ===")
	runCommand(sh, ctx, "find /home -name '*.txt'")

	fmt.Println("\n=== Import/Export ===")
	// Create a test file
	os.MkdirAll("/tmp/go-memsh-test", 0755)
	os.WriteFile("/tmp/go-memsh-test/local-file.txt", []byte("This is a local file"), 0644)

	runCommand(sh, ctx, "import-file /tmp/go-memsh-test/local-file.txt /imported.txt")
	runCommand(sh, ctx, "cat /imported.txt")
	runCommand(sh, ctx, "echo 'Modified content' > /export-test.txt")
	runCommand(sh, ctx, "export-file /export-test.txt /tmp/go-memsh-test/exported.txt")

	// Verify export
	content, _ := os.ReadFile("/tmp/go-memsh-test/exported.txt")
	fmt.Printf("Exported file content (from local filesystem): %s\n", string(content))

	fmt.Println("\n=== Demo Complete ===")
}

func runCommand(sh *memsh.Shell, ctx context.Context, cmd string) {
	fmt.Printf("$ %s\n", cmd)
	err := sh.Run(ctx, cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

func runInteractive(sh *memsh.Shell, ctx context.Context) {
	fmt.Println("Welcome to MemSh - Shell running on afero.FS")
	fmt.Println("Type 'exit' or press Ctrl+D to exit")
	fmt.Println()

	err := sh.RunInteractive(ctx)
	if err != nil {
		log.Fatalf("Interactive mode failed: %v", err)
	}
}
