package memsh

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

// TestJqBasicUsage tests basic jq functionality
func TestJqBasicUsage(t *testing.T) {
	tests := []struct {
		name     string
		setup    string
		script   string
		expected string
		wantErr  bool
	}{
		{
			name: "extract field from JSON",
			setup: `echo '{"name":"John","age":30}' > /data.json`,
			script: "jq '.name' /data.json",
			expected: `"John"`,
		},
		{
			name: "extract field with raw output",
			setup: `echo '{"name":"John","age":30}' > /data.json`,
			script: "jq -r '.name' /data.json",
			expected: "John",
		},
		{
			name: "extract number field",
			setup: `echo '{"name":"John","age":30}' > /data.json`,
			script: "jq '.age' /data.json",
			expected: "30",
		},
		{
			name: "jq nested field access",
			setup: `echo '{"user":{"id":123,"name":"Alice"}}' > /data.json`,
			script: "jq '.user.name' /data.json",
			expected: `"Alice"`,
		},
		{
			name: "array indexing",
			setup: `echo '{"items":["a","b","c"]}' > /data.json`,
			script: "jq '.items[1]' /data.json",
			expected: `"b"`,
		},
		{
			name: "array iteration",
			setup: `echo '["apple","banana","cherry"]' > /data.json`,
			script: "jq '.[]' /data.json",
			expected: `"apple"
"banana"
"cherry"`,
		},
		{
			name: "compact output",
			setup: `echo '{"name": "John", "age": 30}' > /data.json`,
			script: "jq -c '.' /data.json",
			expected: `{"age":30,"name":"John"}`,
		},
		{
			name: "select with filter",
			setup: `echo '[{"name":"John","age":30},{"name":"Jane","age":25}]' > /data.json`,
			script: `jq -c '.[] | select(.age > 26)' /data.json`,
			expected: `{"age":30,"name":"John"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			sh, err := NewShell(fs)
			if err != nil {
				t.Fatalf("NewShell() error = %v", err)
			}

			ctx := context.Background()
			if tt.setup != "" {
				if err := sh.Run(ctx, tt.setup); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			var stdout strings.Builder
			sh.SetIO(strings.NewReader(""), &stdout, &stdout)

			err = sh.Run(ctx, tt.script)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				output := strings.TrimSpace(stdout.String())
				if output != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, output)
				}
			}
		})
	}
}

// TestJqEdgeCases tests edge cases for jq
func TestJqEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    string
		script   string
		expected string
		wantErr  bool
	}{
		{
			name:    "missing filter",
			script:  "jq",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			setup:   `echo 'not json' > /data.json`,
			script:  "jq '.' /data.json",
			wantErr: true,
		},
		{
			name:    "invalid filter",
			setup:   `echo '{}' > /data.json`,
			script:  "jq 'invalid syntax' /data.json",
			wantErr: true,
		},
		{
			name:     "empty object",
			setup:    `echo '{}' > /data.json`,
			script:   "jq '.' /data.json",
			expected: "{}",
		},
		{
			name:     "empty array",
			setup:    `echo '[]' > /data.json`,
			script:   "jq '.' /data.json",
			expected: "[]",
		},
		{
			name:     "null value",
			setup:    `echo 'null' > /data.json`,
			script:   "jq '.' /data.json",
			expected: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			sh, err := NewShell(fs)
			if err != nil {
				t.Fatalf("NewShell() error = %v", err)
			}

			ctx := context.Background()
			if tt.setup != "" {
				if err := sh.Run(ctx, tt.setup); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			var stdout strings.Builder
			sh.SetIO(strings.NewReader(""), &stdout, &stdout)

			err = sh.Run(ctx, tt.script)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := strings.TrimSpace(stdout.String())
				if output != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, output)
				}
			}
		})
	}
}

// TestCurlBasicUsage tests basic curl functionality
func TestCurlBasicUsage(t *testing.T) {
	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok","data":"test"}`))
		case "/text":
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("Hello World"))
		case "/echo":
			// Echo back method and headers
			w.Header().Set("Content-Type", "application/json")
			auth := r.Header.Get("Authorization")
			body, _ := io.ReadAll(r.Body)
			response := map[string]string{
				"method": r.Method,
				"auth":   auth,
				"body":   string(body),
			}
			json.NewEncoder(w).Encode(response)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tests := []struct {
		name     string
		script   string
		contains string
		wantErr  bool
	}{
		{
			name:     "GET request",
			script:   "curl -s " + server.URL + "/text",
			contains: "Hello World",
		},
		{
			name:     "JSON response",
			script:   "curl -s " + server.URL + "/json",
			contains: `"status":"ok"`,
		},
		{
			name:     "POST with data",
			script:   `curl -s -X POST -d 'test data' ` + server.URL + "/echo",
			contains: "test data",
		},
		{
			name:     "custom header",
			script:   `curl -s -H 'Authorization: Bearer token123' ` + server.URL + "/echo",
			contains: "Bearer token123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			sh, err := NewShell(fs)
			if err != nil {
				t.Fatalf("NewShell() error = %v", err)
			}

			var stdout strings.Builder
			sh.SetIO(strings.NewReader(""), &stdout, &stdout)

			ctx := context.Background()
			err = sh.Run(ctx, tt.script)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				output := stdout.String()
				if !strings.Contains(output, tt.contains) {
					t.Errorf("Expected output to contain %q, got %q", tt.contains, output)
				}
			}
		})
	}
}

// TestCurlOutputToFile tests curl output redirection
func TestCurlOutputToFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	fs := afero.NewMemMapFs()
	sh, err := NewShell(fs)
	if err != nil {
		t.Fatalf("NewShell() error = %v", err)
	}

	var stdout strings.Builder
	sh.SetIO(strings.NewReader(""), &stdout, &stdout)

	ctx := context.Background()
	script := "curl -s -o /output.txt " + server.URL
	if err := sh.Run(ctx, script); err != nil {
		t.Fatalf("curl failed: %v", err)
	}

	// Check file was created
	content, err := afero.ReadFile(fs, "/output.txt")
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if string(content) != "test content" {
		t.Errorf("Expected 'test content', got %q", string(content))
	}
}

// TestCurlEdgeCases tests edge cases for curl
func TestCurlEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		wantErr bool
	}{
		{
			name:    "missing URL",
			script:  "curl",
			wantErr: true,
		},
		{
			name:    "invalid URL scheme",
			script:  "curl ://invalid",
			wantErr: true,
		},
		{
			name:    "missing flag argument",
			script:  "curl -X",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			sh, err := NewShell(fs)
			if err != nil {
				t.Fatalf("NewShell() error = %v", err)
			}

			var stdout strings.Builder
			sh.SetIO(strings.NewReader(""), &stdout, &stdout)

			ctx := context.Background()
			err = sh.Run(ctx, tt.script)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestJqAndCurlIntegration tests jq and curl working together
func TestJqAndCurlIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"users":[{"name":"Alice","age":30},{"name":"Bob","age":25}]}`))
	}))
	defer server.Close()

	fs := afero.NewMemMapFs()
	sh, err := NewShell(fs)
	if err != nil {
		t.Fatalf("NewShell() error = %v", err)
	}

	var stdout strings.Builder
	sh.SetIO(strings.NewReader(""), &stdout, &stdout)

	ctx := context.Background()

	// Fetch JSON and save to file
	script1 := "curl -s -o /data.json " + server.URL
	if err := sh.Run(ctx, script1); err != nil {
		t.Fatalf("curl failed: %v", err)
	}

	// Query with jq
	stdout.Reset()
	script2 := "jq -r '.users[0].name' /data.json"
	if err := sh.Run(ctx, script2); err != nil {
		t.Fatalf("jq failed: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output != "Alice" {
		t.Errorf("Expected 'Alice', got %q", output)
	}
}
