package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
)

// Transport interface for MCP communication.
type Transport interface {
	// Send sends a request and returns the response.
	Send(ctx context.Context, method string, params any) (json.RawMessage, error)
	// Notify sends a notification (no response expected).
	Notify(ctx context.Context, method string, params any) error
	// Close closes the transport.
	Close() error
}

// HTTPTransport implements MCP over HTTP.
type HTTPTransport struct {
	url     string
	headers map[string]string
	client  *http.Client
	nextID  int64
}

// NewHTTPTransport creates a new HTTP transport.
func NewHTTPTransport(url string, headers map[string]string) (*HTTPTransport, error) {
	if url == "" {
		return nil, fmt.Errorf("URL is required")
	}
	return &HTTPTransport{
		url:     url,
		headers: headers,
		client:  &http.Client{},
	}, nil
}

// Send sends a request over HTTP.
func (t *HTTPTransport) Send(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := atomic.AddInt64(&t.nextID, 1)

	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, fmt.Errorf("MCP error %d: %s", result.Error.Code, result.Error.Message)
	}

	return result.Result, nil
}

// Notify sends a notification over HTTP.
func (t *HTTPTransport) Notify(ctx context.Context, method string, params any) error {
	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

// Close closes the HTTP transport.
func (t *HTTPTransport) Close() error {
	return nil
}

// StdioTransport implements MCP over stdio.
type StdioTransport struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	mu      sync.Mutex
	nextID  int64
	pending map[int64]chan *JSONRPCResponse
	closed  bool
	closeMu sync.RWMutex
}

// NewStdioTransport creates a new stdio transport.
func NewStdioTransport(ctx context.Context, command []string, env map[string]string) (*StdioTransport, error) {
	if len(command) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	t := &StdioTransport{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewReader(stdout),
		pending: make(map[int64]chan *JSONRPCResponse),
	}

	// Start reading responses
	go t.readLoop()

	return t, nil
}

// readLoop reads responses from the server.
func (t *StdioTransport) readLoop() {
	for {
		t.closeMu.RLock()
		if t.closed {
			t.closeMu.RUnlock()
			return
		}
		t.closeMu.RUnlock()

		line, err := t.stdout.ReadBytes('\n')
		if err != nil {
			t.closeMu.Lock()
			t.closed = true
			// Close all pending channels
			t.mu.Lock()
			for _, ch := range t.pending {
				close(ch)
			}
			t.pending = make(map[int64]chan *JSONRPCResponse)
			t.mu.Unlock()
			t.closeMu.Unlock()
			return
		}

		var resp JSONRPCResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue // Skip invalid JSON
		}

		if resp.ID != 0 {
			t.mu.Lock()
			if ch, ok := t.pending[resp.ID]; ok {
				ch <- &resp
				delete(t.pending, resp.ID)
			}
			t.mu.Unlock()
		}
	}
}

// Send sends a request and waits for a response.
func (t *StdioTransport) Send(ctx context.Context, method string, params any) (json.RawMessage, error) {
	t.closeMu.RLock()
	if t.closed {
		t.closeMu.RUnlock()
		return nil, fmt.Errorf("connection closed")
	}
	t.closeMu.RUnlock()

	id := atomic.AddInt64(&t.nextID, 1)

	ch := make(chan *JSONRPCResponse, 1)
	t.mu.Lock()
	t.pending[id] = ch
	t.mu.Unlock()

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
	}
	if params != nil {
		req.Params = params
	}

	if err := t.writeMessage(req); err != nil {
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, err
	}

	// Wait for response
	select {
	case resp := <-ch:
		if resp == nil {
			return nil, fmt.Errorf("connection closed")
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("MCP error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	case <-ctx.Done():
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, ctx.Err()
	}
}

// Notify sends a notification (no response expected).
func (t *StdioTransport) Notify(ctx context.Context, method string, params any) error {
	t.closeMu.RLock()
	if t.closed {
		t.closeMu.RUnlock()
		return fmt.Errorf("connection closed")
	}
	t.closeMu.RUnlock()

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
	}
	if params != nil {
		req.Params = params
	}

	return t.writeMessage(req)
}

// writeMessage writes a JSON-RPC message.
func (t *StdioTransport) writeMessage(msg any) error {
	reqJSON, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Write newline-delimited JSON
	if _, err := t.stdin.Write(append(reqJSON, '\n')); err != nil {
		return err
	}

	return nil
}

// Close closes the stdio transport.
func (t *StdioTransport) Close() error {
	t.closeMu.Lock()
	t.closed = true
	t.closeMu.Unlock()

	t.stdin.Close()
	if t.cmd.Process != nil {
		return t.cmd.Process.Kill()
	}
	return nil
}

// IsClosed returns whether the transport is closed.
func (t *StdioTransport) IsClosed() bool {
	t.closeMu.RLock()
	defer t.closeMu.RUnlock()
	return t.closed
}
