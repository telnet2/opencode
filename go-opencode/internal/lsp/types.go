// Package lsp provides Language Server Protocol client functionality.
package lsp

import "encoding/json"

// ServerConfig defines a language server configuration.
type ServerConfig struct {
	ID         string   `json:"id"`
	Extensions []string `json:"extensions"` // File extensions handled
	Command    []string `json:"command"`    // Command to spawn server
}

// ServerStatus represents the status of a language server.
type ServerStatus struct {
	ID     string `json:"id"`
	Root   string `json:"root"`
	Key    string `json:"key"`
	Active bool   `json:"active"`
}

// Symbol represents a code symbol.
type Symbol struct {
	Name     string         `json:"name"`
	Kind     SymbolKind     `json:"kind"`
	Location SymbolLocation `json:"location"`
}

// SymbolLocation represents a location in a document.
type SymbolLocation struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Range represents a range in a text document.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Position represents a position in a text document.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// SymbolKind represents the kind of a symbol.
type SymbolKind int

const (
	SymbolKindFile        SymbolKind = 1
	SymbolKindModule      SymbolKind = 2
	SymbolKindNamespace   SymbolKind = 3
	SymbolKindPackage     SymbolKind = 4
	SymbolKindClass       SymbolKind = 5
	SymbolKindMethod      SymbolKind = 6
	SymbolKindProperty    SymbolKind = 7
	SymbolKindField       SymbolKind = 8
	SymbolKindConstructor SymbolKind = 9
	SymbolKindEnum        SymbolKind = 10
	SymbolKindInterface   SymbolKind = 11
	SymbolKindFunction    SymbolKind = 12
	SymbolKindVariable    SymbolKind = 13
	SymbolKindConstant    SymbolKind = 14
	SymbolKindString      SymbolKind = 15
	SymbolKindNumber      SymbolKind = 16
	SymbolKindBoolean     SymbolKind = 17
	SymbolKindArray       SymbolKind = 18
	SymbolKindObject      SymbolKind = 19
	SymbolKindKey         SymbolKind = 20
	SymbolKindNull        SymbolKind = 21
	SymbolKindEnumMember  SymbolKind = 22
	SymbolKindStruct      SymbolKind = 23
	SymbolKindEvent       SymbolKind = 24
	SymbolKindOperator    SymbolKind = 25
	SymbolKindTypeParam   SymbolKind = 26
)

// String returns the string representation of a SymbolKind.
func (sk SymbolKind) String() string {
	switch sk {
	case SymbolKindFile:
		return "File"
	case SymbolKindModule:
		return "Module"
	case SymbolKindNamespace:
		return "Namespace"
	case SymbolKindPackage:
		return "Package"
	case SymbolKindClass:
		return "Class"
	case SymbolKindMethod:
		return "Method"
	case SymbolKindProperty:
		return "Property"
	case SymbolKindField:
		return "Field"
	case SymbolKindConstructor:
		return "Constructor"
	case SymbolKindEnum:
		return "Enum"
	case SymbolKindInterface:
		return "Interface"
	case SymbolKindFunction:
		return "Function"
	case SymbolKindVariable:
		return "Variable"
	case SymbolKindConstant:
		return "Constant"
	case SymbolKindString:
		return "String"
	case SymbolKindNumber:
		return "Number"
	case SymbolKindBoolean:
		return "Boolean"
	case SymbolKindArray:
		return "Array"
	case SymbolKindObject:
		return "Object"
	case SymbolKindStruct:
		return "Struct"
	default:
		return "Unknown"
	}
}

// Diagnostic represents a code diagnostic.
type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Code     string `json:"code,omitempty"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
}

// DiagnosticSeverity represents the severity of a diagnostic.
const (
	DiagnosticSeverityError       = 1
	DiagnosticSeverityWarning     = 2
	DiagnosticSeverityInformation = 3
	DiagnosticSeverityHint        = 4
)

// HoverResult represents the result of a hover request.
type HoverResult struct {
	Contents string `json:"contents"`
	Range    *Range `json:"range,omitempty"`
}

// JSONRPCRequest represents a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// InitializeParams represents the parameters for the initialize request.
type InitializeParams struct {
	ProcessID    int                `json:"processId"`
	RootURI      string             `json:"rootUri"`
	Capabilities ClientCapabilities `json:"capabilities"`
}

// ClientCapabilities represents the client's capabilities.
type ClientCapabilities struct {
	TextDocument TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Workspace    WorkspaceClientCapabilities    `json:"workspace,omitempty"`
}

// TextDocumentClientCapabilities represents text document capabilities.
type TextDocumentClientCapabilities struct {
	Hover          *HoverCapability          `json:"hover,omitempty"`
	DocumentSymbol *DocumentSymbolCapability `json:"documentSymbol,omitempty"`
}

// HoverCapability represents hover capabilities.
type HoverCapability struct {
	ContentFormat []string `json:"contentFormat,omitempty"`
}

// DocumentSymbolCapability represents document symbol capabilities.
type DocumentSymbolCapability struct {
	SymbolKind *SymbolKindCapability `json:"symbolKind,omitempty"`
}

// SymbolKindCapability represents symbol kind capabilities.
type SymbolKindCapability struct {
	ValueSet []SymbolKind `json:"valueSet,omitempty"`
}

// WorkspaceClientCapabilities represents workspace capabilities.
type WorkspaceClientCapabilities struct {
	Symbol *WorkspaceSymbolCapability `json:"symbol,omitempty"`
}

// WorkspaceSymbolCapability represents workspace symbol capabilities.
type WorkspaceSymbolCapability struct {
	SymbolKind *SymbolKindCapability `json:"symbolKind,omitempty"`
}

// TextDocumentIdentifier represents a text document identifier.
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// TextDocumentItem represents a text document item.
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// TextDocumentPositionParams represents parameters for position-based requests.
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// DocumentSymbolParams represents parameters for document symbol requests.
type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// WorkspaceSymbolParams represents parameters for workspace symbol requests.
type WorkspaceSymbolParams struct {
	Query string `json:"query"`
}

// DidOpenTextDocumentParams represents parameters for textDocument/didOpen.
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// SymbolInformation represents symbol information from the server.
type SymbolInformation struct {
	Name          string     `json:"name"`
	Kind          SymbolKind `json:"kind"`
	Location      Location   `json:"location"`
	ContainerName string     `json:"containerName,omitempty"`
}

// Location represents a location in a document.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// AllSymbolKinds returns all symbol kinds.
func AllSymbolKinds() []SymbolKind {
	return []SymbolKind{
		SymbolKindFile,
		SymbolKindModule,
		SymbolKindNamespace,
		SymbolKindPackage,
		SymbolKindClass,
		SymbolKindMethod,
		SymbolKindProperty,
		SymbolKindField,
		SymbolKindConstructor,
		SymbolKindEnum,
		SymbolKindInterface,
		SymbolKindFunction,
		SymbolKindVariable,
		SymbolKindConstant,
		SymbolKindString,
		SymbolKindNumber,
		SymbolKindBoolean,
		SymbolKindArray,
		SymbolKindObject,
		SymbolKindKey,
		SymbolKindNull,
		SymbolKindEnumMember,
		SymbolKindStruct,
		SymbolKindEvent,
		SymbolKindOperator,
		SymbolKindTypeParam,
	}
}
