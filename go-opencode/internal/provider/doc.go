// Package provider provides LLM provider abstraction layer for OpenCode.
//
// This package implements a unified interface for different Large Language Model
// providers using the Eino framework. It supports multiple providers including
// Anthropic Claude, OpenAI GPT, and Volcengine ARK models.
//
// # Core Components
//
// The package is built around several key interfaces and types:
//
//   - Provider: Core interface that all LLM providers must implement
//   - Registry: Manages and coordinates multiple providers
//   - CompletionRequest/CompletionStream: Handles streaming chat completions
//   - Tool conversion utilities for function calling
//
// # Supported Providers
//
// ## Anthropic (Claude)
//
// Supports Claude models including Claude 4 Sonnet, Claude 4 Opus, and Claude 3.5 series.
// Features include:
//
//   - Direct API access or AWS Bedrock integration
//
//   - Extended thinking support for reasoning tasks
//
//   - Prompt caching for improved performance
//
//   - Vision and tool calling capabilities
//
//     provider, err := NewAnthropicProvider(ctx, &AnthropicConfig{
//     ID:        "anthropic",
//     APIKey:    "sk-...",
//     Model:     "claude-sonnet-4-20250514",
//     MaxTokens: 8192,
//     })
//
// ## OpenAI (GPT)
//
// Supports OpenAI models and OpenAI-compatible endpoints including:
//
//   - Native OpenAI API access
//
//   - Azure OpenAI Service
//
//   - Local and self-hosted OpenAI-compatible servers
//
//     provider, err := NewOpenAIProvider(ctx, &OpenAIConfig{
//     ID:        "openai",
//     APIKey:    "sk-...",
//     Model:     "gpt-4o",
//     MaxTokens: 4096,
//     })
//
// ## Volcengine ARK
//
// Supports Volcengine's ARK platform for accessing Chinese language models:
//
//	provider, err := NewArkProvider(ctx, &ArkConfig{
//	    APIKey:    "...",
//	    Model:     "endpoint-id",
//	    MaxTokens: 4096,
//	})
//
// # Registry Usage
//
// The Registry manages all configured providers and provides unified access:
//
//	registry := NewRegistry(config)
//
//	// Get a specific provider
//	provider, err := registry.Get("anthropic")
//
//	// Get a specific model
//	model, err := registry.GetModel("anthropic", "claude-sonnet-4-20250514")
//
//	// Get default model based on configuration
//	model, err := registry.DefaultModel()
//
//	// List all available models across providers
//	models := registry.AllModels()
//
// # Configuration
//
// Providers can be configured through:
//
//  1. Configuration file with provider sections
//  2. Environment variables (auto-discovery)
//  3. Programmatic registration
//
// Configuration supports npm package mapping for TypeScript compatibility:
//
//	[provider.anthropic]
//	npm = "@ai-sdk/anthropic"
//	model = "claude-sonnet-4-20250514"
//	[provider.anthropic.options]
//	apiKey = "sk-..."
//
// # Streaming Completions
//
// All providers support streaming chat completions through a unified interface:
//
//	stream, err := provider.CreateCompletion(ctx, &CompletionRequest{
//	    Model:     "claude-sonnet-4-20250514",
//	    Messages:  messages,
//	    Tools:     tools,
//	    MaxTokens: 4096,
//	})
//
//	for {
//	    msg, err := stream.Recv()
//	    if err != nil {
//	        break
//	    }
//	    // Process message chunk
//	}
//	stream.Close()
//
// # Tool Calling
//
// The package provides utilities for converting between different tool calling formats:
//
//	// Convert internal tool definitions to Eino format
//	einoTools := ConvertToEinoTools(tools)
//
//	// Convert messages between formats
//	einoMessages := ConvertToEinoMessages(messages, parts)
//
// # Error Handling
//
// The package uses Go's standard error handling patterns. Common error scenarios:
//   - Missing API keys or credentials
//   - Invalid model configurations
//   - Network connectivity issues
//   - Provider-specific API errors
//
// Most functions return meaningful error messages that can be used for debugging
// and user feedback.
//
// # Integration with Eino
//
// This package is built on top of the Eino framework (https://github.com/cloudwego/eino),
// which provides:
//   - Standardized LLM interfaces
//   - Built-in tool calling support
//   - Streaming capabilities
//   - Message schema definitions
//
// The abstraction allows OpenCode to support multiple providers through a single,
// consistent interface while leveraging Eino's robust foundation.
package provider
