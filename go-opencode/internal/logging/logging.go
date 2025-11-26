// Package logging provides structured logging using zerolog.
package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Logger is the global logger instance.
var Logger zerolog.Logger

// logFile holds the current log file if logging to file.
var logFile *os.File

// Level represents log levels.
type Level = zerolog.Level

// Log levels exposed for convenience.
const (
	DebugLevel = zerolog.DebugLevel
	InfoLevel  = zerolog.InfoLevel
	WarnLevel  = zerolog.WarnLevel
	ErrorLevel = zerolog.ErrorLevel
	FatalLevel = zerolog.FatalLevel
)

// Config holds logger configuration.
type Config struct {
	// Level is the minimum log level to output.
	Level Level
	// Output is where logs are written. Defaults to os.Stderr.
	Output io.Writer
	// Pretty enables human-readable console output.
	Pretty bool
	// TimeFormat specifies the time format. Defaults to RFC3339.
	TimeFormat string
	// LogToFile enables logging to a timestamped file in /tmp.
	LogToFile bool
	// LogDir is the directory for log files. Defaults to /tmp.
	LogDir string
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		Level:      InfoLevel,
		Output:     os.Stderr,
		Pretty:     false,
		TimeFormat: time.RFC3339,
		LogToFile:  false,
		LogDir:     "/tmp",
	}
}

// Init initializes the global logger with the given configuration.
func Init(cfg Config) {
	if cfg.Output == nil {
		cfg.Output = os.Stderr
	}
	if cfg.TimeFormat == "" {
		cfg.TimeFormat = time.RFC3339
	}
	if cfg.LogDir == "" {
		cfg.LogDir = "/tmp"
	}

	zerolog.TimeFieldFormat = cfg.TimeFormat

	var writers []io.Writer

	// Console output
	var consoleOutput io.Writer = cfg.Output
	if cfg.Pretty {
		consoleOutput = zerolog.ConsoleWriter{
			Out:        cfg.Output,
			TimeFormat: cfg.TimeFormat,
		}
	}
	writers = append(writers, consoleOutput)

	// File output
	if cfg.LogToFile {
		// Close previous log file if any
		if logFile != nil {
			logFile.Close()
		}

		// Create timestamped log file
		timestamp := time.Now().Format("20060102-150405")
		logPath := filepath.Join(cfg.LogDir, fmt.Sprintf("opencode-%s.log", timestamp))

		var err error
		logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			writers = append(writers, logFile)
		}
	}

	// Create multi-writer
	var output io.Writer
	if len(writers) == 1 {
		output = writers[0]
	} else {
		output = zerolog.MultiLevelWriter(writers...)
	}

	Logger = zerolog.New(output).
		Level(cfg.Level).
		With().
		Timestamp().
		Logger()
}

// GetLogFilePath returns the current log file path, or empty string if not logging to file.
func GetLogFilePath() string {
	if logFile != nil {
		return logFile.Name()
	}
	return ""
}

// Close closes the log file if one is open.
func Close() {
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

// ParseLevel parses a log level string (case-insensitive).
// Supported values: DEBUG, INFO, WARN, ERROR, FATAL.
// Returns InfoLevel if the string is not recognized.
func ParseLevel(level string) Level {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "DEBUG":
		return DebugLevel
	case "INFO":
		return InfoLevel
	case "WARN", "WARNING":
		return WarnLevel
	case "ERROR":
		return ErrorLevel
	case "FATAL":
		return FatalLevel
	default:
		return InfoLevel
	}
}

// Debug starts a new debug level log message.
func Debug() *zerolog.Event {
	return Logger.Debug()
}

// Info starts a new info level log message.
func Info() *zerolog.Event {
	return Logger.Info()
}

// Warn starts a new warn level log message.
func Warn() *zerolog.Event {
	return Logger.Warn()
}

// Error starts a new error level log message.
func Error() *zerolog.Event {
	return Logger.Error()
}

// Fatal starts a new fatal level log message.
// Calling Msg or Send on the returned event will call os.Exit(1).
func Fatal() *zerolog.Event {
	return Logger.Fatal()
}

// With creates a child logger with the given fields.
func With() zerolog.Context {
	return Logger.With()
}

// init sets up a default logger so the package is usable without explicit initialization.
func init() {
	Init(DefaultConfig())
}
