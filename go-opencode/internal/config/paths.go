// Package config provides configuration loading and path management.
package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// Paths contains the standard paths for OpenCode data.
type Paths struct {
	Data   string // ~/.local/share/opencode
	Config string // ~/.config/opencode
	Cache  string // ~/.cache/opencode
	State  string // ~/.local/state/opencode
}

// GetPaths returns the standard paths for OpenCode data.
func GetPaths() *Paths {
	return &Paths{
		Data:   filepath.Join(getEnvOrDefault("XDG_DATA_HOME", defaultDataHome()), "opencode"),
		Config: filepath.Join(getEnvOrDefault("XDG_CONFIG_HOME", defaultConfigHome()), "opencode"),
		Cache:  filepath.Join(getEnvOrDefault("XDG_CACHE_HOME", defaultCacheHome()), "opencode"),
		State:  filepath.Join(getEnvOrDefault("XDG_STATE_HOME", defaultStateHome()), "opencode"),
	}
}

// EnsurePaths creates all required directories.
func (p *Paths) EnsurePaths() error {
	for _, dir := range []string{p.Data, p.Config, p.Cache, p.State} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// StoragePath returns the path to the storage directory.
func (p *Paths) StoragePath() string {
	return filepath.Join(p.Data, "storage")
}

// AuthPath returns the path to the auth file.
func (p *Paths) AuthPath() string {
	return filepath.Join(p.Data, "auth.json")
}

// getEnvOrDefault returns the environment variable value or a default.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func defaultDataHome() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("APPDATA")
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "share")
}

func defaultConfigHome() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("APPDATA")
	}
	return filepath.Join(os.Getenv("HOME"), ".config")
}

func defaultCacheHome() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "cache")
	}
	return filepath.Join(os.Getenv("HOME"), ".cache")
}

func defaultStateHome() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("APPDATA")
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "state")
}

// GlobalConfigPath returns the path to the global config file.
func GlobalConfigPath() string {
	return filepath.Join(GetPaths().Config, "opencode.json")
}

// ProjectConfigPath returns the path to the project config file.
func ProjectConfigPath(directory string) string {
	return filepath.Join(directory, ".opencode", "opencode.json")
}
