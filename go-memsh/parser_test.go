package memsh

import (
	"strings"
	"testing"

	"mvdan.cc/sh/v3/syntax"
)

// TestProcessSubstitutionParsing tests if mvdan/sh can parse process substitution syntax
func TestProcessSubstitutionParsing(t *testing.T) {
	tests := []struct {
		name   string
		script string
	}{
		{
			name:   "simple input substitution",
			script: `cat <(echo "hello")`,
		},
		{
			name:   "output substitution",
			script: `echo "test" >(cat)`,
		},
		{
			name:   "diff with two substitutions",
			script: `diff <(cat file1) <(cat file2)`,
		},
		{
			name:   "complex pipeline in substitution",
			script: `cat <(ls | grep txt)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := syntax.NewParser()
			_, err := parser.Parse(strings.NewReader(tt.script), "")
			if err != nil {
				t.Fatalf("Parser failed to parse '%s': %v", tt.script, err)
			}
			t.Logf("Successfully parsed: %s", tt.script)
		})
	}
}

// TestProcessSubstitutionAST examines the AST structure for process substitution
func TestProcessSubstitutionAST(t *testing.T) {
	script := `cat <(echo "hello")`

	parser := syntax.NewParser()
	prog, err := parser.Parse(strings.NewReader(script), "")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Walk the AST and print structure
	syntax.Walk(prog, func(node syntax.Node) bool {
		t.Logf("Node type: %T, Value: %+v", node, node)
		return true
	})
}
