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
