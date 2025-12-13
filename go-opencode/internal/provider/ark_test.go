package provider

import (
	"context"
	"os"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

func TestArkProvider_Integration(t *testing.T) {
	// Load .env file from project root
	_ = godotenv.Load("../../.env")

	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		t.Skip("ARK_API_KEY not set, skipping integration test")
	}

	modelID := os.Getenv("ARK_MODEL_ID")
	if modelID == "" {
		t.Skip("ARK_MODEL_ID not set, skipping integration test")
	}

	baseURL := os.Getenv("ARK_BASE_URL")

	ctx := context.Background()

	// Create provider
	provider, err := NewArkProvider(ctx, &ArkConfig{
		APIKey:    apiKey,
		BaseURL:   baseURL,
		Model:     modelID,
		MaxTokens: 1024,
	})
	if err != nil {
		t.Fatalf("Failed to create ARK provider: %v", err)
	}

	// Verify provider properties
	if provider.ID() != "ark" {
		t.Errorf("Expected ID 'ark', got '%s'", provider.ID())
	}
	if provider.Name() != "ARK" {
		t.Errorf("Expected Name 'ARK', got '%s'", provider.Name())
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

		t.Logf("ARK Response: %s", fullResponse)
	})
}

func TestArkProvider_EmptyContentHandling(t *testing.T) {
	// Load .env file from project root
	_ = godotenv.Load("../../.env")

	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		t.Skip("ARK_API_KEY not set, skipping integration test")
	}

	modelID := os.Getenv("ARK_MODEL_ID")
	if modelID == "" {
		t.Skip("ARK_MODEL_ID not set, skipping integration test")
	}

	baseURL := os.Getenv("ARK_BASE_URL")

	ctx := context.Background()

	provider, err := NewArkProvider(ctx, &ArkConfig{
		APIKey:    apiKey,
		BaseURL:   baseURL,
		Model:     modelID,
		MaxTokens: 1024,
	})
	if err != nil {
		t.Fatalf("Failed to create ARK provider: %v", err)
	}

	// Test 1: Empty first message content - ARK behavior
	// Note: ARK (OpenAI-compatible) may handle empty content differently
	t.Run("EmptyFirstMessageContent", func(t *testing.T) {
		req := &CompletionRequest{
			Model: modelID,
			Messages: []*schema.Message{
				{
					Role:    schema.User,
					Content: "", // Empty content
				},
			},
			MaxTokens:   100,
			Temperature: 0.0,
		}

		stream, err := provider.CreateCompletion(ctx, req)
		if err != nil {
			// ARK may reject empty content with an error
			t.Logf("Got error for empty content (this may be expected): %v", err)
			return
		}
		defer stream.Close()

		// If stream was created, try to receive
		var fullResponse string
		var recvErr error
		for {
			msg, err := stream.Recv()
			if err != nil {
				recvErr = err
				break
			}
			if msg != nil {
				fullResponse += msg.Content
			}
		}

		// Log the behavior for documentation
		if recvErr != nil {
			t.Logf("ARK response to empty content - Error: %v", recvErr)
		} else if fullResponse == "" {
			t.Logf("ARK response to empty content - Empty response")
		} else {
			t.Logf("ARK response to empty content - Got response: %s", fullResponse)
		}
	})

	// Test 2: Non-empty first message should work correctly
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

	// Test 3: Multiple messages with non-empty content should work
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
