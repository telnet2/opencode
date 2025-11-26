package provider_test

import (
	"context"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
	"github.com/opencode-ai/opencode/internal/provider"
)

func TestProviderSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Provider Suite")
}

var _ = BeforeSuite(func() {
	_ = godotenv.Load("../../.env")
})

var _ = Describe("ArkProvider", func() {
	var (
		ctx        context.Context
		arkProvider *provider.ArkProvider
		apiKey     string
		modelID    string
		baseURL    string
	)

	BeforeEach(func() {
		apiKey = os.Getenv("ARK_API_KEY")
		modelID = os.Getenv("ARK_MODEL_ID")
		baseURL = os.Getenv("ARK_BASE_URL")

		if apiKey == "" || modelID == "" {
			Skip("ARK environment variables not set")
		}

		ctx = context.Background()
		var err error
		arkProvider, err = provider.NewArkProvider(ctx, &provider.ArkConfig{
			APIKey:    apiKey,
			BaseURL:   baseURL,
			Model:     modelID,
			MaxTokens: 1024,
		})
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Provider Properties", func() {
		It("should return correct ID", func() {
			Expect(arkProvider.ID()).To(Equal("ark"))
		})

		It("should return correct Name", func() {
			Expect(arkProvider.Name()).To(Equal("ARK"))
		})

		It("should return at least one model", func() {
			models := arkProvider.Models()
			Expect(len(models)).To(BeNumerically(">", 0))
		})

		It("should have correct provider ID in models", func() {
			models := arkProvider.Models()
			for _, m := range models {
				Expect(m.ProviderID).To(Equal("ark"))
			}
		})

		It("should return chat model", func() {
			chatModel := arkProvider.ChatModel()
			Expect(chatModel).NotTo(BeNil())
		})
	})

	Describe("CreateCompletion", func() {
		Context("Basic Completion", func() {
			It("should return a response for simple prompt", func() {
				req := &provider.CompletionRequest{
					Model: modelID,
					Messages: []*schema.Message{
						{Role: schema.User, Content: "Say 'Hello' and nothing else."},
					},
					MaxTokens:   50,
					Temperature: 0.0,
				}

				stream, err := arkProvider.CreateCompletion(ctx, req)
				Expect(err).NotTo(HaveOccurred())
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

				Expect(fullResponse).NotTo(BeEmpty())
				Expect(strings.ToLower(fullResponse)).To(ContainSubstring("hello"))
			})

			It("should stream response chunks", func() {
				req := &provider.CompletionRequest{
					Model: modelID,
					Messages: []*schema.Message{
						{Role: schema.User, Content: "Count from 1 to 5, one number per line."},
					},
					MaxTokens:   100,
					Temperature: 0.0,
				}

				stream, err := arkProvider.CreateCompletion(ctx, req)
				Expect(err).NotTo(HaveOccurred())
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

				// Should have received multiple chunks
				Expect(chunkCount).To(BeNumerically(">", 0))
			})

			It("should respect max_tokens limit", func() {
				req := &provider.CompletionRequest{
					Model: modelID,
					Messages: []*schema.Message{
						{Role: schema.User, Content: "Write a very long essay about anything."},
					},
					MaxTokens:   10,
					Temperature: 0.0,
				}

				stream, err := arkProvider.CreateCompletion(ctx, req)
				Expect(err).NotTo(HaveOccurred())
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

				// Response should be relatively short due to max_tokens
				// Note: token count != word count, so we use a rough estimate
				Expect(len(fullResponse)).To(BeNumerically("<", 500))
			})
		})

		Context("Multi-turn Conversation", func() {
			It("should handle conversation history", func() {
				req := &provider.CompletionRequest{
					Model: modelID,
					Messages: []*schema.Message{
						{Role: schema.User, Content: "Remember the number 42."},
						{Role: schema.Assistant, Content: "I'll remember the number 42."},
						{Role: schema.User, Content: "What number did I ask you to remember?"},
					},
					MaxTokens:   50,
					Temperature: 0.0,
				}

				stream, err := arkProvider.CreateCompletion(ctx, req)
				Expect(err).NotTo(HaveOccurred())
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

				Expect(fullResponse).To(ContainSubstring("42"))
			})
		})

		Context("Error Handling", func() {
			It("should handle context cancellation", func() {
				cancelCtx, cancel := context.WithCancel(ctx)
				cancel() // Cancel immediately

				req := &provider.CompletionRequest{
					Model: modelID,
					Messages: []*schema.Message{
						{Role: schema.User, Content: "Hello"},
					},
					MaxTokens: 50,
				}

				_, err := arkProvider.CreateCompletion(cancelCtx, req)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Tool Binding", func() {
		It("should bind tools without error", func() {
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

			chatModel := arkProvider.ChatModel()
			boundModel, err := chatModel.WithTools(tools)
			Expect(err).NotTo(HaveOccurred())
			Expect(boundModel).NotTo(BeNil())
		})
	})
})

var _ = Describe("Provider Initialization", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	Context("with invalid configuration", func() {
		It("should fail with empty API key when env var not set", func() {
			// Temporarily unset env vars
			oldKey := os.Getenv("ARK_API_KEY")
			oldModel := os.Getenv("ARK_MODEL_ID")
			os.Unsetenv("ARK_API_KEY")
			os.Unsetenv("ARK_MODEL_ID")
			defer func() {
				if oldKey != "" {
					os.Setenv("ARK_API_KEY", oldKey)
				}
				if oldModel != "" {
					os.Setenv("ARK_MODEL_ID", oldModel)
				}
			}()

			_, err := provider.NewArkProvider(ctx, &provider.ArkConfig{
				APIKey:  "",
				Model:   "test-model",
				BaseURL: "https://example.com",
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("API_KEY"))
		})

		It("should fail with empty model ID when env var not set", func() {
			// Temporarily unset env vars
			oldKey := os.Getenv("ARK_API_KEY")
			oldModel := os.Getenv("ARK_MODEL_ID")
			os.Unsetenv("ARK_API_KEY")
			os.Unsetenv("ARK_MODEL_ID")
			defer func() {
				if oldKey != "" {
					os.Setenv("ARK_API_KEY", oldKey)
				}
				if oldModel != "" {
					os.Setenv("ARK_MODEL_ID", oldModel)
				}
			}()

			_, err := provider.NewArkProvider(ctx, &provider.ArkConfig{
				APIKey:  "test-key",
				Model:   "",
				BaseURL: "https://example.com",
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("MODEL_ID"))
		})
	})

	Context("with environment variables", func() {
		It("should read API key from environment", func() {
			apiKey := os.Getenv("ARK_API_KEY")
			modelID := os.Getenv("ARK_MODEL_ID")

			if apiKey == "" || modelID == "" {
				Skip("ARK environment variables not set")
			}

			// Create with empty config - should read from env
			p, err := provider.NewArkProvider(ctx, &provider.ArkConfig{})
			Expect(err).NotTo(HaveOccurred())
			Expect(p).NotTo(BeNil())
		})
	})
})
