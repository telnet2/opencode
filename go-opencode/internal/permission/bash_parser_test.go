package permission

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBashCommand_Simple(t *testing.T) {
	commands, err := ParseBashCommand("ls -la")
	require.NoError(t, err)
	require.Len(t, commands, 1)

	assert.Equal(t, "ls", commands[0].Name)
	assert.Equal(t, []string{"-la"}, commands[0].Args)
}

func TestParseBashCommand_NoArgs(t *testing.T) {
	commands, err := ParseBashCommand("pwd")
	require.NoError(t, err)
	require.Len(t, commands, 1)

	assert.Equal(t, "pwd", commands[0].Name)
	assert.Empty(t, commands[0].Args)
}

func TestParseBashCommand_Pipeline(t *testing.T) {
	commands, err := ParseBashCommand("cat file.txt | grep pattern")
	require.NoError(t, err)
	require.Len(t, commands, 2)

	assert.Equal(t, "cat", commands[0].Name)
	assert.Equal(t, []string{"file.txt"}, commands[0].Args)

	assert.Equal(t, "grep", commands[1].Name)
	assert.Equal(t, []string{"pattern"}, commands[1].Args)
}

func TestParseBashCommand_AndChain(t *testing.T) {
	commands, err := ParseBashCommand("git add . && git commit -m 'message'")
	require.NoError(t, err)
	require.Len(t, commands, 2)

	assert.Equal(t, "git", commands[0].Name)
	assert.Equal(t, "add", commands[0].Subcommand)
	assert.Contains(t, commands[0].Args, ".")

	assert.Equal(t, "git", commands[1].Name)
	assert.Equal(t, "commit", commands[1].Subcommand)
}

func TestParseBashCommand_OrChain(t *testing.T) {
	commands, err := ParseBashCommand("test -f file.txt || touch file.txt")
	require.NoError(t, err)
	require.Len(t, commands, 2)

	assert.Equal(t, "test", commands[0].Name)
	assert.Equal(t, "touch", commands[1].Name)
}

func TestParseBashCommand_Semicolon(t *testing.T) {
	commands, err := ParseBashCommand("echo hello; echo world")
	require.NoError(t, err)
	require.Len(t, commands, 2)

	assert.Equal(t, "echo", commands[0].Name)
	assert.Equal(t, "echo", commands[1].Name)
}

func TestParseBashCommand_Subshell(t *testing.T) {
	commands, err := ParseBashCommand("echo $(pwd)")
	require.NoError(t, err)
	// Should capture both echo and pwd
	assert.GreaterOrEqual(t, len(commands), 2)

	foundEcho := false
	foundPwd := false
	for _, cmd := range commands {
		if cmd.Name == "echo" {
			foundEcho = true
		}
		if cmd.Name == "pwd" {
			foundPwd = true
		}
	}
	assert.True(t, foundEcho, "should find echo command")
	assert.True(t, foundPwd, "should find pwd command")
}

func TestParseBashCommand_DangerousCommand(t *testing.T) {
	commands, err := ParseBashCommand("rm -rf /tmp/test")
	require.NoError(t, err)
	require.Len(t, commands, 1)

	assert.True(t, IsDangerousCommand(commands[0].Name))
	paths := ExtractPaths(commands[0])
	assert.Equal(t, []string{"/tmp/test"}, paths)
}

func TestParseBashCommand_QuotedStrings(t *testing.T) {
	commands, err := ParseBashCommand(`echo "hello world" 'single quoted'`)
	require.NoError(t, err)
	require.Len(t, commands, 1)

	assert.Equal(t, "echo", commands[0].Name)
	assert.Contains(t, commands[0].Args, "hello world")
	assert.Contains(t, commands[0].Args, "single quoted")
}

func TestParseBashCommand_Git(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		subcommand string
	}{
		{"git commit", "git commit -m 'msg'", "commit"},
		{"git push", "git push origin main", "push"},
		{"git pull", "git pull --rebase", "pull"},
		{"git status", "git status", "status"},
		{"git add", "git add .", "add"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands, err := ParseBashCommand(tt.command)
			require.NoError(t, err)
			require.NotEmpty(t, commands)
			assert.Equal(t, "git", commands[0].Name)
			assert.Equal(t, tt.subcommand, commands[0].Subcommand)
		})
	}
}

func TestParseBashCommand_ComplexGitCommit(t *testing.T) {
	commands, err := ParseBashCommand(`git commit -m "$(cat <<'EOF'
Fix bug in parser
EOF
)"`)
	require.NoError(t, err)
	require.NotEmpty(t, commands)
	assert.Equal(t, "git", commands[0].Name)
}

func TestParseBashCommand_Environment(t *testing.T) {
	commands, err := ParseBashCommand("FOO=bar ./script.sh")
	require.NoError(t, err)
	// The assignment may or may not create a command depending on shell interpretation
	// We mainly care this doesn't error
	assert.NotNil(t, commands)
}

func TestParseBashCommand_Redirect(t *testing.T) {
	commands, err := ParseBashCommand("echo test > output.txt")
	require.NoError(t, err)
	require.NotEmpty(t, commands)
	assert.Equal(t, "echo", commands[0].Name)
}

func TestParseBashCommand_Invalid(t *testing.T) {
	// Unclosed quote
	_, err := ParseBashCommand(`echo "unclosed`)
	assert.Error(t, err)
}

func TestIsDangerousCommand(t *testing.T) {
	dangerous := []string{"rm", "mv", "cp", "chmod", "chown", "mkdir", "touch", "rmdir", "dd"}
	safe := []string{"ls", "cat", "echo", "grep", "find", "git", "npm"}

	for _, cmd := range dangerous {
		assert.True(t, IsDangerousCommand(cmd), "%s should be dangerous", cmd)
	}

	for _, cmd := range safe {
		assert.False(t, IsDangerousCommand(cmd), "%s should not be dangerous", cmd)
	}
}

func TestExtractPaths(t *testing.T) {
	tests := []struct {
		name     string
		cmd      BashCommand
		expected []string
	}{
		{
			name:     "rm with paths",
			cmd:      BashCommand{Name: "rm", Args: []string{"-rf", "/tmp/test", "./local"}},
			expected: []string{"/tmp/test", "./local"},
		},
		{
			name:     "cp source and dest",
			cmd:      BashCommand{Name: "cp", Args: []string{"-r", "src/", "dst/"}},
			expected: []string{"src/", "dst/"},
		},
		{
			name:     "chmod with mode",
			cmd:      BashCommand{Name: "chmod", Args: []string{"+x", "script.sh"}},
			expected: []string{"script.sh"},
		},
		{
			name:     "chmod with numeric mode",
			cmd:      BashCommand{Name: "chmod", Args: []string{"755", "script.sh"}},
			expected: []string{"script.sh"},
		},
		{
			name:     "mv with flags",
			cmd:      BashCommand{Name: "mv", Args: []string{"-v", "old.txt", "new.txt"}},
			expected: []string{"old.txt", "new.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := ExtractPaths(tt.cmd)
			assert.Equal(t, tt.expected, paths)
		})
	}
}

func TestIsWithinDir(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		dir      string
		expected bool
	}{
		{"same dir", "/home/user/project", "/home/user/project", true},
		{"subdirectory", "/home/user/project/src", "/home/user/project", true},
		{"nested deep", "/home/user/project/src/pkg/file.go", "/home/user/project", true},
		{"parent dir", "/home/user", "/home/user/project", false},
		{"sibling dir", "/home/user/other", "/home/user/project", false},
		{"absolute outside", "/tmp/test", "/home/user/project", false},
		{"with trailing slash", "/home/user/project/src/", "/home/user/project", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWithinDir(tt.path, tt.dir)
			assert.Equal(t, tt.expected, result, "IsWithinDir(%s, %s)", tt.path, tt.dir)
		})
	}
}
