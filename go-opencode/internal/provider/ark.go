package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"

	"github.com/opencode-ai/opencode/pkg/types"
)

// ArkProvider implements Provider for Volcengine ARK models.
type ArkProvider struct {
	chatModel model.ToolCallingChatModel
	models    []types.Model
	config    *ArkConfig
}

// ArkConfig holds configuration for ARK provider.
type ArkConfig struct {
	APIKey    string
	BaseURL   string
	Model     string // Endpoint ID on ARK platform
	MaxTokens int
}

// NewArkProvider creates a new ARK provider.
func NewArkProvider(ctx context.Context, config *ArkConfig) (*ArkProvider, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ARK_API_KEY")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("ARK_API_KEY not set")
	}

	modelID := config.Model
	if modelID == "" {
		modelID = os.Getenv("ARK_MODEL_ID")
	}

	if modelID == "" {
		return nil, fmt.Errorf("ARK_MODEL_ID not set")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = os.Getenv("ARK_BASE_URL")
	}

	maxTokens := config.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	cfg := &ark.ChatModelConfig{
		APIKey:    apiKey,
		Model:     modelID,
		MaxTokens: &maxTokens,
	}

	if baseURL != "" {
		cfg.BaseURL = baseURL
	}

	chatModel, err := ark.NewChatModel(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create ARK model: %w", err)
	}

	return &ArkProvider{
		chatModel: chatModel,
		models:    arkModels(modelID),
		config:    config,
	}, nil
}

// ID returns the provider identifier.
func (p *ArkProvider) ID() string { return "ark" }

// Name returns the human-readable provider name.
func (p *ArkProvider) Name() string { return "ARK" }

// Models returns the list of available models.
func (p *ArkProvider) Models() []types.Model {
	return p.models
}

// ChatModel returns the Eino ChatModel.
func (p *ArkProvider) ChatModel() model.ToolCallingChatModel {
	return p.chatModel
}

// CreateCompletion creates a streaming completion.
func (p *ArkProvider) CreateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionStream, error) {
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

// arkModels returns the list of ARK models.
func arkModels(endpointID string) []types.Model {
	return []types.Model{
		{
			ID:              endpointID,
			Name:            "ARK Model",
			ProviderID:      "ark",
			ContextLength:   128000,
			MaxOutputTokens: 4096,
			SupportsTools:   true,
			SupportsVision:  true,
			InputPrice:      0.0,  // Pricing varies by endpoint
			OutputPrice:     0.0,
		},
	}
}
