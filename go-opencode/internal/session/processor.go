package session

import (
	"context"
	"fmt"
	"sync"

	"github.com/opencode-ai/opencode/internal/permission"
	"github.com/opencode-ai/opencode/internal/provider"
	"github.com/opencode-ai/opencode/internal/storage"
	"github.com/opencode-ai/opencode/internal/tool"
	"github.com/opencode-ai/opencode/pkg/types"
)

// Processor handles message processing and the agentic loop.
type Processor struct {
	mu sync.Mutex

	providerRegistry  *provider.Registry
	toolRegistry      *tool.Registry
	storage           *storage.Storage
	permissionChecker *permission.Checker

	// Default provider and model to use when not specified
	defaultProviderID string
	defaultModelID    string

	// Active sessions being processed
	sessions map[string]*sessionState
}

// sessionState tracks the state of an active session being processed.
type sessionState struct {
	ctx      context.Context
	cancel   context.CancelFunc
	message  *types.Message
	parts    []types.Part
	waiters  []chan error
	step     int
	retries  int
}

// ProcessCallback is called with message updates during processing.
type ProcessCallback func(msg *types.Message, parts []types.Part)

// NewProcessor creates a new session processor.
func NewProcessor(
	providerReg *provider.Registry,
	toolReg *tool.Registry,
	store *storage.Storage,
	permChecker *permission.Checker,
	defaultProviderID string,
	defaultModelID string,
) *Processor {
	// Use reasonable defaults if not specified
	if defaultProviderID == "" {
		defaultProviderID = "anthropic"
	}
	if defaultModelID == "" {
		defaultModelID = "claude-sonnet-4-20250514"
	}
	return &Processor{
		providerRegistry:  providerReg,
		toolRegistry:      toolReg,
		storage:           store,
		permissionChecker: permChecker,
		defaultProviderID: defaultProviderID,
		defaultModelID:    defaultModelID,
		sessions:          make(map[string]*sessionState),
	}
}

// Process handles a new user message and generates an assistant response.
// This is the main entry point for the agentic loop.
func (p *Processor) Process(ctx context.Context, sessionID string, agent *Agent, callback ProcessCallback) error {
	p.mu.Lock()

	// Check if session is already processing
	if state, ok := p.sessions[sessionID]; ok {
		// Queue this request
		waiter := make(chan error, 1)
		state.waiters = append(state.waiters, waiter)
		p.mu.Unlock()

		// Wait for current processing to complete
		select {
		case err := <-waiter:
			if err != nil {
				return err
			}
			// Retry processing
			return p.Process(ctx, sessionID, agent, callback)
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Create new session state
	loopCtx, cancel := context.WithCancel(ctx)
	state := &sessionState{
		ctx:    loopCtx,
		cancel: cancel,
	}
	p.sessions[sessionID] = state
	p.mu.Unlock()

	// Ensure cleanup
	defer func() {
		p.mu.Lock()
		delete(p.sessions, sessionID)

		// Notify waiters
		for _, waiter := range state.waiters {
			waiter <- nil
		}
		p.mu.Unlock()
	}()

	// Run the agentic loop
	return p.runLoop(loopCtx, sessionID, state, agent, callback)
}

// Abort cancels processing for a session.
func (p *Processor) Abort(sessionID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	state, ok := p.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not processing: %s", sessionID)
	}

	state.cancel()
	return nil
}

// IsProcessing returns whether a session is currently processing.
func (p *Processor) IsProcessing(sessionID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.sessions[sessionID]
	return ok
}

// GetActiveState returns the current state for a processing session.
func (p *Processor) GetActiveState(sessionID string) (*types.Message, []types.Part, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	state, ok := p.sessions[sessionID]
	if !ok {
		return nil, nil, false
	}

	return state.message, state.parts, true
}
