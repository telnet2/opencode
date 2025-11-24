package memsh

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/afero"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

// cmdPwd implements the pwd command
func (s *Shell) cmdPwd(ctx context.Context, args []string) error {
	_, stdout, _ := s.stdio(ctx)
	fmt.Fprintln(stdout, s.cwd)
	return nil
}

// cmdCd implements the cd command
func (s *Shell) cmdCd(ctx context.Context, args []string) error {
	_, stdout, _ := s.stdio(ctx)
	var dir string
	if len(args) < 2 {
		// POSIX: cd with no arguments goes to $HOME
		home := s.env.Get("HOME").Str
		if home == "" {
			dir = "/"
		} else {
			dir = home
		}
	} else if args[1] == "-" {
		// POSIX: cd - goes to previous directory and prints it
		if s.prevDir == "" {
			return fmt.Errorf("cd: OLDPWD not set")
		}
		dir = s.prevDir
		fmt.Fprintln(stdout, dir)
	} else {
		dir = args[1]
	}

	// Save current directory before changing
	oldDir := s.cwd

	err := s.SetCwd(dir)
	if err != nil {
		return err
	}

	// Update previous directory and OLDPWD
	s.prevDir = oldDir
	s.env.Set("OLDPWD", expand.Variable{
		Exported: true,
		Kind:     expand.String,
		Str:      oldDir,
	})

	return nil
}

// cmdLs implements the ls command
func (s *Shell) cmdLs(ctx context.Context, args []string) error {
	_, stdout, stderr := s.stdio(ctx)
	paths := args[1:]
	if len(paths) == 0 {
		paths = []string{"."}
	}

	showAll := false
	showLong := false
	recursive := false
	actualPaths := []string{}

	for _, arg := range paths {
		if strings.HasPrefix(arg, "-") {
			if strings.Contains(arg, "a") {
				showAll = true
			}
			if strings.Contains(arg, "l") {
				showLong = true
			}
			if strings.Contains(arg, "R") {
				recursive = true
			}
		} else {
			actualPaths = append(actualPaths, arg)
		}
	}

	if len(actualPaths) == 0 {
		actualPaths = []string{"."}
	}

	for i, path := range actualPaths {
		if i > 0 {
			fmt.Fprintln(stdout)
		}
		err := s.lsPath(ctx, path, showAll, showLong, recursive, len(actualPaths) > 1 || recursive, "")
		if err != nil {
			fmt.Fprintf(stderr, "ls: %v\n", err)
		}
	}

	return nil
}

// lsPath lists a single path, potentially recursively
func (s *Shell) lsPath(ctx context.Context, path string, showAll, showLong, recursive, showHeader bool, prefix string) error {
	_, stdout, stderr := s.stdio(ctx)
	path = s.resolvePath(path)
	info, err := s.fs.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot access '%s': %v", path, err)
	}

	if !info.IsDir() {
		// Single file
		if showLong {
			mode := info.Mode().String()
			size := info.Size()
			modTime := info.ModTime().Format("Jan 02 15:04")
			fmt.Fprintf(stdout, "%s %8d %s %s\n", mode, size, modTime, info.Name())
		} else {
			fmt.Fprintln(stdout, info.Name())
		}
		return nil
	}

	// Directory
	entries, err := afero.ReadDir(s.fs, path)
	if err != nil {
		return fmt.Errorf("cannot read directory '%s': %v", path, err)
	}

	if showHeader {
		fmt.Fprintf(stdout, "%s:\n", path)
	}

	// Sort entries by name
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	// Collect subdirectories for recursive listing
	var subdirs []string

	for _, entry := range entries {
		if !showAll && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		if showLong {
			mode := entry.Mode().String()
			size := entry.Size()
			modTime := entry.ModTime().Format("Jan 02 15:04")
			fmt.Fprintf(stdout, "%s %8d %s %s\n", mode, size, modTime, entry.Name())
		} else {
			fmt.Fprintln(stdout, entry.Name())
		}

		// Track subdirectories for recursive listing
		if recursive && entry.IsDir() {
			subdirs = append(subdirs, filepath.Join(path, entry.Name()))
		}
	}

	// Recursively list subdirectories
	if recursive {
		for _, subdir := range subdirs {
			fmt.Fprintln(stdout)
			err := s.lsPath(ctx, subdir, showAll, showLong, recursive, true, prefix)
			if err != nil {
				fmt.Fprintf(stderr, "ls: %v\n", err)
			}
		}
	}

	return nil
}

// cmdCat implements the cat command
func (s *Shell) cmdCat(ctx context.Context, args []string) error {
	stdin, stdout, stderr := s.stdio(ctx)
	if len(args) < 2 {
		// Read from stdin
		_, err := io.Copy(stdout, stdin)
		return err
	}

	for _, path := range args[1:] {
		path = s.resolvePath(path)

		// Check if path is a directory using our helper that handles virtual pipes
		info, err := s.statFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "cat: %s: %v\n", path, err)
			return err
		}
		if info.IsDir() {
			err := fmt.Errorf("Is a directory")
			fmt.Fprintf(stderr, "cat: %s: %v\n", path, err)
			return err
		}

		// Use our helper that handles virtual pipes
		file, err := s.openFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "cat: %s: %v\n", path, err)
			return err
		}
		defer file.Close()

		_, err = io.Copy(stdout, file)
		if err != nil {
			fmt.Fprintf(stderr, "cat: %s: %v\n", path, err)
			return err
		}
	}

	return nil
}

// cmdEcho implements the echo command
func (s *Shell) cmdEcho(ctx context.Context, args []string) error {
	_, stdout, _ := s.stdio(ctx)
	// POSIX: -n flag suppresses trailing newline
	// -e flag enables interpretation of backslash escapes
	noNewline := false
	interpretEscapes := false
	startIndex := 1

	// Parse flags
	for startIndex < len(args) && strings.HasPrefix(args[startIndex], "-") {
		flag := args[startIndex]
		if flag == "-n" {
			noNewline = true
			startIndex++
		} else if flag == "-e" {
			interpretEscapes = true
			startIndex++
		} else if flag == "-en" || flag == "-ne" {
			noNewline = true
			interpretEscapes = true
			startIndex++
		} else {
			break
		}
	}

	output := strings.Join(args[startIndex:], " ")

	// Interpret escape sequences if -e flag is present
	if interpretEscapes {
		output = strings.ReplaceAll(output, "\\n", "\n")
		output = strings.ReplaceAll(output, "\\t", "\t")
		output = strings.ReplaceAll(output, "\\r", "\r")
		output = strings.ReplaceAll(output, "\\\\", "\\")
	}

	if noNewline {
		fmt.Fprint(stdout, output)
	} else {
		fmt.Fprintln(stdout, output)
	}

	return nil
}

// cmdMkdir implements the mkdir command
func (s *Shell) cmdMkdir(ctx context.Context, args []string) error {
	_, _, stderr := s.stdio(ctx)
	if len(args) < 2 {
		return fmt.Errorf("mkdir: missing operand")
	}

	createParents := false
	paths := []string{}

	for _, arg := range args[1:] {
		if arg == "-p" {
			createParents = true
		} else {
			paths = append(paths, arg)
		}
	}

	for _, path := range paths {
		path = s.resolvePath(path)
		var err error
		if createParents {
			err = s.fs.MkdirAll(path, 0755)
		} else {
			err = s.fs.Mkdir(path, 0755)
		}
		if err != nil {
			fmt.Fprintf(stderr, "mkdir: cannot create directory '%s': %v\n", path, err)
			return err
		}
	}

	return nil
}

// cmdRm implements the rm command
func (s *Shell) cmdRm(ctx context.Context, args []string) error {
	stdin, stdout, stderr := s.stdio(ctx)
	if len(args) < 2 {
		return fmt.Errorf("rm: missing operand")
	}

	recursive := false
	force := false
	interactive := false
	paths := []string{}

	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "-") {
			if strings.Contains(arg, "r") || strings.Contains(arg, "R") {
				recursive = true
			}
			if strings.Contains(arg, "f") {
				force = true
			}
			if strings.Contains(arg, "i") {
				interactive = true
			}
		} else {
			paths = append(paths, arg)
		}
	}

	for _, path := range paths {
		path = s.resolvePath(path)
		info, err := s.fs.Stat(path)
		if err != nil {
			if force {
				continue
			}
			fmt.Fprintf(stderr, "rm: cannot remove '%s': %v\n", path, err)
			return err
		}

		if info.IsDir() && !recursive {
			err := fmt.Errorf("is a directory")
			if !force {
				fmt.Fprintf(stderr, "rm: cannot remove '%s': %v\n", path, err)
			}
			return err
		}

		// Interactive confirmation
		if interactive && !force {
			var fileType string
			if info.IsDir() {
				fileType = "directory"
			} else {
				fileType = "file"
			}
			fmt.Fprintf(stdout, "rm: remove %s '%s'? ", fileType, path)

			// Read response from stdin
			scanner := bufio.NewScanner(stdin)
			if !scanner.Scan() {
				continue
			}
			response := strings.TrimSpace(scanner.Text())
			if response != "y" && response != "Y" && response != "yes" {
				continue
			}
		}

		err = s.fs.RemoveAll(path)
		if err != nil && !force {
			fmt.Fprintf(stderr, "rm: cannot remove '%s': %v\n", path, err)
			return err
		}
	}

	return nil
}

// cmdTouch implements the touch command
func (s *Shell) cmdTouch(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("touch: missing file operand")
	}

	for _, path := range args[1:] {
		path = s.resolvePath(path)

		// Check if file exists
		exists, err := afero.Exists(s.fs, path)
		if err != nil {
			return fmt.Errorf("touch: %s: %v", path, err)
		}

		if exists {
			// Update modification time
			now := time.Now()
			err = s.fs.Chtimes(path, now, now)
			if err != nil {
				return fmt.Errorf("touch: %s: %v", path, err)
			}
		} else {
			// Create file
			file, err := s.fs.Create(path)
			if err != nil {
				return fmt.Errorf("touch: %s: %v", path, err)
			}
			file.Close()
		}
	}

	return nil
}

// cmdCp implements the cp command
func (s *Shell) cmdCp(ctx context.Context, args []string) error {
	_, _, stderr := s.stdio(ctx)
	if len(args) < 3 {
		return fmt.Errorf("cp: missing file operand")
	}

	recursive := false
	preserve := false
	sources := []string{}
	var dest string

	for i, arg := range args[1:] {
		if strings.HasPrefix(arg, "-") {
			if strings.Contains(arg, "r") || strings.Contains(arg, "R") {
				recursive = true
			}
			if strings.Contains(arg, "p") {
				preserve = true
			}
		} else if i == len(args)-2 {
			dest = arg
		} else {
			sources = append(sources, arg)
		}
	}

	dest = s.resolvePath(dest)

	for _, src := range sources {
		src = s.resolvePath(src)
		if err := s.copyFileOrDir(src, dest, recursive, preserve); err != nil {
			fmt.Fprintf(stderr, "cp: %v\n", err)
			return err
		}
	}

	return nil
}

// copyFileOrDir copies a file or directory
func (s *Shell) copyFileOrDir(src, dest string, recursive, preserve bool) error {
	srcInfo, err := s.fs.Stat(src)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		if !recursive {
			return fmt.Errorf("cannot copy directory '%s' without -r", src)
		}
		return s.copyDir(src, dest, preserve)
	}

	return s.copyFile(src, dest, preserve)
}

// copyFile copies a single file
func (s *Shell) copyFile(src, dest string, preserve bool) error {
	// Get source file info
	srcInfo, err := s.fs.Stat(src)
	if err != nil {
		return err
	}

	// Check if dest is a directory
	destInfo, err := s.fs.Stat(dest)
	if err == nil && destInfo.IsDir() {
		dest = filepath.Join(dest, filepath.Base(src))
	}

	srcFile, err := s.fs.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Use appropriate permissions based on preserve flag
	perm := os.FileMode(0644)
	if preserve {
		perm = srcInfo.Mode()
	}

	destFile, err := s.fs.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	_, err = io.Copy(destFile, srcFile)
	destFile.Close() // Close explicitly before Chtimes
	if err != nil {
		return err
	}

	// Preserve timestamps if requested
	if preserve {
		// Use access time = mod time for simplicity
		if err := s.fs.Chtimes(dest, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
			return err
		}
	}

	return nil
}

// copyDir copies a directory recursively
func (s *Shell) copyDir(src, dest string, preserve bool) error {
	srcInfo, err := s.fs.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	err = s.fs.MkdirAll(dest, srcInfo.Mode())
	if err != nil {
		return err
	}

	// Preserve directory attributes if requested
	if preserve {
		if err := s.fs.Chtimes(dest, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
			return err
		}
	}

	entries, err := afero.ReadDir(s.fs, src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		if entry.IsDir() {
			err = s.copyDir(srcPath, destPath, preserve)
		} else {
			err = s.copyFile(srcPath, destPath, preserve)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// cmdMv implements the mv command
func (s *Shell) cmdMv(ctx context.Context, args []string) error {
	_, _, stderr := s.stdio(ctx)
	if len(args) < 3 {
		return fmt.Errorf("mv: missing file operand")
	}

	src := s.resolvePath(args[1])
	dest := s.resolvePath(args[2])

	// Check if dest is a directory
	destInfo, err := s.fs.Stat(dest)
	if err == nil && destInfo.IsDir() {
		dest = filepath.Join(dest, filepath.Base(src))
	}

	err = s.fs.Rename(src, dest)
	if err != nil {
		fmt.Fprintf(stderr, "mv: %v\n", err)
		return err
	}

	return nil
}

// cmdTest implements the test command
func (s *Shell) cmdTest(ctx context.Context, args []string) error {
	// Remove trailing "]" for [ command
	if args[0] == "[" {
		if len(args) < 2 || args[len(args)-1] != "]" {
			return fmt.Errorf("[: missing ']'")
		}
		args = args[:len(args)-1]
	}

	if len(args) < 2 {
		return interp.NewExitStatus(1)
	}

	// Handle unary operators
	if len(args) == 3 {
		op := args[1]
		path := s.resolvePath(args[2])

		info, err := s.fs.Stat(path)
		exists := err == nil

		switch op {
		case "-e", "-a":
			// File exists
			if exists {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-f":
			// Regular file
			if exists && !info.IsDir() {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-d":
			// Directory
			if exists && info.IsDir() {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-h", "-L":
			// Symbolic link (afero may not fully support, check mode)
			if exists && (info.Mode()&os.ModeSymlink) != 0 {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-r":
			// Readable (simplified - just check if exists)
			if exists {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-w":
			// Writable (simplified - just check if exists)
			if exists {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-x":
			// Executable
			if exists && (info.Mode()&0111) != 0 {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-s":
			// Non-empty file
			if exists && info.Size() > 0 {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-b":
			// Block special file
			if exists && (info.Mode()&os.ModeDevice) != 0 && (info.Mode()&os.ModeCharDevice) == 0 {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-c":
			// Character special file
			if exists && (info.Mode()&os.ModeCharDevice) != 0 {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-p":
			// Named pipe (FIFO)
			if exists && (info.Mode()&os.ModeNamedPipe) != 0 {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-S":
			// Socket
			if exists && (info.Mode()&os.ModeSocket) != 0 {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-z":
			// String is empty
			if args[2] == "" {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-n":
			// String is not empty
			if args[2] != "" {
				return nil
			}
			return interp.NewExitStatus(1)
		}
	}

	// Handle binary operators
	if len(args) == 4 {
		left := args[1]
		op := args[2]
		right := args[3]

		switch op {
		case "=", "==":
			if left == right {
				return nil
			}
			return interp.NewExitStatus(1)
		case "!=":
			if left != right {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-eq":
			l, err1 := strconv.Atoi(left)
			r, err2 := strconv.Atoi(right)
			if err1 == nil && err2 == nil && l == r {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-ne":
			l, err1 := strconv.Atoi(left)
			r, err2 := strconv.Atoi(right)
			if err1 == nil && err2 == nil && l != r {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-lt":
			l, err1 := strconv.Atoi(left)
			r, err2 := strconv.Atoi(right)
			if err1 == nil && err2 == nil && l < r {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-le":
			l, err1 := strconv.Atoi(left)
			r, err2 := strconv.Atoi(right)
			if err1 == nil && err2 == nil && l <= r {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-gt":
			l, err1 := strconv.Atoi(left)
			r, err2 := strconv.Atoi(right)
			if err1 == nil && err2 == nil && l > r {
				return nil
			}
			return interp.NewExitStatus(1)
		case "-ge":
			l, err1 := strconv.Atoi(left)
			r, err2 := strconv.Atoi(right)
			if err1 == nil && err2 == nil && l >= r {
				return nil
			}
			return interp.NewExitStatus(1)
		}
	}

	return interp.NewExitStatus(1)
}
