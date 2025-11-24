package memsh

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/afero"
)

// cmdGrep implements the grep command
func (s *Shell) cmdGrep(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("grep: missing pattern")
	}

	stdin, stdout, stderr := s.stdio(ctx)
	origIn, origOut, origErr := s.stdin, s.stdout, s.stderr
	s.stdin, s.stdout, s.stderr = stdin, stdout, stderr
	defer func() {
		s.stdin, s.stdout, s.stderr = origIn, origOut, origErr
	}()

	ignoreCase := false
	invert := false
	lineNumbers := false
	count := false
	quiet := false
	pattern := ""
	files := []string{}

	// Parse flags and arguments
	i := 1
	for i < len(args) {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			// First non-flag argument is the pattern
			pattern = arg
			i++
			break
		}

		// Handle combined flags like -qi, -in, etc.
		if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			for _, ch := range arg[1:] {
				switch ch {
				case 'i':
					ignoreCase = true
				case 'v':
					invert = true
				case 'n':
					lineNumbers = true
				case 'c':
					count = true
				case 'q':
					quiet = true
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

	// Compile regex
	if ignoreCase {
		pattern = "(?i)" + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("grep: invalid pattern: %v", err)
	}

	// If no files, read from stdin
	if len(files) == 0 {
		return s.grepReader(re, s.stdin, "", lineNumbers, invert, count, quiet)
	}

	// Process each file
	hadError := false
	matchFound := false
	for _, path := range files {
		path = s.resolvePath(path)
		file, err := s.openFile(path)
		if err != nil {
			if !quiet {
				fmt.Fprintf(s.stderr, "grep: %s: %v\n", path, err)
			}
			hadError = true
			continue
		}

		err = s.grepReader(re, file, path, lineNumbers, invert, count, quiet)
		file.Close()
		if err == nil {
			matchFound = true
		}
		if err != nil && err.Error() != "no match found" {
			return err
		}
	}

	if hadError {
		return fmt.Errorf("grep: one or more files could not be opened")
	}
	if !matchFound && quiet {
		// Exit with non-zero status if no matches found in quiet mode
		return fmt.Errorf("no match found")
	}
	return nil
}

// grepReader performs grep on a reader
func (s *Shell) grepReader(re *regexp.Regexp, r io.Reader, filename string, lineNumbers, invert, countOnly, quiet bool) error {
	if r == nil {
		return fmt.Errorf("no input")
	}

	scanner := bufio.NewScanner(r)
	lineNum := 0
	matchCount := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		matches := re.MatchString(line)

		if invert {
			matches = !matches
		}

		if matches {
			matchCount++
			// In quiet mode, just exit success on first match
			if quiet {
				return nil
			}
			if !countOnly {
				prefix := ""
				if filename != "" {
					prefix = filename + ":"
				}
				if lineNumbers {
					prefix += fmt.Sprintf("%d:", lineNum)
				}
				fmt.Fprintf(s.stdout, "%s%s\n", prefix, line)
			}
		}
	}

	if countOnly && !quiet {
		prefix := ""
		if filename != "" {
			prefix = filename + ":"
		}
		fmt.Fprintf(s.stdout, "%s%d\n", prefix, matchCount)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// If no matches found, return error (for quiet mode exit status)
	if matchCount == 0 {
		return fmt.Errorf("no match found")
	}

	return nil
}

// cmdHead implements the head command
func (s *Shell) cmdHead(ctx context.Context, args []string) error {
	stdin, stdout, stderr := s.stdio(ctx)
	origIn, origOut, origErr := s.stdin, s.stdout, s.stderr
	s.stdin, s.stdout, s.stderr = stdin, stdout, stderr
	defer func() {
		s.stdin, s.stdout, s.stderr = origIn, origOut, origErr
	}()

	lines := 10
	files := []string{}

	// Parse arguments
	i := 1
	for i < len(args) {
		arg := args[i]
		if arg == "-n" && i+1 < len(args) {
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("head: invalid number of lines: %s", args[i+1])
			}
			lines = n
			i += 2
		} else if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			// -10 format
			n, err := strconv.Atoi(arg[1:])
			if err == nil {
				lines = n
				i++
			} else {
				return fmt.Errorf("head: invalid option: %s", arg)
			}
		} else {
			files = append(files, arg)
			i++
		}
	}

	// If no files, read from stdin
	if len(files) == 0 {
		return s.headReader(s.stdin, "", lines, false)
	}

	// Process each file
	showFilename := len(files) > 1
	for i, path := range files {
		if i > 0 && showFilename {
			fmt.Fprintln(s.stdout)
		}

		path = s.resolvePath(path)
		file, err := s.openFile(path)
		if err != nil {
			fmt.Fprintf(s.stderr, "head: %s: %v\n", path, err)
			continue
		}

		err = s.headReader(file, path, lines, showFilename)
		file.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// headReader reads the first n lines from a reader
func (s *Shell) headReader(r io.Reader, filename string, lines int, showFilename bool) error {
	if showFilename {
		fmt.Fprintf(s.stdout, "==> %s <==\n", filename)
	}

	scanner := bufio.NewScanner(r)
	count := 0

	for scanner.Scan() && count < lines {
		fmt.Fprintln(s.stdout, scanner.Text())
		count++
	}

	return scanner.Err()
}

// cmdTail implements the tail command
func (s *Shell) cmdTail(ctx context.Context, args []string) error {
	stdin, stdout, stderr := s.stdio(ctx)
	origIn, origOut, origErr := s.stdin, s.stdout, s.stderr
	s.stdin, s.stdout, s.stderr = stdin, stdout, stderr
	defer func() {
		s.stdin, s.stdout, s.stderr = origIn, origOut, origErr
	}()

	lines := 10
	files := []string{}

	// Parse arguments
	i := 1
	for i < len(args) {
		arg := args[i]
		if arg == "-n" && i+1 < len(args) {
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("tail: invalid number of lines: %s", args[i+1])
			}
			lines = n
			i += 2
		} else if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			// -10 format
			n, err := strconv.Atoi(arg[1:])
			if err == nil {
				lines = n
				i++
			} else {
				return fmt.Errorf("tail: invalid option: %s", arg)
			}
		} else {
			files = append(files, arg)
			i++
		}
	}

	// If no files, read from stdin
	if len(files) == 0 {
		return s.tailReader(s.stdin, "", lines, false)
	}

	// Process each file
	showFilename := len(files) > 1
	for i, path := range files {
		if i > 0 && showFilename {
			fmt.Fprintln(s.stdout)
		}

		path = s.resolvePath(path)
		file, err := s.openFile(path)
		if err != nil {
			fmt.Fprintf(s.stderr, "tail: %s: %v\n", path, err)
			continue
		}

		err = s.tailReader(file, path, lines, showFilename)
		file.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// tailReader reads the last n lines from a reader
func (s *Shell) tailReader(r io.Reader, filename string, lines int, showFilename bool) error {
	if showFilename {
		fmt.Fprintf(s.stdout, "==> %s <==\n", filename)
	}

	scanner := bufio.NewScanner(r)
	buffer := make([]string, 0, lines)

	for scanner.Scan() {
		buffer = append(buffer, scanner.Text())
		if len(buffer) > lines {
			buffer = buffer[1:]
		}
	}

	for _, line := range buffer {
		fmt.Fprintln(s.stdout, line)
	}

	return scanner.Err()
}

// cmdWc implements the wc command
func (s *Shell) cmdWc(ctx context.Context, args []string) error {
	stdin, stdout, stderr := s.stdio(ctx)
	origIn, origOut, origErr := s.stdin, s.stdout, s.stderr
	s.stdin, s.stdout, s.stderr = stdin, stdout, stderr
	defer func() {
		s.stdin, s.stdout, s.stderr = origIn, origOut, origErr
	}()

	showLines := true
	showWords := true
	showBytes := true
	files := []string{}

	// Parse arguments
	hasFlags := false
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			hasFlags = true
			if strings.Contains(arg, "l") {
				showLines = true
				showWords = false
				showBytes = false
			}
			if strings.Contains(arg, "w") {
				showWords = true
				if !strings.Contains(arg, "l") {
					showLines = false
				}
				showBytes = false
			}
			if strings.Contains(arg, "c") {
				showBytes = true
				if !strings.Contains(arg, "l") && !strings.Contains(arg, "w") {
					showLines = false
					showWords = false
				}
			}
		} else {
			files = append(files, arg)
		}
	}

	if hasFlags && !showLines && !showWords && !showBytes {
		showLines = true
		showWords = true
		showBytes = true
	}

	// If no files, read from stdin
	if len(files) == 0 {
		lines, words, bytes := s.wcReader(s.stdin)
		s.printWc(lines, words, bytes, "", showLines, showWords, showBytes)
		return nil
	}

	// Process each file
	totalLines := 0
	totalWords := 0
	totalBytes := 0

	for _, path := range files {
		path = s.resolvePath(path)
		file, err := s.openFile(path)
		if err != nil {
			fmt.Fprintf(s.stderr, "wc: %s: %v\n", path, err)
			continue
		}

		lines, words, bytes := s.wcReader(file)
		file.Close()

		s.printWc(lines, words, bytes, path, showLines, showWords, showBytes)

		totalLines += lines
		totalWords += words
		totalBytes += bytes
	}

	if len(files) > 1 {
		s.printWc(totalLines, totalWords, totalBytes, "total", showLines, showWords, showBytes)
	}

	return nil
}

// wcReader counts lines, words, and bytes in a reader
func (s *Shell) wcReader(r io.Reader) (lines, words, bytes int) {
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		lines++
		text := scanner.Text()
		bytes += len(text) + 1 // +1 for newline
		fields := strings.Fields(text)
		words += len(fields)
	}

	return
}

// printWc prints wc output
func (s *Shell) printWc(lines, words, bytes int, filename string, showLines, showWords, showBytes bool) {
	parts := []string{}

	if showLines {
		parts = append(parts, fmt.Sprintf("%7d", lines))
	}
	if showWords {
		parts = append(parts, fmt.Sprintf("%7d", words))
	}
	if showBytes {
		parts = append(parts, fmt.Sprintf("%7d", bytes))
	}

	output := strings.Join(parts, " ")
	if filename != "" {
		output += " " + filename
	}

	fmt.Fprintln(s.stdout, output)
}

// cmdSort implements the sort command
func (s *Shell) cmdSort(ctx context.Context, args []string) error {
	stdin, stdout, stderr := s.stdio(ctx)
	origIn, origOut, origErr := s.stdin, s.stdout, s.stderr
	s.stdin, s.stdout, s.stderr = stdin, stdout, stderr
	defer func() {
		s.stdin, s.stdout, s.stderr = origIn, origOut, origErr
	}()

	reverse := false
	unique := false
	numeric := false
	files := []string{}

	// Parse arguments
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			if strings.Contains(arg, "r") {
				reverse = true
			}
			if strings.Contains(arg, "u") {
				unique = true
			}
			if strings.Contains(arg, "n") {
				numeric = true
			}
		} else {
			files = append(files, arg)
		}
	}

	// Collect all lines
	var lines []string

	if len(files) == 0 {
		scanner := bufio.NewScanner(s.stdin)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	} else {
		for _, path := range files {
			path = s.resolvePath(path)
			file, err := s.openFile(path)
			if err != nil {
				fmt.Fprintf(s.stderr, "sort: %s: %v\n", path, err)
				continue
			}

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
			file.Close()

			if err := scanner.Err(); err != nil {
				return err
			}
		}
	}

	// Sort lines
	if numeric {
		sort.Slice(lines, func(i, j int) bool {
			ni, _ := strconv.ParseFloat(lines[i], 64)
			nj, _ := strconv.ParseFloat(lines[j], 64)
			if reverse {
				return ni > nj
			}
			return ni < nj
		})
	} else {
		sort.Strings(lines)
		if reverse {
			for i := 0; i < len(lines)/2; i++ {
				j := len(lines) - 1 - i
				lines[i], lines[j] = lines[j], lines[i]
			}
		}
	}

	// Remove duplicates if -u
	if unique {
		uniqueLines := []string{}
		prev := ""
		for _, line := range lines {
			if line != prev {
				uniqueLines = append(uniqueLines, line)
				prev = line
			}
		}
		lines = uniqueLines
	}

	// Output
	for _, line := range lines {
		fmt.Fprintln(s.stdout, line)
	}

	return nil
}

// cmdUniq implements the uniq command
func (s *Shell) cmdUniq(ctx context.Context, args []string) error {
	stdin, stdout, stderr := s.stdio(ctx)
	origIn, origOut, origErr := s.stdin, s.stdout, s.stderr
	s.stdin, s.stdout, s.stderr = stdin, stdout, stderr
	defer func() {
		s.stdin, s.stdout, s.stderr = origIn, origOut, origErr
	}()

	count := false
	files := []string{}

	// Parse arguments
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if arg == "-c" {
			count = true
		} else {
			files = append(files, arg)
		}
	}

	var reader io.Reader

	if len(files) == 0 {
		reader = s.stdin
	} else {
		path := s.resolvePath(files[0])
		file, err := s.openFile(path)
		if err != nil {
			return fmt.Errorf("uniq: %s: %v", path, err)
		}
		defer file.Close()
		reader = file
	}

	scanner := bufio.NewScanner(reader)
	prev := ""
	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()

		if line != prev {
			if prev != "" {
				if count {
					fmt.Fprintf(s.stdout, "%7d %s\n", lineCount, prev)
				} else {
					fmt.Fprintln(s.stdout, prev)
				}
			}
			prev = line
			lineCount = 1
		} else {
			lineCount++
		}
	}

	// Print last line
	if prev != "" {
		if count {
			fmt.Fprintf(s.stdout, "%7d %s\n", lineCount, prev)
		} else {
			fmt.Fprintln(s.stdout, prev)
		}
	}

	return scanner.Err()
}

// cmdFind implements the find command
func (s *Shell) cmdFind(ctx context.Context, args []string) error {
	stdin, stdout, stderr := s.stdio(ctx)
	origIn, origOut, origErr := s.stdin, s.stdout, s.stderr
	s.stdin, s.stdout, s.stderr = stdin, stdout, stderr
	defer func() {
		s.stdin, s.stdout, s.stderr = origIn, origOut, origErr
	}()

	paths := []string{"."}
	namePattern := ""
	fileType := "" // f for file, d for directory

	// Parse arguments
	i := 1
	for i < len(args) {
		arg := args[i]

		if arg == "-name" && i+1 < len(args) {
			namePattern = args[i+1]
			i += 2
		} else if arg == "-type" && i+1 < len(args) {
			fileType = args[i+1]
			i += 2
		} else if !strings.HasPrefix(arg, "-") {
			paths = []string{arg}
			i++
		} else {
			i++
		}
	}

	// Compile name pattern if provided
	var nameRe *regexp.Regexp
	if namePattern != "" {
		// Convert glob pattern to regex
		pattern := strings.ReplaceAll(namePattern, ".", "\\.")
		pattern = strings.ReplaceAll(pattern, "*", ".*")
		pattern = strings.ReplaceAll(pattern, "?", ".")
		pattern = "^" + pattern + "$"

		var err error
		nameRe, err = regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("find: invalid pattern: %v", err)
		}
	}

	// Search each path
	for _, path := range paths {
		path = s.resolvePath(path)
		err := s.findWalk(path, nameRe, fileType)
		if err != nil {
			return err
		}
	}

	return nil
}

// findWalk recursively walks a directory
func (s *Shell) findWalk(path string, nameRe *regexp.Regexp, fileType string) error {
	info, err := s.fs.Stat(path)
	if err != nil {
		return err
	}

	// Check if this entry matches the criteria
	matches := true

	if nameRe != nil {
		matches = matches && nameRe.MatchString(info.Name())
	}

	if fileType != "" {
		if fileType == "f" {
			matches = matches && !info.IsDir()
		} else if fileType == "d" {
			matches = matches && info.IsDir()
		}
	}

	if matches {
		fmt.Fprintln(s.stdout, path)
	}

	// Recurse into directories
	if info.IsDir() {
		entries, err := afero.ReadDir(s.fs, path)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			entryPath := path + "/" + entry.Name()
			err = s.findWalk(entryPath, nameRe, fileType)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
