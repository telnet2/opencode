package provider

import (
	"context"
	"os"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

func TestAnthropicProvider_Integration(t *testing.T) {
	// Load .env file from project root
	_ = godotenv.Load("../../.env")

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping integration test")
	}

	modelID := os.Getenv("ANTHROPIC_MODEL_ID")
	if modelID == "" {
		modelID = "claude-3-5-haiku-20241022" // Default to Haiku for cheaper testing
	}

	ctx := context.Background()

	// Create provider
	provider, err := NewAnthropicProvider(ctx, &AnthropicConfig{
		APIKey:    apiKey,
		MaxTokens: 1024,
	})
	if err != nil {
		t.Fatalf("Failed to create Anthropic provider: %v", err)
	}

	// Verify provider properties
	if provider.ID() != "anthropic" {
		t.Errorf("Expected ID 'anthropic', got '%s'", provider.ID())
	}
	if provider.Name() != "Anthropic" {
		t.Errorf("Expected Name 'Anthropic', got '%s'", provider.Name())
	}

	models := provider.Models()
	if len(models) == 0 {
		t.Error("Expected at least one model")
	}

	// Test a simple completion
	t.Run("SimpleCompletion", func(t *testing.T) {
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

		t.Logf("Anthropic Response: %s", fullResponse)
	})

	// Test streaming chunks
	t.Run("StreamingChunks", func(t *testing.T) {
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
		t.Logf("Received %d chunks", chunkCount)
	})

	// Test multi-turn conversation
	t.Run("MultiTurnConversation", func(t *testing.T) {
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
		t.Logf("Anthropic Response: %s", fullResponse)
	})

	// Test tool binding
	t.Run("ToolBinding", func(t *testing.T) {
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
	})
}

func TestAnthropicProvider_CustomID(t *testing.T) {
	// Load .env file from project root
	_ = godotenv.Load("../../.env")

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping test")
	}

	ctx := context.Background()

	// Create provider with custom ID
	provider, err := NewAnthropicProvider(ctx, &AnthropicConfig{
		ID:        "claude",
		APIKey:    apiKey,
		MaxTokens: 1024,
	})
	if err != nil {
		t.Fatalf("Failed to create Anthropic provider: %v", err)
	}

	// Verify custom ID
	if provider.ID() != "claude" {
		t.Errorf("Expected ID 'claude', got '%s'", provider.ID())
	}
}

func TestAnthropicProvider_NoAPIKey(t *testing.T) {
	ctx := context.Background()

	// Clear env var temporarily
	originalKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	defer os.Setenv("ANTHROPIC_API_KEY", originalKey)

	// Create provider without API key should fail
	_, err := NewAnthropicProvider(ctx, &AnthropicConfig{
		MaxTokens: 1024,
	})
	if err == nil {
		t.Error("Expected error when API key is not set")
	}
}

func TestAnthropicProvider_EmptyContentHandling(t *testing.T) {
	// Load .env file from project root
	_ = godotenv.Load("../../.env")

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping integration test")
	}

	modelID := os.Getenv("ANTHROPIC_MODEL_ID")
	if modelID == "" {
		modelID = "claude-3-5-haiku-20241022" // Default to Haiku for cheaper testing
	}

	ctx := context.Background()

	provider, err := NewAnthropicProvider(ctx, &AnthropicConfig{
		APIKey:    apiKey,
		MaxTokens: 1024,
	})
	if err != nil {
		t.Fatalf("Failed to create Anthropic provider: %v", err)
	}

	// Test 1: Empty first message content should return an error
	// This reproduces the bug where a user message without content causes:
	// "messages.0.content: Field required"
	t.Run("EmptyFirstMessageContentReturnsError", func(t *testing.T) {
		req := &CompletionRequest{
			Model: modelID,
			Messages: []*schema.Message{
				{
					Role:    schema.User,
					Content: "", // Empty content - should cause error
				},
			},
			MaxTokens:   100,
			Temperature: 0.0,
		}

		stream, err := provider.CreateCompletion(ctx, req)
		if err == nil {
			// If we got a stream, try to read from it - it should fail
			if stream != nil {
				defer stream.Close()
				_, recvErr := stream.Recv()
				if recvErr == nil {
					t.Error("Expected error for empty first message content, but received successful response")
				} else {
					t.Logf("Got expected error on Recv: %v", recvErr)
				}
			}
		} else {
			// Error during CreateCompletion is expected
			t.Logf("Got expected error: %v", err)
			if err.Error() == "" {
				t.Error("Expected non-empty error message")
			}
		}
	})

	// Test 2: Empty first message followed by non-empty message should also fail
	// because Anthropic API requires content in every user message
	t.Run("EmptyFirstMessageWithFollowupReturnsError", func(t *testing.T) {
		req := &CompletionRequest{
			Model: modelID,
			Messages: []*schema.Message{
				{
					Role:    schema.User,
					Content: "", // Empty content - should cause error
				},
				{
					Role:    schema.User,
					Content: "Say hello",
				},
			},
			MaxTokens:   100,
			Temperature: 0.0,
		}

		stream, err := provider.CreateCompletion(ctx, req)
		if err == nil {
			if stream != nil {
				defer stream.Close()
				_, recvErr := stream.Recv()
				if recvErr == nil {
					t.Error("Expected error for empty first message content, but received successful response")
				} else {
					t.Logf("Got expected error on Recv: %v", recvErr)
				}
			}
		} else {
			t.Logf("Got expected error: %v", err)
		}
	})

	// Test 3: Non-empty first message should work correctly
	t.Run("NonEmptyFirstMessageWorks", func(t *testing.T) {
		req := &CompletionRequest{
			Model: modelID,
			Messages: []*schema.Message{
				{
					Role:    schema.User,
					Content: "Say 'test' and nothing else.",
				},
			},
			MaxTokens:   50,
			Temperature: 0.0,
		}

		stream, err := provider.CreateCompletion(ctx, req)
		if err != nil {
			t.Fatalf("Expected no error for non-empty content, got: %v", err)
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
			t.Error("Expected non-empty response for non-empty first message")
		}
		t.Logf("Response: %s", fullResponse)
	})

	// Test 4: Multiple messages with non-empty content should work
	t.Run("MultipleNonEmptyMessagesWork", func(t *testing.T) {
		req := &CompletionRequest{
			Model: modelID,
			Messages: []*schema.Message{
				{
					Role:    schema.User,
					Content: "Remember X=5",
				},
				{
					Role:    schema.Assistant,
					Content: "I'll remember X=5.",
				},
				{
					Role:    schema.User,
					Content: "What is X? Reply with just the number.",
				},
			},
			MaxTokens:   50,
			Temperature: 0.0,
		}

		stream, err := provider.CreateCompletion(ctx, req)
		if err != nil {
			t.Fatalf("Expected no error for conversation with non-empty content, got: %v", err)
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
		t.Logf("Response: %s", fullResponse)
	})
}
