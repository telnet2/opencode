package client

import (
	"fmt"
	"strings"
	"time"
)

// SessionOptions configures a session
type SessionOptions struct {
	// BaseURL is the server URL (e.g., "http://localhost:8080")
	BaseURL string
	// SessionID to connect to an existing session (optional)
	SessionID string
	// Timeout for operations (default: 30s)
	Timeout time.Duration
	// AutoReconnect enables automatic reconnection
	AutoReconnect bool
}

// Session represents an active shell session
type Session struct {
	client  *Client
	info    *SessionInfo
	options SessionOptions
}

// NewSession creates a new session or connects to an existing one
func NewSession(opts SessionOptions) (*Session, error) {
	clientOpts := ClientOptions{
		BaseURL:       opts.BaseURL,
		Timeout:       opts.Timeout,
		AutoReconnect: opts.AutoReconnect,
	}

	client := NewClient(clientOpts)

	session := &Session{
		client:  client,
		options: opts,
	}

	if err := session.Init(); err != nil {
		return nil, err
	}

	return session, nil
}

// Init initializes the session
func (s *Session) Init() error {
	if s.options.SessionID != "" {
		// Use existing session
		sessions, err := s.client.ListSessions()
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}

		for _, sess := range sessions {
			if sess.ID == s.options.SessionID {
				s.info = &sess
				break
			}
		}

		if s.info == nil {
			return fmt.Errorf("session not found: %s", s.options.SessionID)
		}
	} else {
		// Create new session
		info, err := s.client.CreateSession()
		if err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
		s.info = info
	}

	// Connect to WebSocket
	if err := s.client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	return nil
}

// ID returns the session ID
func (s *Session) ID() string {
	if s.info == nil {
		return ""
	}
	return s.info.ID
}

// Cwd returns the current working directory
func (s *Session) Cwd() string {
	if s.info == nil {
		return "/"
	}
	return s.info.Cwd
}

// Info returns the session info
func (s *Session) Info() *SessionInfo {
	return s.info
}

// Connected returns true if the session is connected
func (s *Session) Connected() bool {
	return s.client.State() == StateConnected
}

// Execute executes a shell command
func (s *Session) Execute(command string) (*ExecuteCommandResult, error) {
	if s.info == nil {
		return nil, fmt.Errorf("session not initialized")
	}

	result, err := s.client.ExecuteCommand(s.info.ID, command)
	if err != nil {
		return nil, err
	}

	// Update cwd
	if result.Cwd != "" {
		s.info.Cwd = result.Cwd
		s.info.LastUsed = time.Now()
	}

	return result, nil
}

// Run executes a command and returns the output as a string
func (s *Session) Run(command string) (string, error) {
	result, err := s.Execute(command)
	if err != nil {
		return "", err
	}

	if result.Error != "" {
		return "", fmt.Errorf("command error: %s", result.Error)
	}

	return strings.Join(result.Output, "\n"), nil
}

// RunSafe executes a command and returns output and error separately
func (s *Session) RunSafe(command string) (output string, cmdErr string, cwd string, err error) {
	result, err := s.Execute(command)
	if err != nil {
		return "", "", "", err
	}

	return strings.Join(result.Output, "\n"), result.Error, result.Cwd, nil
}

// Cd changes the working directory
func (s *Session) Cd(path string) (string, error) {
	result, err := s.Execute(fmt.Sprintf("cd %s", escapePath(path)))
	if err != nil {
		return "", err
	}

	if result.Error != "" {
		return "", fmt.Errorf("cd error: %s", result.Error)
	}

	return result.Cwd, nil
}

// Pwd returns the current working directory
func (s *Session) Pwd() (string, error) {
	output, err := s.Run("pwd")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// ReadFile reads a file's contents
func (s *Session) ReadFile(path string) (string, error) {
	return s.Run(fmt.Sprintf("cat %s", escapePath(path)))
}

// WriteFile writes content to a file
func (s *Session) WriteFile(path, content string) error {
	// Use heredoc to write content
	cmd := fmt.Sprintf("cat > %s << 'MEMSH_EOF'\n%s\nMEMSH_EOF", escapePath(path), content)
	_, err := s.Run(cmd)
	return err
}

// AppendFile appends content to a file
func (s *Session) AppendFile(path, content string) error {
	cmd := fmt.Sprintf("cat >> %s << 'MEMSH_EOF'\n%s\nMEMSH_EOF", escapePath(path), content)
	_, err := s.Run(cmd)
	return err
}

// Exists checks if a path exists
func (s *Session) Exists(path string) (bool, error) {
	output, _, _, err := s.RunSafe(fmt.Sprintf("test -e %s && echo exists", escapePath(path)))
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) == "exists", nil
}

// IsDirectory checks if a path is a directory
func (s *Session) IsDirectory(path string) (bool, error) {
	output, _, _, err := s.RunSafe(fmt.Sprintf("test -d %s && echo dir", escapePath(path)))
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) == "dir", nil
}

// IsFile checks if a path is a file
func (s *Session) IsFile(path string) (bool, error) {
	output, _, _, err := s.RunSafe(fmt.Sprintf("test -f %s && echo file", escapePath(path)))
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) == "file", nil
}

// Mkdir creates a directory
func (s *Session) Mkdir(path string, recursive bool) error {
	flags := ""
	if recursive {
		flags = "-p"
	}
	_, err := s.Run(fmt.Sprintf("mkdir %s %s", flags, escapePath(path)))
	return err
}

// Rm removes a file or directory
func (s *Session) Rm(path string, recursive, force bool) error {
	flags := ""
	if recursive {
		flags += "-r"
	}
	if force {
		flags += "f"
	}
	_, err := s.Run(fmt.Sprintf("rm %s %s", flags, escapePath(path)))
	return err
}

// Ls lists directory contents
func (s *Session) Ls(path string, all, long bool) ([]string, error) {
	flags := ""
	if all {
		flags += "-a"
	}
	if long {
		flags += "l"
	}

	targetPath := "."
	if path != "" {
		targetPath = escapePath(path)
	}

	output, err := s.Run(fmt.Sprintf("ls %s %s", flags, targetPath))
	if err != nil {
		return nil, err
	}

	lines := strings.Split(output, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			result = append(result, line)
		}
	}

	return result, nil
}

// Close closes the session
func (s *Session) Close(removeSession bool) error {
	if removeSession && s.info != nil {
		if err := s.client.RemoveSession(s.info.ID); err != nil {
			return err
		}
	}
	s.client.Close()
	return nil
}

// escapePath escapes a path for shell usage
func escapePath(path string) string {
	// Wrap in single quotes and escape single quotes within
	escaped := strings.ReplaceAll(path, "'", "'\\''")
	return "'" + escaped + "'"
}
