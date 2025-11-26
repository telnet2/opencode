package lsp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// WorkspaceSymbol searches for symbols in the workspace.
func (c *Client) WorkspaceSymbol(ctx context.Context, query string) ([]Symbol, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var allSymbols []Symbol

	for _, client := range c.clients {
		symbols, err := client.workspaceSymbol(ctx, query)
		if err != nil {
			continue // Skip failed clients
		}
		allSymbols = append(allSymbols, symbols...)
	}

	return allSymbols, nil
}

func (lc *languageClient) workspaceSymbol(ctx context.Context, query string) ([]Symbol, error) {
	params := WorkspaceSymbolParams{
		Query: query,
	}

	var result []SymbolInformation
	if err := lc.conn.call(ctx, "workspace/symbol", params, &result); err != nil {
		return nil, err
	}

	symbols := make([]Symbol, len(result))
	for i, s := range result {
		symbols[i] = Symbol{
			Name: s.Name,
			Kind: s.Kind,
			Location: SymbolLocation{
				URI: s.Location.URI,
				Range: Range{
					Start: Position{
						Line:      s.Location.Range.Start.Line,
						Character: s.Location.Range.Start.Character,
					},
					End: Position{
						Line:      s.Location.Range.End.Line,
						Character: s.Location.Range.End.Character,
					},
				},
			},
		}
	}

	return symbols, nil
}

// Hover returns hover information for a position.
func (c *Client) Hover(ctx context.Context, file string, line, character int) (*HoverResult, error) {
	client, err := c.GetClient(ctx, file)
	if err != nil {
		return nil, err
	}

	return client.hover(ctx, file, line, character)
}

func (lc *languageClient) hover(ctx context.Context, file string, line, character int) (*HoverResult, error) {
	params := TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{
			URI: "file://" + file,
		},
		Position: Position{
			Line:      line,
			Character: character,
		},
	}

	var result struct {
		Contents any    `json:"contents"`
		Range    *Range `json:"range,omitempty"`
	}

	if err := lc.conn.call(ctx, "textDocument/hover", params, &result); err != nil {
		return nil, err
	}

	if result.Contents == nil {
		return nil, nil
	}

	// Extract text from hover contents
	var contents string
	switch v := result.Contents.(type) {
	case string:
		contents = v
	case map[string]any:
		if value, ok := v["value"].(string); ok {
			contents = value
		}
	case []any:
		var parts []string
		for _, p := range v {
			if s, ok := p.(string); ok {
				parts = append(parts, s)
			} else if m, ok := p.(map[string]any); ok {
				if value, ok := m["value"].(string); ok {
					parts = append(parts, value)
				}
			}
		}
		contents = strings.Join(parts, "\n")
	}

	return &HoverResult{
		Contents: contents,
		Range:    result.Range,
	}, nil
}

// DocumentSymbol returns symbols in a document.
func (c *Client) DocumentSymbol(ctx context.Context, file string) ([]Symbol, error) {
	client, err := c.GetClient(ctx, file)
	if err != nil {
		return nil, err
	}

	return client.documentSymbol(ctx, file)
}

func (lc *languageClient) documentSymbol(ctx context.Context, file string) ([]Symbol, error) {
	params := DocumentSymbolParams{
		TextDocument: TextDocumentIdentifier{
			URI: "file://" + file,
		},
	}

	var result []SymbolInformation
	if err := lc.conn.call(ctx, "textDocument/documentSymbol", params, &result); err != nil {
		return nil, err
	}

	symbols := make([]Symbol, len(result))
	for i, s := range result {
		symbols[i] = Symbol{
			Name: s.Name,
			Kind: s.Kind,
			Location: SymbolLocation{
				URI: s.Location.URI,
				Range: Range{
					Start: Position{
						Line:      s.Location.Range.Start.Line,
						Character: s.Location.Range.Start.Character,
					},
					End: Position{
						Line:      s.Location.Range.End.Line,
						Character: s.Location.Range.End.Character,
					},
				},
			},
		}
	}

	return symbols, nil
}

// TouchFile notifies the server of file changes (opens the file).
func (c *Client) TouchFile(ctx context.Context, file string) error {
	client, err := c.GetClient(ctx, file)
	if err != nil {
		return err
	}

	return client.touchFile(ctx, file)
}

func (lc *languageClient) touchFile(ctx context.Context, file string) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	uri := "file://" + file

	// Check if already open
	if _, ok := lc.openFiles[uri]; ok {
		// Already open, increment version and send change
		lc.openFiles[uri]++
		return nil
	}

	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	params := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: detectLanguageID(file),
			Version:    1,
			Text:       string(content),
		},
	}

	lc.openFiles[uri] = 1
	return lc.conn.notify(ctx, "textDocument/didOpen", params)
}

// CloseFile notifies the server that a file is closed.
func (c *Client) CloseFile(ctx context.Context, file string) error {
	client, err := c.GetClient(ctx, file)
	if err != nil {
		return err
	}

	return client.closeFile(ctx, file)
}

func (lc *languageClient) closeFile(ctx context.Context, file string) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	uri := "file://" + file

	if _, ok := lc.openFiles[uri]; !ok {
		return nil // Not open
	}

	params := struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}

	delete(lc.openFiles, uri)
	return lc.conn.notify(ctx, "textDocument/didClose", params)
}

// Definition returns the definition location for a position.
func (c *Client) Definition(ctx context.Context, file string, line, character int) ([]SymbolLocation, error) {
	client, err := c.GetClient(ctx, file)
	if err != nil {
		return nil, err
	}

	return client.definition(ctx, file, line, character)
}

func (lc *languageClient) definition(ctx context.Context, file string, line, character int) ([]SymbolLocation, error) {
	params := TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{
			URI: "file://" + file,
		},
		Position: Position{
			Line:      line,
			Character: character,
		},
	}

	var result []Location
	if err := lc.conn.call(ctx, "textDocument/definition", params, &result); err != nil {
		// Try single location format
		var single Location
		if err := lc.conn.call(ctx, "textDocument/definition", params, &single); err != nil {
			return nil, err
		}
		result = []Location{single}
	}

	locations := make([]SymbolLocation, len(result))
	for i, loc := range result {
		locations[i] = SymbolLocation{
			URI: loc.URI,
			Range: Range{
				Start: Position{
					Line:      loc.Range.Start.Line,
					Character: loc.Range.Start.Character,
				},
				End: Position{
					Line:      loc.Range.End.Line,
					Character: loc.Range.End.Character,
				},
			},
		}
	}

	return locations, nil
}

// References returns all references to the symbol at the given position.
func (c *Client) References(ctx context.Context, file string, line, character int, includeDeclaration bool) ([]SymbolLocation, error) {
	client, err := c.GetClient(ctx, file)
	if err != nil {
		return nil, err
	}

	return client.references(ctx, file, line, character, includeDeclaration)
}

func (lc *languageClient) references(ctx context.Context, file string, line, character int, includeDeclaration bool) ([]SymbolLocation, error) {
	params := struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Position     Position               `json:"position"`
		Context      struct {
			IncludeDeclaration bool `json:"includeDeclaration"`
		} `json:"context"`
	}{
		TextDocument: TextDocumentIdentifier{
			URI: "file://" + file,
		},
		Position: Position{
			Line:      line,
			Character: character,
		},
	}
	params.Context.IncludeDeclaration = includeDeclaration

	var result []Location
	if err := lc.conn.call(ctx, "textDocument/references", params, &result); err != nil {
		return nil, err
	}

	locations := make([]SymbolLocation, len(result))
	for i, loc := range result {
		locations[i] = SymbolLocation{
			URI: loc.URI,
			Range: Range{
				Start: Position{
					Line:      loc.Range.Start.Line,
					Character: loc.Range.Start.Character,
				},
				End: Position{
					Line:      loc.Range.End.Line,
					Character: loc.Range.End.Character,
				},
			},
		}
	}

	return locations, nil
}

// detectLanguageID detects the language ID from a file path.
func detectLanguageID(file string) string {
	ext := strings.ToLower(filepath.Ext(file))
	switch ext {
	case ".go":
		return "go"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "typescriptreact"
	case ".js":
		return "javascript"
	case ".jsx":
		return "javascriptreact"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".h", ".hpp":
		return "cpp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".cs":
		return "csharp"
	case ".swift":
		return "swift"
	case ".kt", ".kts":
		return "kotlin"
	case ".scala":
		return "scala"
	case ".lua":
		return "lua"
	case ".sh", ".bash":
		return "shellscript"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	case ".scss":
		return "scss"
	case ".less":
		return "less"
	case ".md":
		return "markdown"
	case ".sql":
		return "sql"
	default:
		return "plaintext"
	}
}
