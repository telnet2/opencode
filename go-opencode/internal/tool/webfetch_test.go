package tool

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebFetchTool_Properties(t *testing.T) {
	tool := NewWebFetchTool("/tmp")

	if tool.ID() != "webfetch" {
		t.Errorf("Expected ID 'webfetch', got %q", tool.ID())
	}

	desc := tool.Description()
	if !strings.Contains(desc, "URL") {
		t.Error("Description should mention 'URL'")
	}

	params := tool.Parameters()
	if len(params) == 0 {
		t.Error("Parameters should not be empty")
	}

	// Verify JSON schema is valid
	var schema map[string]any
	if err := json.Unmarshal(params, &schema); err != nil {
		t.Errorf("Parameters should be valid JSON: %v", err)
	}

	// Check required properties
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Error("Schema should have properties")
	}
	if _, ok := props["url"]; !ok {
		t.Error("Schema should have url property")
	}
	if _, ok := props["format"]; !ok {
		t.Error("Schema should have format property")
	}
}

func TestWebFetchTool_URLValidation(t *testing.T) {
	tool := NewWebFetchTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{"valid https", "https://example.com", false, ""},
		{"valid http", "http://example.com", false, ""},
		{"missing protocol", "example.com", true, "http:// or https://"},
		{"ftp protocol", "ftp://example.com", true, "http:// or https://"},
		{"file protocol", "file:///etc/passwd", true, "http:// or https://"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock server for valid URLs
			if !tt.wantErr {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/plain")
					w.Write([]byte("test content"))
				}))
				defer server.Close()
				tt.url = server.URL
			}

			input := json.RawMessage(`{"url": "` + tt.url + `", "format": "text"}`)
			_, err := tool.Execute(ctx, input, toolCtx)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for URL %q", tt.url)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Error should contain %q, got: %v", tt.errMsg, err)
				}
			}
		})
	}
}

func TestWebFetchTool_FormatValidation(t *testing.T) {
	tool := NewWebFetchTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	tests := []struct {
		format  string
		wantErr bool
	}{
		{"text", false},
		{"markdown", false},
		{"html", false},
		{"json", true},
		{"xml", true},
		{"", true},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("test"))
	}))
	defer server.Close()

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			input := json.RawMessage(`{"url": "` + server.URL + `", "format": "` + tt.format + `"}`)
			_, err := tool.Execute(ctx, input, toolCtx)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error for format %q", tt.format)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error for format %q: %v", tt.format, err)
			}
		})
	}
}

func TestWebFetchTool_HTMLToMarkdown(t *testing.T) {
	tool := NewWebFetchTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	htmlContent := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1>Hello World</h1>
<p>This is a <strong>test</strong> paragraph.</p>
<ul>
<li>Item 1</li>
<li>Item 2</li>
</ul>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	input := json.RawMessage(`{"url": "` + server.URL + `", "format": "markdown"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check markdown conversion
	if !strings.Contains(result.Output, "# Hello World") {
		t.Error("Output should contain markdown heading")
	}
	if !strings.Contains(result.Output, "**test**") {
		t.Error("Output should contain bold text")
	}
	if !strings.Contains(result.Output, "- Item 1") {
		t.Error("Output should contain list items")
	}
}

func TestWebFetchTool_HTMLToText(t *testing.T) {
	tool := NewWebFetchTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	htmlContent := `<!DOCTYPE html>
<html>
<head>
<title>Test</title>
<script>alert('bad');</script>
<style>body { color: red; }</style>
</head>
<body>
<h1>Hello World</h1>
<p>This is a test.</p>
<script>console.log('hidden');</script>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	input := json.RawMessage(`{"url": "` + server.URL + `", "format": "text"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check text extraction
	if !strings.Contains(result.Output, "Hello World") {
		t.Error("Output should contain heading text")
	}
	if !strings.Contains(result.Output, "This is a test") {
		t.Error("Output should contain paragraph text")
	}

	// Script content should be removed
	if strings.Contains(result.Output, "alert") {
		t.Error("Output should not contain script content")
	}
	if strings.Contains(result.Output, "console.log") {
		t.Error("Output should not contain script content")
	}
	if strings.Contains(result.Output, "color: red") {
		t.Error("Output should not contain style content")
	}
}

func TestWebFetchTool_HTMLPassthrough(t *testing.T) {
	tool := NewWebFetchTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	htmlContent := `<html><body><h1>Test</h1></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	input := json.RawMessage(`{"url": "` + server.URL + `", "format": "html"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// HTML format should return raw HTML
	if result.Output != htmlContent {
		t.Errorf("Expected raw HTML, got %q", result.Output)
	}
}

func TestWebFetchTool_PlainTextPassthrough(t *testing.T) {
	tool := NewWebFetchTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	plainContent := "This is plain text content."

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(plainContent))
	}))
	defer server.Close()

	// Test all formats with plain text - should all return as-is
	formats := []string{"text", "markdown", "html"}
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			input := json.RawMessage(`{"url": "` + server.URL + `", "format": "` + format + `"}`)
			result, err := tool.Execute(ctx, input, toolCtx)
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			if result.Output != plainContent {
				t.Errorf("Format %s: Expected plain text passthrough, got %q", format, result.Output)
			}
		})
	}
}

func TestWebFetchTool_HTTPError(t *testing.T) {
	tool := NewWebFetchTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	input := json.RawMessage(`{"url": "` + server.URL + `", "format": "text"}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("Error should mention status code, got: %v", err)
	}
}

func TestWebFetchTool_InvalidInput(t *testing.T) {
	tool := NewWebFetchTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	// Invalid JSON
	input := json.RawMessage(`{invalid json}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	if err == nil {
		t.Error("Expected error for invalid JSON input")
	}
}

func TestWebFetchTool_Timeout(t *testing.T) {
	tool := NewWebFetchTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("fast response"))
	}))
	defer server.Close()

	// Test with explicit timeout
	input := json.RawMessage(`{"url": "` + server.URL + `", "format": "text", "timeout": 5}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Output != "fast response" {
		t.Errorf("Expected 'fast response', got %q", result.Output)
	}
}

func TestWebFetchTool_ResultMetadata(t *testing.T) {
	tool := NewWebFetchTool("/tmp")
	ctx := context.Background()
	toolCtx := testContext()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<html><body>Test</body></html>"))
	}))
	defer server.Close()

	input := json.RawMessage(`{"url": "` + server.URL + `", "format": "text"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check title format
	if !strings.Contains(result.Title, server.URL) {
		t.Error("Title should contain URL")
	}
	if !strings.Contains(result.Title, "text/html") {
		t.Error("Title should contain content type")
	}
}

func TestWebFetchTool_EinoTool(t *testing.T) {
	tool := NewWebFetchTool("/tmp")
	einoTool := tool.EinoTool()

	if einoTool == nil {
		t.Error("EinoTool should not return nil")
	}

	info, err := einoTool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if info.Name != "webfetch" {
		t.Errorf("Expected name 'webfetch', got %q", info.Name)
	}
}

func TestExtractTextFromHTML(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		wantText string
		wantNot  []string
	}{
		{
			name:     "basic text",
			html:     "<html><body><p>Hello World</p></body></html>",
			wantText: "Hello World",
			wantNot:  []string{},
		},
		{
			name:     "skip script",
			html:     "<html><body><p>Text</p><script>alert('bad')</script></body></html>",
			wantText: "Text",
			wantNot:  []string{"alert", "bad"},
		},
		{
			name:     "skip style",
			html:     "<html><head><style>body{color:red}</style></head><body><p>Text</p></body></html>",
			wantText: "Text",
			wantNot:  []string{"color", "red"},
		},
		{
			name:     "skip noscript",
			html:     "<html><body><p>Text</p><noscript>Enable JS</noscript></body></html>",
			wantText: "Text",
			wantNot:  []string{"Enable JS"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractTextFromHTML(tt.html)
			if err != nil {
				t.Fatalf("extractTextFromHTML failed: %v", err)
			}

			if !strings.Contains(result, tt.wantText) {
				t.Errorf("Expected text %q not found in result: %q", tt.wantText, result)
			}

			for _, notWant := range tt.wantNot {
				if strings.Contains(result, notWant) {
					t.Errorf("Unexpected text %q found in result: %q", notWant, result)
				}
			}
		})
	}
}

func TestConvertHTMLToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		wantMD   []string
		wantNot  []string
	}{
		{
			name:    "heading",
			html:    "<h1>Title</h1>",
			wantMD:  []string{"# Title"},
			wantNot: []string{},
		},
		{
			name:    "bold",
			html:    "<p><strong>Bold</strong></p>",
			wantMD:  []string{"**Bold**"},
			wantNot: []string{},
		},
		{
			name:    "italic",
			html:    "<p><em>Italic</em></p>",
			wantMD:  []string{"*Italic*"},
			wantNot: []string{},
		},
		{
			name:    "list",
			html:    "<ul><li>Item 1</li><li>Item 2</li></ul>",
			wantMD:  []string{"- Item 1", "- Item 2"},
			wantNot: []string{},
		},
		{
			name:    "skip script",
			html:    "<p>Text</p><script>bad()</script>",
			wantMD:  []string{"Text"},
			wantNot: []string{"bad", "script"},
		},
		{
			name:    "horizontal rule",
			html:    "<p>Above</p><hr><p>Below</p>",
			wantMD:  []string{"---"},
			wantNot: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertHTMLToMarkdown(tt.html)
			if err != nil {
				t.Fatalf("convertHTMLToMarkdown failed: %v", err)
			}

			for _, want := range tt.wantMD {
				if !strings.Contains(result, want) {
					t.Errorf("Expected markdown %q not found in result: %q", want, result)
				}
			}

			for _, notWant := range tt.wantNot {
				if strings.Contains(result, notWant) {
					t.Errorf("Unexpected text %q found in result: %q", notWant, result)
				}
			}
		})
	}
}
