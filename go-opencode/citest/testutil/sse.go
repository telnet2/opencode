package testutil

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// SSEClient provides SSE client utilities for testing
type SSEClient struct {
	BaseURL    string
	HTTPClient *http.Client

	mu       sync.Mutex
	events   []SSEEvent
	eventsCh chan SSEEvent
	errCh    chan error
	cancel   context.CancelFunc
	body     io.ReadCloser
}

// NewSSEClient creates a new SSE test client
func NewSSEClient(baseURL string) *SSEClient {
	return &SSEClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 0, // No timeout for SSE
		},
		eventsCh: make(chan SSEEvent, 100),
		errCh:    make(chan error, 1),
	}
}

// Connect starts the SSE connection
func (c *SSEClient) Connect(ctx context.Context, path string) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/event-stream") {
		resp.Body.Close()
		return fmt.Errorf("unexpected content type: %s", contentType)
	}

	c.body = resp.Body

	// Start reading events in background
	go c.readEvents(resp.Body)

	return nil
}

// readEvents reads SSE events from the connection
func (c *SSEClient) readEvents(body io.Reader) {
	defer func() {
		close(c.eventsCh)
		close(c.errCh)
	}()

	reader := bufio.NewReader(body)
	var eventType string
	var eventData strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF && err != context.Canceled {
				c.errCh <- err
			}
			return
		}

		line = strings.TrimRight(line, "\r\n")

		// Empty line = event complete
		if line == "" {
			if eventData.Len() > 0 {
				data := eventData.String()
				evt := SSEEvent{
					Type: eventType,
					Data: json.RawMessage(data),
				}

				c.mu.Lock()
				c.events = append(c.events, evt)
				c.mu.Unlock()

				select {
				case c.eventsCh <- evt:
				default:
					// Channel full, drop event
				}
			}
			eventType = ""
			eventData.Reset()
			continue
		}

		// Comment (heartbeat)
		if strings.HasPrefix(line, ":") {
			// Record heartbeat as special event
			evt := SSEEvent{Type: "heartbeat"}
			c.mu.Lock()
			c.events = append(c.events, evt)
			c.mu.Unlock()
			select {
			case c.eventsCh <- evt:
			default:
			}
			continue
		}

		// Parse field
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimSpace(data)
			eventData.WriteString(data)
		}
	}
}

// Events returns the event channel
func (c *SSEClient) Events() <-chan SSEEvent {
	return c.eventsCh
}

// Errors returns the error channel
func (c *SSEClient) Errors() <-chan error {
	return c.errCh
}

// WaitForEvent waits for a specific event type with timeout
func (c *SSEClient) WaitForEvent(eventType string, timeout time.Duration) (*SSEEvent, error) {
	deadline := time.After(timeout)
	for {
		select {
		case evt, ok := <-c.eventsCh:
			if !ok {
				return nil, fmt.Errorf("connection closed")
			}
			if evt.Type == eventType {
				return &evt, nil
			}
		case err := <-c.errCh:
			return nil, err
		case <-deadline:
			return nil, fmt.Errorf("timeout waiting for event: %s", eventType)
		}
	}
}

// WaitForHeartbeat waits for a heartbeat with timeout
func (c *SSEClient) WaitForHeartbeat(timeout time.Duration) error {
	_, err := c.WaitForEvent("heartbeat", timeout)
	return err
}

// WaitForAnyEvent waits for any event with timeout
func (c *SSEClient) WaitForAnyEvent(timeout time.Duration) (*SSEEvent, error) {
	deadline := time.After(timeout)
	select {
	case evt, ok := <-c.eventsCh:
		if !ok {
			return nil, fmt.Errorf("connection closed")
		}
		return &evt, nil
	case err := <-c.errCh:
		return nil, err
	case <-deadline:
		return nil, fmt.Errorf("timeout waiting for event")
	}
}

// CollectEvents collects events for a duration
func (c *SSEClient) CollectEvents(duration time.Duration) []SSEEvent {
	var collected []SSEEvent
	deadline := time.After(duration)
	for {
		select {
		case evt, ok := <-c.eventsCh:
			if !ok {
				return collected
			}
			collected = append(collected, evt)
		case <-deadline:
			return collected
		}
	}
}

// GetAllEvents returns all received events
func (c *SSEClient) GetAllEvents() []SSEEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]SSEEvent, len(c.events))
	copy(result, c.events)
	return result
}

// HasEventType checks if an event type was received
func (c *SSEClient) HasEventType(eventType string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, evt := range c.events {
		if evt.Type == eventType {
			return true
		}
	}
	return false
}

// CountEventType counts events of a specific type
func (c *SSEClient) CountEventType(eventType string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	count := 0
	for _, evt := range c.events {
		if evt.Type == eventType {
			count++
		}
	}
	return count
}

// Close closes the SSE connection
func (c *SSEClient) Close() {
	if c.cancel != nil {
		c.cancel()
	}
	if c.body != nil {
		c.body.Close()
	}
}

// ---- SSE Event Data Helpers ----

// SessionEventData represents session event data
type SessionEventData struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Directory string `json:"directory"`
}

// MessageEventData represents message event data
type MessageEventData struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	Role      string `json:"role"`
	Content   string `json:"content"`
}

// PartEventData represents part event data
type PartEventData struct {
	SessionID string `json:"sessionID"`
	MessageID string `json:"messageID"`
	PartIndex int    `json:"partIndex"`
	Delta     string `json:"delta,omitempty"`
}

// ParseSessionEvent parses session event data
func (evt *SSEEvent) ParseSessionEvent() (*SessionEventData, error) {
	var wrapper struct {
		Type string           `json:"type"`
		Data SessionEventData `json:"data"`
	}
	if err := json.Unmarshal(evt.Data, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Data, nil
}

// ParseMessageEvent parses message event data
func (evt *SSEEvent) ParseMessageEvent() (*MessageEventData, error) {
	var wrapper struct {
		Type string           `json:"type"`
		Data MessageEventData `json:"data"`
	}
	if err := json.Unmarshal(evt.Data, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Data, nil
}

// ParsePartEvent parses part event data
func (evt *SSEEvent) ParsePartEvent() (*PartEventData, error) {
	var wrapper struct {
		Type string        `json:"type"`
		Data PartEventData `json:"data"`
	}
	if err := json.Unmarshal(evt.Data, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Data, nil
}
