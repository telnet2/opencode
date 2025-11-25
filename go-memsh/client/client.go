// Package client provides a Go SDK for connecting to go-memsh service.
// It provides the same features as the TypeScript memsh-cli package.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// ClientOptions configures the memsh client
type ClientOptions struct {
	// BaseURL is the server URL (e.g., "http://localhost:8080")
	BaseURL string
	// Timeout for HTTP requests (default: 30s)
	Timeout time.Duration
	// AutoReconnect enables automatic WebSocket reconnection
	AutoReconnect bool
	// MaxReconnectAttempts limits reconnection attempts (default: 5)
	MaxReconnectAttempts int
	// ReconnectDelay is the initial delay between reconnection attempts (default: 1s)
	ReconnectDelay time.Duration
}

// ConnectionState represents the WebSocket connection state
type ConnectionState string

const (
	StateDisconnected ConnectionState = "disconnected"
	StateConnecting   ConnectionState = "connecting"
	StateConnected    ConnectionState = "connected"
	StateReconnecting ConnectionState = "reconnecting"
)

// Client is the main client for connecting to go-memsh service
type Client struct {
	options           ClientOptions
	httpClient        *http.Client
	ws                *websocket.Conn
	wsMu              sync.Mutex
	state             ConnectionState
	stateMu           sync.RWMutex
	requestID         int64
	pendingRequests   map[int64]chan *JSONRPCResponse
	pendingMu         sync.Mutex
	reconnectAttempts int
	done              chan struct{}
}

// NewClient creates a new memsh client
func NewClient(opts ClientOptions) *Client {
	// Apply defaults
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.MaxReconnectAttempts == 0 {
		opts.MaxReconnectAttempts = 5
	}
	if opts.ReconnectDelay == 0 {
		opts.ReconnectDelay = time.Second
	}

	return &Client{
		options: opts,
		httpClient: &http.Client{
			Timeout: opts.Timeout,
		},
		state:           StateDisconnected,
		pendingRequests: make(map[int64]chan *JSONRPCResponse),
		done:            make(chan struct{}),
	}
}

// State returns the current connection state
func (c *Client) State() ConnectionState {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state
}

func (c *Client) setState(state ConnectionState) {
	c.stateMu.Lock()
	c.state = state
	c.stateMu.Unlock()
}

// wsURL returns the WebSocket URL for REPL connection
func (c *Client) wsURL() (string, error) {
	u, err := url.Parse(c.options.BaseURL)
	if err != nil {
		return "", err
	}

	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}

	return fmt.Sprintf("%s://%s/api/v1/session/repl", scheme, u.Host), nil
}

// CreateSession creates a new shell session
func (c *Client) CreateSession() (*SessionInfo, error) {
	resp, err := c.httpClient.Post(
		c.options.BaseURL+"/api/v1/session/create",
		"application/json",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to create session: %s", resp.Status)
	}

	var result CreateSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result.Session, nil
}

// ListSessions lists all active sessions
func (c *Client) ListSessions() ([]SessionInfo, error) {
	resp, err := c.httpClient.Post(
		c.options.BaseURL+"/api/v1/session/list",
		"application/json",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list sessions: %s", resp.Status)
	}

	var result ListSessionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Sessions, nil
}

// RemoveSession removes a session by ID
func (c *Client) RemoveSession(sessionID string) error {
	reqBody := RemoveSessionRequest{SessionID: sessionID}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.options.BaseURL+"/api/v1/session/remove",
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		return fmt.Errorf("failed to remove session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to remove session: %s", resp.Status)
	}

	return nil
}

// Connect establishes a WebSocket connection for REPL
func (c *Client) Connect() error {
	state := c.State()
	if state == StateConnected || state == StateConnecting {
		return nil
	}

	c.setState(StateConnecting)

	wsURL, err := c.wsURL()
	if err != nil {
		c.setState(StateDisconnected)
		return err
	}

	c.wsMu.Lock()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		c.wsMu.Unlock()
		c.setState(StateDisconnected)
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	c.ws = conn
	c.wsMu.Unlock()

	c.setState(StateConnected)
	c.reconnectAttempts = 0

	// Start message reader
	go c.readMessages()

	return nil
}

// Disconnect closes the WebSocket connection
func (c *Client) Disconnect() {
	c.wsMu.Lock()
	if c.ws != nil {
		c.ws.Close()
		c.ws = nil
	}
	c.wsMu.Unlock()

	c.setState(StateDisconnected)
	c.clearPendingRequests(fmt.Errorf("client disconnected"))
}

// Execute executes a shell command
func (c *Client) Execute(params ExecuteCommandParams) (*ExecuteCommandResult, error) {
	if c.State() != StateConnected {
		if err := c.Connect(); err != nil {
			return nil, err
		}
	}

	return c.sendRequest("shell.execute", params)
}

// ExecuteCommand is a convenience method to execute a command string
func (c *Client) ExecuteCommand(sessionID, command string) (*ExecuteCommandResult, error) {
	return c.Execute(ExecuteCommandParams{
		SessionID: sessionID,
		Command:   command,
		Args:      nil,
	})
}

// sendRequest sends a JSON-RPC request and waits for response
func (c *Client) sendRequest(method string, params interface{}) (*ExecuteCommandResult, error) {
	id := atomic.AddInt64(&c.requestID, 1)

	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsBytes,
		ID:      id,
	}

	// Create response channel
	respChan := make(chan *JSONRPCResponse, 1)
	c.pendingMu.Lock()
	c.pendingRequests[id] = respChan
	c.pendingMu.Unlock()

	// Cleanup on exit
	defer func() {
		c.pendingMu.Lock()
		delete(c.pendingRequests, id)
		c.pendingMu.Unlock()
	}()

	// Send request
	c.wsMu.Lock()
	if c.ws == nil {
		c.wsMu.Unlock()
		return nil, fmt.Errorf("WebSocket not connected")
	}
	err = c.ws.WriteJSON(request)
	c.wsMu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Wait for response with timeout
	select {
	case resp := <-respChan:
		if resp.Error != nil {
			return nil, fmt.Errorf("JSON-RPC error [%d]: %s", resp.Error.Code, resp.Error.Message)
		}

		// Parse result
		var result ExecuteCommandResult
		resultBytes, err := json.Marshal(resp.Result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}
		if err := json.Unmarshal(resultBytes, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result: %w", err)
		}

		return &result, nil

	case <-time.After(c.options.Timeout):
		return nil, fmt.Errorf("request timeout")

	case <-c.done:
		return nil, fmt.Errorf("client closed")
	}
}

// readMessages reads incoming WebSocket messages
func (c *Client) readMessages() {
	for {
		c.wsMu.Lock()
		ws := c.ws
		c.wsMu.Unlock()

		if ws == nil {
			return
		}

		var response JSONRPCResponse
		err := ws.ReadJSON(&response)
		if err != nil {
			c.handleDisconnect()
			return
		}

		// Dispatch response to waiting request
		if response.ID != 0 {
			c.pendingMu.Lock()
			if ch, ok := c.pendingRequests[response.ID]; ok {
				ch <- &response
			}
			c.pendingMu.Unlock()
		}
	}
}

// handleDisconnect handles WebSocket disconnection
func (c *Client) handleDisconnect() {
	wasConnected := c.State() == StateConnected
	c.setState(StateDisconnected)

	c.wsMu.Lock()
	c.ws = nil
	c.wsMu.Unlock()

	c.clearPendingRequests(fmt.Errorf("connection lost"))

	// Attempt reconnection if enabled
	if wasConnected && c.options.AutoReconnect && c.reconnectAttempts < c.options.MaxReconnectAttempts {
		c.reconnectAttempts++
		c.setState(StateReconnecting)

		delay := c.options.ReconnectDelay * time.Duration(c.reconnectAttempts)
		time.Sleep(delay)

		if err := c.Connect(); err != nil {
			// Reconnection failed, will be handled by next attempt
		}
	}
}

// clearPendingRequests clears all pending requests with an error
func (c *Client) clearPendingRequests(err error) {
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()

	for _, ch := range c.pendingRequests {
		close(ch)
	}
	c.pendingRequests = make(map[int64]chan *JSONRPCResponse)
}

// Close closes the client and all connections
func (c *Client) Close() {
	close(c.done)
	c.Disconnect()
}
