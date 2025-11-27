package testutil

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// MockLLMConfig defines the YAML configuration schema for MockLLM scenarios.
type MockLLMConfig struct {
	Settings  MockSettings      `yaml:"settings"`
	Defaults  MockDefaults      `yaml:"defaults"`
	Responses []ResponseRule    `yaml:"responses"`
	ToolRules []ToolRule        `yaml:"tool_rules"`
}

// MockSettings configures MockLLM server behavior.
type MockSettings struct {
	LagMS           int  `yaml:"lag_ms"`            // Artificial delay in milliseconds
	EnableStreaming bool `yaml:"enable_streaming"`  // Whether to stream responses
	ChunkDelayMS    int  `yaml:"chunk_delay_ms"`    // Delay between streaming chunks
}

// MockDefaults defines fallback behavior.
type MockDefaults struct {
	Fallback string `yaml:"fallback"` // Response when no rules match
}

// ResponseRule defines a prompt-to-response mapping.
type ResponseRule struct {
	Name     string       `yaml:"name"`     // Optional rule name for debugging
	Match    MatchConfig  `yaml:"match"`    // How to match the prompt
	Response string       `yaml:"response"` // The response to return
	Priority int          `yaml:"priority"` // Higher priority rules are checked first
}

// MatchConfig defines how to match a prompt.
type MatchConfig struct {
	// Simple string matching (case-insensitive contains)
	Contains string `yaml:"contains"`

	// All strings must be present (case-insensitive)
	ContainsAll []string `yaml:"contains_all"`

	// Any string must be present (case-insensitive)
	ContainsAny []string `yaml:"contains_any"`

	// Exact match (case-insensitive)
	Exact string `yaml:"exact"`

	// Regex pattern
	Regex string `yaml:"regex"`
}

// ToolRule defines when to generate a tool call.
type ToolRule struct {
	Name      string          `yaml:"name"`       // Optional rule name
	Match     MatchConfig     `yaml:"match"`      // How to match the prompt
	Tool      string          `yaml:"tool"`       // Tool name (must be available in request)
	ToolCall  ToolCallConfig  `yaml:"tool_call"`  // Tool call configuration
	Response  string          `yaml:"response"`   // Optional text response alongside tool call
	Priority  int             `yaml:"priority"`   // Higher priority rules are checked first
}

// ToolCallConfig defines a tool call to generate.
type ToolCallConfig struct {
	ID        string            `yaml:"id"`        // Tool call ID (auto-generated if empty)
	Arguments map[string]string `yaml:"arguments"` // Tool arguments
}

// DefaultMockLLMConfig returns the default configuration with common scenarios.
func DefaultMockLLMConfig() *MockLLMConfig {
	return &MockLLMConfig{
		Settings: MockSettings{
			LagMS:           0,
			EnableStreaming: true,
			ChunkDelayMS:    5,
		},
		Defaults: MockDefaults{
			Fallback: "I understand your request. Let me help you with that.",
		},
		Responses: []ResponseRule{
			{
				Name:     "hello-world",
				Match:    MatchConfig{Contains: "hello, world"},
				Response: "Hello, World!",
				Priority: 10,
			},
			{
				Name:     "math-2plus2",
				Match:    MatchConfig{ContainsAny: []string{"2+2", "2 + 2"}},
				Response: "4",
				Priority: 10,
			},
			{
				Name:     "remember-42",
				Match:    MatchConfig{ContainsAll: []string{"remember", "42"}},
				Response: "OK",
				Priority: 10,
			},
			{
				Name:     "recall-number",
				Match:    MatchConfig{ContainsAll: []string{"what number", "remember"}},
				Response: "42",
				Priority: 10,
			},
			{
				Name:     "greet-alice",
				Match:    MatchConfig{ContainsAll: []string{"alice", "name"}},
				Response: "Nice to meet you, Alice",
				Priority: 5,
			},
			{
				Name:     "ask-name",
				Match:    MatchConfig{ContainsAll: []string{"what", "name"}},
				Response: "Alice",
				Priority: 4,
			},
			{
				Name:     "simple-hello",
				Match:    MatchConfig{Contains: "hello"},
				Response: "Hello! How can I help you today?",
				Priority: 1,
			},
		},
		ToolRules: []ToolRule{
			{
				Name:     "echo-hello-world",
				Match:    MatchConfig{Contains: "echo hello world"},
				Tool:     "bash",
				ToolCall: ToolCallConfig{
					ID:        "call_bash_001",
					Arguments: map[string]string{"command": "echo hello world"},
				},
				Response: "I'll run that bash command for you.",
				Priority: 10,
			},
		},
	}
}

// LoadMockLLMConfig loads configuration from a YAML file.
func LoadMockLLMConfig(path string) (*MockLLMConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config MockLLMConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// LoadMockLLMConfigFromDir looks for mockllm.yaml in the given directory.
func LoadMockLLMConfigFromDir(dir string) (*MockLLMConfig, error) {
	path := filepath.Join(dir, "mockllm.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Try mockllm.yml as alternative
		path = filepath.Join(dir, "mockllm.yml")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, err
		}
	}
	return LoadMockLLMConfig(path)
}

// SaveMockLLMConfig saves configuration to a YAML file.
func SaveMockLLMConfig(config *MockLLMConfig, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Matches checks if the prompt matches this rule.
func (m *MatchConfig) Matches(prompt string) bool {
	promptLower := strings.ToLower(prompt)

	// Exact match
	if m.Exact != "" {
		return strings.EqualFold(prompt, m.Exact)
	}

	// Contains single string
	if m.Contains != "" {
		return strings.Contains(promptLower, strings.ToLower(m.Contains))
	}

	// Contains all strings
	if len(m.ContainsAll) > 0 {
		for _, s := range m.ContainsAll {
			if !strings.Contains(promptLower, strings.ToLower(s)) {
				return false
			}
		}
		return true
	}

	// Contains any string
	if len(m.ContainsAny) > 0 {
		for _, s := range m.ContainsAny {
			if strings.Contains(promptLower, strings.ToLower(s)) {
				return true
			}
		}
		return false
	}

	// Regex matching (if needed in the future)
	// if m.Regex != "" { ... }

	return false
}

// FindMatchingResponse finds the first matching response rule for a prompt.
func (c *MockLLMConfig) FindMatchingResponse(prompt string) (string, bool) {
	// Sort by priority (higher first) - for simplicity, we assume they're pre-sorted
	// or we iterate and track the highest priority match
	var bestMatch *ResponseRule
	bestPriority := -1

	for i := range c.Responses {
		rule := &c.Responses[i]
		if rule.Match.Matches(prompt) {
			if rule.Priority > bestPriority {
				bestMatch = rule
				bestPriority = rule.Priority
			}
		}
	}

	if bestMatch != nil {
		return bestMatch.Response, true
	}

	return c.Defaults.Fallback, false
}

// FindMatchingToolRule finds a matching tool rule for a prompt and available tools.
func (c *MockLLMConfig) FindMatchingToolRule(prompt string, availableTools []string) *ToolRule {
	toolSet := make(map[string]bool)
	for _, t := range availableTools {
		toolSet[t] = true
	}

	var bestMatch *ToolRule
	bestPriority := -1

	for i := range c.ToolRules {
		rule := &c.ToolRules[i]
		// Check if the required tool is available
		if !toolSet[rule.Tool] {
			continue
		}
		if rule.Match.Matches(prompt) {
			if rule.Priority > bestPriority {
				bestMatch = &c.ToolRules[i]
				bestPriority = rule.Priority
			}
		}
	}

	return bestMatch
}
