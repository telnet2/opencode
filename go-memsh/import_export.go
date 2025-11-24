package memsh

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// cmdImportFile implements the import-file command
// Usage: import-file <local-path> <memfs-path>
func (s *Shell) cmdImportFile(ctx context.Context, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("import-file: usage: import-file <local-path> <memfs-path>")
	}

	localPath := args[1]
	memfsPath := s.resolvePath(args[2])

	// Open local file
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("import-file: cannot open local file '%s': %v", localPath, err)
	}
	defer localFile.Close()

	// Check if it's a directory
	localInfo, err := localFile.Stat()
	if err != nil {
		return fmt.Errorf("import-file: cannot stat local file '%s': %v", localPath, err)
	}

	if localInfo.IsDir() {
		return fmt.Errorf("import-file: '%s' is a directory, use import-dir instead", localPath)
	}

	// Create directory structure in memfs if needed
	memfsDir := filepath.Dir(memfsPath)
	if err := s.fs.MkdirAll(memfsDir, 0755); err != nil {
		return fmt.Errorf("import-file: cannot create directory '%s': %v", memfsDir, err)
	}

	// Create memfs file
	memfsFile, err := s.fs.Create(memfsPath)
	if err != nil {
		return fmt.Errorf("import-file: cannot create memfs file '%s': %v", memfsPath, err)
	}
	defer memfsFile.Close()

	// Copy contents
	_, err = io.Copy(memfsFile, localFile)
	if err != nil {
		return fmt.Errorf("import-file: cannot copy file: %v", err)
	}

	fmt.Fprintf(s.stdout, "Imported '%s' to '%s'\n", localPath, memfsPath)
	return nil
}

// cmdImportDir implements the import-dir command
// Usage: import-dir <local-path> <memfs-path>
func (s *Shell) cmdImportDir(ctx context.Context, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("import-dir: usage: import-dir <local-path> <memfs-path>")
	}

	localPath := args[1]
	memfsPath := s.resolvePath(args[2])

	// Check if local path is a directory
	localInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("import-dir: cannot access local path '%s': %v", localPath, err)
	}

	if !localInfo.IsDir() {
		return fmt.Errorf("import-dir: '%s' is not a directory, use import-file instead", localPath)
	}

	// Import directory recursively
	err = s.importDirRecursive(localPath, memfsPath)
	if err != nil {
		return fmt.Errorf("import-dir: %v", err)
	}

	fmt.Fprintf(s.stdout, "Imported directory '%s' to '%s'\n", localPath, memfsPath)
	return nil
}

// importDirRecursive recursively imports a directory from local filesystem to memfs
func (s *Shell) importDirRecursive(localPath, memfsPath string) error {
	// Create directory in memfs
	err := s.fs.MkdirAll(memfsPath, 0755)
	if err != nil {
		return fmt.Errorf("cannot create directory '%s': %v", memfsPath, err)
	}

	// Read local directory
	entries, err := os.ReadDir(localPath)
	if err != nil {
		return fmt.Errorf("cannot read directory '%s': %v", localPath, err)
	}

	// Process each entry
	for _, entry := range entries {
		localEntryPath := filepath.Join(localPath, entry.Name())
		memfsEntryPath := filepath.Join(memfsPath, entry.Name())

		if entry.IsDir() {
			// Recursively import subdirectory
			err = s.importDirRecursive(localEntryPath, memfsEntryPath)
			if err != nil {
				return err
			}
		} else {
			// Import file
			localFile, err := os.Open(localEntryPath)
			if err != nil {
				return fmt.Errorf("cannot open file '%s': %v", localEntryPath, err)
			}

			memfsFile, err := s.fs.Create(memfsEntryPath)
			if err != nil {
				localFile.Close()
				return fmt.Errorf("cannot create file '%s': %v", memfsEntryPath, err)
			}

			_, err = io.Copy(memfsFile, localFile)
			localFile.Close()
			memfsFile.Close()

			if err != nil {
				return fmt.Errorf("cannot copy file '%s': %v", localEntryPath, err)
			}
		}
	}

	return nil
}

// cmdExportFile implements the export-file command
// Usage: export-file <memfs-path> <local-path>
func (s *Shell) cmdExportFile(ctx context.Context, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("export-file: usage: export-file <memfs-path> <local-path>")
	}

	memfsPath := s.resolvePath(args[1])
	localPath := args[2]

	// Check if memfs path is a file
	memfsInfo, err := s.fs.Stat(memfsPath)
	if err != nil {
		return fmt.Errorf("export-file: cannot access memfs path '%s': %v", memfsPath, err)
	}

	if memfsInfo.IsDir() {
		return fmt.Errorf("export-file: '%s' is a directory, use export-dir instead", memfsPath)
	}

	// Open memfs file
	memfsFile, err := s.fs.Open(memfsPath)
	if err != nil {
		return fmt.Errorf("export-file: cannot open memfs file '%s': %v", memfsPath, err)
	}
	defer memfsFile.Close()

	// Create directory structure in local filesystem if needed
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("export-file: cannot create local directory '%s': %v", localDir, err)
	}

	// Create local file
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("export-file: cannot create local file '%s': %v", localPath, err)
	}
	defer localFile.Close()

	// Copy contents
	_, err = io.Copy(localFile, memfsFile)
	if err != nil {
		return fmt.Errorf("export-file: cannot copy file: %v", err)
	}

	fmt.Fprintf(s.stdout, "Exported '%s' to '%s'\n", memfsPath, localPath)
	return nil
}

// cmdExportDir implements the export-dir command
// Usage: export-dir <memfs-path> <local-path>
func (s *Shell) cmdExportDir(ctx context.Context, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("export-dir: usage: export-dir <memfs-path> <local-path>")
	}

	memfsPath := s.resolvePath(args[1])
	localPath := args[2]

	// Check if memfs path is a directory
	memfsInfo, err := s.fs.Stat(memfsPath)
	if err != nil {
		return fmt.Errorf("export-dir: cannot access memfs path '%s': %v", memfsPath, err)
	}

	if !memfsInfo.IsDir() {
		return fmt.Errorf("export-dir: '%s' is not a directory, use export-file instead", memfsPath)
	}

	// Export directory recursively
	err = s.exportDirRecursive(memfsPath, localPath)
	if err != nil {
		return fmt.Errorf("export-dir: %v", err)
	}

	fmt.Fprintf(s.stdout, "Exported directory '%s' to '%s'\n", memfsPath, localPath)
	return nil
}

// exportDirRecursive recursively exports a directory from memfs to local filesystem
func (s *Shell) exportDirRecursive(memfsPath, localPath string) error {
	// Create local directory
	err := os.MkdirAll(localPath, 0755)
	if err != nil {
		return fmt.Errorf("cannot create local directory '%s': %v", localPath, err)
	}

	// Read memfs directory
	entries, err := afero.ReadDir(s.fs, memfsPath)
	if err != nil {
		return fmt.Errorf("cannot read memfs directory '%s': %v", memfsPath, err)
	}

	// Process each entry
	for _, entry := range entries {
		memfsEntryPath := filepath.Join(memfsPath, entry.Name())
		localEntryPath := filepath.Join(localPath, entry.Name())

		if entry.IsDir() {
			// Recursively export subdirectory
			err = s.exportDirRecursive(memfsEntryPath, localEntryPath)
			if err != nil {
				return err
			}
		} else {
			// Export file
			memfsFile, err := s.fs.Open(memfsEntryPath)
			if err != nil {
				return fmt.Errorf("cannot open memfs file '%s': %v", memfsEntryPath, err)
			}

			localFile, err := os.Create(localEntryPath)
			if err != nil {
				memfsFile.Close()
				return fmt.Errorf("cannot create local file '%s': %v", localEntryPath, err)
			}

			_, err = io.Copy(localFile, memfsFile)
			memfsFile.Close()
			localFile.Close()

			if err != nil {
				return fmt.Errorf("cannot copy file '%s': %v", memfsEntryPath, err)
			}
		}
	}

	return nil
}
