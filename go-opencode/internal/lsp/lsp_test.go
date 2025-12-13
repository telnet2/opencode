package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	client := NewClient("/tmp", false)
	assert.NotNil(t, client)
	assert.False(t, client.IsDisabled())
	assert.NotEmpty(t, client.GetServers())
}

func TestNewClient_Disabled(t *testing.T) {
	client := NewClient("/tmp", true)
	assert.True(t, client.IsDisabled())
}

func TestClient_SetDisabled(t *testing.T) {
	client := NewClient("/tmp", false)

	client.SetDisabled(true)
	assert.True(t, client.IsDisabled())

	client.SetDisabled(false)
	assert.False(t, client.IsDisabled())
}

func TestBuiltInServers(t *testing.T) {
	servers := builtInServers()

	// Verify expected servers exist
	expectedServers := []string{"typescript", "go", "python", "rust"}
	for _, name := range expectedServers {
		server, ok := servers[name]
		assert.True(t, ok, "expected server %s to exist", name)
		assert.NotEmpty(t, server.Extensions)
		assert.NotEmpty(t, server.Command)
	}

	// Verify typescript server
	ts := servers["typescript"]
	assert.Contains(t, ts.Extensions, ".ts")
	assert.Contains(t, ts.Extensions, ".tsx")
	assert.Contains(t, ts.Extensions, ".js")
	assert.Contains(t, ts.Extensions, ".jsx")

	// Verify go server
	go_ := servers["go"]
	assert.Contains(t, go_.Extensions, ".go")

	// Verify python server
	py := servers["python"]
	assert.Contains(t, py.Extensions, ".py")

	// Verify rust server
	rs := servers["rust"]
	assert.Contains(t, rs.Extensions, ".rs")
}

func TestClient_AddServer(t *testing.T) {
	client := NewClient("/tmp", false)

	config := &ServerConfig{
		ID:         "custom",
		Extensions: []string{".custom"},
		Command:    []string{"custom-server", "--stdio"},
	}

	client.AddServer(config)

	servers := client.GetServers()
	assert.Contains(t, servers, "custom")
	assert.Equal(t, ".custom", servers["custom"].Extensions[0])
}

func TestClient_Status_Empty(t *testing.T) {
	client := NewClient("/tmp", false)
	status := client.Status()
	assert.Empty(t, status)
}

func TestDetectLanguageID(t *testing.T) {
	tests := []struct {
		file     string
		expected string
	}{
		{"main.go", "go"},
		{"index.ts", "typescript"},
		{"App.tsx", "typescriptreact"},
		{"script.js", "javascript"},
		{"Component.jsx", "javascriptreact"},
		{"app.py", "python"},
		{"lib.rs", "rust"},
		{"Main.java", "java"},
		{"program.c", "c"},
		{"program.cpp", "cpp"},
		{"header.h", "cpp"},
		{"script.rb", "ruby"},
		{"index.php", "php"},
		{"Program.cs", "csharp"},
		{"app.swift", "swift"},
		{"Main.kt", "kotlin"},
		{"App.scala", "scala"},
		{"script.lua", "lua"},
		{"script.sh", "shellscript"},
		{"config.yaml", "yaml"},
		{"config.yml", "yaml"},
		{"data.json", "json"},
		{"config.xml", "xml"},
		{"index.html", "html"},
		{"style.css", "css"},
		{"style.scss", "scss"},
		{"style.less", "less"},
		{"README.md", "markdown"},
		{"query.sql", "sql"},
		{"unknown.xyz", "plaintext"},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			result := detectLanguageID(tt.file)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSymbolKind_String(t *testing.T) {
	tests := []struct {
		kind     SymbolKind
		expected string
	}{
		{SymbolKindFile, "File"},
		{SymbolKindModule, "Module"},
		{SymbolKindNamespace, "Namespace"},
		{SymbolKindPackage, "Package"},
		{SymbolKindClass, "Class"},
		{SymbolKindMethod, "Method"},
		{SymbolKindProperty, "Property"},
		{SymbolKindField, "Field"},
		{SymbolKindConstructor, "Constructor"},
		{SymbolKindEnum, "Enum"},
		{SymbolKindInterface, "Interface"},
		{SymbolKindFunction, "Function"},
		{SymbolKindVariable, "Variable"},
		{SymbolKindConstant, "Constant"},
		{SymbolKindStruct, "Struct"},
		{SymbolKind(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.kind.String())
		})
	}
}

func TestAllSymbolKinds(t *testing.T) {
	kinds := AllSymbolKinds()
	assert.Len(t, kinds, 26)
	assert.Contains(t, kinds, SymbolKindFile)
	assert.Contains(t, kinds, SymbolKindFunction)
	assert.Contains(t, kinds, SymbolKindClass)
	assert.Contains(t, kinds, SymbolKindMethod)
}

func TestClient_FindProjectRoot(t *testing.T) {
	client := NewClient("/default", false)

	// When no markers found, should return workDir
	root := client.findProjectRoot("/some/unknown/path/file.go", "go")
	assert.Equal(t, "/default", root)
}

func TestClient_Close(t *testing.T) {
	client := NewClient("/tmp", false)

	// Should not panic on empty client
	err := client.Close()
	assert.NoError(t, err)
}

func TestClient_GetServers(t *testing.T) {
	client := NewClient("/tmp", false)
	servers := client.GetServers()

	// Verify it returns a copy
	servers["new"] = &ServerConfig{ID: "new"}

	originalServers := client.GetServers()
	_, exists := originalServers["new"]
	assert.False(t, exists, "GetServers should return a copy")
}

func TestJSONRPCRequest(t *testing.T) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
		Params:  map[string]string{"key": "value"},
	}

	assert.Equal(t, "2.0", req.JSONRPC)
	assert.Equal(t, int64(1), req.ID)
	assert.Equal(t, "test", req.Method)
}

func TestInitializeParams(t *testing.T) {
	params := InitializeParams{
		ProcessID: 12345,
		RootURI:   "file:///project",
		Capabilities: ClientCapabilities{
			TextDocument: TextDocumentClientCapabilities{
				Hover: &HoverCapability{
					ContentFormat: []string{"plaintext", "markdown"},
				},
			},
		},
	}

	assert.Equal(t, 12345, params.ProcessID)
	assert.Equal(t, "file:///project", params.RootURI)
	assert.NotNil(t, params.Capabilities.TextDocument.Hover)
}

func TestServerConfig(t *testing.T) {
	config := ServerConfig{
		ID:         "test",
		Extensions: []string{".test"},
		Command:    []string{"test-server", "--stdio"},
	}

	assert.Equal(t, "test", config.ID)
	assert.Contains(t, config.Extensions, ".test")
	assert.Equal(t, "test-server", config.Command[0])
}

func TestSymbol(t *testing.T) {
	symbol := Symbol{
		Name: "TestFunction",
		Kind: SymbolKindFunction,
		Location: SymbolLocation{
			URI: "file:///test.go",
			Range: Range{
				Start: Position{Line: 10, Character: 5},
				End:   Position{Line: 10, Character: 20},
			},
		},
	}

	assert.Equal(t, "TestFunction", symbol.Name)
	assert.Equal(t, SymbolKindFunction, symbol.Kind)
	assert.Equal(t, 10, symbol.Location.Range.Start.Line)
	assert.Equal(t, 5, symbol.Location.Range.Start.Character)
}

func TestDiagnostic(t *testing.T) {
	diag := Diagnostic{
		Range: Range{
			Start: Position{Line: 5, Character: 0},
			End:   Position{Line: 5, Character: 10},
		},
		Severity: DiagnosticSeverityError,
		Code:     "E001",
		Source:   "linter",
		Message:  "Test error",
	}

	assert.Equal(t, DiagnosticSeverityError, diag.Severity)
	assert.Equal(t, "Test error", diag.Message)
	assert.Equal(t, "linter", diag.Source)
}

func TestHoverResult(t *testing.T) {
	result := HoverResult{
		Contents: "Test hover content",
		Range: &Range{
			Start: Position{Line: 1, Character: 0},
			End:   Position{Line: 1, Character: 10},
		},
	}

	assert.Equal(t, "Test hover content", result.Contents)
	assert.NotNil(t, result.Range)
}
