package memsh

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// Shell represents a shell interpreter running on afero.FS
type Shell struct {
	fs          afero.Fs
	runner      *interp.Runner
	env         *EnvironMap
	cwd         string
	prevDir     string // Previous directory for cd -
	stdin       io.Reader
	stdout      io.Writer
	stderr      io.Writer
	pipeManager *PipeManager // Manages virtual pipes for process substitution
	config      ShellConfig
}

// ShellConfig controls optional shell behaviors.
type ShellConfig struct {
	// MergeScriptEnv determines whether environment changes made inside sh scripts
	// are merged back into the parent shell. When false (default), scripts are
	// isolated and any mutations are discarded after execution.
	MergeScriptEnv bool
}

// NewShell creates a new shell interpreter with the given afero.FS
func NewShell(fs afero.Fs) (*Shell, error) {
	return NewShellWithConfig(fs, ShellConfig{})
}

// NewShellWithConfig creates a shell with optional configuration overrides.
func NewShellWithConfig(fs afero.Fs, cfg ShellConfig) (*Shell, error) {
	if fs == nil {
		fs = afero.NewMemMapFs()
	}

	shell := &Shell{
		fs:          fs,
		cwd:         "/",
		prevDir:     "/",
		stdin:       os.Stdin,
		stdout:      os.Stdout,
		stderr:      os.Stderr,
		env:         NewEnvironMap(os.Environ()),
		pipeManager: NewPipeManager(),
		config:      cfg,
	}

	// Create runner with our custom handlers
	runner, err := interp.New(
		interp.StdIO(shell.stdin, shell.stdout, shell.stderr),
		interp.Env(shell.env),
		interp.Dir(shell.cwd),
		interp.CallHandler(shell.callHandler),
		interp.ExecHandlers(shell.execHandler),
		interp.OpenHandler(shell.openHandler),
		interp.StatHandler(shell.statHandler),
		interp.ReadDirHandler(shell.readDirHandler),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}

	shell.runner = runner

	return shell, nil
}

// SetIO sets the stdin, stdout, and stderr for the shell
func (s *Shell) SetIO(stdin io.Reader, stdout, stderr io.Writer) {
	s.stdin = stdin
	s.stdout = stdout
	s.stderr = stderr
	s.runner.Reset()
	interp.StdIO(stdin, stdout, stderr)(s.runner)
}

// Run executes a shell script
func (s *Shell) Run(ctx context.Context, script string) error {
	// Use Bash variant to support process substitution and other Bash features
	parser := syntax.NewParser(syntax.Variant(syntax.LangBash))
	prog, err := parser.Parse(strings.NewReader(script), "")
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	// Process any process substitutions in the script
	if err := s.processProcessSubstitutions(ctx, prog); err != nil {
		return err
	}

	// Reset runner state and configure for execution
	// This is safe because Run() is only called at the top level, never for
	// individual commands within a pipeline (those use execHandler directly).
	// mvdan/sh will handle setting up pipes BETWEEN commands in a pipeline,
	// but the runner needs its top-level stdin/stdout/stderr configured.
	s.runner.Reset()
	interp.StdIO(s.stdin, s.stdout, s.stderr)(s.runner)
	interp.Dir(s.cwd)(s.runner)
	interp.Env(s.env)(s.runner)

	return s.runner.Run(ctx, prog)
}

// stdio returns the active stdio streams for the current execution context.
// If the context belongs to a pipeline stage, the pipeline-specific readers and
// writers are used; otherwise, the shell's default stdio is returned.
func (s *Shell) stdio(ctx context.Context) (io.Reader, io.Writer, io.Writer) {
	if hc := interp.HandlerCtx(ctx); hc != nil {
		in := hc.Stdin
		if in == nil {
			in = s.stdin
		}
		out := hc.Stdout
		if out == nil {
			out = s.stdout
		}
		errW := hc.Stderr
		if errW == nil {
			errW = s.stderr
		}
		return in, out, errW
	}
	return s.stdin, s.stdout, s.stderr
}

// processProcessSubstitutions walks the AST to find and set up process substitutions
func (s *Shell) processProcessSubstitutions(ctx context.Context, prog *syntax.File) error {
	// First pass: collect all ProcSubst nodes
	var procSubsts []*syntax.ProcSubst

	syntax.Walk(prog, func(node syntax.Node) bool {
		if procSubst, ok := node.(*syntax.ProcSubst); ok {
			procSubsts = append(procSubsts, procSubst)
		}
		return true
	})

	if len(procSubsts) == 0 {
		return nil
	}

	// Set up pipes for each ProcSubst
	procSubstMap := make(map[*syntax.ProcSubst]*VirtualPipe)
	for _, procSubst := range procSubsts {
		pipe, err := s.setupProcessSubstitution(ctx, procSubst)
		if err != nil {
			// Clean up any pipes we've created so far
			for _, p := range procSubstMap {
				s.pipeManager.ClosePipe(p.id)
			}
			return fmt.Errorf("process substitution setup failed: %w", err)
		}
		procSubstMap[procSubst] = pipe
	}

	// Second pass: replace ProcSubst nodes with Lit nodes in all words
	syntax.Walk(prog, func(node syntax.Node) bool {
		if word, ok := node.(*syntax.Word); ok {
			replaceProcSubstInWord(word, procSubstMap)
		}
		return true
	})

	return nil
}

// RunInteractive starts an interactive shell session
func (s *Shell) RunInteractive(ctx context.Context) error {
	// Use Bash variant to support process substitution and other Bash features
	parser := syntax.NewParser(syntax.Variant(syntax.LangBash))
	fmt.Fprintf(s.stdout, "memsh> ")

	err := parser.Interactive(s.stdin, func(stmts []*syntax.Stmt) bool {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		if parser.Incomplete() {
			fmt.Fprintf(s.stdout, "> ")
			return true
		}

		for _, stmt := range stmts {
			s.runner.Reset()
			interp.Dir(s.cwd)(s.runner)
			interp.Env(s.env)(s.runner)

			if err := s.runner.Run(ctx, stmt); err != nil {
				fmt.Fprintf(s.stderr, "%v\n", err)
			}

			// Update cwd from runner
			s.cwd = s.runner.Dir
		}

		fmt.Fprintf(s.stdout, "memsh> ")
		return true
	})

	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

// callHandler intercepts all command calls, including builtins
// It allows us to override built-in commands like cd and pwd
func (s *Shell) callHandler(ctx context.Context, args []string) ([]string, error) {
	if len(args) == 0 {
		return args, nil
	}

	// For certain builtins, we prepend a marker to prevent the builtin from running
	// The exec handler will recognize and handle these
	switch args[0] {
	case "cd", "pwd", "test", "[":
		// Prepend marker to the command name
		modifiedArgs := make([]string, len(args))
		copy(modifiedArgs, args)
		modifiedArgs[0] = "__memsh_" + args[0] + "__"
		return modifiedArgs, nil
	}

	// For all other commands, return args unchanged
	return args, nil
}

// execHandler handles command execution
func (s *Shell) execHandler(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(ctx context.Context, args []string) error {
		if len(args) == 0 {
			return nil
		}

		// Handle intercepted builtin commands (marked by callHandler)
		if strings.HasPrefix(args[0], "__memsh_") && strings.HasSuffix(args[0], "__") {
			// Extract the original command name
			originalCmd := strings.TrimPrefix(strings.TrimSuffix(args[0], "__"), "__memsh_")
			// Create new args with the original command name
			newArgs := make([]string, len(args))
			copy(newArgs, args)
			newArgs[0] = originalCmd
			args = newArgs
		}

		// Handle built-in commands
		switch args[0] {
		case "help":
			return s.cmdHelp(ctx, args)
		case "pwd":
			return s.cmdPwd(ctx, args)
		case "cd":
			return s.cmdCd(ctx, args)
		case "ls":
			return s.cmdLs(ctx, args)
		case "cat":
			return s.cmdCat(ctx, args)
		case "echo":
			return s.cmdEcho(ctx, args)
		case "mkdir":
			return s.cmdMkdir(ctx, args)
		case "rm":
			return s.cmdRm(ctx, args)
		case "touch":
			return s.cmdTouch(ctx, args)
		case "cp":
			return s.cmdCp(ctx, args)
		case "mv":
			return s.cmdMv(ctx, args)
		case "sleep":
			return s.cmdSleep(ctx, args)
		case "true":
			return nil
		case "false":
			return interp.NewExitStatus(1)
		case "test", "[":
			return s.cmdTest(ctx, args)
		case "env":
			return s.cmdEnv(ctx, args)
		case "set":
			return s.cmdSet(ctx, args)
		case "unset":
			return s.cmdUnset(ctx, args)
		case "export":
			return s.cmdExport(ctx, args)
		case "exit":
			return s.cmdExit(ctx, args)
		case "sh":
			return s.cmdSh(ctx, args)
		case "grep":
			return s.cmdGrep(ctx, args)
		case "head":
			return s.cmdHead(ctx, args)
		case "tail":
			return s.cmdTail(ctx, args)
		case "wc":
			return s.cmdWc(ctx, args)
		case "sort":
			return s.cmdSort(ctx, args)
		case "uniq":
			return s.cmdUniq(ctx, args)
		case "find":
			return s.cmdFind(ctx, args)
		case "import-file":
			return s.cmdImportFile(ctx, args)
		case "import-dir":
			return s.cmdImportDir(ctx, args)
		case "export-file":
			return s.cmdExportFile(ctx, args)
		case "export-dir":
			return s.cmdExportDir(ctx, args)
		case "jq":
			return s.cmdJq(ctx, args)
		case "curl":
			return s.cmdCurl(ctx, args)
		default:
			return fmt.Errorf("%s: command not found", args[0])
		}
	}
}

// openHandler handles file opening
func (s *Shell) openHandler(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
	// Check if this is a virtual /dev/fd/N path for process substitution
	if strings.HasPrefix(path, "/dev/fd/") {
		// Extract the file descriptor number
		var fdNum int
		_, err := fmt.Sscanf(path, "/dev/fd/%d", &fdNum)
		if err == nil {
			// Get the virtual pipe
			if pipe, ok := s.pipeManager.GetPipe(fdNum); ok {
				// Return a virtual file that reads from the pipe
				return &virtualFileReadWriteCloser{NewVirtualFile(pipe)}, nil
			}
		}
		return nil, fmt.Errorf("no such file or directory: %s", path)
	}

	path = s.resolvePath(path)
	file, err := s.fs.OpenFile(path, flag, perm)
	if err != nil {
		return nil, err
	}
	// afero.File should implement io.ReadWriteCloser
	return file.(io.ReadWriteCloser), nil
}

// virtualFileReadWriteCloser wraps VirtualFile to implement io.ReadWriteCloser
type virtualFileReadWriteCloser struct {
	*VirtualFile
}

func (v *virtualFileReadWriteCloser) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("virtual pipe is read-only")
}

// statHandler handles file stat operations
func (s *Shell) statHandler(ctx context.Context, name string, followSymlinks bool) (os.FileInfo, error) {
	// Check if this is a virtual /dev/fd/N path
	if strings.HasPrefix(name, "/dev/fd/") {
		var fdNum int
		_, err := fmt.Sscanf(name, "/dev/fd/%d", &fdNum)
		if err == nil {
			if pipe, ok := s.pipeManager.GetPipe(fdNum); ok {
				vf := NewVirtualFile(pipe)
				return vf.Stat()
			}
		}
		return nil, fmt.Errorf("no such file or directory: %s", name)
	}

	name = s.resolvePath(name)
	if followSymlinks {
		return s.fs.Stat(name)
	}
	if lfs, ok := s.fs.(afero.Lstater); ok {
		fi, _, err := lfs.LstatIfPossible(name)
		return fi, err
	}
	return s.fs.Stat(name)
}

// readDirHandler handles directory reading
func (s *Shell) readDirHandler(ctx context.Context, path string) ([]os.FileInfo, error) {
	path = s.resolvePath(path)
	entries, err := afero.ReadDir(s.fs, path)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// resolvePath resolves a path relative to the current working directory
func (s *Shell) resolvePath(path string) string {
	// Don't resolve virtual /dev/fd paths
	if strings.HasPrefix(path, "/dev/fd/") {
		return path
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(s.cwd, path))
}

// openFile opens a file, handling both regular files and virtual /dev/fd paths
func (s *Shell) openFile(path string) (io.ReadCloser, error) {
	// Check if this is a virtual /dev/fd/N path for process substitution
	if strings.HasPrefix(path, "/dev/fd/") {
		var fdNum int
		_, err := fmt.Sscanf(path, "/dev/fd/%d", &fdNum)
		if err == nil {
			if pipe, ok := s.pipeManager.GetPipe(fdNum); ok {
				return NewVirtualFile(pipe), nil
			}
		}
		return nil, fmt.Errorf("no such file or directory: %s", path)
	}

	// Regular file
	return s.fs.Open(path)
}

// statFile stats a file, handling both regular files and virtual /dev/fd paths
func (s *Shell) statFile(path string) (os.FileInfo, error) {
	// Check if this is a virtual /dev/fd/N path
	if strings.HasPrefix(path, "/dev/fd/") {
		var fdNum int
		_, err := fmt.Sscanf(path, "/dev/fd/%d", &fdNum)
		if err == nil {
			if pipe, ok := s.pipeManager.GetPipe(fdNum); ok {
				vf := NewVirtualFile(pipe)
				return vf.Stat()
			}
		}
		return nil, fmt.Errorf("no such file or directory: %s", path)
	}

	// Regular file
	return s.fs.Stat(path)
}

// GetCwd returns the current working directory
func (s *Shell) GetCwd() string {
	return s.cwd
}

// SetCwd sets the current working directory
func (s *Shell) SetCwd(dir string) error {
	dir = s.resolvePath(dir)
	info, err := s.fs.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s: not a directory", dir)
	}
	s.cwd = dir
	s.runner.Reset()
	interp.Dir(s.cwd)(s.runner)
	return nil
}

// cmdSleep implements the sleep command
func (s *Shell) cmdSleep(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("sleep: missing operand")
	}

	duration, err := time.ParseDuration(args[1] + "s")
	if err != nil {
		return fmt.Errorf("sleep: invalid time interval '%s'", args[1])
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return nil
	}
}

// cmdExit implements the exit command
func (s *Shell) cmdExit(ctx context.Context, args []string) error {
	code := 0
	if len(args) > 1 {
		fmt.Sscanf(args[1], "%d", &code)
	}
	return interp.NewExitStatus(uint8(code))
}

// cmdSh implements the sh command to execute shell scripts
func (s *Shell) cmdSh(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("sh: missing script file argument")
	}

	scriptPath := args[1]
	scriptArgs := args[2:] // Positional parameters $1, $2, $3, ... (excluding $0)

	// Resolve and read the script file
	scriptPath = s.resolvePath(scriptPath)
	file, err := s.openFile(scriptPath)
	if err != nil {
		return fmt.Errorf("sh: cannot open %s: %v", args[1], err)
	}
	defer file.Close()

	// Read the entire script
	scriptContent, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("sh: error reading %s: %v", args[1], err)
	}

	// Parse the script using Bash variant
	parser := syntax.NewParser(syntax.Variant(syntax.LangBash))
	prog, err := parser.Parse(strings.NewReader(string(scriptContent)), scriptPath)
	if err != nil {
		return fmt.Errorf("sh: parse error in %s: %v", args[1], err)
	}

	// Save current shell state
	oldEnv := s.env
	oldRunner := s.runner
	oldCwd := s.cwd
	oldPrevDir := s.prevDir

	// Create a copy of the environment for the script
	scriptEnv := s.env.Copy()

	// Create a new runner for the script with its own environment and parameters
	runner, err := interp.New(
		interp.StdIO(s.stdin, s.stdout, s.stderr),
		interp.Env(scriptEnv),
		interp.Dir(s.cwd),
		interp.Params(scriptArgs...),
		interp.CallHandler(s.callHandler),
		interp.ExecHandlers(s.execHandler),
		interp.OpenHandler(s.openHandler),
		interp.StatHandler(s.statHandler),
		interp.ReadDirHandler(s.readDirHandler),
	)
	if err != nil {
		return fmt.Errorf("sh: failed to create runner: %v", err)
	}

	// Temporarily replace shell's environment and runner so builtin commands use them
	s.env = scriptEnv
	s.runner = runner

	// Execute the script
	err = runner.Run(ctx, prog)

	// Merge or discard environment changes based on configuration
	if s.config.MergeScriptEnv {
		oldEnv.ReplaceWith(scriptEnv)
		s.cwd = runner.Dir
	} else {
		s.cwd = oldCwd
		s.prevDir = oldPrevDir
	}

	// Restore original shell state
	s.env = oldEnv
	s.runner = oldRunner
	s.runner.Reset()
	interp.Dir(s.cwd)(s.runner)
	interp.Env(s.env)(s.runner)

	if err != nil {
		// Check if it's an exit status
		if exitErr, ok := err.(interp.ExitStatus); ok {
			return exitErr
		}
		return fmt.Errorf("sh: %v", err)
	}

	return nil
}
