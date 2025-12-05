package session

import (
	"strings"
	"testing"
)

func TestComputeDiff_SingleLineChange(t *testing.T) {
	before := `module github.com/opencode-ai/opencode

go 1.25

require (
	github.com/example/pkg v1.0.0
)`

	after := `module github.com/opencode-ai/opencode

go 1.24

require (
	github.com/example/pkg v1.0.0
)`

	diffText, additions, deletions, err := computeDiff(before, after, "go.mod")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The change from "go 1.25" to "go 1.24" should result in 1 addition and 1 deletion
	if additions != 1 {
		t.Errorf("expected 1 addition, got %d", additions)
	}
	if deletions != 1 {
		t.Errorf("expected 1 deletion, got %d", deletions)
	}

	// diffText should not be empty
	if diffText == "" {
		t.Error("expected non-empty diff text")
	}
}

func TestComputeDiff_MultipleLineChanges(t *testing.T) {
	before := `line1
line2
line3`

	after := `line1
modified2
line3
line4`

	_, additions, deletions, err := computeDiff(before, after, "test.txt")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The diff algorithm groups changes differently:
	// - "line2\nline3" gets replaced with "modified2\nline3\nline4"
	// - This results in 3 lines added and 2 lines deleted
	// The important thing is that additions > 0 when there are additions
	if additions == 0 {
		t.Error("expected non-zero additions")
	}
	if deletions == 0 {
		t.Error("expected non-zero deletions")
	}
	// Net change: +1 line (from 3 to 4 lines)
	if additions-deletions != 1 {
		t.Errorf("expected net change of +1, got %d", additions-deletions)
	}
}

func TestComputeDiff_NoChanges(t *testing.T) {
	content := `same content
on multiple lines`

	diffText, additions, deletions, err := computeDiff(content, content, "file.txt")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if additions != 0 {
		t.Errorf("expected 0 additions, got %d", additions)
	}
	if deletions != 0 {
		t.Errorf("expected 0 deletions, got %d", deletions)
	}

	// No changes means empty diff or only headers
	// Either way, additions and deletions should be 0
	_ = diffText
}

func TestComputeDiff_NewFile(t *testing.T) {
	before := ""
	after := `new content
with two lines`

	_, additions, deletions, err := computeDiff(before, after, "new.txt")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// New file with 2 lines = 2 additions
	if additions != 2 {
		t.Errorf("expected 2 additions, got %d", additions)
	}
	if deletions != 0 {
		t.Errorf("expected 0 deletions, got %d", deletions)
	}
}

func TestComputeDiff_DeletedFile(t *testing.T) {
	before := `content to delete
second line`
	after := ""

	_, additions, deletions, err := computeDiff(before, after, "deleted.txt")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if additions != 0 {
		t.Errorf("expected 0 additions, got %d", additions)
	}
	// Deleted file with 2 lines = 2 deletions
	if deletions != 2 {
		t.Errorf("expected 2 deletions, got %d", deletions)
	}
}

func TestComputeDiff_UnifiedDiffFormat(t *testing.T) {
	before := `line1
line2
line3`

	after := `line1
modified2
line3`

	diffText, _, _, err := computeDiff(before, after, "test.txt")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("Diff output:\n%s", diffText)

	// The diff text should be in proper unified diff format
	// Each deleted line should be prefixed with "-" on its own line
	// Each added line should be prefixed with "+" on its own line

	// Check that diffText contains proper line-by-line format
	// It should NOT have "-line2+modified2" on the same line
	if diffText == "" {
		t.Error("expected non-empty diff text")
	}

	// CRITICAL: The diff should NOT contain URL-encoded characters like %0A
	// The TUI expects raw newlines, not URL-encoded ones
	if strings.Contains(diffText, "%0A") {
		t.Error("diff should not contain URL-encoded newlines (%0A)")
	}
	if strings.Contains(diffText, "%0D") {
		t.Error("diff should not contain URL-encoded carriage returns (%0D)")
	}

	// Verify the diff has proper structure:
	// - Should have "--- test.txt" or "--- a/test.txt" header
	// - Should have "+++ test.txt" or "+++ b/test.txt" header
	// - Should have "-line2" on its own line (not merged with +)
	// - Should have "+modified2" on its own line

	lines := splitLines(diffText)

	hasMinusHeader := false
	hasPlusHeader := false
	foundDeletedLine := false
	foundAddedLine := false

	for _, line := range lines {
		if strings.HasPrefix(line, "--- ") {
			hasMinusHeader = true
		}
		if strings.HasPrefix(line, "+++ ") {
			hasPlusHeader = true
		}
		// Check for proper deleted line format (starts with - but not ---)
		if len(line) > 1 && line[0] == '-' && line[1] != '-' {
			foundDeletedLine = true
			// Verify it's on its own line (doesn't contain + after the content)
			if containsAddedMarker(line) {
				t.Errorf("deleted line should not contain '+' marker: %q", line)
			}
		}
		// Check for proper added line format (starts with + but not +++)
		if len(line) > 1 && line[0] == '+' && line[1] != '+' {
			foundAddedLine = true
		}
	}

	if !hasMinusHeader {
		t.Errorf("diff should have '--- ' header line: %s", diffText)
	}
	if !hasPlusHeader {
		t.Errorf("diff should have '+++ ' header line: %s", diffText)
	}
	if !foundDeletedLine {
		t.Errorf("diff should contain deleted line starting with '-': %s", diffText)
	}
	if !foundAddedLine {
		t.Errorf("diff should contain added line starting with '+': %s", diffText)
	}
}

// splitLines splits text by newlines, similar to strings.Split but handles edge cases
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			lines = append(lines, text[start:i])
			start = i + 1
		}
	}
	if start < len(text) {
		lines = append(lines, text[start:])
	}
	return lines
}

// containsAddedMarker checks if line contains a '+' that's not at the start
func containsAddedMarker(line string) bool {
	for i := 1; i < len(line); i++ {
		if line[i] == '+' {
			return true
		}
	}
	return false
}
