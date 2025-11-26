package memsh

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/afero"
)

// StorageBackend defines the interface for a storage backend that can be used
// as the base layer for a copy-on-write filesystem or as a flush target.
type StorageBackend interface {
	// Name returns the backend's identifier (e.g., "local:/path", "s3://bucket")
	Name() string

	// Open opens a file for reading
	Open(path string) (io.ReadCloser, error)

	// Stat returns file info for the given path
	Stat(path string) (os.FileInfo, error)

	// ReadDir reads a directory and returns its entries
	ReadDir(path string) ([]os.FileInfo, error)

	// Create creates or truncates a file for writing
	Create(path string) (io.WriteCloser, error)

	// MkdirAll creates a directory path and all parents
	MkdirAll(path string, perm os.FileMode) error

	// Remove removes a file or empty directory
	Remove(path string) error

	// RemoveAll removes a path and all its children
	RemoveAll(path string) error

	// Exists checks if a path exists
	Exists(path string) (bool, error)
}

// LocalBackend implements StorageBackend using the local filesystem
type LocalBackend struct {
	basePath string
	fs       afero.Fs
}

// NewLocalBackend creates a new local filesystem backend rooted at basePath
func NewLocalBackend(basePath string) (*LocalBackend, error) {
	// Ensure the base path exists
	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("invalid base path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("cannot access base path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("base path is not a directory: %s", absPath)
	}

	return &LocalBackend{
		basePath: absPath,
		fs:       afero.NewBasePathFs(afero.NewOsFs(), absPath),
	}, nil
}

func (b *LocalBackend) Name() string {
	return "local:" + b.basePath
}

func (b *LocalBackend) resolvePath(path string) string {
	// Ensure path is relative and clean
	path = filepath.Clean(path)
	if filepath.IsAbs(path) {
		path = strings.TrimPrefix(path, "/")
	}
	return path
}

func (b *LocalBackend) Open(path string) (io.ReadCloser, error) {
	return b.fs.Open(b.resolvePath(path))
}

func (b *LocalBackend) Stat(path string) (os.FileInfo, error) {
	return b.fs.Stat(b.resolvePath(path))
}

func (b *LocalBackend) ReadDir(path string) ([]os.FileInfo, error) {
	return afero.ReadDir(b.fs, b.resolvePath(path))
}

func (b *LocalBackend) Create(path string) (io.WriteCloser, error) {
	resolved := b.resolvePath(path)
	// Ensure parent directory exists
	dir := filepath.Dir(resolved)
	if dir != "." && dir != "/" {
		if err := b.fs.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}
	return b.fs.Create(resolved)
}

func (b *LocalBackend) MkdirAll(path string, perm os.FileMode) error {
	return b.fs.MkdirAll(b.resolvePath(path), perm)
}

func (b *LocalBackend) Remove(path string) error {
	return b.fs.Remove(b.resolvePath(path))
}

func (b *LocalBackend) RemoveAll(path string) error {
	return b.fs.RemoveAll(b.resolvePath(path))
}

func (b *LocalBackend) Exists(path string) (bool, error) {
	return afero.Exists(b.fs, b.resolvePath(path))
}

// BasePath returns the absolute path this backend is rooted at
func (b *LocalBackend) BasePath() string {
	return b.basePath
}

// CopyOnWriteFs implements afero.Fs with copy-on-write semantics.
// It uses a memory overlay for modifications while reading from a base backend.
type CopyOnWriteFs struct {
	mu sync.RWMutex

	// Base layer (read-only source)
	base StorageBackend

	// Overlay layer (in-memory modifications)
	overlay afero.Fs

	// Tracking sets for dirty state
	modified map[string]struct{} // Files that were modified (including creates)
	deleted  map[string]struct{} // Files/dirs that were deleted

	// Mount point in the memfs namespace
	mountPoint string

	// Whether to sync deletes on flush
	syncDeletes bool
}

// CopyOnWriteConfig configures a copy-on-write filesystem
type CopyOnWriteConfig struct {
	// MountPoint is where the COW filesystem appears in the memfs namespace
	// If empty, defaults to "/"
	MountPoint string

	// SyncDeletes determines whether deleted files are removed from the target
	// during flush operations. Default is true.
	SyncDeletes bool

	// Lazy determines whether files are loaded on-demand (true) or eagerly (false)
	// Lazy loading is more memory efficient but may have higher latency on first access
	Lazy bool
}

// NewCopyOnWriteFs creates a new copy-on-write filesystem over the given backend
func NewCopyOnWriteFs(base StorageBackend, config CopyOnWriteConfig) (*CopyOnWriteFs, error) {
	if base == nil {
		return nil, fmt.Errorf("base backend cannot be nil")
	}

	mountPoint := config.MountPoint
	if mountPoint == "" {
		mountPoint = "/"
	}
	mountPoint = filepath.Clean(mountPoint)

	cow := &CopyOnWriteFs{
		base:        base,
		overlay:     afero.NewMemMapFs(),
		modified:    make(map[string]struct{}),
		deleted:     make(map[string]struct{}),
		mountPoint:  mountPoint,
		syncDeletes: config.SyncDeletes,
	}

	// If not lazy, load all files from base into overlay
	if !config.Lazy {
		if err := cow.loadFromBase(); err != nil {
			return nil, fmt.Errorf("failed to load from base: %w", err)
		}
	}

	return cow, nil
}

// loadFromBase recursively loads all files from the base backend into the overlay
func (c *CopyOnWriteFs) loadFromBase() error {
	return c.loadDirFromBase("/")
}

func (c *CopyOnWriteFs) loadDirFromBase(path string) error {
	entries, err := c.base.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())

		if entry.IsDir() {
			if err := c.overlay.MkdirAll(entryPath, entry.Mode()); err != nil {
				return err
			}
			if err := c.loadDirFromBase(entryPath); err != nil {
				return err
			}
		} else {
			if err := c.copyFileFromBase(entryPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *CopyOnWriteFs) copyFileFromBase(path string) error {
	src, err := c.base.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := c.overlay.MkdirAll(dir, 0755); err != nil {
		return err
	}

	dst, err := c.overlay.Create(path)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

// isInOverlay checks if a path exists in the overlay
func (c *CopyOnWriteFs) isInOverlay(path string) bool {
	_, err := c.overlay.Stat(path)
	return err == nil
}

// isDeleted checks if a path has been marked as deleted
func (c *CopyOnWriteFs) isDeleted(path string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, deleted := c.deleted[path]
	return deleted
}

// markModified marks a file as modified
func (c *CopyOnWriteFs) markModified(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.modified[path] = struct{}{}
	delete(c.deleted, path) // Undelete if previously deleted
}

// markDeleted marks a file as deleted
func (c *CopyOnWriteFs) markDeleted(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deleted[path] = struct{}{}
	delete(c.modified, path)
}

// ensureInOverlay copies a file from base to overlay if not already present
func (c *CopyOnWriteFs) ensureInOverlay(path string) error {
	if c.isInOverlay(path) {
		return nil
	}

	// Check if file exists in base
	info, err := c.base.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return c.overlay.MkdirAll(path, info.Mode())
	}

	return c.copyFileFromBase(path)
}

// Afero Fs interface implementation

func (c *CopyOnWriteFs) Name() string {
	return fmt.Sprintf("CopyOnWriteFs(%s)", c.base.Name())
}

func (c *CopyOnWriteFs) Create(name string) (afero.File, error) {
	if c.isDeleted(name) {
		// Undelete by creating
		c.mu.Lock()
		delete(c.deleted, name)
		c.mu.Unlock()
	}

	// Ensure parent directory exists
	dir := filepath.Dir(name)
	if dir != "." && dir != "/" {
		if err := c.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	f, err := c.overlay.Create(name)
	if err != nil {
		return nil, err
	}

	c.markModified(name)
	return &cowFile{File: f, cow: c, path: name}, nil
}

func (c *CopyOnWriteFs) Mkdir(name string, perm os.FileMode) error {
	if c.isDeleted(name) {
		c.mu.Lock()
		delete(c.deleted, name)
		c.mu.Unlock()
	}

	err := c.overlay.Mkdir(name, perm)
	if err != nil {
		return err
	}
	c.markModified(name)
	return nil
}

func (c *CopyOnWriteFs) MkdirAll(path string, perm os.FileMode) error {
	// Remove from deleted for this path and all parents
	c.mu.Lock()
	parts := strings.Split(filepath.Clean(path), string(filepath.Separator))
	current := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		if current == "" {
			current = "/"
		}
		delete(c.deleted, "/"+current)
	}
	c.mu.Unlock()

	err := c.overlay.MkdirAll(path, perm)
	if err != nil {
		return err
	}
	c.markModified(path)
	return nil
}

func (c *CopyOnWriteFs) Open(name string) (afero.File, error) {
	if c.isDeleted(name) {
		return nil, os.ErrNotExist
	}

	// Try overlay first
	if c.isInOverlay(name) {
		f, err := c.overlay.Open(name)
		if err != nil {
			return nil, err
		}
		return &cowFile{File: f, cow: c, path: name}, nil
	}

	// Try to copy from base (lazy loading)
	if err := c.ensureInOverlay(name); err != nil {
		return nil, err
	}

	f, err := c.overlay.Open(name)
	if err != nil {
		return nil, err
	}
	return &cowFile{File: f, cow: c, path: name}, nil
}

func (c *CopyOnWriteFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if c.isDeleted(name) {
		if flag&os.O_CREATE == 0 {
			return nil, os.ErrNotExist
		}
		// Creating a new file, undelete
		c.mu.Lock()
		delete(c.deleted, name)
		c.mu.Unlock()
	}

	// For write operations, ensure we have a copy
	if flag&(os.O_WRONLY|os.O_RDWR|os.O_CREATE|os.O_TRUNC) != 0 {
		// Ensure parent directory exists
		dir := filepath.Dir(name)
		if dir != "." && dir != "/" {
			if err := c.MkdirAll(dir, 0755); err != nil {
				return nil, err
			}
		}

		// Try to copy existing file from base if it exists and we're not truncating
		if !c.isInOverlay(name) && flag&os.O_CREATE != 0 && flag&os.O_TRUNC == 0 {
			_ = c.ensureInOverlay(name) // Ignore error, file may not exist
		}

		f, err := c.overlay.OpenFile(name, flag, perm)
		if err != nil {
			return nil, err
		}
		c.markModified(name)
		return &cowFile{File: f, cow: c, path: name}, nil
	}

	// Read-only access
	if !c.isInOverlay(name) {
		if err := c.ensureInOverlay(name); err != nil {
			return nil, err
		}
	}

	f, err := c.overlay.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	return &cowFile{File: f, cow: c, path: name}, nil
}

func (c *CopyOnWriteFs) Remove(name string) error {
	c.markDeleted(name)

	// Also remove from overlay if present
	if c.isInOverlay(name) {
		return c.overlay.Remove(name)
	}

	// Check if it exists in base
	if _, err := c.base.Stat(name); err != nil {
		return err
	}

	return nil
}

func (c *CopyOnWriteFs) RemoveAll(path string) error {
	c.markDeleted(path)

	// Remove from overlay if present
	if c.isInOverlay(path) {
		return c.overlay.RemoveAll(path)
	}

	// Check if it exists in base
	if _, err := c.base.Stat(path); err != nil {
		return err
	}

	return nil
}

func (c *CopyOnWriteFs) Rename(oldname, newname string) error {
	if c.isDeleted(oldname) {
		return os.ErrNotExist
	}

	// Ensure source is in overlay
	if !c.isInOverlay(oldname) {
		if err := c.ensureInOverlay(oldname); err != nil {
			return err
		}
	}

	err := c.overlay.Rename(oldname, newname)
	if err != nil {
		return err
	}

	c.markDeleted(oldname)
	c.markModified(newname)
	return nil
}

func (c *CopyOnWriteFs) Stat(name string) (os.FileInfo, error) {
	if c.isDeleted(name) {
		return nil, os.ErrNotExist
	}

	// Try overlay first
	if info, err := c.overlay.Stat(name); err == nil {
		return info, nil
	}

	// Try base
	return c.base.Stat(name)
}

func (c *CopyOnWriteFs) Chmod(name string, mode os.FileMode) error {
	if c.isDeleted(name) {
		return os.ErrNotExist
	}

	if !c.isInOverlay(name) {
		if err := c.ensureInOverlay(name); err != nil {
			return err
		}
	}

	err := c.overlay.Chmod(name, mode)
	if err != nil {
		return err
	}
	c.markModified(name)
	return nil
}

func (c *CopyOnWriteFs) Chown(name string, uid, gid int) error {
	if c.isDeleted(name) {
		return os.ErrNotExist
	}

	if !c.isInOverlay(name) {
		if err := c.ensureInOverlay(name); err != nil {
			return err
		}
	}

	err := c.overlay.Chown(name, uid, gid)
	if err != nil {
		return err
	}
	c.markModified(name)
	return nil
}

func (c *CopyOnWriteFs) Chtimes(name string, atime, mtime time.Time) error {
	if c.isDeleted(name) {
		return os.ErrNotExist
	}

	if !c.isInOverlay(name) {
		if err := c.ensureInOverlay(name); err != nil {
			return err
		}
	}

	err := c.overlay.Chtimes(name, atime, mtime)
	if err != nil {
		return err
	}
	c.markModified(name)
	return nil
}

// cowFile wraps an afero.File to track modifications
type cowFile struct {
	afero.File
	cow  *CopyOnWriteFs
	path string
}

func (f *cowFile) Write(p []byte) (n int, err error) {
	n, err = f.File.Write(p)
	if n > 0 {
		f.cow.markModified(f.path)
	}
	return
}

func (f *cowFile) WriteAt(p []byte, off int64) (n int, err error) {
	n, err = f.File.WriteAt(p, off)
	if n > 0 {
		f.cow.markModified(f.path)
	}
	return
}

func (f *cowFile) WriteString(s string) (n int, err error) {
	n, err = f.File.WriteString(s)
	if n > 0 {
		f.cow.markModified(f.path)
	}
	return
}

func (f *cowFile) Truncate(size int64) error {
	err := f.File.Truncate(size)
	if err == nil {
		f.cow.markModified(f.path)
	}
	return err
}

// COW-specific methods

// MountPoint returns the mount point of this COW filesystem
func (c *CopyOnWriteFs) MountPoint() string {
	return c.mountPoint
}

// Base returns the underlying storage backend
func (c *CopyOnWriteFs) Base() StorageBackend {
	return c.base
}

// IsDirty returns true if there are uncommitted changes
func (c *CopyOnWriteFs) IsDirty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.modified) > 0 || len(c.deleted) > 0
}

// DirtyFiles returns the list of modified file paths
func (c *CopyOnWriteFs) DirtyFiles() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	files := make([]string, 0, len(c.modified))
	for path := range c.modified {
		files = append(files, path)
	}
	sort.Strings(files)
	return files
}

// DeletedFiles returns the list of deleted file paths
func (c *CopyOnWriteFs) DeletedFiles() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	files := make([]string, 0, len(c.deleted))
	for path := range c.deleted {
		files = append(files, path)
	}
	sort.Strings(files)
	return files
}

// Status returns a summary of the current dirty state
type CowStatus struct {
	MountPoint    string   `json:"mount_point"`
	Backend       string   `json:"backend"`
	IsDirty       bool     `json:"is_dirty"`
	ModifiedCount int      `json:"modified_count"`
	DeletedCount  int      `json:"deleted_count"`
	ModifiedFiles []string `json:"modified_files,omitempty"`
	DeletedFiles  []string `json:"deleted_files,omitempty"`
}

func (c *CopyOnWriteFs) Status() CowStatus {
	return CowStatus{
		MountPoint:    c.mountPoint,
		Backend:       c.base.Name(),
		IsDirty:       c.IsDirty(),
		ModifiedCount: len(c.modified),
		DeletedCount:  len(c.deleted),
		ModifiedFiles: c.DirtyFiles(),
		DeletedFiles:  c.DeletedFiles(),
	}
}

// Flush writes all modifications back to the base backend
func (c *CopyOnWriteFs) Flush() error {
	return c.FlushTo(c.base)
}

// FlushTo writes all modifications to the specified backend
func (c *CopyOnWriteFs) FlushTo(target StorageBackend) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// First, handle deletions if syncing deletes
	if c.syncDeletes {
		for path := range c.deleted {
			if err := target.RemoveAll(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete %s: %w", path, err)
			}
		}
	}

	// Then, copy modified files
	for path := range c.modified {
		info, err := c.overlay.Stat(path)
		if err != nil {
			// File was modified then deleted, skip
			continue
		}

		if info.IsDir() {
			if err := target.MkdirAll(path, info.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", path, err)
			}
			continue
		}

		// Copy file contents
		src, err := c.overlay.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open %s from overlay: %w", path, err)
		}

		dst, err := target.Create(path)
		if err != nil {
			src.Close()
			return fmt.Errorf("failed to create %s in target: %w", path, err)
		}

		_, err = io.Copy(dst, src)
		src.Close()
		dst.Close()

		if err != nil {
			return fmt.Errorf("failed to copy %s: %w", path, err)
		}
	}

	// Clear tracking sets after successful flush
	c.modified = make(map[string]struct{})
	c.deleted = make(map[string]struct{})

	return nil
}

// FlushToLocal is a convenience method to flush to a local directory
func (c *CopyOnWriteFs) FlushToLocal(path string) error {
	backend, err := NewLocalBackend(path)
	if err != nil {
		return err
	}
	return c.FlushTo(backend)
}

// Reset discards all modifications and reloads from the base backend
func (c *CopyOnWriteFs) Reset() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.overlay = afero.NewMemMapFs()
	c.modified = make(map[string]struct{})
	c.deleted = make(map[string]struct{})

	return c.loadFromBase()
}

// Reload refreshes a specific file from the base backend, discarding local changes
func (c *CopyOnWriteFs) Reload(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove from tracking
	delete(c.modified, path)
	delete(c.deleted, path)

	// Remove from overlay
	if c.isInOverlay(path) {
		if err := c.overlay.RemoveAll(path); err != nil {
			return err
		}
	}

	// Copy from base
	info, err := c.base.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return c.loadDirFromBase(path)
	}
	return c.copyFileFromBase(path)
}
