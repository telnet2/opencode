package storage

import (
	"os"
	"sync"
	"syscall"
)

// FileLock provides file-based locking for concurrent access.
type FileLock struct {
	path string
	file *os.File
	mu   sync.Mutex
}

// NewFileLock creates a new file lock.
func NewFileLock(path string) *FileLock {
	return &FileLock{path: path}
}

// Lock acquires an exclusive lock on the file.
func (l *FileLock) Lock() error {
	l.mu.Lock()

	var err error
	l.file, err = os.OpenFile(l.path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		l.mu.Unlock()
		return err
	}

	// Use flock for exclusive lock
	if err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_EX); err != nil {
		l.file.Close()
		l.mu.Unlock()
		return err
	}

	return nil
}

// TryLock attempts to acquire the lock without blocking.
func (l *FileLock) TryLock() bool {
	if !l.mu.TryLock() {
		return false
	}

	var err error
	l.file, err = os.OpenFile(l.path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		l.mu.Unlock()
		return false
	}

	// Use flock with LOCK_NB for non-blocking
	if err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		l.file.Close()
		l.mu.Unlock()
		return false
	}

	return true
}

// Unlock releases the lock.
func (l *FileLock) Unlock() error {
	if l.file == nil {
		return nil
	}

	// Release flock
	syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)

	// Close and remove lock file
	l.file.Close()
	os.Remove(l.path + ".lock")

	l.file = nil
	l.mu.Unlock()

	return nil
}
