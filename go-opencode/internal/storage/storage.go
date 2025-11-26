// Package storage provides file-based JSON storage matching the TypeScript implementation.
package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	ErrNotFound = errors.New("not found")
)

// Storage provides file-based JSON storage.
type Storage struct {
	basePath string
	mu       sync.RWMutex
	locks    map[string]*FileLock
}

// New creates a new Storage instance.
func New(basePath string) *Storage {
	return &Storage{
		basePath: basePath,
		locks:    make(map[string]*FileLock),
	}
}

// pathToFile converts a path slice to a file path.
func (s *Storage) pathToFile(path []string) string {
	parts := append([]string{s.basePath}, path...)
	return filepath.Join(parts...) + ".json"
}

// pathToDir converts a path slice to a directory path.
func (s *Storage) pathToDir(path []string) string {
	parts := append([]string{s.basePath}, path...)
	return filepath.Join(parts...)
}

// Get retrieves a value from storage.
func (s *Storage) Get(ctx context.Context, path []string, v any) error {
	filePath := s.pathToFile(path)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

	return nil
}

// Put stores a value in storage with file locking.
func (s *Storage) Put(ctx context.Context, path []string, v any) error {
	filePath := s.pathToFile(path)

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Acquire lock
	lock := s.getLock(filePath)
	if err := lock.Lock(); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer lock.Unlock()

	// Marshal data
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	// Write to temp file first, then rename (atomic operation)
	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath) // Clean up temp file
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// Delete removes a value from storage.
func (s *Storage) Delete(ctx context.Context, path []string) error {
	filePath := s.pathToFile(path)

	// Acquire lock
	lock := s.getLock(filePath)
	if err := lock.Lock(); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer lock.Unlock()

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// List returns all items at a path.
func (s *Storage) List(ctx context.Context, path []string) ([]string, error) {
	dirPath := s.pathToDir(path)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var items []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			items = append(items, name)
		} else if strings.HasSuffix(name, ".json") {
			items = append(items, strings.TrimSuffix(name, ".json"))
		}
	}

	return items, nil
}

// Scan iterates over all items at a path.
func (s *Storage) Scan(ctx context.Context, path []string, fn func(key string, data json.RawMessage) error) error {
	dirPath := s.pathToDir(path)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to scan
		}
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}

		filePath := filepath.Join(dirPath, name)
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue // Skip files that can't be read
		}

		key := strings.TrimSuffix(name, ".json")
		if err := fn(key, json.RawMessage(data)); err != nil {
			return err
		}
	}

	return nil
}

// Exists checks if a path exists.
func (s *Storage) Exists(ctx context.Context, path []string) bool {
	filePath := s.pathToFile(path)
	_, err := os.Stat(filePath)
	return err == nil
}

// getLock returns a file lock for a path.
func (s *Storage) getLock(filePath string) *FileLock {
	s.mu.Lock()
	defer s.mu.Unlock()

	lock, ok := s.locks[filePath]
	if !ok {
		lock = NewFileLock(filePath)
		s.locks[filePath] = lock
	}

	return lock
}
