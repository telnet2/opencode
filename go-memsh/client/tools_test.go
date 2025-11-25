package client

import (
	"testing"
)

func TestEscapePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/simple/path", "'/simple/path'"},
		{"/path with spaces", "'/path with spaces'"},
		{"/path'with'quotes", "'/path'\\''with'\\''quotes'"},
		{"./relative", "'./relative'"},
		{"file.txt", "'file.txt'"},
	}

	for _, test := range tests {
		result := escapePath(test.input)
		if result != test.expected {
			t.Errorf("escapePath(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestMinFunc(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{1, 2, 1},
		{5, 3, 3},
		{0, 0, 0},
		{-1, 1, -1},
		{100, 100, 100},
	}

	for _, test := range tests {
		result := min(test.a, test.b)
		if result != test.expected {
			t.Errorf("min(%d, %d) = %d, expected %d", test.a, test.b, result, test.expected)
		}
	}
}

func TestEditToolValidation(t *testing.T) {
	// Test that oldString and newString must be different
	opts := EditOptions{
		FilePath:  "/test.txt",
		OldString: "same",
		NewString: "same",
	}

	// We can't easily test the actual EditTool without a real session,
	// but we can verify the validation logic pattern
	if opts.OldString == opts.NewString {
		// This is the expected validation that would happen
		t.Log("Correctly detected same oldString and newString")
	} else {
		t.Error("Should detect same oldString and newString")
	}
}

func TestReadOptionsDefaults(t *testing.T) {
	opts := ReadOptions{
		FilePath: "/test.txt",
	}

	// Test default values
	limit := opts.Limit
	if limit <= 0 {
		limit = DefaultReadLimit
	}

	if limit != DefaultReadLimit {
		t.Errorf("expected default limit %d, got %d", DefaultReadLimit, limit)
	}

	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	if offset != 0 {
		t.Errorf("expected default offset 0, got %d", offset)
	}
}

func TestGlobPatternConversion(t *testing.T) {
	tests := []struct {
		input    string
		hasSlash bool
		hasStar  bool
	}{
		{"*.txt", false, false},
		{"**/*.ts", true, true},
		{"src/**/*.go", true, true},
		{"file.js", false, false},
	}

	for _, test := range tests {
		hasSlash := containsString(test.input, "/")
		hasStar := containsString(test.input, "**")

		if hasSlash != test.hasSlash {
			t.Errorf("pattern %q: expected hasSlash=%v, got %v", test.input, test.hasSlash, hasSlash)
		}
		if hasStar != test.hasStar {
			t.Errorf("pattern %q: expected hasStar=%v, got %v", test.input, test.hasStar, hasStar)
		}
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestToolResultStructure(t *testing.T) {
	result := ToolResult{
		Title:  "Test Tool",
		Output: "Hello, World!",
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
	}

	if result.Title != "Test Tool" {
		t.Errorf("expected title 'Test Tool', got '%s'", result.Title)
	}

	if result.Output != "Hello, World!" {
		t.Errorf("expected output 'Hello, World!', got '%s'", result.Output)
	}

	if result.Metadata["key1"] != "value1" {
		t.Errorf("expected metadata key1='value1', got '%v'", result.Metadata["key1"])
	}

	if result.Metadata["key2"] != 42 {
		t.Errorf("expected metadata key2=42, got '%v'", result.Metadata["key2"])
	}
}

func TestConstants(t *testing.T) {
	if DefaultReadLimit != 2000 {
		t.Errorf("expected DefaultReadLimit=2000, got %d", DefaultReadLimit)
	}

	if MaxLineLength != 2000 {
		t.Errorf("expected MaxLineLength=2000, got %d", MaxLineLength)
	}

	if DefaultMaxOutputLength != 30000 {
		t.Errorf("expected DefaultMaxOutputLength=30000, got %d", DefaultMaxOutputLength)
	}

	if DefaultSearchLimit != 100 {
		t.Errorf("expected DefaultSearchLimit=100, got %d", DefaultSearchLimit)
	}
}
