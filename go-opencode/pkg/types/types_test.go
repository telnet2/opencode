package types

import (
	"encoding/json"
	"testing"
)

func TestSession_JSON(t *testing.T) {
	session := Session{
		ID:        "session-123",
		ProjectID: "project-456",
		Directory: "/home/user/project",
		Title:     "Test Session",
		Version:   "1.0.0",
		Summary: SessionSummary{
			Additions: 100,
			Deletions: 50,
			Files:     5,
		},
		Time: SessionTime{
			Created: 1700000000000,
			Updated: 1700000001000,
		},
	}

	// Marshal
	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var decoded Session
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify fields
	if decoded.ID != session.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.ID, session.ID)
	}
	if decoded.ProjectID != session.ProjectID {
		t.Errorf("ProjectID mismatch: got %s, want %s", decoded.ProjectID, session.ProjectID)
	}
	if decoded.Summary.Additions != session.Summary.Additions {
		t.Errorf("Additions mismatch: got %d, want %d", decoded.Summary.Additions, session.Summary.Additions)
	}
}

func TestSession_OptionalFields(t *testing.T) {
	// Test with optional ParentID
	parentID := "parent-123"
	session := Session{
		ID:       "session-123",
		ParentID: &parentID,
	}

	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify parentID is included
	var raw map[string]any
	json.Unmarshal(data, &raw)
	if _, ok := raw["parentID"]; !ok {
		t.Error("parentID should be present when set")
	}

	// Test without parentID
	session2 := Session{ID: "session-456"}
	data2, _ := json.Marshal(session2)
	var raw2 map[string]any
	json.Unmarshal(data2, &raw2)
	if _, ok := raw2["parentID"]; ok {
		t.Error("parentID should be omitted when nil")
	}
}

func TestMessage_JSON(t *testing.T) {
	msg := Message{
		ID:        "msg-123",
		SessionID: "session-456",
		Role:      "assistant",
		ModelID:   "claude-3-opus",
		ProviderID: "anthropic",
		Cost:      0.05,
		Tokens: &TokenUsage{
			Input:  1000,
			Output: 500,
			Cache: CacheUsage{
				Read:  100,
				Write: 50,
			},
		},
		Time: MessageTime{
			Created: 1700000000000,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Role != "assistant" {
		t.Errorf("Role mismatch: got %s, want assistant", decoded.Role)
	}
	if decoded.Tokens.Input != 1000 {
		t.Errorf("Tokens.Input mismatch: got %d, want 1000", decoded.Tokens.Input)
	}
}

func TestMessage_UserFields(t *testing.T) {
	system := "You are a helpful assistant"
	msg := Message{
		ID:        "msg-user-1",
		SessionID: "session-1",
		Role:      "user",
		Agent:     "main",
		Model: &ModelRef{
			ProviderID: "anthropic",
			ModelID:    "claude-3-opus",
		},
		System: &system,
		Tools: map[string]bool{
			"Read":  true,
			"Write": true,
			"Bash":  false,
		},
		Time: MessageTime{Created: 1700000000000},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Agent != "main" {
		t.Errorf("Agent mismatch: got %s, want main", decoded.Agent)
	}
	if decoded.Model.ProviderID != "anthropic" {
		t.Errorf("Model.ProviderID mismatch")
	}
	if !decoded.Tools["Read"] {
		t.Error("Tools[Read] should be true")
	}
	if decoded.Tools["Bash"] {
		t.Error("Tools[Bash] should be false")
	}
}

func TestFileDiff_JSON(t *testing.T) {
	diff := FileDiff{
		Path:      "/src/main.go",
		Additions: 10,
		Deletions: 5,
		Before:    "func old() {}",
		After:     "func new() {}",
	}

	data, err := json.Marshal(diff)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded FileDiff
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Path != diff.Path {
		t.Errorf("Path mismatch: got %s, want %s", decoded.Path, diff.Path)
	}
}

func TestSessionSummary_EmptyDiffs(t *testing.T) {
	summary := SessionSummary{
		Additions: 0,
		Deletions: 0,
		Files:     0,
	}

	data, _ := json.Marshal(summary)
	var raw map[string]any
	json.Unmarshal(data, &raw)

	// Diffs should be omitted when nil/empty
	if _, ok := raw["diffs"]; ok {
		t.Error("diffs should be omitted when nil")
	}
}

func TestCustomPrompt_JSON(t *testing.T) {
	loadedAt := int64(1700000000000)
	prompt := CustomPrompt{
		Type:     "file",
		Value:    "/path/to/prompt.md",
		LoadedAt: &loadedAt,
		Variables: map[string]string{
			"project": "myapp",
			"version": "1.0.0",
		},
	}

	data, err := json.Marshal(prompt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded CustomPrompt
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Type != "file" {
		t.Errorf("Type mismatch: got %s, want file", decoded.Type)
	}
	if decoded.Variables["project"] != "myapp" {
		t.Error("Variables[project] mismatch")
	}
}

func TestMessageError_JSON(t *testing.T) {
	msgErr := MessageError{
		Name: "UnknownError",
		Data: MessageErrorData{Message: "Rate limit exceeded"},
	}

	data, err := json.Marshal(msgErr)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded MessageError
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Name != "UnknownError" {
		t.Errorf("Name mismatch: got %s, want UnknownError", decoded.Name)
	}
}
