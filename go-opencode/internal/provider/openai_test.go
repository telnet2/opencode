package provider

import (
	"context"
	"os"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

func TestOpenAIProvider_Integration(t *testing.T) {
	// Load .env file from project root
	_ = godotenv.Load("../../.env")

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	modelID := os.Getenv("OPENAI_MODEL_ID")
	if modelID == "" {
		modelID = "gpt-4o-mini" // Default to gpt-4o-mini for cheaper testing
	}

	ctx := context.Background()

	// Create provider
	provider, err := NewOpenAIProvider(ctx, &OpenAIConfig{
		APIKey:    apiKey,
		Model:     modelID,
		MaxTokens: 1024,
	})
	if err != nil {
		t.Fatalf("Failed to create OpenAI provider: %v", err)
	}

	// Verify provider properties
	if provider.ID() != "openai" {
		t.Errorf("Expected ID 'openai', got '%s'", provider.ID())
	}
	if provider.Name() != "OpenAI" {
		t.Errorf("Expected Name 'OpenAI', got '%s'", provider.Name())
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
			MaxTokens: 100,
			// Note: GPT-5 models don't accept custom temperature (fixed at 1)
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

		t.Logf("OpenAI Response: %s", fullResponse)
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
			MaxTokens: 100,
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
			MaxTokens: 50,
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
		t.Logf("OpenAI Response: %s", fullResponse)
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
