package memsh

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

// EnvironMap implements expand.Environ with a map backend
type EnvironMap struct {
	vars map[string]expand.Variable
}

// NewEnvironMap creates a new environment map from os environment
func NewEnvironMap(pairs []string) *EnvironMap {
	env := &EnvironMap{
		vars: make(map[string]expand.Variable),
	}

	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			env.vars[parts[0]] = expand.Variable{
				Exported: true,
				Kind:     expand.String,
				Str:      parts[1],
			}
		}
	}

	return env
}

// Get retrieves a variable by name
func (e *EnvironMap) Get(name string) expand.Variable {
	if v, ok := e.vars[name]; ok {
		return v
	}
	return expand.Variable{}
}

// Each iterates over all variables
func (e *EnvironMap) Each(fn func(name string, vr expand.Variable) bool) {
	for name, vr := range e.vars {
		if !fn(name, vr) {
			break
		}
	}
}

// Set sets a variable
func (e *EnvironMap) Set(name string, vr expand.Variable) {
	e.vars[name] = vr
}

// Unset removes a variable
func (e *EnvironMap) Unset(name string) {
	delete(e.vars, name)
}

// Copy creates a deep copy of the environment map
func (e *EnvironMap) Copy() *EnvironMap {
	newEnv := &EnvironMap{
		vars: make(map[string]expand.Variable),
	}
	e.Each(func(name string, vr expand.Variable) bool {
		newEnv.Set(name, vr)
		return true
	})
	return newEnv
}

// ReplaceWith overwrites the current environment with a copy of another one
// while keeping the receiver pointer stable.
func (e *EnvironMap) ReplaceWith(other *EnvironMap) {
	e.vars = make(map[string]expand.Variable, len(other.vars))
	other.Each(func(name string, vr expand.Variable) bool {
		e.vars[name] = vr
		return true
	})
}

// ToSlice converts the environment to a slice of "key=value" strings
func (e *EnvironMap) ToSlice() []string {
	var result []string
	e.Each(func(name string, vr expand.Variable) bool {
		if vr.Exported {
			result = append(result, name+"="+vr.Str)
		}
		return true
	})
	sort.Strings(result)
	return result
}

// cmdEnv implements the env command
func (s *Shell) cmdEnv(ctx context.Context, args []string) error {
	_, stdout, _ := s.stdio(ctx)
	// If no arguments, list all exported variables
	if len(args) == 1 {
		vars := s.env.ToSlice()
		for _, v := range vars {
			fmt.Fprintln(stdout, v)
		}
		return nil
	}

	// POSIX: env can run command with modified environment
	// env [-i] [-u name] [name=value ...] [command [args...]]
	ignoreEnv := false
	unsetVars := []string{}
	setVars := map[string]string{}
	commandIndex := -1

	for i := 1; i < len(args); i++ {
		arg := args[i]

		if arg == "-i" || arg == "--ignore-environment" {
			ignoreEnv = true
		} else if arg == "-u" || arg == "--unset" {
			if i+1 >= len(args) {
				return fmt.Errorf("env: option requires an argument -- 'u'")
			}
			i++
			unsetVars = append(unsetVars, args[i])
		} else if strings.Contains(arg, "=") {
			// name=value
			parts := strings.SplitN(arg, "=", 2)
			setVars[parts[0]] = parts[1]
		} else {
			// First non-assignment, non-flag argument is the command
			commandIndex = i
			break
		}
	}

	// If no command specified, just print the modified environment
	if commandIndex == -1 {
		// Build modified environment
		env := &EnvironMap{vars: make(map[string]expand.Variable)}

		if !ignoreEnv {
			// Copy current environment
			s.env.Each(func(name string, vr expand.Variable) bool {
				env.Set(name, vr)
				return true
			})
		}

		// Apply unsets
		for _, name := range unsetVars {
			env.Unset(name)
		}

		// Apply sets
		for name, value := range setVars {
			env.Set(name, expand.Variable{
				Exported: true,
				Kind:     expand.String,
				Str:      value,
			})
		}

		// Print modified environment
		vars := env.ToSlice()
		for _, v := range vars {
			fmt.Fprintln(stdout, v)
		}
		return nil
	}

	// Run command with modified environment
	// Create a new shell with modified environment
	oldEnv := s.env

	newEnv := &EnvironMap{vars: make(map[string]expand.Variable)}
	if !ignoreEnv {
		s.env.Each(func(name string, vr expand.Variable) bool {
			newEnv.Set(name, vr)
			return true
		})
	}

	// Apply unsets
	for _, name := range unsetVars {
		newEnv.Unset(name)
	}

	// Apply sets
	for name, value := range setVars {
		newEnv.Set(name, expand.Variable{
			Exported: true,
			Kind:     expand.String,
			Str:      value,
		})
	}

	// Temporarily replace environment
	s.env = newEnv
	s.runner.Reset()
	interp.Env(s.env)(s.runner)

	// Build command string from remaining args
	command := strings.Join(args[commandIndex:], " ")

	// Execute command
	err := s.Run(ctx, command)

	// Restore original environment
	s.env = oldEnv
	s.runner.Reset()
	interp.Env(s.env)(s.runner)

	return err
}

// cmdSet implements the set command
func (s *Shell) cmdSet(ctx context.Context, args []string) error {
	_, stdout, _ := s.stdio(ctx)
	if len(args) == 1 {
		// List all variables (including non-exported)
		var vars []string
		s.env.Each(func(name string, vr expand.Variable) bool {
			vars = append(vars, name+"="+vr.Str)
			return true
		})
		sort.Strings(vars)
		for _, v := range vars {
			fmt.Fprintln(stdout, v)
		}
		return nil
	}

	// Parse variable assignment: set VAR=value
	for _, arg := range args[1:] {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("set: invalid syntax: %s", arg)
		}

		name := parts[0]
		value := parts[1]

		// Set as non-exported by default
		s.env.Set(name, expand.Variable{
			Exported: false,
			Kind:     expand.String,
			Str:      value,
		})
	}

	// Update runner environment
	s.runner.Reset()
	interp.Env(s.env)(s.runner)

	return nil
}

// cmdUnset implements the unset command
func (s *Shell) cmdUnset(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("unset: missing variable name")
	}

	for _, name := range args[1:] {
		s.env.Unset(name)
	}

	// Update runner environment
	s.runner.Reset()
	interp.Env(s.env)(s.runner)

	return nil
}

// cmdExport implements the export command
func (s *Shell) cmdExport(ctx context.Context, args []string) error {
	_, stdout, _ := s.stdio(ctx)
	if len(args) == 1 {
		// List all exported variables
		var vars []string
		s.env.Each(func(name string, vr expand.Variable) bool {
			if vr.Exported {
				vars = append(vars, fmt.Sprintf("export %s=%s", name, vr.Str))
			}
			return true
		})
		sort.Strings(vars)
		for _, v := range vars {
			fmt.Fprintln(stdout, v)
		}
		return nil
	}

	// Parse variable assignment or mark as exported
	for _, arg := range args[1:] {
		parts := strings.SplitN(arg, "=", 2)
		name := parts[0]

		if len(parts) == 2 {
			// export VAR=value
			value := parts[1]
			s.env.Set(name, expand.Variable{
				Exported: true,
				Kind:     expand.String,
				Str:      value,
			})
		} else {
			// export VAR (mark existing as exported)
			vr := s.env.Get(name)
			vr.Exported = true
			s.env.Set(name, vr)
		}
	}

	// Update runner environment
	s.runner.Reset()
	interp.Env(s.env)(s.runner)

	return nil
}
