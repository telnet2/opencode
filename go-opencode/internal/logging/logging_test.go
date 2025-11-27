package logging

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Level != InfoLevel {
		t.Errorf("expected Level to be InfoLevel, got %v", cfg.Level)
	}
	if cfg.Output != os.Stderr {
		t.Errorf("expected Output to be os.Stderr")
	}
	if cfg.Pretty != false {
		t.Errorf("expected Pretty to be false")
	}
	if cfg.TimeFormat != time.RFC3339 {
		t.Errorf("expected TimeFormat to be RFC3339, got %s", cfg.TimeFormat)
	}
	if cfg.LogToFile != false {
		t.Errorf("expected LogToFile to be false")
	}
	if cfg.LogDir != "/tmp" {
		t.Errorf("expected LogDir to be /tmp, got %s", cfg.LogDir)
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"DEBUG", DebugLevel},
		{"debug", DebugLevel},
		{"  DEBUG  ", DebugLevel},
		{"INFO", InfoLevel},
		{"info", InfoLevel},
		{"WARN", WarnLevel},
		{"warn", WarnLevel},
		{"WARNING", WarnLevel},
		{"warning", WarnLevel},
		{"ERROR", ErrorLevel},
		{"error", ErrorLevel},
		{"FATAL", FatalLevel},
		{"fatal", FatalLevel},
		{"unknown", InfoLevel},
		{"", InfoLevel},
		{"INVALID", InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("ParseLevel(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestInitWithDefaults(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  InfoLevel,
		Output: &buf,
		Pretty: false,
	}

	Init(cfg)

	Info().Msg("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("expected output to contain 'test message', got %s", output)
	}
	if !strings.Contains(output, "info") {
		t.Errorf("expected output to contain 'info' level, got %s", output)
	}
}

func TestInitWithPrettyOutput(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  InfoLevel,
		Output: &buf,
		Pretty: true,
	}

	Init(cfg)

	Info().Msg("pretty test")

	output := buf.String()
	if !strings.Contains(output, "pretty test") {
		t.Errorf("expected output to contain 'pretty test', got %s", output)
	}
}

func TestLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  WarnLevel,
		Output: &buf,
		Pretty: false,
	}

	Init(cfg)

	// These should NOT appear (below WarnLevel)
	Debug().Msg("debug message")
	Info().Msg("info message")

	// These should appear
	Warn().Msg("warn message")
	Error().Msg("error message")

	output := buf.String()

	if strings.Contains(output, "debug message") {
		t.Error("debug message should not appear when level is Warn")
	}
	if strings.Contains(output, "info message") {
		t.Error("info message should not appear when level is Warn")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("warn message should appear when level is Warn")
	}
	if !strings.Contains(output, "error message") {
		t.Error("error message should appear when level is Warn")
	}
}

func TestLogToFile(t *testing.T) {
	// Create temp directory for log files
	tempDir, err := os.MkdirTemp("", "logging-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := Config{
		Level:     InfoLevel,
		Output:    &bytes.Buffer{}, // Suppress console output
		LogToFile: true,
		LogDir:    tempDir,
	}

	Init(cfg)
	defer Close()

	Info().Msg("file log test")

	// Check log file was created
	logPath := GetLogFilePath()
	if logPath == "" {
		t.Fatal("expected log file path to be set")
	}

	// Verify file is in correct directory
	if !strings.HasPrefix(logPath, tempDir) {
		t.Errorf("log file path %s should be in %s", logPath, tempDir)
	}

	// Verify file name pattern
	fileName := filepath.Base(logPath)
	if !strings.HasPrefix(fileName, "opencode-") || !strings.HasSuffix(fileName, ".log") {
		t.Errorf("unexpected log file name: %s", fileName)
	}

	// Verify file contents
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !strings.Contains(string(content), "file log test") {
		t.Errorf("log file should contain 'file log test', got: %s", string(content))
	}
}

func TestClose(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logging-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := Config{
		Level:     InfoLevel,
		Output:    &bytes.Buffer{},
		LogToFile: true,
		LogDir:    tempDir,
	}

	Init(cfg)

	logPath := GetLogFilePath()
	if logPath == "" {
		t.Fatal("expected log file path before close")
	}

	Close()

	if GetLogFilePath() != "" {
		t.Error("expected empty log file path after close")
	}
}

func TestGetLogFilePathWhenNotLoggingToFile(t *testing.T) {
	cfg := Config{
		Level:     InfoLevel,
		Output:    &bytes.Buffer{},
		LogToFile: false,
	}

	Close() // Ensure no previous log file
	Init(cfg)

	if GetLogFilePath() != "" {
		t.Error("expected empty log file path when not logging to file")
	}
}

func TestWithContext(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  InfoLevel,
		Output: &buf,
		Pretty: false,
	}

	Init(cfg)

	childLogger := With().Str("component", "test").Logger()
	childLogger.Info().Msg("with context")

	output := buf.String()
	if !strings.Contains(output, "component") {
		t.Errorf("expected output to contain 'component' field, got %s", output)
	}
	if !strings.Contains(output, "test") {
		t.Errorf("expected output to contain 'test' value, got %s", output)
	}
}

func TestLogWithFields(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  InfoLevel,
		Output: &buf,
		Pretty: false,
	}

	Init(cfg)

	Info().
		Str("key", "value").
		Int("count", 42).
		Bool("enabled", true).
		Msg("message with fields")

	output := buf.String()
	if !strings.Contains(output, `"key":"value"`) {
		t.Errorf("expected output to contain key field, got %s", output)
	}
	if !strings.Contains(output, `"count":42`) {
		t.Errorf("expected output to contain count field, got %s", output)
	}
	if !strings.Contains(output, `"enabled":true`) {
		t.Errorf("expected output to contain enabled field, got %s", output)
	}
}

func TestInitWithNilOutput(t *testing.T) {
	// Should default to os.Stderr without panic
	cfg := Config{
		Level:  InfoLevel,
		Output: nil,
	}

	// This should not panic
	Init(cfg)
}

func TestInitWithEmptyTimeFormat(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:      InfoLevel,
		Output:     &buf,
		TimeFormat: "",
	}

	Init(cfg)
	Info().Msg("time format test")

	// Should still work, using default RFC3339
	output := buf.String()
	if !strings.Contains(output, "time format test") {
		t.Errorf("expected output to contain message, got %s", output)
	}
}

func TestInitWithEmptyLogDir(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:     InfoLevel,
		Output:    &buf,
		LogToFile: true,
		LogDir:    "",
	}

	Init(cfg)
	defer Close()

	// Should default to /tmp
	logPath := GetLogFilePath()
	if logPath != "" && !strings.HasPrefix(logPath, "/tmp") {
		t.Errorf("expected log path to start with /tmp, got %s", logPath)
	}
}

func TestReinitClosePreviousLogFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logging-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// First init
	cfg1 := Config{
		Level:     InfoLevel,
		Output:    &bytes.Buffer{},
		LogToFile: true,
		LogDir:    tempDir,
	}
	Init(cfg1)
	firstLogPath := GetLogFilePath()

	// Wait a bit to get different timestamp
	time.Sleep(time.Second)

	// Second init should close the first file
	cfg2 := Config{
		Level:     InfoLevel,
		Output:    &bytes.Buffer{},
		LogToFile: true,
		LogDir:    tempDir,
	}
	Init(cfg2)
	defer Close()

	secondLogPath := GetLogFilePath()

	// Paths should be different (different timestamps)
	if firstLogPath == secondLogPath {
		t.Error("expected different log paths on reinit")
	}

	// Both files should exist
	if _, err := os.Stat(firstLogPath); os.IsNotExist(err) {
		t.Errorf("first log file should still exist: %s", firstLogPath)
	}
	if _, err := os.Stat(secondLogPath); os.IsNotExist(err) {
		t.Errorf("second log file should exist: %s", secondLogPath)
	}
}

func TestDebugLevel(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  DebugLevel,
		Output: &buf,
	}

	Init(cfg)

	Debug().Msg("debug test")

	output := buf.String()
	if !strings.Contains(output, "debug test") {
		t.Errorf("expected debug message in output, got %s", output)
	}
}

func TestErrorLevel(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  InfoLevel,
		Output: &buf,
	}

	Init(cfg)

	Error().Err(os.ErrNotExist).Msg("error test")

	output := buf.String()
	if !strings.Contains(output, "error test") {
		t.Errorf("expected error message in output, got %s", output)
	}
	if !strings.Contains(output, "file does not exist") {
		t.Errorf("expected error details in output, got %s", output)
	}
}
