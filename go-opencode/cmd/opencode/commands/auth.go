package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage provider credentials",
	Long: `Manage authentication credentials for AI providers.

Subcommands:
  list     List all configured providers and their status
  login    Log in to a provider
  logout   Log out from a provider`,
}

var authListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all providers and their status",
	RunE:    runAuthList,
}

var authLoginCmd = &cobra.Command{
	Use:   "login [provider]",
	Short: "Log in to a provider",
	Long: `Log in to a provider by providing an API key.

Supported providers:
  anthropic    Anthropic (Claude)
  openai       OpenAI (GPT-4, etc.)
  google       Google AI (Gemini)`,
	RunE: runAuthLogin,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout [provider]",
	Short: "Log out from a provider",
	RunE:  runAuthLogout,
}

func init() {
	authCmd.AddCommand(authListCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
}

// Auth represents stored authentication data
type Auth struct {
	Providers map[string]AuthProvider `json:"providers"`
}

type AuthProvider struct {
	APIKey string `json:"apiKey,omitempty"`
}

func runAuthList(cmd *cobra.Command, args []string) error {
	paths := config.GetPaths()

	// Load auth file
	auth, _ := loadAuth(paths.AuthPath())

	// Known providers and their environment variables
	providers := map[string]string{
		"anthropic": "ANTHROPIC_API_KEY",
		"openai":    "OPENAI_API_KEY",
		"google":    "GOOGLE_API_KEY",
		"bedrock":   "AWS_ACCESS_KEY_ID",
	}

	fmt.Println("Provider Authentication Status:")
	fmt.Println()

	for provider, envVar := range providers {
		status := "not configured"

		// Check environment variable
		if os.Getenv(envVar) != "" {
			status = fmt.Sprintf("configured (via %s)", envVar)
		}

		// Check auth file
		if auth != nil && auth.Providers != nil {
			if p, ok := auth.Providers[provider]; ok && p.APIKey != "" {
				status = "configured (via auth file)"
			}
		}

		fmt.Printf("  %-12s %s\n", provider, status)
	}

	fmt.Println()
	fmt.Printf("Auth file: %s\n", paths.AuthPath())

	return nil
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("provider name required. Use: opencode auth login <provider>")
	}

	provider := args[0]
	paths := config.GetPaths()

	// Prompt for API key
	fmt.Printf("Enter API key for %s: ", provider)
	reader := bufio.NewReader(os.Stdin)
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Load existing auth
	auth, _ := loadAuth(paths.AuthPath())
	if auth == nil {
		auth = &Auth{Providers: make(map[string]AuthProvider)}
	}
	if auth.Providers == nil {
		auth.Providers = make(map[string]AuthProvider)
	}

	// Save API key
	auth.Providers[provider] = AuthProvider{APIKey: apiKey}

	if err := saveAuth(paths.AuthPath(), auth); err != nil {
		return fmt.Errorf("failed to save auth: %w", err)
	}

	fmt.Printf("Successfully logged in to %s\n", provider)
	return nil
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("provider name required. Use: opencode auth logout <provider>")
	}

	provider := args[0]
	paths := config.GetPaths()

	// Load existing auth
	auth, err := loadAuth(paths.AuthPath())
	if err != nil {
		return fmt.Errorf("no auth file found")
	}

	if auth.Providers == nil {
		return fmt.Errorf("not logged in to %s", provider)
	}

	if _, ok := auth.Providers[provider]; !ok {
		return fmt.Errorf("not logged in to %s", provider)
	}

	// Remove provider
	delete(auth.Providers, provider)

	if err := saveAuth(paths.AuthPath(), auth); err != nil {
		return fmt.Errorf("failed to save auth: %w", err)
	}

	fmt.Printf("Successfully logged out from %s\n", provider)
	return nil
}

func loadAuth(path string) (*Auth, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var auth Auth
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, err
	}

	return &auth, nil
}

func saveAuth(path string, auth *Auth) error {
	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(config.GetPaths().Data, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
