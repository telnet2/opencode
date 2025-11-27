package server

import (
	"bufio"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/opencode-ai/opencode/internal/lsp"
)

// FileInfo represents file information.
type FileInfo struct {
	Name        string `json:"name"`
	IsDirectory bool   `json:"isDirectory"`
	Size        int64  `json:"size"`
}

// listFiles handles GET /file
func (s *Server) listFiles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = getDirectory(r.Context())
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
		return
	}

	var files []FileInfo
	for _, entry := range entries {
		info, _ := entry.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		files = append(files, FileInfo{
			Name:        entry.Name(),
			IsDirectory: entry.IsDir(),
			Size:        size,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"files": files})
}

// readFile handles GET /file/content
func (s *Server) readFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "path required")
		return
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 2000
	}

	file, err := os.Open(path)
	if err != nil {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, "File not found")
		return
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		if lineNum < offset {
			continue
		}
		if len(lines) >= limit {
			break
		}
		lines = append(lines, scanner.Text())
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"content":   strings.Join(lines, "\n"),
		"lines":     len(lines),
		"truncated": lineNum > offset+limit,
	})
}

// gitStatus handles GET /file/status
func (s *Server) gitStatus(w http.ResponseWriter, r *http.Request) {
	directory := r.URL.Query().Get("directory")
	if directory == "" {
		directory = getDirectory(r.Context())
	}

	// Get current branch
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = directory
	branch, _ := cmd.Output()

	// Get status
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = directory
	output, _ := cmd.Output()

	var staged, unstaged, untracked []string
	for _, line := range strings.Split(string(output), "\n") {
		if len(line) < 3 {
			continue
		}
		status := line[:2]
		file := strings.TrimSpace(line[3:])

		switch {
		case status[0] != ' ' && status[0] != '?':
			staged = append(staged, file)
		case status[1] != ' ' && status[1] != '?':
			unstaged = append(unstaged, file)
		case status == "??":
			untracked = append(untracked, file)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"branch":    strings.TrimSpace(string(branch)),
		"staged":    staged,
		"unstaged":  unstaged,
		"untracked": untracked,
	})
}

// searchText handles GET /find
func (s *Server) searchText(w http.ResponseWriter, r *http.Request) {
	pattern := r.URL.Query().Get("pattern")
	if pattern == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "pattern required")
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		path = getDirectory(r.Context())
	}

	include := r.URL.Query().Get("include")

	args := []string{
		"--line-number",
		"--with-filename",
		"--color=never",
	}

	if include != "" {
		args = append(args, "--glob", include)
	}

	args = append(args, pattern, path)

	cmd := exec.Command("rg", args...)
	output, _ := cmd.Output()

	type SearchMatch struct {
		File    string `json:"file"`
		Line    int    `json:"line"`
		Content string `json:"content"`
	}

	var matches []SearchMatch
	for _, line := range strings.Split(string(output), "\n") {
		if line == "" {
			continue
		}

		// Parse: file:line:content
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}

		lineNum, _ := strconv.Atoi(parts[1])
		matches = append(matches, SearchMatch{
			File:    parts[0],
			Line:    lineNum,
			Content: parts[2],
		})
	}

	// Limit results
	const maxMatches = 100
	truncated := false
	if len(matches) > maxMatches {
		matches = matches[:maxMatches]
		truncated = true
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"matches":   matches,
		"count":     len(matches),
		"truncated": truncated,
	})
}

// searchFiles handles GET /find/file
func (s *Server) searchFiles(w http.ResponseWriter, r *http.Request) {
	pattern := r.URL.Query().Get("pattern")
	if pattern == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "pattern required")
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		path = getDirectory(r.Context())
	}

	cmd := exec.Command("rg", "--files", "--glob", pattern)
	cmd.Dir = path
	output, _ := cmd.Output()

	files := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Filter empty strings
	var result []string
	for _, f := range files {
		if f != "" {
			result = append(result, filepath.Clean(f))
		}
	}

	// Limit results
	const maxFiles = 100
	if len(result) > maxFiles {
		result = result[:maxFiles]
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"files": result,
		"count": len(result),
	})
}

// Symbol kinds to include in results (matching TypeScript implementation).
// See: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#symbolKind
var symbolKindsFilter = map[lsp.SymbolKind]bool{
	lsp.SymbolKindClass:     true, // 5
	lsp.SymbolKindMethod:    true, // 6
	lsp.SymbolKindEnum:      true, // 10
	lsp.SymbolKindInterface: true, // 11
	lsp.SymbolKindFunction:  true, // 12
	lsp.SymbolKindVariable:  true, // 13
	lsp.SymbolKindConstant:  true, // 14
	lsp.SymbolKindStruct:    true, // 23
}

// searchSymbols handles GET /find/symbol
func (s *Server) searchSymbols(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "query parameter required")
		return
	}

	// Check if LSP is available
	if s.lspClient == nil || s.lspClient.IsDisabled() {
		writeJSON(w, http.StatusOK, []lsp.Symbol{})
		return
	}

	ctx := r.Context()
	symbols, err := s.lspClient.WorkspaceSymbol(ctx, query)
	if err != nil {
		// Log error but return empty array (matching TS behavior)
		writeJSON(w, http.StatusOK, []lsp.Symbol{})
		return
	}

	// Filter by symbol kinds (matching TypeScript)
	filtered := make([]lsp.Symbol, 0, len(symbols))
	for _, sym := range symbols {
		if symbolKindsFilter[sym.Kind] {
			filtered = append(filtered, sym)
		}
	}

	// Limit to 10 results (matching TypeScript)
	if len(filtered) > 10 {
		filtered = filtered[:10]
	}

	writeJSON(w, http.StatusOK, filtered)
}
