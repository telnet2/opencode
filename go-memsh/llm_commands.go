package memsh

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/afero"
)

// StatResult represents the JSON output of the stat command
type StatResult struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime string `json:"mtime"`
	IsDir   bool   `json:"is_dir"`
	Perm    string `json:"perm"`
}

// cmdStat implements the stat command - returns file metadata as JSON
func (s *Shell) cmdStat(ctx context.Context, args []string) error {
	_, stdout, stderr := s.stdio(ctx)

	if len(args) < 2 {
		return fmt.Errorf("stat: missing file operand")
	}

	// Parse options
	jsonOutput := true // Default to JSON for LLM compatibility
	i := 1
	for i < len(args) && strings.HasPrefix(args[i], "-") {
		switch args[i] {
		case "--json", "-j":
			jsonOutput = true
		case "--human", "-h":
			jsonOutput = false
		default:
			// Unknown option, treat as filename
			break
		}
		i++
	}

	paths := args[i:]
	if len(paths) == 0 {
		return fmt.Errorf("stat: missing file operand")
	}

	results := make([]StatResult, 0, len(paths))
	var lastErr error

	for _, path := range paths {
		resolvedPath := s.resolvePath(path)
		info, err := s.fs.Stat(resolvedPath)
		if err != nil {
			fmt.Fprintf(stderr, "stat: cannot stat '%s': %v\n", path, err)
			lastErr = err
			continue
		}

		result := StatResult{
			Name:    info.Name(),
			Path:    resolvedPath,
			Size:    info.Size(),
			Mode:    info.Mode().String(),
			ModTime: info.ModTime().Format(time.RFC3339),
			IsDir:   info.IsDir(),
			Perm:    fmt.Sprintf("%04o", info.Mode().Perm()),
		}

		if jsonOutput {
			results = append(results, result)
		} else {
			// Human-readable format
			fileType := "regular file"
			if info.IsDir() {
				fileType = "directory"
			}
			fmt.Fprintf(stdout, "  File: %s\n", resolvedPath)
			fmt.Fprintf(stdout, "  Size: %d\t\tBlocks: -\t\tIO Block: -\t%s\n", info.Size(), fileType)
			fmt.Fprintf(stdout, "Access: (%s/%s)\n", result.Perm, info.Mode().String())
			fmt.Fprintf(stdout, "Modify: %s\n", info.ModTime().Format("2006-01-02 15:04:05.000000000 -0700"))
		}
	}

	if jsonOutput && len(results) > 0 {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if len(results) == 1 {
			encoder.Encode(results[0])
		} else {
			encoder.Encode(results)
		}
	}

	return lastErr
}

// cmdReadfile implements the readfile command - returns raw file content
// This is optimized for LLM tools that need exact file content without formatting
func (s *Shell) cmdReadfile(ctx context.Context, args []string) error {
	_, stdout, stderr := s.stdio(ctx)

	if len(args) < 2 {
		return fmt.Errorf("readfile: missing file operand")
	}

	// Parse options
	offset := 0
	limit := -1 // -1 means no limit
	i := 1

	for i < len(args) {
		switch args[i] {
		case "--offset", "-o":
			if i+1 >= len(args) {
				return fmt.Errorf("readfile: --offset requires an argument")
			}
			var err error
			offset, err = strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("readfile: invalid offset: %s", args[i+1])
			}
			i += 2
		case "--limit", "-l":
			if i+1 >= len(args) {
				return fmt.Errorf("readfile: --limit requires an argument")
			}
			var err error
			limit, err = strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("readfile: invalid limit: %s", args[i+1])
			}
			i += 2
		default:
			if strings.HasPrefix(args[i], "-") {
				return fmt.Errorf("readfile: unknown option: %s", args[i])
			}
			// First non-option argument is the file path
			goto readFile
		}
	}

readFile:
	if i >= len(args) {
		return fmt.Errorf("readfile: missing file operand")
	}

	path := s.resolvePath(args[i])

	// Check if path is a directory
	info, err := s.fs.Stat(path)
	if err != nil {
		fmt.Fprintf(stderr, "readfile: %s: %v\n", args[i], err)
		return err
	}
	if info.IsDir() {
		err := fmt.Errorf("Is a directory")
		fmt.Fprintf(stderr, "readfile: %s: %v\n", args[i], err)
		return err
	}

	// Read file content
	content, err := afero.ReadFile(s.fs, path)
	if err != nil {
		fmt.Fprintf(stderr, "readfile: %s: %v\n", args[i], err)
		return err
	}

	// Apply offset and limit
	if offset > 0 {
		if offset >= len(content) {
			return nil // Nothing to output
		}
		content = content[offset:]
	}

	if limit > 0 && limit < len(content) {
		content = content[:limit]
	}

	// Write raw content to stdout
	stdout.Write(content)

	return nil
}

// cmdWritefile implements the writefile command - writes stdin to a file
// This is optimized for LLM tools that need to write exact content
func (s *Shell) cmdWritefile(ctx context.Context, args []string) error {
	stdin, _, stderr := s.stdio(ctx)

	if len(args) < 2 {
		return fmt.Errorf("writefile: missing file operand")
	}

	// Parse options
	appendMode := false
	createDirs := false
	i := 1

	for i < len(args) {
		switch args[i] {
		case "--append", "-a":
			appendMode = true
			i++
		case "--parents", "-p":
			createDirs = true
			i++
		default:
			if strings.HasPrefix(args[i], "-") && args[i] != "-" {
				return fmt.Errorf("writefile: unknown option: %s", args[i])
			}
			goto writeFile
		}
	}

writeFile:
	if i >= len(args) {
		return fmt.Errorf("writefile: missing file operand")
	}

	path := s.resolvePath(args[i])

	// Create parent directories if requested
	if createDirs {
		dir := filepath.Dir(path)
		if err := s.fs.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(stderr, "writefile: cannot create directory '%s': %v\n", dir, err)
			return err
		}
	}

	// Read content from stdin
	content, err := io.ReadAll(stdin)
	if err != nil {
		fmt.Fprintf(stderr, "writefile: error reading input: %v\n", err)
		return err
	}

	// Write to file
	var flag int
	if appendMode {
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	} else {
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}

	file, err := s.fs.OpenFile(path, flag, 0644)
	if err != nil {
		fmt.Fprintf(stderr, "writefile: cannot open '%s': %v\n", args[i], err)
		return err
	}
	defer file.Close()

	_, err = file.Write(content)
	if err != nil {
		fmt.Fprintf(stderr, "writefile: error writing to '%s': %v\n", args[i], err)
		return err
	}

	return nil
}

// cmdFindEx implements enhanced find command with additional filters
func (s *Shell) cmdFindEx(ctx context.Context, args []string) error {
	stdin, stdout, stderr := s.stdio(ctx)
	origIn, origOut, origErr := s.stdin, s.stdout, s.stderr
	s.stdin, s.stdout, s.stderr = stdin, stdout, stderr
	defer func() {
		s.stdin, s.stdout, s.stderr = origIn, origOut, origErr
	}()

	// Default options
	opts := &findOptions{
		paths:     []string{"."},
		maxDepth:  -1, // unlimited
		minDepth:  0,
		printNull: false,
	}

	// Parse arguments
	i := 1
	for i < len(args) {
		arg := args[i]

		switch arg {
		case "-name":
			if i+1 >= len(args) {
				return fmt.Errorf("find: -name requires an argument")
			}
			opts.namePattern = args[i+1]
			i += 2
		case "-iname":
			if i+1 >= len(args) {
				return fmt.Errorf("find: -iname requires an argument")
			}
			opts.namePattern = args[i+1]
			opts.nameIgnoreCase = true
			i += 2
		case "-type":
			if i+1 >= len(args) {
				return fmt.Errorf("find: -type requires an argument")
			}
			opts.fileType = args[i+1]
			i += 2
		case "-maxdepth":
			if i+1 >= len(args) {
				return fmt.Errorf("find: -maxdepth requires an argument")
			}
			d, err := strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("find: invalid maxdepth: %s", args[i+1])
			}
			opts.maxDepth = d
			i += 2
		case "-mindepth":
			if i+1 >= len(args) {
				return fmt.Errorf("find: -mindepth requires an argument")
			}
			d, err := strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("find: invalid mindepth: %s", args[i+1])
			}
			opts.minDepth = d
			i += 2
		case "-mtime":
			if i+1 >= len(args) {
				return fmt.Errorf("find: -mtime requires an argument")
			}
			opts.mtimeStr = args[i+1]
			i += 2
		case "-size":
			if i+1 >= len(args) {
				return fmt.Errorf("find: -size requires an argument")
			}
			opts.sizeStr = args[i+1]
			i += 2
		case "-empty":
			opts.empty = true
			i++
		case "-print0":
			opts.printNull = true
			i++
		case "-path":
			if i+1 >= len(args) {
				return fmt.Errorf("find: -path requires an argument")
			}
			opts.pathPattern = args[i+1]
			i += 2
		default:
			if !strings.HasPrefix(arg, "-") {
				opts.paths = []string{arg}
				i++
			} else {
				// Unknown option
				i++
			}
		}
	}

	// Compile name pattern if provided
	if opts.namePattern != "" {
		pattern := globToRegex(opts.namePattern)
		if opts.nameIgnoreCase {
			pattern = "(?i)" + pattern
		}
		var err error
		opts.nameRe, err = regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("find: invalid pattern: %v", err)
		}
	}

	// Compile path pattern if provided
	if opts.pathPattern != "" {
		pattern := globToRegex(opts.pathPattern)
		var err error
		opts.pathRe, err = regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("find: invalid path pattern: %v", err)
		}
	}

	// Search each path
	for _, path := range opts.paths {
		path = s.resolvePath(path)
		err := s.findWalkEx(path, opts, 0)
		if err != nil {
			return err
		}
	}

	return nil
}

type findOptions struct {
	paths          []string
	namePattern    string
	nameIgnoreCase bool
	nameRe         *regexp.Regexp
	pathPattern    string
	pathRe         *regexp.Regexp
	fileType       string
	maxDepth       int
	minDepth       int
	mtimeStr       string
	sizeStr        string
	empty          bool
	printNull      bool
}

func (s *Shell) findWalkEx(path string, opts *findOptions, depth int) error {
	// Check depth limits
	if opts.maxDepth >= 0 && depth > opts.maxDepth {
		return nil
	}

	info, err := s.fs.Stat(path)
	if err != nil {
		return err
	}

	// Check if this entry matches the criteria
	matches := true

	// Check depth
	if depth < opts.minDepth {
		matches = false
	}

	// Check name pattern
	if matches && opts.nameRe != nil {
		matches = opts.nameRe.MatchString(info.Name())
	}

	// Check path pattern
	if matches && opts.pathRe != nil {
		matches = opts.pathRe.MatchString(path)
	}

	// Check file type
	if matches && opts.fileType != "" {
		switch opts.fileType {
		case "f":
			matches = !info.IsDir()
		case "d":
			matches = info.IsDir()
		case "l":
			matches = (info.Mode() & os.ModeSymlink) != 0
		}
	}

	// Check mtime
	if matches && opts.mtimeStr != "" {
		matches = s.checkMtime(info.ModTime(), opts.mtimeStr)
	}

	// Check size
	if matches && opts.sizeStr != "" {
		matches = s.checkSize(info.Size(), opts.sizeStr)
	}

	// Check empty
	if matches && opts.empty {
		if info.IsDir() {
			entries, err := afero.ReadDir(s.fs, path)
			matches = err == nil && len(entries) == 0
		} else {
			matches = info.Size() == 0
		}
	}

	if matches {
		if opts.printNull {
			fmt.Fprintf(s.stdout, "%s\x00", path)
		} else {
			fmt.Fprintln(s.stdout, path)
		}
	}

	// Recurse into directories
	if info.IsDir() {
		entries, err := afero.ReadDir(s.fs, path)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			entryPath := filepath.Join(path, entry.Name())
			err = s.findWalkEx(entryPath, opts, depth+1)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// checkMtime checks if the file modification time matches the mtime expression
// +n: more than n days ago, -n: less than n days ago, n: exactly n days ago
func (s *Shell) checkMtime(modTime time.Time, mtimeStr string) bool {
	now := time.Now()
	days, err := parseMtimeExpr(mtimeStr)
	if err != nil {
		return true // Invalid expression, don't filter
	}

	daysDiff := int(now.Sub(modTime).Hours() / 24)

	if strings.HasPrefix(mtimeStr, "+") {
		return daysDiff > days
	} else if strings.HasPrefix(mtimeStr, "-") {
		return daysDiff < days
	}
	return daysDiff == days
}

func parseMtimeExpr(expr string) (int, error) {
	expr = strings.TrimPrefix(expr, "+")
	expr = strings.TrimPrefix(expr, "-")
	return strconv.Atoi(expr)
}

// checkSize checks if the file size matches the size expression
// +n: larger than n, -n: smaller than n, n: exactly n
// Suffixes: c (bytes), k (kilobytes), M (megabytes), G (gigabytes)
func (s *Shell) checkSize(fileSize int64, sizeStr string) bool {
	size, err := parseSizeExpr(sizeStr)
	if err != nil {
		return true // Invalid expression, don't filter
	}

	if strings.HasPrefix(sizeStr, "+") {
		return fileSize > size
	} else if strings.HasPrefix(sizeStr, "-") {
		return fileSize < size
	}
	return fileSize == size
}

func parseSizeExpr(expr string) (int64, error) {
	expr = strings.TrimPrefix(expr, "+")
	expr = strings.TrimPrefix(expr, "-")

	multiplier := int64(512) // Default: 512-byte blocks
	if strings.HasSuffix(expr, "c") {
		multiplier = 1
		expr = strings.TrimSuffix(expr, "c")
	} else if strings.HasSuffix(expr, "k") {
		multiplier = 1024
		expr = strings.TrimSuffix(expr, "k")
	} else if strings.HasSuffix(expr, "M") {
		multiplier = 1024 * 1024
		expr = strings.TrimSuffix(expr, "M")
	} else if strings.HasSuffix(expr, "G") {
		multiplier = 1024 * 1024 * 1024
		expr = strings.TrimSuffix(expr, "G")
	}

	n, err := strconv.ParseInt(expr, 10, 64)
	if err != nil {
		return 0, err
	}

	return n * multiplier, nil
}

// globToRegex converts a shell glob pattern to a regex pattern
func globToRegex(pattern string) string {
	pattern = regexp.QuoteMeta(pattern)
	pattern = strings.ReplaceAll(pattern, `\*`, ".*")
	pattern = strings.ReplaceAll(pattern, `\?`, ".")
	return "^" + pattern + "$"
}

// cmdGrepEx implements enhanced grep command with additional options
func (s *Shell) cmdGrepEx(ctx context.Context, args []string) error {
	stdin, stdout, stderr := s.stdio(ctx)
	origIn, origOut, origErr := s.stdin, s.stdout, s.stderr
	s.stdin, s.stdout, s.stderr = stdin, stdout, stderr
	defer func() {
		s.stdin, s.stdout, s.stderr = origIn, origOut, origErr
	}()

	opts := &grepOptions{
		afterContext:  0,
		beforeContext: 0,
	}

	pattern := ""
	files := []string{}

	// Parse flags and arguments
	i := 1
	for i < len(args) {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			// First non-flag argument is the pattern
			pattern = arg
			i++
			break
		}

		// Handle long options
		if strings.HasPrefix(arg, "--") {
			switch {
			case arg == "--ignore-case":
				opts.ignoreCase = true
			case arg == "--invert-match":
				opts.invert = true
			case arg == "--line-number":
				opts.lineNumbers = true
			case arg == "--count":
				opts.count = true
			case arg == "--quiet", arg == "--silent":
				opts.quiet = true
			case arg == "--files-with-matches":
				opts.filesOnly = true
			case arg == "--files-without-match":
				opts.filesWithout = true
			case arg == "--recursive":
				opts.recursive = true
			case strings.HasPrefix(arg, "--after-context="):
				n, _ := strconv.Atoi(strings.TrimPrefix(arg, "--after-context="))
				opts.afterContext = n
			case strings.HasPrefix(arg, "--before-context="):
				n, _ := strconv.Atoi(strings.TrimPrefix(arg, "--before-context="))
				opts.beforeContext = n
			case strings.HasPrefix(arg, "--context="):
				n, _ := strconv.Atoi(strings.TrimPrefix(arg, "--context="))
				opts.afterContext = n
				opts.beforeContext = n
			case strings.HasPrefix(arg, "--include="):
				opts.includeGlob = strings.TrimPrefix(arg, "--include=")
			case strings.HasPrefix(arg, "--exclude="):
				opts.excludeGlob = strings.TrimPrefix(arg, "--exclude=")
			}
			i++
			continue
		}

		// Handle short options (can be combined like -inr)
		for j := 1; j < len(arg); j++ {
			ch := arg[j]
			switch ch {
			case 'i':
				opts.ignoreCase = true
			case 'v':
				opts.invert = true
			case 'n':
				opts.lineNumbers = true
			case 'c':
				opts.count = true
			case 'q':
				opts.quiet = true
			case 'l':
				opts.filesOnly = true
			case 'L':
				opts.filesWithout = true
			case 'r', 'R':
				opts.recursive = true
			case 'A':
				// -A requires a number
				if j+1 < len(arg) {
					n, _ := strconv.Atoi(arg[j+1:])
					opts.afterContext = n
					j = len(arg) // Skip rest of arg
				} else if i+1 < len(args) {
					n, _ := strconv.Atoi(args[i+1])
					opts.afterContext = n
					i++
				}
			case 'B':
				if j+1 < len(arg) {
					n, _ := strconv.Atoi(arg[j+1:])
					opts.beforeContext = n
					j = len(arg)
				} else if i+1 < len(args) {
					n, _ := strconv.Atoi(args[i+1])
					opts.beforeContext = n
					i++
				}
			case 'C':
				if j+1 < len(arg) {
					n, _ := strconv.Atoi(arg[j+1:])
					opts.afterContext = n
					opts.beforeContext = n
					j = len(arg)
				} else if i+1 < len(args) {
					n, _ := strconv.Atoi(args[i+1])
					opts.afterContext = n
					opts.beforeContext = n
					i++
				}
			}
		}
		i++
	}

	// Remaining args are files
	for i < len(args) {
		files = append(files, args[i])
		i++
	}

	if pattern == "" {
		return fmt.Errorf("grep: missing pattern")
	}

	// Compile regex
	if opts.ignoreCase {
		pattern = "(?i)" + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("grep: invalid pattern: %v", err)
	}
	opts.re = re

	// If no files and recursive, use current directory
	if len(files) == 0 && opts.recursive {
		files = []string{"."}
	}

	// If no files, read from stdin
	if len(files) == 0 {
		return s.grepReaderEx(re, s.stdin, "", opts)
	}

	// Collect files (handle recursive)
	var allFiles []string
	for _, path := range files {
		path = s.resolvePath(path)
		info, err := s.fs.Stat(path)
		if err != nil {
			fmt.Fprintf(s.stderr, "grep: %s: %v\n", path, err)
			continue
		}

		if info.IsDir() {
			if opts.recursive {
				collected, err := s.collectFiles(path, opts)
				if err != nil {
					fmt.Fprintf(s.stderr, "grep: %s: %v\n", path, err)
					continue
				}
				allFiles = append(allFiles, collected...)
			} else {
				fmt.Fprintf(s.stderr, "grep: %s: Is a directory\n", path)
			}
		} else {
			allFiles = append(allFiles, path)
		}
	}

	// Process each file
	matchFound := false
	showFilename := len(allFiles) > 1

	for _, path := range allFiles {
		file, err := s.openFile(path)
		if err != nil {
			if !opts.quiet {
				fmt.Fprintf(s.stderr, "grep: %s: %v\n", path, err)
			}
			continue
		}

		displayPath := path
		if showFilename {
			opts.showFilename = true
		}

		err = s.grepReaderEx(opts.re, file, displayPath, opts)
		file.Close()

		if err == nil {
			matchFound = true
		}
	}

	if !matchFound && opts.quiet {
		return fmt.Errorf("no match found")
	}
	return nil
}

type grepOptions struct {
	ignoreCase    bool
	invert        bool
	lineNumbers   bool
	count         bool
	quiet         bool
	filesOnly     bool
	filesWithout  bool
	recursive     bool
	afterContext  int
	beforeContext int
	includeGlob   string
	excludeGlob   string
	re            *regexp.Regexp
	showFilename  bool
}

func (s *Shell) collectFiles(dir string, opts *grepOptions) ([]string, error) {
	var files []string

	entries, err := afero.ReadDir(s.fs, dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			subFiles, err := s.collectFiles(path, opts)
			if err != nil {
				continue
			}
			files = append(files, subFiles...)
		} else {
			// Apply include/exclude patterns
			if opts.includeGlob != "" {
				matched, _ := filepath.Match(opts.includeGlob, entry.Name())
				if !matched {
					continue
				}
			}
			if opts.excludeGlob != "" {
				matched, _ := filepath.Match(opts.excludeGlob, entry.Name())
				if matched {
					continue
				}
			}
			files = append(files, path)
		}
	}

	return files, nil
}

func (s *Shell) grepReaderEx(re *regexp.Regexp, r io.Reader, filename string, opts *grepOptions) error {
	if r == nil {
		return fmt.Errorf("no input")
	}

	// Read all lines for context support
	content, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")

	matchCount := 0
	matchedInFile := false
	printedLines := make(map[int]bool)

	for lineNum, line := range lines {
		matches := re.MatchString(line)
		if opts.invert {
			matches = !matches
		}

		if matches {
			matchCount++
			matchedInFile = true

			if opts.quiet {
				return nil
			}
			if opts.filesOnly {
				fmt.Fprintln(s.stdout, filename)
				return nil
			}
			if opts.count {
				continue
			}

			// Print before context
			if opts.beforeContext > 0 {
				start := lineNum - opts.beforeContext
				if start < 0 {
					start = 0
				}
				for i := start; i < lineNum; i++ {
					if !printedLines[i] {
						s.printGrepLine(filename, i+1, lines[i], opts, "-")
						printedLines[i] = true
					}
				}
			}

			// Print matching line
			if !printedLines[lineNum] {
				s.printGrepLine(filename, lineNum+1, line, opts, ":")
				printedLines[lineNum] = true
			}

			// Print after context
			if opts.afterContext > 0 {
				end := lineNum + opts.afterContext + 1
				if end > len(lines) {
					end = len(lines)
				}
				for i := lineNum + 1; i < end; i++ {
					if !printedLines[i] {
						s.printGrepLine(filename, i+1, lines[i], opts, "-")
						printedLines[i] = true
					}
				}
			}
		}
	}

	if opts.count && !opts.quiet {
		prefix := ""
		if filename != "" && opts.showFilename {
			prefix = filename + ":"
		}
		fmt.Fprintf(s.stdout, "%s%d\n", prefix, matchCount)
	}

	if opts.filesWithout && !matchedInFile {
		fmt.Fprintln(s.stdout, filename)
	}

	if matchCount == 0 {
		return fmt.Errorf("no match found")
	}

	return nil
}

func (s *Shell) printGrepLine(filename string, lineNum int, line string, opts *grepOptions, sep string) {
	prefix := ""
	if filename != "" && opts.showFilename {
		prefix = filename + sep
	}
	if opts.lineNumbers {
		prefix += fmt.Sprintf("%d%s", lineNum, sep)
	}
	fmt.Fprintf(s.stdout, "%s%s\n", prefix, line)
}

// cmdExists implements file existence check - returns exit code 0 if exists, 1 otherwise
func (s *Shell) cmdExists(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("exists: missing file operand")
	}

	checkDir := false
	checkFile := false
	i := 1

	for i < len(args) && strings.HasPrefix(args[i], "-") {
		switch args[i] {
		case "-d":
			checkDir = true
		case "-f":
			checkFile = true
		}
		i++
	}

	if i >= len(args) {
		return fmt.Errorf("exists: missing file operand")
	}

	path := s.resolvePath(args[i])
	info, err := s.fs.Stat(path)

	if err != nil {
		return fmt.Errorf("exists: %s does not exist", args[i])
	}

	if checkDir && !info.IsDir() {
		return fmt.Errorf("exists: %s is not a directory", args[i])
	}

	if checkFile && info.IsDir() {
		return fmt.Errorf("exists: %s is not a file", args[i])
	}

	return nil
}
