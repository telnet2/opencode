package tool

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// buildDiffMetadata calculates a unified diff and line counts to enrich tool metadata.
// It returns the diff text (prefixed with file headers when a path is provided),
// the number of added lines, and the number of deleted lines.
func buildDiffMetadata(path, before, after, baseDir string) (string, int, int) {
	if before == after {
		return "", 0, 0
	}

	relPath := relativePath(path, baseDir)

	dmp := diffmatchpatch.New()
	a, b, lineArray := dmp.DiffLinesToChars(before, after)
	diffs := dmp.DiffMain(a, b, false)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)

	additions, deletions := 0, 0
	for _, d := range diffs {
		switch d.Type {
		case diffmatchpatch.DiffInsert:
			additions += countLines(d.Text)
		case diffmatchpatch.DiffDelete:
			deletions += countLines(d.Text)
		}
	}

	patches := dmp.PatchMake(before, diffs)
	diffText := dmp.PatchToText(patches)
	if diffText == "" {
		return "", additions, deletions
	}

	var builder strings.Builder
	if relPath != "" {
		builder.WriteString(fmt.Sprintf("--- %s\n", relPath))
		builder.WriteString(fmt.Sprintf("+++ %s\n", relPath))
	}
	builder.WriteString(diffText)

	return builder.String(), additions, deletions
}

func relativePath(path, baseDir string) string {
	if path == "" {
		return ""
	}
	if baseDir == "" {
		return path
	}
	if rel, err := filepath.Rel(baseDir, path); err == nil {
		return rel
	}
	return path
}

func countLines(text string) int {
	if text == "" {
		return 0
	}
	lines := strings.Count(text, "\n")
	if !strings.HasSuffix(text, "\n") {
		lines++
	}
	return lines
}
