// Package comparative provides comparative testing infrastructure.
package comparative_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// ComparativeHarness manages both TypeScript and Go servers for comparative testing.
type ComparativeHarness struct {
	MockLLM    *MockLLMServer
	TSServer   *TSServerManager
	GoServer   *GoServerManager
	WorkDir    string
	Config     *HarnessConfig
	mu         sync.Mutex
}

// HarnessConfig holds configuration for both servers.
type HarnessConfig struct {
	// MockLLM configuration
	MockResponses map[string]MockResponse

	// Shared configuration
	Model      string
	Provider   string
	WorkDir    string
	StateDir   string
	ConfigDir  string

	// Test settings
	Timeout    time.Duration
	EnableLogs bool
}

// DefaultHarnessConfig returns default configuration.
func DefaultHarnessConfig() *HarnessConfig {
	return &HarnessConfig{
		MockResponses: map[string]MockResponse{
			"hello": {Content: "Hello! How can I help you?"},
			"test":  {Content: "This is a test response."},
		},
		Model:    "mock/gpt-4",
		Provider: "mock",
		Timeout:  30 * time.Second,
	}
}

// TSServerManager manages the TypeScript server.
type TSServerManager struct {
	cmd     *exec.Cmd
	baseURL string
	port    int
	workDir string
	envVars []string
}

// GoServerManager manages the Go server.
type GoServerManager struct {
	cmd     *exec.Cmd
	baseURL string
	port    int
	workDir string
	envVars []string
}

// NewComparativeHarness creates a new comparative testing harness.
func NewComparativeHarness(config *HarnessConfig) (*ComparativeHarness, error) {
	// Create temp directories
	workDir, err := os.MkdirTemp("", "opencode-compare-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create work dir: %w", err)
	}

	if config.StateDir == "" {
		config.StateDir = filepath.Join(workDir, "state")
	}
	if config.ConfigDir == "" {
		config.ConfigDir = filepath.Join(workDir, "config")
	}
	if err := os.MkdirAll(config.StateDir, 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(config.ConfigDir, 0755); err != nil {
		return nil, err
	}

	// Create MockLLM server
	mockConfig := &MockLLMConfig{
		Responses: config.MockResponses,
		Defaults: MockDefaults{
			Fallback: "I understand your request.",
		},
		Settings: MockSettings{
			LagMS:           0,
			EnableStreaming: true,
		},
	}
	mockServer := NewMockLLMServer(mockConfig)

	return &ComparativeHarness{
		MockLLM: mockServer,
		WorkDir: workDir,
		Config:  config,
	}, nil
}

// Start launches both servers.
func (h *ComparativeHarness) Start(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Start Go server
	if err := h.startGoServer(ctx); err != nil {
		return fmt.Errorf("failed to start Go server: %w", err)
	}

	// Optionally start TypeScript server
	if os.Getenv("COMPARE_WITH_TS") == "true" {
		if err := h.startTSServer(ctx); err != nil {
			h.GoServer.Stop()
			return fmt.Errorf("failed to start TS server: %w", err)
		}
	}

	return nil
}

// Stop shuts down both servers.
func (h *ComparativeHarness) Stop() error {
	var errs []error

	if h.TSServer != nil {
		if err := h.TSServer.Stop(); err != nil {
			errs = append(errs, err)
		}
	}

	if h.GoServer != nil {
		if err := h.GoServer.Stop(); err != nil {
			errs = append(errs, err)
		}
	}

	if h.MockLLM != nil {
		h.MockLLM.Close()
	}

	if h.WorkDir != "" {
		os.RemoveAll(h.WorkDir)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors stopping servers: %v", errs)
	}
	return nil
}

// startGoServer starts the Go server with MockLLM backend.
func (h *ComparativeHarness) startGoServer(ctx context.Context) error {
	// For now, we'll use the test server infrastructure
	// In production, this would build and start the actual Go binary

	h.GoServer = &GoServerManager{
		workDir: h.WorkDir,
		envVars: []string{
			fmt.Sprintf("OPENAI_API_KEY=mock-key"),
			fmt.Sprintf("OPENAI_BASE_URL=%s/v1", h.MockLLM.URL()),
			fmt.Sprintf("OPENCODE_STATE_DIR=%s/go-state", h.WorkDir),
			fmt.Sprintf("OPENCODE_CONFIG_DIR=%s/go-config", h.WorkDir),
			fmt.Sprintf("OPENCODE_DIRECTORY=%s", h.WorkDir),
		},
	}

	// For testing, we use the testutil.StartTestServer approach
	// This is a placeholder showing the environment setup
	return nil
}

// startTSServer starts the TypeScript server with MockLLM backend.
func (h *ComparativeHarness) startTSServer(ctx context.Context) error {
	h.TSServer = &TSServerManager{
		workDir: h.WorkDir,
		envVars: []string{
			fmt.Sprintf("OPENAI_API_KEY=mock-key"),
			fmt.Sprintf("OPENAI_BASE_URL=%s/v1", h.MockLLM.URL()),
			fmt.Sprintf("OPENCODE_STATE_DIR=%s/ts-state", h.WorkDir),
			fmt.Sprintf("OPENCODE_CONFIG_DIR=%s/ts-config", h.WorkDir),
		},
	}

	// Start bun server
	// This would be: bun run start:server in packages/opencode
	return nil
}

// Stop stops the TypeScript server.
func (m *TSServerManager) Stop() error {
	if m.cmd != nil && m.cmd.Process != nil {
		return m.cmd.Process.Kill()
	}
	return nil
}

// Stop stops the Go server.
func (m *GoServerManager) Stop() error {
	if m.cmd != nil && m.cmd.Process != nil {
		return m.cmd.Process.Kill()
	}
	return nil
}

// URL returns the server URL.
func (m *GoServerManager) URL() string {
	return m.baseURL
}

// URL returns the server URL.
func (m *TSServerManager) URL() string {
	return m.baseURL
}

// DualClient makes parallel requests to both servers.
type DualClient struct {
	tsURL      string
	goURL      string
	httpClient *http.Client
}

// DualResponse contains responses from both servers.
type DualResponse struct {
	TS     *ServerResponse
	Go     *ServerResponse
	TSErr  error
	GoErr  error
	Timing DualTiming
}

// DualTiming captures latency for both servers.
type DualTiming struct {
	TSLatency time.Duration
	GoLatency time.Duration
}

// ServerResponse represents a response from a server.
type ServerResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// NewDualClient creates a new dual client.
func NewDualClient(tsURL, goURL string) *DualClient {
	return &DualClient{
		tsURL: tsURL,
		goURL: goURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Get performs parallel GET requests to both servers.
func (c *DualClient) Get(ctx context.Context, path string) *DualResponse {
	return c.do(ctx, http.MethodGet, path, nil)
}

// Post performs parallel POST requests to both servers.
func (c *DualClient) Post(ctx context.Context, path string, body interface{}) *DualResponse {
	return c.do(ctx, http.MethodPost, path, body)
}

// do performs the actual parallel requests.
func (c *DualClient) do(ctx context.Context, method, path string, body interface{}) *DualResponse {
	var wg sync.WaitGroup
	result := &DualResponse{}

	wg.Add(2)

	// TypeScript request
	go func() {
		defer wg.Done()
		if c.tsURL == "" {
			result.TSErr = fmt.Errorf("TS server not available")
			return
		}
		start := time.Now()
		result.TS, result.TSErr = c.request(ctx, c.tsURL+path, method, body)
		result.Timing.TSLatency = time.Since(start)
	}()

	// Go request
	go func() {
		defer wg.Done()
		if c.goURL == "" {
			result.GoErr = fmt.Errorf("Go server not available")
			return
		}
		start := time.Now()
		result.Go, result.GoErr = c.request(ctx, c.goURL+path, method, body)
		result.Timing.GoLatency = time.Since(start)
	}()

	wg.Wait()
	return result
}

// request performs a single HTTP request.
func (c *DualClient) request(ctx context.Context, url, method string, body interface{}) (*ServerResponse, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &ServerResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       respBody,
	}, nil
}

// CompareJSON compares two JSON responses.
func CompareJSON(ts, go_ []byte, tolerances *Tolerances) ([]Difference, error) {
	var tsData, goData interface{}

	if err := json.Unmarshal(ts, &tsData); err != nil {
		return nil, fmt.Errorf("failed to parse TS response: %w", err)
	}

	if err := json.Unmarshal(go_, &goData); err != nil {
		return nil, fmt.Errorf("failed to parse Go response: %w", err)
	}

	var diffs []Difference
	compareValues("$", tsData, goData, tolerances, &diffs)
	return diffs, nil
}

// Difference represents a difference between two values.
type Difference struct {
	Path     string
	Type     DiffType
	TSValue  interface{}
	GoValue  interface{}
	Severity Severity
}

// DiffType describes the type of difference.
type DiffType string

const (
	DiffTypeValueMismatch DiffType = "value_mismatch"
	DiffTypeMissingInTS   DiffType = "missing_in_ts"
	DiffTypeMissingInGo   DiffType = "missing_in_go"
	DiffTypeTypeMismatch  DiffType = "type_mismatch"
)

// Severity describes the importance of a difference.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

// Tolerances defines acceptable differences.
type Tolerances struct {
	IgnorePaths      []string
	KnownDifferences map[string]Severity
}

// DefaultTolerances returns sensible defaults.
func DefaultTolerances() *Tolerances {
	return &Tolerances{
		IgnorePaths: []string{
			"$.time.created",
			"$.time.updated",
			"$.id",
		},
		KnownDifferences: map[string]Severity{
			"$.version": SeverityInfo,
		},
	}
}

// ShouldIgnore checks if a path should be ignored.
func (t *Tolerances) ShouldIgnore(path string) bool {
	for _, p := range t.IgnorePaths {
		if p == path {
			return true
		}
	}
	return false
}

// GetSeverity returns the severity for a difference.
func (t *Tolerances) GetSeverity(path string, diffType DiffType) Severity {
	if sev, ok := t.KnownDifferences[path]; ok {
		return sev
	}

	switch diffType {
	case DiffTypeMissingInGo:
		return SeverityCritical
	case DiffTypeMissingInTS:
		return SeverityWarning
	case DiffTypeTypeMismatch:
		return SeverityCritical
	default:
		return SeverityWarning
	}
}

// compareValues recursively compares two values.
func compareValues(path string, ts, go_ interface{}, tolerances *Tolerances, diffs *[]Difference) {
	if tolerances != nil && tolerances.ShouldIgnore(path) {
		return
	}

	if ts == nil && go_ == nil {
		return
	}

	if ts == nil {
		*diffs = append(*diffs, Difference{
			Path:     path,
			Type:     DiffTypeMissingInTS,
			GoValue:  go_,
			Severity: tolerances.GetSeverity(path, DiffTypeMissingInTS),
		})
		return
	}

	if go_ == nil {
		*diffs = append(*diffs, Difference{
			Path:     path,
			Type:     DiffTypeMissingInGo,
			TSValue:  ts,
			Severity: tolerances.GetSeverity(path, DiffTypeMissingInGo),
		})
		return
	}

	// Compare maps
	tsMap, tsIsMap := ts.(map[string]interface{})
	goMap, goIsMap := go_.(map[string]interface{})
	if tsIsMap && goIsMap {
		compareObjects(path, tsMap, goMap, tolerances, diffs)
		return
	}

	// Compare slices
	tsSlice, tsIsSlice := ts.([]interface{})
	goSlice, goIsSlice := go_.([]interface{})
	if tsIsSlice && goIsSlice {
		compareArrays(path, tsSlice, goSlice, tolerances, diffs)
		return
	}

	// Compare primitives
	if ts != go_ {
		*diffs = append(*diffs, Difference{
			Path:     path,
			Type:     DiffTypeValueMismatch,
			TSValue:  ts,
			GoValue:  go_,
			Severity: tolerances.GetSeverity(path, DiffTypeValueMismatch),
		})
	}
}

// compareObjects compares two maps.
func compareObjects(path string, ts, go_ map[string]interface{}, tolerances *Tolerances, diffs *[]Difference) {
	allKeys := make(map[string]bool)
	for k := range ts {
		allKeys[k] = true
	}
	for k := range go_ {
		allKeys[k] = true
	}

	for key := range allKeys {
		keyPath := path + "." + key
		tsVal, tsOk := ts[key]
		goVal, goOk := go_[key]

		if !tsOk {
			if !tolerances.ShouldIgnore(keyPath) {
				*diffs = append(*diffs, Difference{
					Path:     keyPath,
					Type:     DiffTypeMissingInTS,
					GoValue:  goVal,
					Severity: tolerances.GetSeverity(keyPath, DiffTypeMissingInTS),
				})
			}
			continue
		}
		if !goOk {
			if !tolerances.ShouldIgnore(keyPath) {
				*diffs = append(*diffs, Difference{
					Path:     keyPath,
					Type:     DiffTypeMissingInGo,
					TSValue:  tsVal,
					Severity: tolerances.GetSeverity(keyPath, DiffTypeMissingInGo),
				})
			}
			continue
		}

		compareValues(keyPath, tsVal, goVal, tolerances, diffs)
	}
}

// compareArrays compares two slices.
func compareArrays(path string, ts, go_ []interface{}, tolerances *Tolerances, diffs *[]Difference) {
	maxLen := len(ts)
	if len(go_) > maxLen {
		maxLen = len(go_)
	}

	for i := 0; i < maxLen; i++ {
		elemPath := fmt.Sprintf("%s[%d]", path, i)

		if i >= len(ts) {
			*diffs = append(*diffs, Difference{
				Path:     elemPath,
				Type:     DiffTypeMissingInTS,
				GoValue:  go_[i],
				Severity: tolerances.GetSeverity(elemPath, DiffTypeMissingInTS),
			})
			continue
		}
		if i >= len(go_) {
			*diffs = append(*diffs, Difference{
				Path:     elemPath,
				Type:     DiffTypeMissingInGo,
				TSValue:  ts[i],
				Severity: tolerances.GetSeverity(elemPath, DiffTypeMissingInGo),
			})
			continue
		}

		compareValues(elemPath, ts[i], go_[i], tolerances, diffs)
	}
}

// FilterBySeverity filters differences by severity.
func FilterBySeverity(diffs []Difference, severity Severity) []Difference {
	var filtered []Difference
	for _, d := range diffs {
		if d.Severity == severity {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// ===== Harness Integration Tests =====

var _ = Describe("Comparative Harness", func() {
	Describe("JSON Comparison", func() {
		It("should detect value mismatches", func() {
			ts := []byte(`{"name": "alice", "age": 30}`)
			go_ := []byte(`{"name": "bob", "age": 30}`)

			diffs, err := CompareJSON(ts, go_, DefaultTolerances())
			Expect(err).NotTo(HaveOccurred())
			Expect(len(diffs)).To(Equal(1))
			Expect(diffs[0].Path).To(Equal("$.name"))
			Expect(diffs[0].Type).To(Equal(DiffTypeValueMismatch))
		})

		It("should detect missing fields in Go", func() {
			ts := []byte(`{"name": "alice", "email": "alice@test.com"}`)
			go_ := []byte(`{"name": "alice"}`)

			diffs, err := CompareJSON(ts, go_, DefaultTolerances())
			Expect(err).NotTo(HaveOccurred())
			Expect(len(diffs)).To(Equal(1))
			Expect(diffs[0].Type).To(Equal(DiffTypeMissingInGo))
		})

		It("should detect missing fields in TS", func() {
			ts := []byte(`{"name": "alice"}`)
			go_ := []byte(`{"name": "alice", "extra": "field"}`)

			diffs, err := CompareJSON(ts, go_, DefaultTolerances())
			Expect(err).NotTo(HaveOccurred())
			Expect(len(diffs)).To(Equal(1))
			Expect(diffs[0].Type).To(Equal(DiffTypeMissingInTS))
		})

		It("should ignore configured paths", func() {
			ts := []byte(`{"id": "ts-123", "name": "alice"}`)
			go_ := []byte(`{"id": "go-456", "name": "alice"}`)

			tolerances := DefaultTolerances()
			diffs, err := CompareJSON(ts, go_, tolerances)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(diffs)).To(Equal(0))
		})

		It("should compare nested objects", func() {
			ts := []byte(`{"user": {"name": "alice", "settings": {"theme": "dark"}}}`)
			go_ := []byte(`{"user": {"name": "alice", "settings": {"theme": "light"}}}`)

			diffs, err := CompareJSON(ts, go_, DefaultTolerances())
			Expect(err).NotTo(HaveOccurred())
			Expect(len(diffs)).To(Equal(1))
			Expect(diffs[0].Path).To(Equal("$.user.settings.theme"))
		})

		It("should compare arrays", func() {
			ts := []byte(`{"items": [1, 2, 3]}`)
			go_ := []byte(`{"items": [1, 2, 4]}`)

			diffs, err := CompareJSON(ts, go_, DefaultTolerances())
			Expect(err).NotTo(HaveOccurred())
			Expect(len(diffs)).To(Equal(1))
			Expect(diffs[0].Path).To(Equal("$.items[2]"))
		})
	})

	Describe("Tolerances", func() {
		It("should correctly assign severity", func() {
			tolerances := DefaultTolerances()

			Expect(tolerances.GetSeverity("$.unknown", DiffTypeMissingInGo)).To(Equal(SeverityCritical))
			Expect(tolerances.GetSeverity("$.unknown", DiffTypeMissingInTS)).To(Equal(SeverityWarning))
			Expect(tolerances.GetSeverity("$.version", DiffTypeValueMismatch)).To(Equal(SeverityInfo))
		})
	})

	Describe("MockLLM Integration", func() {
		var mockServer *MockLLMServer

		BeforeEach(func() {
			config := &MockLLMConfig{
				Responses: map[string]MockResponse{
					"create a file": {
						Content: "I'll create that file for you.",
						ToolCalls: []MockToolCall{
							{
								ID:   "call_write",
								Type: "function",
								Function: MockFunctionCall{
									Name:      "write_file",
									Arguments: `{"path": "/test.txt", "content": "hello"}`,
								},
							},
						},
					},
				},
				Defaults: MockDefaults{
					Fallback: "I understand.",
				},
				Settings: MockSettings{
					EnableStreaming: true,
				},
			}
			mockServer = NewMockLLMServer(config)
		})

		AfterEach(func() {
			mockServer.Close()
		})

		It("should provide deterministic responses", func() {
			// First request
			body1 := map[string]interface{}{
				"model": "gpt-4",
				"messages": []map[string]interface{}{
					{"role": "user", "content": "please create a file"},
				},
			}
			jsonBody1, _ := json.Marshal(body1)
			resp1, err := http.Post(mockServer.URL()+"/v1/chat/completions", "application/json", bytes.NewReader(jsonBody1))
			Expect(err).NotTo(HaveOccurred())

			var result1 map[string]interface{}
			json.NewDecoder(resp1.Body).Decode(&result1)
			resp1.Body.Close()

			// Second identical request
			jsonBody2, _ := json.Marshal(body1)
			resp2, err := http.Post(mockServer.URL()+"/v1/chat/completions", "application/json", bytes.NewReader(jsonBody2))
			Expect(err).NotTo(HaveOccurred())

			var result2 map[string]interface{}
			json.NewDecoder(resp2.Body).Decode(&result2)
			resp2.Body.Close()

			// Responses should have same content (ignoring dynamic fields like id, created)
			choices1 := result1["choices"].([]interface{})
			choices2 := result2["choices"].([]interface{})
			msg1 := choices1[0].(map[string]interface{})["message"].(map[string]interface{})
			msg2 := choices2[0].(map[string]interface{})["message"].(map[string]interface{})

			Expect(msg1["content"]).To(Equal(msg2["content"]))
		})

		It("should return tool calls when configured", func() {
			body := map[string]interface{}{
				"model": "gpt-4",
				"messages": []map[string]interface{}{
					{"role": "user", "content": "please create a file"},
				},
			}
			jsonBody, _ := json.Marshal(body)
			resp, err := http.Post(mockServer.URL()+"/v1/chat/completions", "application/json", bytes.NewReader(jsonBody))
			Expect(err).NotTo(HaveOccurred())

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)
			resp.Body.Close()

			choices := result["choices"].([]interface{})
			msg := choices[0].(map[string]interface{})["message"].(map[string]interface{})
			toolCalls := msg["tool_calls"].([]interface{})

			Expect(len(toolCalls)).To(Equal(1))
			tc := toolCalls[0].(map[string]interface{})
			fn := tc["function"].(map[string]interface{})
			Expect(fn["name"]).To(Equal("write_file"))
		})
	})
})
