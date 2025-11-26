package memsh

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// COW filesystem management for the shell

// cowMount represents a mounted COW filesystem
type cowMount struct {
	fs         *CopyOnWriteFs
	mountPoint string
}

// CowManager manages COW filesystem mounts for a shell
type CowManager struct {
	mounts map[string]*cowMount // mountPoint -> mount
}

// NewCowManager creates a new COW manager
func NewCowManager() *CowManager {
	return &CowManager{
		mounts: make(map[string]*cowMount),
	}
}

// Mount mounts a COW filesystem at the given mount point
func (m *CowManager) Mount(mountPoint string, cowFs *CopyOnWriteFs) error {
	mountPoint = filepath.Clean(mountPoint)
	if !strings.HasPrefix(mountPoint, "/") {
		mountPoint = "/" + mountPoint
	}

	if _, exists := m.mounts[mountPoint]; exists {
		return fmt.Errorf("mount point %s already in use", mountPoint)
	}

	m.mounts[mountPoint] = &cowMount{
		fs:         cowFs,
		mountPoint: mountPoint,
	}
	return nil
}

// Unmount unmounts a COW filesystem
func (m *CowManager) Unmount(mountPoint string) error {
	mountPoint = filepath.Clean(mountPoint)
	if !strings.HasPrefix(mountPoint, "/") {
		mountPoint = "/" + mountPoint
	}

	mount, exists := m.mounts[mountPoint]
	if !exists {
		return fmt.Errorf("no mount at %s", mountPoint)
	}

	// Check for dirty state and warn
	if mount.fs.IsDirty() {
		return fmt.Errorf("mount at %s has uncommitted changes; use cow-flush first or cow-unmount -f to force", mountPoint)
	}

	delete(m.mounts, mountPoint)
	return nil
}

// ForceUnmount unmounts a COW filesystem, discarding any uncommitted changes
func (m *CowManager) ForceUnmount(mountPoint string) error {
	mountPoint = filepath.Clean(mountPoint)
	if !strings.HasPrefix(mountPoint, "/") {
		mountPoint = "/" + mountPoint
	}

	if _, exists := m.mounts[mountPoint]; !exists {
		return fmt.Errorf("no mount at %s", mountPoint)
	}

	delete(m.mounts, mountPoint)
	return nil
}

// Get returns the COW filesystem at the given mount point
func (m *CowManager) Get(mountPoint string) (*CopyOnWriteFs, bool) {
	mountPoint = filepath.Clean(mountPoint)
	if !strings.HasPrefix(mountPoint, "/") {
		mountPoint = "/" + mountPoint
	}

	mount, exists := m.mounts[mountPoint]
	if !exists {
		return nil, false
	}
	return mount.fs, true
}

// List returns all mount points
func (m *CowManager) List() []string {
	points := make([]string, 0, len(m.mounts))
	for mp := range m.mounts {
		points = append(points, mp)
	}
	return points
}

// FindMount finds which COW mount (if any) a path belongs to
func (m *CowManager) FindMount(path string) (*CopyOnWriteFs, string, bool) {
	path = filepath.Clean(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	var bestMatch *cowMount
	var bestLen int

	for mp, mount := range m.mounts {
		if strings.HasPrefix(path, mp) && len(mp) > bestLen {
			bestMatch = mount
			bestLen = len(mp)
		}
	}

	if bestMatch != nil {
		relativePath := strings.TrimPrefix(path, bestMatch.mountPoint)
		if relativePath == "" {
			relativePath = "/"
		}
		return bestMatch.fs, relativePath, true
	}

	return nil, "", false
}

// Shell command implementations

// cmdCowMount implements the cow-mount command
// Usage: cow-mount <local-path> [mount-point]
// Mounts a local directory as a copy-on-write filesystem
func (s *Shell) cmdCowMount(ctx context.Context, args []string) error {
	_, stdout, stderr := s.stdio(ctx)

	if len(args) < 2 {
		fmt.Fprintln(stderr, "Usage: cow-mount <local-path> [mount-point] [--lazy]")
		fmt.Fprintln(stderr, "")
		fmt.Fprintln(stderr, "Mounts a local directory as a copy-on-write filesystem.")
		fmt.Fprintln(stderr, "Changes are tracked in memory until explicitly flushed.")
		fmt.Fprintln(stderr, "")
		fmt.Fprintln(stderr, "Options:")
		fmt.Fprintln(stderr, "  --lazy    Load files on-demand instead of eagerly")
		fmt.Fprintln(stderr, "")
		fmt.Fprintln(stderr, "Examples:")
		fmt.Fprintln(stderr, "  cow-mount /path/to/project /project")
		fmt.Fprintln(stderr, "  cow-mount /path/to/data --lazy")
		return nil
	}

	localPath := args[1]
	mountPoint := "/"
	lazy := false

	// Parse remaining args
	for i := 2; i < len(args); i++ {
		if args[i] == "--lazy" {
			lazy = true
		} else if !strings.HasPrefix(args[i], "-") && mountPoint == "/" {
			mountPoint = args[i]
		}
	}

	// Resolve mount point
	mountPoint = s.resolvePath(mountPoint)

	// Create local backend
	backend, err := NewLocalBackend(localPath)
	if err != nil {
		return fmt.Errorf("cow-mount: %w", err)
	}

	// Create COW filesystem
	cowFs, err := NewCopyOnWriteFs(backend, CopyOnWriteConfig{
		MountPoint:  mountPoint,
		SyncDeletes: true,
		Lazy:        lazy,
	})
	if err != nil {
		return fmt.Errorf("cow-mount: failed to create COW filesystem: %w", err)
	}

	// Register with COW manager
	if err := s.cowManager.Mount(mountPoint, cowFs); err != nil {
		return fmt.Errorf("cow-mount: %w", err)
	}

	// Import files into the shell's filesystem at the mount point
	if !lazy {
		if err := s.syncCowToMemfs(cowFs, mountPoint); err != nil {
			s.cowManager.ForceUnmount(mountPoint)
			return fmt.Errorf("cow-mount: failed to sync files: %w", err)
		}
	}

	fmt.Fprintf(stdout, "Mounted %s at %s (lazy=%v)\n", backend.Name(), mountPoint, lazy)
	return nil
}

// syncCowToMemfs syncs files from a COW filesystem to the shell's memfs
func (s *Shell) syncCowToMemfs(cowFs *CopyOnWriteFs, mountPoint string) error {
	// Create mount point directory
	if err := s.fs.MkdirAll(mountPoint, 0755); err != nil {
		return err
	}

	// Walk the COW filesystem and copy to memfs
	return s.syncDirToMemfs(cowFs, "/", mountPoint)
}

func (s *Shell) syncDirToMemfs(cowFs *CopyOnWriteFs, cowPath, memfsPath string) error {
	// Read directory from COW overlay
	dir, err := cowFs.overlay.Open(cowPath)
	if err != nil {
		// Directory doesn't exist in overlay, might be lazy loading
		return nil
	}
	defer dir.Close()

	entries, err := dir.Readdir(-1)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		cowEntryPath := filepath.Join(cowPath, entry.Name())
		memfsEntryPath := filepath.Join(memfsPath, entry.Name())

		if entry.IsDir() {
			if err := s.fs.MkdirAll(memfsEntryPath, 0755); err != nil {
				return err
			}
			if err := s.syncDirToMemfs(cowFs, cowEntryPath, memfsEntryPath); err != nil {
				return err
			}
		} else {
			if err := s.copyFileToMemfs(cowFs, cowEntryPath, memfsEntryPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Shell) copyFileToMemfs(cowFs *CopyOnWriteFs, cowPath, memfsPath string) error {
	src, err := cowFs.Open(cowPath)
	if err != nil {
		return err
	}
	defer src.Close()

	// Ensure parent directory exists
	if err := s.fs.MkdirAll(filepath.Dir(memfsPath), 0755); err != nil {
		return err
	}

	dst, err := s.fs.Create(memfsPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

// cmdCowUnmount implements the cow-unmount command
// Usage: cow-unmount <mount-point> [-f]
func (s *Shell) cmdCowUnmount(ctx context.Context, args []string) error {
	_, stdout, _ := s.stdio(ctx)

	if len(args) < 2 {
		return fmt.Errorf("cow-unmount: usage: cow-unmount <mount-point> [-f]")
	}

	mountPoint := s.resolvePath(args[1])
	force := len(args) > 2 && args[2] == "-f"

	var err error
	if force {
		err = s.cowManager.ForceUnmount(mountPoint)
	} else {
		err = s.cowManager.Unmount(mountPoint)
	}

	if err != nil {
		return fmt.Errorf("cow-unmount: %w", err)
	}

	fmt.Fprintf(stdout, "Unmounted %s\n", mountPoint)
	return nil
}

// cmdCowStatus implements the cow-status command
// Usage: cow-status [mount-point] [--json]
func (s *Shell) cmdCowStatus(ctx context.Context, args []string) error {
	_, stdout, _ := s.stdio(ctx)

	mountPoint := ""
	jsonOutput := false

	for i := 1; i < len(args); i++ {
		if args[i] == "--json" {
			jsonOutput = true
		} else if !strings.HasPrefix(args[i], "-") {
			mountPoint = s.resolvePath(args[i])
		}
	}

	if mountPoint != "" {
		// Show status for specific mount
		cowFs, ok := s.cowManager.Get(mountPoint)
		if !ok {
			return fmt.Errorf("cow-status: no mount at %s", mountPoint)
		}

		status := cowFs.Status()
		if jsonOutput {
			data, _ := json.MarshalIndent(status, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			s.printCowStatus(stdout, status)
		}
		return nil
	}

	// Show status for all mounts
	mounts := s.cowManager.List()
	if len(mounts) == 0 {
		fmt.Fprintln(stdout, "No COW mounts active")
		return nil
	}

	var allStatuses []CowStatus
	for _, mp := range mounts {
		cowFs, _ := s.cowManager.Get(mp)
		allStatuses = append(allStatuses, cowFs.Status())
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(allStatuses, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		for i, status := range allStatuses {
			if i > 0 {
				fmt.Fprintln(stdout, "")
			}
			s.printCowStatus(stdout, status)
		}
	}

	return nil
}

func (s *Shell) printCowStatus(w io.Writer, status CowStatus) {
	fmt.Fprintf(w, "Mount: %s\n", status.MountPoint)
	fmt.Fprintf(w, "Backend: %s\n", status.Backend)
	if status.IsDirty {
		fmt.Fprintf(w, "Status: DIRTY (%d modified, %d deleted)\n",
			status.ModifiedCount, status.DeletedCount)
		if len(status.ModifiedFiles) > 0 {
			fmt.Fprintln(w, "Modified files:")
			for _, f := range status.ModifiedFiles {
				fmt.Fprintf(w, "  M %s\n", f)
			}
		}
		if len(status.DeletedFiles) > 0 {
			fmt.Fprintln(w, "Deleted files:")
			for _, f := range status.DeletedFiles {
				fmt.Fprintf(w, "  D %s\n", f)
			}
		}
	} else {
		fmt.Fprintln(w, "Status: CLEAN")
	}
}

// cmdCowFlush implements the cow-flush command
// Usage: cow-flush [mount-point] [--to <target-path>]
func (s *Shell) cmdCowFlush(ctx context.Context, args []string) error {
	_, stdout, stderr := s.stdio(ctx)

	if len(args) < 2 {
		fmt.Fprintln(stderr, "Usage: cow-flush <mount-point> [--to <target-path>]")
		fmt.Fprintln(stderr, "")
		fmt.Fprintln(stderr, "Flushes all changes from a COW mount back to storage.")
		fmt.Fprintln(stderr, "Without --to, flushes to the original source directory.")
		fmt.Fprintln(stderr, "")
		fmt.Fprintln(stderr, "Examples:")
		fmt.Fprintln(stderr, "  cow-flush /project           # Flush to original location")
		fmt.Fprintln(stderr, "  cow-flush /project --to /backup/project")
		return nil
	}

	mountPoint := s.resolvePath(args[1])
	targetPath := ""

	for i := 2; i < len(args); i++ {
		if args[i] == "--to" && i+1 < len(args) {
			targetPath = args[i+1]
			i++
		}
	}

	cowFs, ok := s.cowManager.Get(mountPoint)
	if !ok {
		return fmt.Errorf("cow-flush: no mount at %s", mountPoint)
	}

	// First sync from memfs back to COW overlay
	if err := s.syncMemfsToCow(cowFs, mountPoint); err != nil {
		return fmt.Errorf("cow-flush: failed to sync from memfs: %w", err)
	}

	var err error
	if targetPath != "" {
		err = cowFs.FlushToLocal(targetPath)
		if err == nil {
			fmt.Fprintf(stdout, "Flushed %s to %s\n", mountPoint, targetPath)
		}
	} else {
		err = cowFs.Flush()
		if err == nil {
			fmt.Fprintf(stdout, "Flushed %s to %s\n", mountPoint, cowFs.Base().Name())
		}
	}

	return err
}

// syncMemfsToCow syncs changes from memfs back to the COW overlay
func (s *Shell) syncMemfsToCow(cowFs *CopyOnWriteFs, mountPoint string) error {
	// Walk the memfs at mount point and update COW overlay
	return s.walkAndSync(mountPoint, "/", cowFs)
}

func (s *Shell) walkAndSync(memfsPath, cowPath string, cowFs *CopyOnWriteFs) error {
	info, err := s.fs.Stat(memfsPath)
	if err != nil {
		return nil // Path doesn't exist in memfs
	}

	if !info.IsDir() {
		// Sync file
		return s.syncFileFromMemfs(memfsPath, cowPath, cowFs)
	}

	// Read memfs directory
	entries, err := aferoReadDir(s.fs, memfsPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		memfsEntryPath := filepath.Join(memfsPath, entry.Name())
		cowEntryPath := filepath.Join(cowPath, entry.Name())

		if err := s.walkAndSync(memfsEntryPath, cowEntryPath, cowFs); err != nil {
			return err
		}
	}

	return nil
}

func (s *Shell) syncFileFromMemfs(memfsPath, cowPath string, cowFs *CopyOnWriteFs) error {
	src, err := s.fs.Open(memfsPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := cowFs.Create(cowPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

// aferoReadDir is a helper to read directories from afero.Fs
func aferoReadDir(fs afero.Fs, path string) ([]FileInfo, error) {
	dir, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	return dir.Readdir(-1)
}

// cmdCowReset implements the cow-reset command
// Usage: cow-reset <mount-point>
func (s *Shell) cmdCowReset(ctx context.Context, args []string) error {
	_, stdout, _ := s.stdio(ctx)

	if len(args) < 2 {
		return fmt.Errorf("cow-reset: usage: cow-reset <mount-point>")
	}

	mountPoint := s.resolvePath(args[1])

	cowFs, ok := s.cowManager.Get(mountPoint)
	if !ok {
		return fmt.Errorf("cow-reset: no mount at %s", mountPoint)
	}

	if err := cowFs.Reset(); err != nil {
		return fmt.Errorf("cow-reset: %w", err)
	}

	// Re-sync to memfs
	if err := s.syncCowToMemfs(cowFs, mountPoint); err != nil {
		return fmt.Errorf("cow-reset: failed to re-sync: %w", err)
	}

	fmt.Fprintf(stdout, "Reset %s from %s\n", mountPoint, cowFs.Base().Name())
	return nil
}

// cmdCowDiff implements the cow-diff command
// Usage: cow-diff <mount-point> [path]
// Shows differences between the COW overlay and base
func (s *Shell) cmdCowDiff(ctx context.Context, args []string) error {
	_, stdout, _ := s.stdio(ctx)

	if len(args) < 2 {
		return fmt.Errorf("cow-diff: usage: cow-diff <mount-point> [path]")
	}

	mountPoint := s.resolvePath(args[1])
	filePath := ""
	if len(args) > 2 {
		filePath = args[2]
	}

	cowFs, ok := s.cowManager.Get(mountPoint)
	if !ok {
		return fmt.Errorf("cow-diff: no mount at %s", mountPoint)
	}

	status := cowFs.Status()

	if filePath != "" {
		// Show diff for specific file
		// Check if file is in modified list
		found := false
		for _, f := range status.ModifiedFiles {
			if f == filePath || filepath.Join(mountPoint, f) == filePath {
				found = true
				fmt.Fprintf(stdout, "--- base/%s\n", f)
				fmt.Fprintf(stdout, "+++ modified/%s\n", f)
				// Would show actual diff here if we had a diff library
				fmt.Fprintln(stdout, "(file content diff not implemented yet)")
				break
			}
		}
		if !found {
			for _, f := range status.DeletedFiles {
				if f == filePath || filepath.Join(mountPoint, f) == filePath {
					fmt.Fprintf(stdout, "--- base/%s\n", f)
					fmt.Fprintln(stdout, "(deleted)")
					found = true
					break
				}
			}
		}
		if !found {
			fmt.Fprintf(stdout, "File %s has no changes\n", filePath)
		}
	} else {
		// Show summary
		if !status.IsDirty {
			fmt.Fprintln(stdout, "No changes")
			return nil
		}

		for _, f := range status.ModifiedFiles {
			fmt.Fprintf(stdout, "M %s\n", f)
		}
		for _, f := range status.DeletedFiles {
			fmt.Fprintf(stdout, "D %s\n", f)
		}
	}

	return nil
}

// FileInfo type alias for os.FileInfo
type FileInfo = os.FileInfo

// GetCowManager returns the shell's COW manager for external access
func (s *Shell) GetCowManager() *CowManager {
	return s.cowManager
}
