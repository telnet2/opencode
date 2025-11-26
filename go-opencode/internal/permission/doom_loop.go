package permission

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
)

// DoomLoopThreshold is the number of identical calls before triggering.
const DoomLoopThreshold = 3

// DoomLoopDetector tracks repeated tool calls to detect infinite loops.
type DoomLoopDetector struct {
	mu      sync.RWMutex
	history map[string][]string // sessionID -> last N tool call hashes
}

// NewDoomLoopDetector creates a new doom loop detector.
func NewDoomLoopDetector() *DoomLoopDetector {
	return &DoomLoopDetector{
		history: make(map[string][]string),
	}
}

// Check checks if a tool call is a doom loop (same tool + input N times in a row).
// Returns true if this appears to be a doom loop.
func (d *DoomLoopDetector) Check(sessionID, toolName string, input any) bool {
	hash := d.hashCall(toolName, input)

	d.mu.Lock()
	defer d.mu.Unlock()

	history := d.history[sessionID]

	// Check if we have enough history and all recent calls match
	if len(history) >= DoomLoopThreshold-1 {
		allSame := true
		start := len(history) - (DoomLoopThreshold - 1)
		for i := start; i < len(history); i++ {
			if history[i] != hash {
				allSame = false
				break
			}
		}

		if allSame {
			// This is a doom loop - update history and return true
			d.history[sessionID] = append(history, hash)
			// Keep only last 10 entries to prevent unbounded growth
			if len(d.history[sessionID]) > 10 {
				d.history[sessionID] = d.history[sessionID][len(d.history[sessionID])-10:]
			}
			return true
		}
	}

	// Not a doom loop - update history
	d.history[sessionID] = append(history, hash)
	// Keep only last 10 entries
	if len(d.history[sessionID]) > 10 {
		d.history[sessionID] = d.history[sessionID][len(d.history[sessionID])-10:]
	}

	return false
}

// hashCall creates a hash of the tool name and input.
func (d *DoomLoopDetector) hashCall(toolName string, input any) string {
	data, _ := json.Marshal(map[string]any{
		"tool":  toolName,
		"input": input,
	})
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// Clear clears the history for a session.
func (d *DoomLoopDetector) Clear(sessionID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.history, sessionID)
}

// Reset resets the detector for a session after a different call breaks the loop.
func (d *DoomLoopDetector) Reset(sessionID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.history[sessionID] = nil
}
