package session

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/opencode-ai/opencode/internal/storage"
	"github.com/opencode-ai/opencode/internal/tool"
	"github.com/opencode-ai/opencode/pkg/types"
)

func TestNewProcessor(t *testing.T) {
	store := storage.New(t.TempDir())

	toolReg := tool.NewRegistry(t.TempDir())

	proc := NewProcessor(nil, toolReg, store, nil, "", "")

	assert.NotNil(t, proc)
	assert.NotNil(t, proc.sessions)
	assert.Empty(t, proc.sessions)
}

func TestProcessor_IsProcessing(t *testing.T) {
	store := storage.New(t.TempDir())

	toolReg := tool.NewRegistry(t.TempDir())
	proc := NewProcessor(nil, toolReg, store, nil, "", "")

	// Initially not processing
	assert.False(t, proc.IsProcessing("session1"))

	// Manually add session state
	proc.mu.Lock()
	proc.sessions["session1"] = &sessionState{}
	proc.mu.Unlock()

	// Now should be processing
	assert.True(t, proc.IsProcessing("session1"))
}

func TestProcessor_Abort(t *testing.T) {
	store := storage.New(t.TempDir())

	toolReg := tool.NewRegistry(t.TempDir())
	proc := NewProcessor(nil, toolReg, store, nil, "", "")

	// Try to abort non-existent session
	err := proc.Abort("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not processing")

	// Create a session state with cancel function
	ctx, cancel := context.WithCancel(context.Background())
	proc.mu.Lock()
	proc.sessions["session1"] = &sessionState{
		ctx:    ctx,
		cancel: cancel,
	}
	proc.mu.Unlock()

	// Abort should succeed
	err = proc.Abort("session1")
	assert.NoError(t, err)

	// Context should be cancelled
	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Fatal("context should be cancelled")
	}
}

func TestProcessor_GetActiveState(t *testing.T) {
	store := storage.New(t.TempDir())

	toolReg := tool.NewRegistry(t.TempDir())
	proc := NewProcessor(nil, toolReg, store, nil, "", "")

	// No active session
	msg, parts, ok := proc.GetActiveState("session1")
	assert.False(t, ok)
	assert.Nil(t, msg)
	assert.Nil(t, parts)

	// Add session state
	testMsg := &types.Message{ID: "msg1", Role: "assistant"}
	testParts := []types.Part{&types.TextPart{ID: "part1", Type: "text", Text: "Hello"}}

	proc.mu.Lock()
	proc.sessions["session1"] = &sessionState{
		message: testMsg,
		parts:   testParts,
	}
	proc.mu.Unlock()

	// Now should return state
	msg, parts, ok = proc.GetActiveState("session1")
	assert.True(t, ok)
	assert.Equal(t, testMsg, msg)
	assert.Equal(t, testParts, parts)
}

func TestAgent_ToolEnabled(t *testing.T) {
	tests := []struct {
		name     string
		agent    *Agent
		toolID   string
		expected bool
	}{
		{
			name:     "empty agent allows all tools",
			agent:    &Agent{},
			toolID:   "Read",
			expected: true,
		},
		{
			name: "explicitly enabled tool",
			agent: &Agent{
				Tools: []string{"Read", "Write"},
			},
			toolID:   "Read",
			expected: true,
		},
		{
			name: "tool not in enabled list",
			agent: &Agent{
				Tools: []string{"Read", "Write"},
			},
			toolID:   "Bash",
			expected: false,
		},
		{
			name: "explicitly disabled tool",
			agent: &Agent{
				DisabledTools: []string{"Bash"},
			},
			toolID:   "Bash",
			expected: false,
		},
		{
			name: "disabled takes precedence",
			agent: &Agent{
				Tools:         []string{"Bash"},
				DisabledTools: []string{"Bash"},
			},
			toolID:   "Bash",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.agent.ToolEnabled(tt.toolID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultAgent(t *testing.T) {
	agent := DefaultAgent()

	assert.Equal(t, "default", agent.Name)
	assert.Equal(t, 0.7, agent.Temperature)
	assert.Equal(t, 1.0, agent.TopP)
	assert.Equal(t, 50, agent.MaxSteps)
	assert.Equal(t, "ask", agent.Permission.DoomLoop)
	assert.Equal(t, "ask", agent.Permission.Bash)
	assert.Equal(t, "ask", agent.Permission.Write)
}

func TestCodeAgent(t *testing.T) {
	agent := CodeAgent()

	assert.Equal(t, "code", agent.Name)
	assert.Equal(t, 0.3, agent.Temperature)
	assert.Equal(t, 100, agent.MaxSteps)
	assert.NotEmpty(t, agent.Prompt)
	assert.Equal(t, "allow", agent.Permission.Write)
}

func TestPlanAgent(t *testing.T) {
	agent := PlanAgent()

	assert.Equal(t, "plan", agent.Name)
	assert.Equal(t, 0.5, agent.Temperature)
	assert.Equal(t, 20, agent.MaxSteps)
	assert.Contains(t, agent.DisabledTools, "Write")
	assert.Contains(t, agent.DisabledTools, "Edit")
	assert.Contains(t, agent.DisabledTools, "Bash")
	assert.Equal(t, "deny", agent.Permission.Write)
}

func TestSystemPrompt_Build(t *testing.T) {
	session := &types.Session{
		ID:        "test-session",
		Directory: t.TempDir(),
	}
	agent := DefaultAgent()

	prompt := NewSystemPrompt(session, agent, "anthropic", "claude-sonnet-4")
	result := prompt.Build()

	// Should contain provider header
	assert.Contains(t, result, "Claude")
	assert.Contains(t, result, "Anthropic")

	// Should contain environment info
	assert.Contains(t, result, "Environment Information")
	assert.Contains(t, result, "Working Directory")
	assert.Contains(t, result, "Platform")

	// Should contain tool instructions
	assert.Contains(t, result, "Tool Usage Guidelines")
	assert.Contains(t, result, "File Operations")
}

func TestSystemPrompt_ProviderHeaders(t *testing.T) {
	tests := []struct {
		provider string
		expected string
	}{
		{"anthropic", "Claude"},
		{"openai", "helpful AI assistant"},
		{"google", "helpful AI assistant"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			prompt := NewSystemPrompt(nil, DefaultAgent(), tt.provider, "test-model")
			result := prompt.Build()
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestCompactionConfig(t *testing.T) {
	config := DefaultCompactionConfig

	assert.Equal(t, 4, config.MinMessagesToKeep)
	assert.Equal(t, 2000, config.SummaryMaxTokens)
	assert.Equal(t, 0.75, config.ContextThreshold)
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"", 0},
		{"Hello", 1},
		{"Hello World", 2},
		{"This is a test message with some words", 9},
	}

	for _, tt := range tests {
		result := estimateTokens(tt.text)
		assert.Equal(t, tt.expected, result, "text: %s", tt.text)
	}
}

func TestGeneratePartID(t *testing.T) {
	id1 := generatePartID()
	id2 := generatePartID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Len(t, id1, 26) // ULID length
}

func TestPtr(t *testing.T) {
	s := "test"
	p := ptr(s)
	assert.NotNil(t, p)
	assert.Equal(t, s, *p)

	n := 42
	pn := ptr(n)
	assert.NotNil(t, pn)
	assert.Equal(t, n, *pn)
}

func TestToolState(t *testing.T) {
	assert.Equal(t, ToolState("pending"), ToolStatePending)
	assert.Equal(t, ToolState("running"), ToolStateRunning)
	assert.Equal(t, ToolState("completed"), ToolStateCompleted)
	assert.Equal(t, ToolState("error"), ToolStateError)
}

func TestCompactionPart(t *testing.T) {
	part := &CompactionPart{
		ID:      "test-id",
		Type:    "compaction",
		Summary: "This is a summary",
		Count:   5,
	}

	assert.Equal(t, "compaction", part.PartType())
	assert.Equal(t, "test-id", part.PartID())
}

func TestSessionState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	state := &sessionState{
		ctx:     ctx,
		cancel:  cancel,
		step:    0,
		retries: 0,
	}

	assert.NotNil(t, state.ctx)
	assert.NotNil(t, state.cancel)
	assert.Equal(t, 0, state.step)
	assert.Equal(t, 0, state.retries)
}

func TestProcessCallback(t *testing.T) {
	var callCount int
	var lastMsg *types.Message
	var lastParts []types.Part

	callback := ProcessCallback(func(msg *types.Message, parts []types.Part) {
		callCount++
		lastMsg = msg
		lastParts = parts
	})

	msg := &types.Message{ID: "test"}
	parts := []types.Part{&types.TextPart{ID: "p1"}}

	callback(msg, parts)

	assert.Equal(t, 1, callCount)
	assert.Equal(t, msg, lastMsg)
	assert.Equal(t, parts, lastParts)
}

func TestConstants(t *testing.T) {
	assert.Equal(t, 50, MaxSteps)
	assert.Equal(t, 3, MaxRetries)
	assert.Equal(t, time.Second, RetryInitialInterval)
	assert.Equal(t, 30*time.Second, RetryMaxInterval)
	assert.Equal(t, 2*time.Minute, RetryMaxElapsedTime)
	assert.Equal(t, 150000, MaxContextTokens)
}

func TestNewRetryBackoff(t *testing.T) {
	ctx := context.Background()
	b := newRetryBackoff(ctx)

	// First backoff should be around RetryInitialInterval (with jitter)
	interval1 := b.NextBackOff()
	assert.NotEqual(t, interval1, time.Duration(0))

	// Second backoff should be longer due to exponential increase
	interval2 := b.NextBackOff()
	assert.NotEqual(t, interval2, time.Duration(0))

	// Third backoff
	interval3 := b.NextBackOff()
	assert.NotEqual(t, interval3, time.Duration(0))

	// Fourth should hit max retries (MaxRetries = 3)
	interval4 := b.NextBackOff()
	// After max retries, it should return backoff.Stop (-1)
	assert.Less(t, interval4, time.Duration(0))
}

func TestNewRetryBackoff_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	b := newRetryBackoff(ctx)

	// First backoff should work
	interval1 := b.NextBackOff()
	assert.Greater(t, interval1, time.Duration(0))

	// Cancel the context
	cancel()

	// After cancellation, should return backoff.Stop
	interval2 := b.NextBackOff()
	assert.Less(t, interval2, time.Duration(0))
}
