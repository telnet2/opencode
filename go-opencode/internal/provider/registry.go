package provider

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/opencode-ai/opencode/pkg/types"
)

// Registry manages all available providers.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
	config    *types.Config
}

// NewRegistry creates a new provider registry.
func NewRegistry(config *types.Config) *Registry {
	return &Registry{
		providers: make(map[string]Provider),
		config:    config,
	}
}

// Register adds a provider to the registry.
func (r *Registry) Register(provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.ID()] = provider
}

// Get retrieves a provider by ID.
func (r *Registry) Get(providerID string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, ok := r.providers[providerID]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", providerID)
	}
	return provider, nil
}

// List returns all available providers.
func (r *Registry) List() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	return providers
}

// GetModel retrieves a specific model from a provider.
func (r *Registry) GetModel(providerID, modelID string) (*types.Model, error) {
	provider, err := r.Get(providerID)
	if err != nil {
		return nil, err
	}

	for _, model := range provider.Models() {
		if model.ID == modelID {
			return &model, nil
		}
	}

	return nil, fmt.Errorf("model not found: %s/%s", providerID, modelID)
}

// AllModels returns all models from all providers.
func (r *Registry) AllModels() []types.Model {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var models []types.Model
	for _, p := range r.providers {
		models = append(models, p.Models()...)
	}

	// Sort by quality/priority
	sort.Slice(models, func(i, j int) bool {
		return modelPriority(models[i].ID) > modelPriority(models[j].ID)
	})

	return models
}

// DefaultModel returns the default model.
func (r *Registry) DefaultModel() (*types.Model, error) {
	if r.config != nil && r.config.Model != "" {
		providerID, modelID := ParseModelString(r.config.Model)
		return r.GetModel(providerID, modelID)
	}

	// Default to Claude Sonnet if available
	model, err := r.GetModel("anthropic", "claude-sonnet-4-20250514")
	if err == nil {
		return model, nil
	}

	// Fall back to first available model
	models := r.AllModels()
	if len(models) == 0 {
		return nil, fmt.Errorf("no models available")
	}
	return &models[0], nil
}

// ParseModelString parses "provider/model" format.
func ParseModelString(s string) (providerID, modelID string) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", s
}

// modelPriority returns sorting priority for models.
func modelPriority(modelID string) int {
	switch {
	case strings.Contains(modelID, "gpt-5"):
		return 100
	case strings.Contains(modelID, "claude-sonnet-4"):
		return 90
	case strings.Contains(modelID, "claude-opus"):
		return 85
	case strings.Contains(modelID, "gpt-4o"):
		return 80
	case strings.Contains(modelID, "claude-3-5"):
		return 75
	case strings.Contains(modelID, "gemini-2"):
		return 70
	default:
		return 50
	}
}

// Npm package to provider type mapping
const (
	NpmOpenAI           = "@ai-sdk/openai"
	NpmOpenAICompatible = "@ai-sdk/openai-compatible"
	NpmAnthropic        = "@ai-sdk/anthropic"
)

// InitializeProviders creates and registers all providers from config.
func InitializeProviders(ctx context.Context, config *types.Config) (*Registry, error) {
	registry := NewRegistry(config)

	// Track which providers are configured
	configuredProviders := make(map[string]bool)

	// Iterate through all configured providers
	for name, cfg := range config.Provider {
		if cfg.Disable {
			continue
		}

		configuredProviders[name] = true
		apiKey, baseURL := getProviderCredentials(cfg)

		// Determine provider type from npm field or provider name
		npm := cfg.Npm
		if npm == "" {
			// Fall back to provider name for backward compatibility
			npm = inferNpmFromProviderName(name)
		}

		var provider Provider
		var err error

		switch npm {
		case NpmAnthropic:
			if apiKey != "" {
				provider, err = NewAnthropicProvider(ctx, &AnthropicConfig{
					ID:        name,
					APIKey:    apiKey,
					BaseURL:   baseURL,
					Model:     cfg.Model,
					MaxTokens: 8192,
				})
			}

		case NpmOpenAI, NpmOpenAICompatible:
			// OpenAI and OpenAI-compatible use the same provider with different baseURL
			if apiKey != "" || baseURL != "" {
				// OpenAI-compatible may not require API key for local models
				provider, err = NewOpenAIProvider(ctx, &OpenAIConfig{
					ID:        name,
					APIKey:    apiKey,
					BaseURL:   baseURL,
					Model:     cfg.Model,
					MaxTokens: 4096,
				})
			}

		default:
			// Try to infer from well-known provider names
			switch name {
			case "ark":
				if apiKey != "" {
					provider, err = NewArkProvider(ctx, &ArkConfig{
						APIKey:    apiKey,
						BaseURL:   baseURL,
						Model:     cfg.Model,
						MaxTokens: 4096,
					})
				}
			}
		}

		if err != nil {
			// Log error but continue with other providers
			continue
		}
		if provider != nil {
			registry.Register(provider)
		}
	}

	// Auto-register default providers from environment variables if not already configured
	if !configuredProviders["anthropic"] {
		if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
			fmt.Printf("[provider] Auto-registering anthropic provider from ANTHROPIC_API_KEY\n")
			provider, err := NewAnthropicProvider(ctx, &AnthropicConfig{
				ID:        "anthropic",
				APIKey:    apiKey,
				MaxTokens: 8192,
			})
			if err != nil {
				fmt.Printf("[provider] Failed to create anthropic provider: %v\n", err)
			} else if provider != nil {
				registry.Register(provider)
				fmt.Printf("[provider] Successfully registered anthropic provider\n")
			}
		} else {
			fmt.Printf("[provider] ANTHROPIC_API_KEY not set\n")
		}
	}

	if !configuredProviders["openai"] {
		if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
			provider, err := NewOpenAIProvider(ctx, &OpenAIConfig{
				ID:        "openai",
				APIKey:    apiKey,
				MaxTokens: 4096,
			})
			if err == nil && provider != nil {
				registry.Register(provider)
			}
		}
	}

	return registry, nil
}

// inferNpmFromProviderName maps well-known provider names to npm packages.
func inferNpmFromProviderName(name string) string {
	switch name {
	case "anthropic", "claude":
		return NpmAnthropic
	case "openai":
		return NpmOpenAI
	default:
		return ""
	}
}

// getProviderCredentials extracts API key and base URL from provider config.
func getProviderCredentials(cfg types.ProviderConfig) (apiKey, baseURL string) {
	if cfg.Options != nil {
		apiKey = cfg.Options.APIKey
		baseURL = cfg.Options.BaseURL
	}
	return apiKey, baseURL
}
