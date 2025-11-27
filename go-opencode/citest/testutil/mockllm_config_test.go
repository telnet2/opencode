package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMockLLMConfig(t *testing.T) {
	// Test loading the default config file
	configPath := filepath.Join("..", "config", "mockllm.yaml")
	config, err := LoadMockLLMConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify settings
	if !config.Settings.EnableStreaming {
		t.Error("Expected streaming to be enabled")
	}

	// Verify there are responses
	if len(config.Responses) == 0 {
		t.Error("Expected responses to be loaded")
	}

	// Verify there are tool rules
	if len(config.ToolRules) == 0 {
		t.Error("Expected tool rules to be loaded")
	}

	// Test FindMatchingResponse
	response, found := config.FindMatchingResponse("hello, world")
	if !found {
		t.Error("Expected to find matching response for 'hello, world'")
	}
	if response != "Hello, World!" {
		t.Errorf("Unexpected response: %s", response)
	}

	// Test FindMatchingToolRule
	toolRule := config.FindMatchingToolRule("echo hello world", []string{"bash", "read"})
	if toolRule == nil {
		t.Error("Expected to find matching tool rule")
	}
	if toolRule != nil && toolRule.Tool != "bash" {
		t.Errorf("Expected bash tool, got: %s", toolRule.Tool)
	}
}

func TestDefaultMockLLMConfig(t *testing.T) {
	config := DefaultMockLLMConfig()

	// Verify default config has expected structure
	if config.Settings.ChunkDelayMS != 5 {
		t.Errorf("Expected chunk delay of 5, got: %d", config.Settings.ChunkDelayMS)
	}

	if config.Defaults.Fallback == "" {
		t.Error("Expected fallback to be set")
	}

	// Test matching responses
	tests := []struct {
		prompt   string
		expected string
	}{
		{"hello, world", "Hello, World!"},
		{"2+2", "4"},
		{"2 + 2", "4"},
		{"remember 42", "OK"},
		{"hello there", "Hello! How can I help you today?"},
	}

	for _, tc := range tests {
		response, _ := config.FindMatchingResponse(tc.prompt)
		if response != tc.expected {
			t.Errorf("For prompt '%s': expected '%s', got '%s'", tc.prompt, tc.expected, response)
		}
	}
}

func TestMatchConfig(t *testing.T) {
	tests := []struct {
		name   string
		match  MatchConfig
		prompt string
		want   bool
	}{
		{"contains match", MatchConfig{Contains: "hello"}, "say hello world", true},
		{"contains no match", MatchConfig{Contains: "hello"}, "say hi world", false},
		{"exact match", MatchConfig{Exact: "hello"}, "hello", true},
		{"exact no match", MatchConfig{Exact: "hello"}, "HELLO", true}, // case-insensitive
		{"exact different", MatchConfig{Exact: "hello"}, "hello world", false},
		{"contains_all match", MatchConfig{ContainsAll: []string{"hello", "world"}}, "hello beautiful world", true},
		{"contains_all partial", MatchConfig{ContainsAll: []string{"hello", "world"}}, "hello there", false},
		{"contains_any match first", MatchConfig{ContainsAny: []string{"hello", "world"}}, "hello there", true},
		{"contains_any match second", MatchConfig{ContainsAny: []string{"hello", "world"}}, "world peace", true},
		{"contains_any no match", MatchConfig{ContainsAny: []string{"hello", "world"}}, "hi there", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.match.Matches(tc.prompt)
			if got != tc.want {
				t.Errorf("Matches(%q) = %v, want %v", tc.prompt, got, tc.want)
			}
		})
	}
}

func TestSaveMockLLMConfig(t *testing.T) {
	config := DefaultMockLLMConfig()

	// Save to temp file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	err := SaveMockLLMConfig(config, configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load it back
	loaded, err := LoadMockLLMConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	// Verify it matches
	if len(loaded.Responses) != len(config.Responses) {
		t.Errorf("Response count mismatch: got %d, want %d", len(loaded.Responses), len(config.Responses))
	}
}

func TestChunkSplitting(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		mode       string
		size       int
		maxChunks  int
		wantCount  int
		wantChunks []string
	}{
		{
			name:       "word mode default",
			content:    "Hello World",
			mode:       "word",
			wantCount:  2,
			wantChunks: []string{"Hello ", "World"},
		},
		{
			name:       "word mode with max chunks",
			content:    "one two three four",
			mode:       "word",
			maxChunks:  2,
			wantCount:  2,
			wantChunks: []string{"one ", "two three four"},
		},
		{
			name:       "char mode",
			content:    "Hello World",
			mode:       "char",
			size:       5,
			wantCount:  3,
			wantChunks: []string{"Hello", " Worl", "d"},
		},
		{
			name:       "char mode with max",
			content:    "abcdefghij",
			mode:       "char",
			size:       2,
			maxChunks:  3,
			wantCount:  3,
			wantChunks: []string{"ab", "cd", "efghij"},
		},
		{
			name:      "fixed mode 3 chunks",
			content:   "123456789",
			mode:      "fixed",
			maxChunks: 3,
			wantCount: 3,
		},
		{
			name:      "fixed mode 2 chunks",
			content:   "Hello World!",
			mode:      "fixed",
			maxChunks: 2,
			wantCount: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := DefaultMockLLMConfig()
			config.Settings.ChunkMode = tc.mode
			config.Settings.ChunkSize = tc.size
			config.Settings.MaxChunks = tc.maxChunks

			server := NewMockLLMServerWithConfig(config)
			defer server.Close()

			chunks := server.splitIntoChunks(tc.content)

			if len(chunks) != tc.wantCount {
				t.Errorf("chunk count: got %d, want %d", len(chunks), tc.wantCount)
			}

			if tc.wantChunks != nil {
				for i, want := range tc.wantChunks {
					if i >= len(chunks) {
						t.Errorf("missing chunk %d: want %q", i, want)
						continue
					}
					if chunks[i] != want {
						t.Errorf("chunk[%d]: got %q, want %q", i, chunks[i], want)
					}
				}
			}

			// Verify all content is preserved
			joined := ""
			for _, c := range chunks {
				joined += c
			}
			if joined != tc.content {
				t.Errorf("content not preserved: got %q, want %q", joined, tc.content)
			}
		})
	}
}
