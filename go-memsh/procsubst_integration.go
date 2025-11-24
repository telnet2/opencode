package memsh

import (
	"context"
	"fmt"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// processCmdForSubstitution scans command arguments for process substitution
// and sets up virtual pipes for them
func (s *Shell) processCmdForSubstitution(ctx context.Context, words []*syntax.Word) ([]*VirtualPipe, error) {
	var pipes []*VirtualPipe

	for _, word := range words {
		// Walk through the word parts looking for ProcSubst nodes
		for _, part := range word.Parts {
			if procSubst, ok := part.(*syntax.ProcSubst); ok {
				pipe, err := s.setupProcessSubstitution(ctx, procSubst)
				if err != nil {
					// Clean up any pipes we've already created
					for _, p := range pipes {
						s.pipeManager.ClosePipe(p.id)
					}
					return nil, err
				}
				pipes = append(pipes, pipe)
			}
		}
	}

	return pipes, nil
}

// setupProcessSubstitution creates a virtual pipe and starts executing the command
func (s *Shell) setupProcessSubstitution(ctx context.Context, procSubst *syntax.ProcSubst) (*VirtualPipe, error) {
	// Create a virtual pipe
	pipe := s.pipeManager.CreatePipe()

	// Convert the statements to a command string
	var cmdBuilder strings.Builder
	syntax.NewPrinter().Print(&cmdBuilder, &syntax.File{Stmts: procSubst.Stmts})
	cmdStr := cmdBuilder.String()

	// Create ProcessSubstitution
	ps := &ProcessSubstitution{
		Command: cmdStr,
		IsInput: procSubst.Op.String() == "<(", // <(...) for input, >(...) for output
		Pipe:    pipe,
	}

	// Execute in background
	go func() {
		if err := ps.ExecuteInBackground(ctx, s); err != nil {
			fmt.Fprintf(s.stderr, "process substitution error: %v\n", err)
		}
	}()

	return pipe, nil
}

// replaceProcSubstInWord replaces process substitution nodes with /dev/fd/N paths
func replaceProcSubstInWord(word *syntax.Word, pipes map[*syntax.ProcSubst]*VirtualPipe) {
	newParts := make([]syntax.WordPart, 0, len(word.Parts))

	for _, part := range word.Parts {
		if procSubst, ok := part.(*syntax.ProcSubst); ok {
			if pipe, found := pipes[procSubst]; found {
				// Replace with a literal containing the virtual path
				newParts = append(newParts, &syntax.Lit{
					Value: pipe.GetPath(),
				})
				continue
			}
		}
		newParts = append(newParts, part)
	}

	word.Parts = newParts
}
