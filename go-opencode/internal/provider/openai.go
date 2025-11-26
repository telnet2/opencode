package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino-ext/components/model/openai"

	"github.com/opencode-ai/opencode/pkg/types"
)

// OpenAIProvider implements Provider for OpenAI models.
type OpenAIProvider struct {
	chatModel model.ToolCallingChatModel
	models    []types.Model
	config    *OpenAIConfig
}

// OpenAIConfig holds configuration for OpenAI provider.
type OpenAIConfig struct {
	APIKey    string
	BaseURL   string
	MaxTokens int

	// Azure configuration
	UseAzure   bool
	APIVersion string
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(ctx context.Context, config *OpenAIConfig) (*OpenAIProvider, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		if config.UseAzure {
			apiKey = os.Getenv("AZURE_OPENAI_API_KEY")
		} else {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
	}

	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not set")
	}

	maxTokens := config.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	cfg := &openai.ChatModelConfig{
		APIKey:    apiKey,
		Model:     "gpt-4o",
		MaxTokens: &maxTokens,
	}

	if config.BaseURL != "" {
		cfg.BaseURL = config.BaseURL
	}

	if config.UseAzure {
		cfg.ByAzure = true
		if config.APIVersion != "" {
			cfg.APIVersion = config.APIVersion
		} else {
			cfg.APIVersion = "2024-02-15-preview"
		}
	}

	chatModel, err := openai.NewChatModel(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI model: %w", err)
	}

	return &OpenAIProvider{
		chatModel: chatModel,
		models:    openAIModels(),
		config:    config,
	}, nil
}

// ID returns the provider identifier.
func (p *OpenAIProvider) ID() string { return "openai" }

// Name returns the human-readable provider name.
func (p *OpenAIProvider) Name() string { return "OpenAI" }

// Models returns the list of available models.
func (p *OpenAIProvider) Models() []types.Model {
	return p.models
}

// ChatModel returns the Eino ChatModel.
func (p *OpenAIProvider) ChatModel() model.ToolCallingChatModel {
	return p.chatModel
}

// CreateCompletion creates a streaming completion.
func (p *OpenAIProvider) CreateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionStream, error) {
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

// openAIModels returns the list of OpenAI models.
func openAIModels() []types.Model {
	return []types.Model{
		{
			ID:              "gpt-4o",
			Name:            "GPT-4o",
			ProviderID:      "openai",
			ContextLength:   128000,
			MaxOutputTokens: 16384,
			SupportsTools:   true,
			SupportsVision:  true,
			InputPrice:      2.5,
			OutputPrice:     10.0,
		},
		{
			ID:              "gpt-4o-mini",
			Name:            "GPT-4o Mini",
			ProviderID:      "openai",
			ContextLength:   128000,
			MaxOutputTokens: 16384,
			SupportsTools:   true,
			SupportsVision:  true,
			InputPrice:      0.15,
			OutputPrice:     0.6,
		},
		{
			ID:                "o1",
			Name:              "O1",
			ProviderID:        "openai",
			ContextLength:     200000,
			MaxOutputTokens:   100000,
			SupportsTools:     true,
			SupportsReasoning: true,
			InputPrice:        15.0,
			OutputPrice:       60.0,
		},
		{
			ID:                "o1-mini",
			Name:              "O1 Mini",
			ProviderID:        "openai",
			ContextLength:     128000,
			MaxOutputTokens:   65536,
			SupportsTools:     true,
			SupportsReasoning: true,
			InputPrice:        1.1,
			OutputPrice:       4.4,
		},
	}
}
