// Package logging provides structured logging using zerolog.
package logging

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Logger is the global logger instance.
var Logger zerolog.Logger

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
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		Level:      InfoLevel,
		Output:     os.Stderr,
		Pretty:     false,
		TimeFormat: time.RFC3339,
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

	zerolog.TimeFieldFormat = cfg.TimeFormat

	var output io.Writer = cfg.Output
	if cfg.Pretty {
		output = zerolog.ConsoleWriter{
			Out:        cfg.Output,
			TimeFormat: cfg.TimeFormat,
		}
	}

	Logger = zerolog.New(output).
		Level(cfg.Level).
		With().
		Timestamp().
		Logger()
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
