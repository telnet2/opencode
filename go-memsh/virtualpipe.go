package memsh

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// VirtualPipe represents a virtual pipe for process substitution
// It emulates a FIFO by storing data in memory and providing file-like access
type VirtualPipe struct {
	id       int
	buffer   *bytes.Buffer
	mu       sync.RWMutex
	closed   bool
	ready    chan struct{} // Signals when data is available or pipe is closed
	doneChan chan struct{} // Signals when command execution is complete
}

// NewVirtualPipe creates a new virtual pipe
func NewVirtualPipe(id int) *VirtualPipe {
	return &VirtualPipe{
		id:       id,
		buffer:   &bytes.Buffer{},
		ready:    make(chan struct{}),
		doneChan: make(chan struct{}),
	}
}

// Write writes data to the pipe buffer (called by the executing command)
func (vp *VirtualPipe) Write(p []byte) (n int, err error) {
	vp.mu.Lock()
	defer vp.mu.Unlock()

	if vp.closed {
		return 0, io.ErrClosedPipe
	}

	n, err = vp.buffer.Write(p)

	// Signal that data is ready
	select {
	case <-vp.ready:
		// Already signaled
	default:
		close(vp.ready)
	}

	return n, err
}

// Read reads data from the pipe buffer (called by the consuming command)
func (vp *VirtualPipe) Read(p []byte) (n int, err error) {
	// Wait for data to be available or pipe to close
	<-vp.ready

	vp.mu.RLock()
	defer vp.mu.RUnlock()

	if vp.buffer.Len() == 0 && vp.closed {
		return 0, io.EOF
	}

	return vp.buffer.Read(p)
}

// Close marks the pipe as closed
func (vp *VirtualPipe) Close() error {
	vp.mu.Lock()
	defer vp.mu.Unlock()

	if vp.closed {
		return nil
	}

	vp.closed = true

	// Signal ready in case any readers are waiting
	select {
	case <-vp.ready:
		// Already signaled
	default:
		close(vp.ready)
	}

	return nil
}

// Done signals that the command execution is complete
func (vp *VirtualPipe) Done() {
	select {
	case <-vp.doneChan:
		// Already done
	default:
		close(vp.doneChan)
	}
}

// Wait waits for the command execution to complete
func (vp *VirtualPipe) Wait() {
	<-vp.doneChan
}

// GetPath returns the virtual path for this pipe (e.g., /dev/fd/3)
func (vp *VirtualPipe) GetPath() string {
	return fmt.Sprintf("/dev/fd/%d", vp.id)
}

// GetContents returns all buffered contents (for reading the complete output)
func (vp *VirtualPipe) GetContents() []byte {
	vp.mu.RLock()
	defer vp.mu.RUnlock()
	return vp.buffer.Bytes()
}

// PipeManager manages virtual pipes for process substitution
type PipeManager struct {
	pipes   map[int]*VirtualPipe
	nextID  int
	mu      sync.Mutex
	timeout time.Duration
}

// NewPipeManager creates a new pipe manager
func NewPipeManager() *PipeManager {
	return &PipeManager{
		pipes:   make(map[int]*VirtualPipe),
		nextID:  3, // Start at 3 (0=stdin, 1=stdout, 2=stderr)
		timeout: 30 * time.Second,
	}
}

// CreatePipe creates a new virtual pipe and returns it
func (pm *PipeManager) CreatePipe() *VirtualPipe {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	id := pm.nextID
	pm.nextID++

	pipe := NewVirtualPipe(id)
	pm.pipes[id] = pipe

	return pipe
}

// GetPipe retrieves a pipe by ID
func (pm *PipeManager) GetPipe(id int) (*VirtualPipe, bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pipe, ok := pm.pipes[id]
	return pipe, ok
}

// ClosePipe closes and removes a pipe
func (pm *PipeManager) ClosePipe(id int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pipe, ok := pm.pipes[id]; ok {
		pipe.Close()
		delete(pm.pipes, id)
	}
}

// CloseAll closes all pipes
func (pm *PipeManager) CloseAll() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for id, pipe := range pm.pipes {
		pipe.Close()
		delete(pm.pipes, id)
	}
}

// VirtualFile represents a virtual file descriptor that reads from a pipe
type VirtualFile struct {
	pipe *VirtualPipe
	pos  int
}

// NewVirtualFile creates a file-like wrapper around a virtual pipe
func NewVirtualFile(pipe *VirtualPipe) *VirtualFile {
	return &VirtualFile{
		pipe: pipe,
		pos:  0,
	}
}

// Read implements io.Reader for VirtualFile
func (vf *VirtualFile) Read(p []byte) (n int, err error) {
	// Wait for the command to complete writing
	vf.pipe.Wait()

	// Get all contents
	contents := vf.pipe.GetContents()

	// Read from current position
	if vf.pos >= len(contents) {
		return 0, io.EOF
	}

	n = copy(p, contents[vf.pos:])
	vf.pos += n

	return n, nil
}

// Close closes the virtual file
func (vf *VirtualFile) Close() error {
	return nil // The pipe will be closed by the manager
}

// Stat returns minimal file info for the virtual file
func (vf *VirtualFile) Stat() (os.FileInfo, error) {
	return &VirtualFileInfo{
		name: vf.pipe.GetPath(),
		size: int64(len(vf.pipe.GetContents())),
	}, nil
}

// VirtualFileInfo implements os.FileInfo for virtual files
type VirtualFileInfo struct {
	name string
	size int64
}

func (vfi *VirtualFileInfo) Name() string       { return vfi.name }
func (vfi *VirtualFileInfo) Size() int64        { return vfi.size }
func (vfi *VirtualFileInfo) Mode() os.FileMode  { return 0444 | os.ModeNamedPipe }
func (vfi *VirtualFileInfo) ModTime() time.Time { return time.Now() }
func (vfi *VirtualFileInfo) IsDir() bool        { return false }
func (vfi *VirtualFileInfo) Sys() interface{}   { return nil }

// ProcessSubstitution represents a process substitution `<(command)` or `>(command)`
type ProcessSubstitution struct {
	Command string
	IsInput bool // true for <(cmd), false for >(cmd)
	Pipe    *VirtualPipe
}

// ExecuteInBackground executes the command and writes output to the pipe
func (ps *ProcessSubstitution) ExecuteInBackground(ctx context.Context, shell *Shell) error {
	defer ps.Pipe.Done()
	defer ps.Pipe.Close()

	// Create a new shell instance with the pipe as stdout
	// This avoids modifying the shared shell's stdout
	subShell, err := NewShell(shell.fs)
	if err != nil {
		return fmt.Errorf("failed to create sub-shell: %v", err)
	}

	// Copy environment and working directory
	subShell.env = shell.env
	subShell.cwd = shell.cwd
	subShell.pipeManager = shell.pipeManager

	// Set stdout to the pipe, use an empty stdin for process substitution
	// (the process substitution command shouldn't need stdin)
	var stdin io.Reader = shell.stdin
	if stdin == nil {
		stdin = strings.NewReader("")
	}
	subShell.SetIO(stdin, ps.Pipe, shell.stderr)

	// Execute the command
	err = subShell.Run(ctx, ps.Command)
	if err != nil {
		return fmt.Errorf("process substitution failed: %v", err)
	}

	return nil
}
