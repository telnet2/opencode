package provider

import (
	"context"
	"os"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"

	"github.com/opencode-ai/opencode/pkg/types"
)

// ProviderTestConfig defines a provider configuration for table-driven tests
type ProviderTestConfig struct {
	Name           string                        // Test name
	ProviderID     string                        // Provider ID in registry
	Npm            string                        // NPM package type
	APIKeyEnv      string                        // Env var for API key
	BaseURLEnv     string                        // Env var for base URL (optional)
	ModelIDEnv     string                        // Env var for model ID
	DefaultModelID string                        // Default model if env not set
	SkipToolTest   bool                          // Some providers don't support tools well
}

// providerTestConfigs defines all providers to test via registry
var providerTestConfigs = []ProviderTestConfig{
	{
		Name:           "Anthropic",
		ProviderID:     "anthropic",
		Npm:            NpmAnthropic,
		APIKeyEnv:      "ANTHROPIC_API_KEY",
		ModelIDEnv:     "ANTHROPIC_MODEL_ID",
		DefaultModelID: "claude-3-5-haiku-20241022",
	},
	{
		Name:           "OpenAI",
		ProviderID:     "openai",
		Npm:            NpmOpenAI,
		APIKeyEnv:      "OPENAI_API_KEY",
		BaseURLEnv:     "OPENAI_BASE_URL",
		ModelIDEnv:     "OPENAI_MODEL_ID",
		DefaultModelID: "gpt-4o-mini",
	},
	{
		Name:           "ARK",
		ProviderID:     "ark",
		Npm:            "", // ARK uses custom handler
		APIKeyEnv:      "ARK_API_KEY",
		BaseURLEnv:     "ARK_BASE_URL",
		ModelIDEnv:     "ARK_MODEL_ID",
		DefaultModelID: "",
		SkipToolTest:   true, // ARK may have limited tool support
	},
}

func TestRegistry_LLMIntegration(t *testing.T) {
	// Load .env file from project root
	_ = godotenv.Load("../../.env")

	for _, tc := range providerTestConfigs {
		tc := tc // capture range variable
		t.Run(tc.Name, func(t *testing.T) {
			// Check if API key is set
			apiKey := os.Getenv(tc.APIKeyEnv)
			if apiKey == "" {
				t.Skipf("%s not set, skipping %s integration test", tc.APIKeyEnv, tc.Name)
			}

			// Get model ID
			modelID := os.Getenv(tc.ModelIDEnv)
			if modelID == "" {
				if tc.DefaultModelID == "" {
					t.Skipf("%s not set and no default, skipping %s test", tc.ModelIDEnv, tc.Name)
				}
				modelID = tc.DefaultModelID
			}

			// Build config for registry
			config := buildTestConfig(tc)

			ctx := context.Background()

			// Initialize providers via registry
			registry, err := InitializeProviders(ctx, config)
			if err != nil {
				t.Fatalf("Failed to initialize providers: %v", err)
			}

			// Get provider from registry
			provider, err := registry.Get(tc.ProviderID)
			if err != nil {
				t.Fatalf("Failed to get provider %s from registry: %v", tc.ProviderID, err)
			}

			// Run integration subtests
			runProviderIntegrationTests(t, provider, modelID, tc.SkipToolTest)
		})
	}
}

// buildTestConfig creates a types.Config for the given provider test config
func buildTestConfig(tc ProviderTestConfig) *types.Config {
	apiKey := os.Getenv(tc.APIKeyEnv)
	baseURL := ""
	if tc.BaseURLEnv != "" {
		baseURL = os.Getenv(tc.BaseURLEnv)
	}
	modelID := os.Getenv(tc.ModelIDEnv)
	if modelID == "" {
		modelID = tc.DefaultModelID
	}

	providerConfig := types.ProviderConfig{
		Npm:   tc.Npm,
		Model: modelID,
		Options: &types.ProviderOptions{
			APIKey:  apiKey,
			BaseURL: baseURL,
		},
	}

	return &types.Config{
		Model: tc.ProviderID + "/" + modelID,
		Provider: map[string]types.ProviderConfig{
			tc.ProviderID: providerConfig,
		},
	}
}

// runProviderIntegrationTests runs the standard integration test suite on a provider
func runProviderIntegrationTests(t *testing.T, provider Provider, modelID string, skipToolTest bool) {
	ctx := context.Background()

	// Verify provider properties
	if provider.ID() == "" {
		t.Error("Expected non-empty provider ID")
	}
	if provider.Name() == "" {
		t.Error("Expected non-empty provider name")
	}

	t.Run("SimpleCompletion", func(t *testing.T) {
		testSimpleCompletion(t, ctx, provider, modelID)
	})

	t.Run("StreamingChunks", func(t *testing.T) {
		testStreamingChunks(t, ctx, provider, modelID)
	})

	t.Run("MultiTurnConversation", func(t *testing.T) {
		testMultiTurnConversation(t, ctx, provider, modelID)
	})

	if !skipToolTest {
		t.Run("ToolBinding", func(t *testing.T) {
			testToolBinding(t, provider)
		})
	}
}

func testSimpleCompletion(t *testing.T, ctx context.Context, provider Provider, modelID string) {
	req := &CompletionRequest{
		Model: modelID,
		Messages: []*schema.Message{
			{
				Role:    schema.User,
				Content: "Say 'Hello, World!' and nothing else.",
			},
		},
		MaxTokens:   100,
		Temperature: 0.0,
	}

	stream, err := provider.CreateCompletion(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create completion: %v", err)
	}
	defer stream.Close()

	var fullResponse string
	for {
		msg, err := stream.Recv()
		if err != nil {
			break
		}
		if msg != nil {
			fullResponse += msg.Content
		}
	}

	if fullResponse == "" {
		t.Error("Expected non-empty response")
	}

	t.Logf("[%s] Response: %s", provider.Name(), fullResponse)
}

func testStreamingChunks(t *testing.T, ctx context.Context, provider Provider, modelID string) {
	req := &CompletionRequest{
		Model: modelID,
		Messages: []*schema.Message{
			{
				Role:    schema.User,
				Content: "Count from 1 to 5, one number per line.",
			},
		},
		MaxTokens:   100,
		Temperature: 0.0,
	}

	stream, err := provider.CreateCompletion(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create completion: %v", err)
	}
	defer stream.Close()

	chunkCount := 0
	for {
		msg, err := stream.Recv()
		if err != nil {
			break
		}
		if msg != nil {
			chunkCount++
		}
	}

	if chunkCount == 0 {
		t.Error("Expected to receive at least one chunk")
	}
	t.Logf("[%s] Received %d chunks", provider.Name(), chunkCount)
}

func testMultiTurnConversation(t *testing.T, ctx context.Context, provider Provider, modelID string) {
	req := &CompletionRequest{
		Model: modelID,
		Messages: []*schema.Message{
			{Role: schema.User, Content: "Remember the number 42."},
			{Role: schema.Assistant, Content: "I'll remember the number 42."},
			{Role: schema.User, Content: "What number did I ask you to remember? Reply with just the number."},
		},
		MaxTokens:   50,
		Temperature: 0.0,
	}

	stream, err := provider.CreateCompletion(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create completion: %v", err)
	}
	defer stream.Close()

	var fullResponse string
	for {
		msg, err := stream.Recv()
		if err != nil {
			break
		}
		if msg != nil {
			fullResponse += msg.Content
		}
	}

	if fullResponse == "" {
		t.Error("Expected non-empty response")
	}
	t.Logf("[%s] Response: %s", provider.Name(), fullResponse)
}

func testToolBinding(t *testing.T, provider Provider) {
	tools := []*schema.ToolInfo{
		{
			Name: "calculator",
			Desc: "Performs arithmetic calculations",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"expression": {
					Type: schema.String,
					Desc: "The mathematical expression to evaluate",
				},
			}),
		},
	}

	chatModel := provider.ChatModel()
	boundModel, err := chatModel.WithTools(tools)
	if err != nil {
		t.Fatalf("Failed to bind tools: %v", err)
	}
	if boundModel == nil {
		t.Error("Expected non-nil bound model")
	}
}

// TestRegistry_MultiProvider tests multiple providers in a single registry
func TestRegistry_MultiProvider(t *testing.T) {
	_ = godotenv.Load("../../.env")

	// Build config with all available providers
	config := &types.Config{
		Provider: make(map[string]types.ProviderConfig),
	}

	availableProviders := []string{}

	for _, tc := range providerTestConfigs {
		apiKey := os.Getenv(tc.APIKeyEnv)
		if apiKey == "" {
			continue
		}

		modelID := os.Getenv(tc.ModelIDEnv)
		if modelID == "" {
			modelID = tc.DefaultModelID
		}
		if modelID == "" {
			continue
		}

		baseURL := ""
		if tc.BaseURLEnv != "" {
			baseURL = os.Getenv(tc.BaseURLEnv)
		}

		config.Provider[tc.ProviderID] = types.ProviderConfig{
			Npm:   tc.Npm,
			Model: modelID,
			Options: &types.ProviderOptions{
				APIKey:  apiKey,
				BaseURL: baseURL,
			},
		}
		availableProviders = append(availableProviders, tc.ProviderID)
	}

	if len(availableProviders) == 0 {
		t.Skip("No provider API keys configured, skipping multi-provider test")
	}

	ctx := context.Background()

	// Initialize all providers at once
	registry, err := InitializeProviders(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize providers: %v", err)
	}

	// Verify all expected providers are registered
	providers := registry.List()
	t.Logf("Registered %d providers: %v", len(providers), availableProviders)

	if len(providers) != len(availableProviders) {
		t.Errorf("Expected %d providers, got %d", len(availableProviders), len(providers))
	}

	// Verify each provider can be retrieved
	for _, providerID := range availableProviders {
		provider, err := registry.Get(providerID)
		if err != nil {
			t.Errorf("Failed to get provider %s: %v", providerID, err)
			continue
		}
		t.Logf("Provider %s: ID=%s, Name=%s, Models=%d",
			providerID, provider.ID(), provider.Name(), len(provider.Models()))
	}
}
