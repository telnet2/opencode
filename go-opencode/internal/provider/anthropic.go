package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino-ext/components/model/claude"

	"github.com/opencode-ai/opencode/pkg/types"
)

// AnthropicProvider implements Provider for Anthropic Claude models.
type AnthropicProvider struct {
	chatModel model.ToolCallingChatModel
	models    []types.Model
	config    *AnthropicConfig
}

// AnthropicConfig holds configuration for Anthropic provider.
type AnthropicConfig struct {
	// ID is the provider identifier (e.g., "anthropic", "claude")
	// If empty, defaults to "anthropic"
	ID        string
	APIKey    string
	BaseURL   string
	Model     string // Model ID (e.g., "claude-sonnet-4-20250514", "claude-3-5-haiku-20241022")
	MaxTokens int

	// Extended thinking support
	Thinking *claude.Thinking

	// Bedrock configuration
	UseBedrock bool
	Region     string
	Profile    string
}

// NewAnthropicProvider creates a new Anthropic provider.
func NewAnthropicProvider(ctx context.Context, config *AnthropicConfig) (*AnthropicProvider, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	if apiKey == "" && !config.UseBedrock {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	// Default model if not specified
	modelID := config.Model
	if modelID == "" {
		modelID = "claude-sonnet-4-20250514"
	}

	var chatModel model.ToolCallingChatModel
	var err error

	if config.UseBedrock {
		// Use AWS Bedrock - convert model ID to Bedrock format
		bedrockModel := "anthropic." + modelID + "-v1:0"
		chatModel, err = claude.NewChatModel(ctx, &claude.Config{
			ByBedrock: true,
			Region:    config.Region,
			Profile:   config.Profile,
			Model:     bedrockModel,
			MaxTokens: config.MaxTokens,
			Thinking:  config.Thinking,
		})
	} else {
		// Use direct API
		cfg := &claude.Config{
			APIKey:    apiKey,
			Model:     modelID,
			MaxTokens: config.MaxTokens,
			Thinking:  config.Thinking,
		}
		if config.BaseURL != "" {
			cfg.BaseURL = &config.BaseURL
		}
		chatModel, err = claude.NewChatModel(ctx, cfg)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Claude model: %w", err)
	}

	return &AnthropicProvider{
		chatModel: chatModel,
		models:    anthropicModels(),
		config:    config,
	}, nil
}

// ID returns the provider identifier.
func (p *AnthropicProvider) ID() string {
	if p.config.ID != "" {
		return p.config.ID
	}
	return "anthropic"
}

// Name returns the human-readable provider name.
func (p *AnthropicProvider) Name() string { return "Anthropic" }

// Models returns the list of available models.
func (p *AnthropicProvider) Models() []types.Model {
	return p.models
}

// ChatModel returns the Eino ChatModel.
func (p *AnthropicProvider) ChatModel() model.ToolCallingChatModel {
	return p.chatModel
}

// CreateCompletion creates a streaming completion.
func (p *AnthropicProvider) CreateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionStream, error) {
	// Bind tools if provided
	chatModel := p.chatModel
	if len(req.Tools) > 0 {
		var err error
		chatModel, err = chatModel.WithTools(req.Tools)
		if err != nil {
			return nil, fmt.Errorf("failed to bind tools: %w", err)
		}
	}

	// Create streaming request
	stream, err := chatModel.Stream(ctx, req.Messages,
		model.WithMaxTokens(req.MaxTokens),
		model.WithTemperature(float32(req.Temperature)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	return NewCompletionStream(stream), nil
}

// anthropicModels returns the list of Anthropic models.
func anthropicModels() []types.Model {
	return []types.Model{
		{
			ID:                "claude-sonnet-4-20250514",
			Name:              "Claude Sonnet 4",
			ProviderID:        "anthropic",
			ContextLength:     200000,
			MaxOutputTokens:   64000,
			SupportsTools:     true,
			SupportsVision:    true,
			SupportsReasoning: false,
			InputPrice:        3.0,
			OutputPrice:       15.0,
			Options: types.ModelOptions{
				PromptCaching:  true,
				ExtendedOutput: true,
			},
		},
		{
			ID:                "claude-opus-4-20250514",
			Name:              "Claude Opus 4",
			ProviderID:        "anthropic",
			ContextLength:     200000,
			MaxOutputTokens:   32000,
			SupportsTools:     true,
			SupportsVision:    true,
			SupportsReasoning: true,
			InputPrice:        15.0,
			OutputPrice:       75.0,
			Options: types.ModelOptions{
				PromptCaching: true,
			},
		},
		{
			ID:                "claude-3-5-sonnet-20241022",
			Name:              "Claude 3.5 Sonnet",
			ProviderID:        "anthropic",
			ContextLength:     200000,
			MaxOutputTokens:   8192,
			SupportsTools:     true,
			SupportsVision:    true,
			InputPrice:        3.0,
			OutputPrice:       15.0,
			Options: types.ModelOptions{
				PromptCaching: true,
			},
		},
		{
			ID:                "claude-3-5-haiku-20241022",
			Name:              "Claude 3.5 Haiku",
			ProviderID:        "anthropic",
			ContextLength:     200000,
			MaxOutputTokens:   8192,
			SupportsTools:     true,
			SupportsVision:    true,
			InputPrice:        0.8,
			OutputPrice:       4.0,
		},
		{
			ID:                "claude-haiku-4-5-20251001",
			Name:              "Claude 4.5 Haiku",
			ProviderID:        "anthropic",
			ContextLength:     200000,
			MaxOutputTokens:   8192,
			SupportsTools:     true,
			SupportsVision:    true,
			InputPrice:        0.8,
			OutputPrice:       4.0,
		},
		// Alias for claude-haiku-4-5-20251001
		{
			ID:                "claude-haiku-4-5",
			Name:              "Claude 4.5 Haiku",
			ProviderID:        "anthropic",
			ContextLength:     200000,
			MaxOutputTokens:   8192,
			SupportsTools:     true,
			SupportsVision:    true,
			InputPrice:        0.8,
			OutputPrice:       4.0,
		},
	}
}
