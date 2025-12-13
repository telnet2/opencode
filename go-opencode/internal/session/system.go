package session

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/opencode-ai/opencode/pkg/types"
)

// SystemPrompt builds the system prompt for the LLM.
type SystemPrompt struct {
	session    *types.Session
	agent      *Agent
	modelID    string
	providerID string
}

// NewSystemPrompt creates a new system prompt builder.
func NewSystemPrompt(session *types.Session, agent *Agent, providerID, modelID string) *SystemPrompt {
	return &SystemPrompt{
		session:    session,
		agent:      agent,
		modelID:    modelID,
		providerID: providerID,
	}
}

// Build constructs the complete system prompt.
func (s *SystemPrompt) Build() string {
	var parts []string

	// 1. Provider-specific header
	if header := s.providerHeader(); header != "" {
		parts = append(parts, header)
	}

	// 2. Base agent prompt
	if s.agent != nil && s.agent.Prompt != "" {
		parts = append(parts, s.agent.Prompt)
	}

	// 3. Model-specific instructions
	if modelPrompt := s.modelPrompt(); modelPrompt != "" {
		parts = append(parts, modelPrompt)
	}

	// 4. Environment context
	parts = append(parts, s.environmentContext())

	// 5. Custom rules (AGENTS.md, CLAUDE.md)
	if rules := s.loadCustomRules(); rules != "" {
		parts = append(parts, rules)
	}

	// 6. Tool instructions
	if toolInstructions := s.toolInstructions(); toolInstructions != "" {
		parts = append(parts, toolInstructions)
	}

	return strings.Join(parts, "\n\n")
}

// providerHeader returns the provider-specific system header.
func (s *SystemPrompt) providerHeader() string {
	switch s.providerID {
	case "anthropic":
		return `You are Claude, an AI assistant made by Anthropic. You are helpful, harmless, and honest.

IMPORTANT: You have access to tools that can read, write, and execute commands on the user's computer. Use them responsibly.`

	case "openai":
		return `You are a helpful AI assistant with access to tools for reading, writing, and executing commands.

Use tools responsibly and follow user instructions carefully.`

	case "google":
		return `You are a helpful AI assistant with tool access.

You can read files, write code, and execute commands to help the user.`

	default:
		return ""
	}
}

// modelPrompt returns model-specific instructions.
func (s *SystemPrompt) modelPrompt() string {
	switch {
	case strings.Contains(s.modelID, "claude"):
		return `When using tools, be decisive and take action. Don't ask for confirmation unless absolutely necessary.

For file operations:
- Read files before editing to understand context
- Make minimal, focused changes
- Preserve existing code style and formatting`

	case strings.Contains(s.modelID, "gpt"):
		return `When working with files:
- Always read files before making changes
- Make precise, targeted edits
- Follow existing code conventions`

	case strings.Contains(s.modelID, "gemini"):
		return `For code tasks:
- Examine existing code structure first
- Make minimal necessary changes
- Maintain code style consistency`

	default:
		return ""
	}
}

// environmentContext returns environment information.
func (s *SystemPrompt) environmentContext() string {
	var env strings.Builder

	env.WriteString("# Environment Information\n\n")

	// Working directory
	workDir := ""
	if s.session != nil {
		workDir = s.session.Directory
	}
	if workDir == "" {
		workDir, _ = os.Getwd()
	}
	env.WriteString(fmt.Sprintf("Working Directory: %s\n", workDir))

	// Current date
	env.WriteString(fmt.Sprintf("Current Date: %s\n", time.Now().Format("2006-01-02")))

	// Platform info
	env.WriteString(fmt.Sprintf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH))

	// Git branch if available
	if branch := s.getGitBranch(workDir); branch != "" {
		env.WriteString(fmt.Sprintf("Git Branch: %s\n", branch))
	}

	// Project type detection
	if projectType := s.detectProjectType(workDir); projectType != "" {
		env.WriteString(fmt.Sprintf("Project Type: %s\n", projectType))
	}

	return env.String()
}

// loadCustomRules loads custom rules from various locations.
func (s *SystemPrompt) loadCustomRules() string {
	workDir := ""
	if s.session != nil {
		workDir = s.session.Directory
	}
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	// Try loading from multiple locations
	locations := []string{
		filepath.Join(workDir, "AGENTS.md"),
		filepath.Join(workDir, "CLAUDE.md"),
		filepath.Join(workDir, ".opencode", "rules.md"),
	}

	// Also check global config
	if home, err := os.UserHomeDir(); err == nil {
		locations = append(locations,
			filepath.Join(home, ".config", "opencode", "rules.md"),
			filepath.Join(home, ".claude", "rules.md"),
		)
	}

	for _, loc := range locations {
		if content, err := os.ReadFile(loc); err == nil && len(content) > 0 {
			return fmt.Sprintf("# Custom Rules\n\n%s", string(content))
		}
	}

	return ""
}

// toolInstructions returns general tool usage guidelines.
func (s *SystemPrompt) toolInstructions() string {
	return `# Tool Usage Guidelines

1. **File Operations**
   - Use the Read tool before editing files
   - Use Edit for surgical changes, Write for new files
   - Always provide absolute paths

2. **Bash Commands**
   - Prefer built-in tools over bash when possible
   - Include a description for every bash command
   - Handle errors gracefully

3. **Search**
   - Use Glob for file discovery
   - Use Grep for content search
   - Be specific with patterns to avoid noise

4. **Best Practices**
   - Work iteratively, verify changes work
   - Don't modify files you haven't read
   - Explain your reasoning before acting`
}

// getGitBranch returns the current git branch.
func (s *SystemPrompt) getGitBranch(dir string) string {
	if dir == "" {
		return ""
	}

	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// detectProjectType detects the project type from files.
func (s *SystemPrompt) detectProjectType(dir string) string {
	if dir == "" {
		return ""
	}

	// Check for common project indicators
	indicators := map[string][]string{
		"Node.js": {"package.json"},
		"Python":  {"pyproject.toml", "setup.py", "requirements.txt"},
		"Go":      {"go.mod"},
		"Rust":    {"Cargo.toml"},
		"Java":    {"pom.xml", "build.gradle"},
		"Ruby":    {"Gemfile"},
		"PHP":     {"composer.json"},
		"C#":      {"*.csproj", "*.sln"},
		"Elixir":  {"mix.exs"},
		"Haskell": {"*.cabal", "stack.yaml"},
	}

	for projectType, files := range indicators {
		for _, pattern := range files {
			matches, _ := filepath.Glob(filepath.Join(dir, pattern))
			if len(matches) > 0 {
				return projectType
			}
		}
	}

	return ""
}

// BuildSystemMessage creates a formatted system message from the prompt.
func (s *SystemPrompt) BuildSystemMessage() string {
	return s.Build()
}

// WithCustomPrompt adds a custom prompt override.
func (s *SystemPrompt) WithCustomPrompt(custom *types.CustomPrompt) *SystemPrompt {
	if custom == nil {
		return s
	}

	switch custom.Type {
	case "file":
		// Load prompt from file
		if content, err := os.ReadFile(custom.Value); err == nil {
			if s.agent == nil {
				s.agent = DefaultAgent()
			}
			s.agent.Prompt = s.replaceVariables(string(content), custom.Variables)
		}
	case "inline":
		// Use inline prompt
		if s.agent == nil {
			s.agent = DefaultAgent()
		}
		s.agent.Prompt = s.replaceVariables(custom.Value, custom.Variables)
	}

	return s
}

// replaceVariables replaces template variables in the prompt.
func (s *SystemPrompt) replaceVariables(prompt string, vars map[string]string) string {
	result := prompt
	for key, value := range vars {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}
