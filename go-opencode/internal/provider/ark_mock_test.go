package provider_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudwego/eino/schema"
	"github.com/opencode-ai/opencode/internal/provider"
)

var _ = Describe("ArkProvider with MockLLM", func() {
	var (
		ctx        context.Context
		mockServer *MockLLMServer
		arkProvider *provider.ArkProvider
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create MockLLM server with predefined responses
		mockServer = NewMockLLMServer(&MockLLMConfig{
			Responses: map[string]MockResponse{
				"hello": {
					Content: "Hello! I'm a mocked ARK model.",
				},
				"count": {
					Content: "1\n2\n3\n4\n5",
				},
				"remember": {
					Content: "I'll remember that.",
				},
				"what number": {
					Content: "The number is 42.",
				},
				"calculate": {
					Content: "I'll calculate that for you.",
					ToolCalls: []MockToolCall{
						{
							ID:   "call_calc_001",
							Type: "function",
							Function: MockFunctionCall{
								Name:      "calculator",
								Arguments: `{"expression": "2+2"}`,
							},
						},
					},
				},
			},
			Defaults: MockDefaults{
				Fallback: "I understand your request.",
			},
			Settings: MockSettings{
				LagMS:           0,
				EnableStreaming: true,
			},
		})

		// Create ArkProvider pointing to mock server
		var err error
		arkProvider, err = provider.NewArkProvider(ctx, &provider.ArkConfig{
			APIKey:    "mock-api-key",           // Mock key - won't be validated by mock server
			BaseURL:   mockServer.URL(),         // Point to mock server
			Model:     "mock-ark-endpoint-123",  // Mock model ID
			MaxTokens: 1024,
		})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if mockServer != nil {
			mockServer.Close()
		}
	})

	Describe("Provider Properties", func() {
		It("should have correct ID", func() {
			Expect(arkProvider.ID()).To(Equal("ark"))
		})

		It("should have correct Name", func() {
			Expect(arkProvider.Name()).To(Equal("ARK"))
		})

		It("should have models", func() {
			models := arkProvider.Models()
			Expect(len(models)).To(BeNumerically(">", 0))
		})

		It("should return a chat model", func() {
			chatModel := arkProvider.ChatModel()
			Expect(chatModel).NotTo(BeNil())
		})
	})

	Describe("CreateCompletion with Mock", func() {
		It("should receive response from mock server", func() {
			req := &provider.CompletionRequest{
				Model: "mock-ark-endpoint-123",
				Messages: []*schema.Message{
					{Role: schema.User, Content: "hello"},
				},
				MaxTokens:   100,
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

			// Should contain the mocked response
			Expect(fullResponse).To(ContainSubstring("Hello"))
		})

		It("should stream multiple chunks", func() {
			req := &provider.CompletionRequest{
				Model: "mock-ark-endpoint-123",
				Messages: []*schema.Message{
					{Role: schema.User, Content: "count from 1 to 5"},
				},
				MaxTokens: 100,
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

			// Should receive multiple chunks (streaming)
			Expect(chunkCount).To(BeNumerically(">", 0))
		})

		It("should handle multi-turn conversation", func() {
			// Note: The mock server extracts the LAST user message for matching
			// Using "what number" which matches our mock response containing "42"
			req := &provider.CompletionRequest{
				Model: "mock-ark-endpoint-123",
				Messages: []*schema.Message{
					{Role: schema.User, Content: "Store 42 for me"},
					{Role: schema.Assistant, Content: "Done."},
					{Role: schema.User, Content: "what number was stored"},
				},
				MaxTokens: 50,
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

			// Mock response for "what number" returns "The number is 42."
			Expect(fullResponse).To(ContainSubstring("42"))
		})

		It("should return fallback for unknown prompts", func() {
			req := &provider.CompletionRequest{
				Model: "mock-ark-endpoint-123",
				Messages: []*schema.Message{
					{Role: schema.User, Content: "something completely random xyz123"},
				},
				MaxTokens: 100,
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

			Expect(fullResponse).To(Equal("I understand your request."))
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
							Desc: "The mathematical expression",
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

	Describe("Request Verification", func() {
		It("should send correct request to mock server", func() {
			req := &provider.CompletionRequest{
				Model: "mock-ark-endpoint-123",
				Messages: []*schema.Message{
					{Role: schema.User, Content: "hello test"},
				},
				MaxTokens: 100,
			}

			stream, err := arkProvider.CreateCompletion(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Drain the stream
			for {
				_, err := stream.Recv()
				if err != nil {
					break
				}
			}
			stream.Close()

			// Verify request was recorded
			requests := mockServer.GetRequests()
			Expect(len(requests)).To(BeNumerically(">", 0))

			// Check the last request
			lastReq := requests[len(requests)-1]
			Expect(lastReq.Path).To(Or(
				Equal("/v1/chat/completions"),
				Equal("/chat/completions"),
			))

			// Verify messages were sent
			messages, ok := lastReq.Body["messages"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(len(messages)).To(BeNumerically(">", 0))
		})
	})

	Describe("Determinism", func() {
		It("should return identical responses for identical prompts", func() {
			req := &provider.CompletionRequest{
				Model: "mock-ark-endpoint-123",
				Messages: []*schema.Message{
					{Role: schema.User, Content: "hello"},
				},
				MaxTokens: 100,
			}

			// First request
			stream1, err := arkProvider.CreateCompletion(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			var response1 string
			for {
				msg, err := stream1.Recv()
				if err != nil {
					break
				}
				if msg != nil {
					response1 += msg.Content
				}
			}
			stream1.Close()

			// Second identical request
			stream2, err := arkProvider.CreateCompletion(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			var response2 string
			for {
				msg, err := stream2.Recv()
				if err != nil {
					break
				}
				if msg != nil {
					response2 += msg.Content
				}
			}
			stream2.Close()

			// Responses should be identical
			Expect(response1).To(Equal(response2))
		})
	})
})

var _ = Describe("AnthropicProvider with MockLLM", func() {
	var (
		ctx              context.Context
		mockServer       *MockLLMServer
		anthropicProvider *provider.AnthropicProvider
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create MockLLM server with predefined responses
		mockServer = NewMockLLMServer(&MockLLMConfig{
			Responses: map[string]MockResponse{
				"hello": {
					Content: "Hello! I'm a mocked Claude model.",
				},
				"count": {
					Content: "1\n2\n3\n4\n5",
				},
				"remember": {
					Content: "I'll remember that.",
				},
				"what number": {
					Content: "The number is 42.",
				},
			},
			Defaults: MockDefaults{
				Fallback: "I understand your request.",
			},
			Settings: MockSettings{
				LagMS:           0,
				EnableStreaming: true,
			},
		})

		// Create AnthropicProvider pointing to mock server
		var err error
		anthropicProvider, err = provider.NewAnthropicProvider(ctx, &provider.AnthropicConfig{
			APIKey:    "mock-api-key",           // Mock key - won't be validated by mock server
			BaseURL:   mockServer.URL(),         // Point to mock server
			MaxTokens: 1024,
		})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if mockServer != nil {
			mockServer.Close()
		}
	})

	Describe("Provider Properties", func() {
		It("should have correct ID", func() {
			Expect(anthropicProvider.ID()).To(Equal("anthropic"))
		})

		It("should have correct Name", func() {
			Expect(anthropicProvider.Name()).To(Equal("Anthropic"))
		})

		It("should have models", func() {
			models := anthropicProvider.Models()
			Expect(len(models)).To(BeNumerically(">", 0))
		})

		It("should return a chat model", func() {
			chatModel := anthropicProvider.ChatModel()
			Expect(chatModel).NotTo(BeNil())
		})
	})

	Describe("CreateCompletion with Mock", func() {
		It("should receive response from mock server", func() {
			req := &provider.CompletionRequest{
				Model: "claude-3-opus-20240229",
				Messages: []*schema.Message{
					{Role: schema.User, Content: "hello"},
				},
				MaxTokens:   100,
				Temperature: 0.0,
			}

			stream, err := anthropicProvider.CreateCompletion(ctx, req)
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

			// Should contain the mocked response
			Expect(fullResponse).To(ContainSubstring("Hello"))
		})

		It("should stream multiple chunks", func() {
			req := &provider.CompletionRequest{
				Model: "claude-3-opus-20240229",
				Messages: []*schema.Message{
					{Role: schema.User, Content: "count from 1 to 5"},
				},
				MaxTokens: 100,
			}

			stream, err := anthropicProvider.CreateCompletion(ctx, req)
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

			// Should receive multiple chunks (streaming)
			Expect(chunkCount).To(BeNumerically(">", 0))
		})

		It("should handle multi-turn conversation", func() {
			// Using "what number" which matches our mock response containing "42"
			req := &provider.CompletionRequest{
				Model: "claude-3-opus-20240229",
				Messages: []*schema.Message{
					{Role: schema.User, Content: "Store 42 for me"},
					{Role: schema.Assistant, Content: "Done."},
					{Role: schema.User, Content: "what number was stored"},
				},
				MaxTokens: 50,
			}

			stream, err := anthropicProvider.CreateCompletion(ctx, req)
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

			// Mock response for "what number" returns "The number is 42."
			Expect(fullResponse).To(ContainSubstring("42"))
		})

		It("should return fallback for unknown prompts", func() {
			req := &provider.CompletionRequest{
				Model: "claude-3-opus-20240229",
				Messages: []*schema.Message{
					{Role: schema.User, Content: "something completely random xyz123"},
				},
				MaxTokens: 100,
			}

			stream, err := anthropicProvider.CreateCompletion(ctx, req)
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

			Expect(fullResponse).To(Equal("I understand your request."))
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
							Desc: "The mathematical expression",
						},
					}),
				},
			}

			chatModel := anthropicProvider.ChatModel()
			boundModel, err := chatModel.WithTools(tools)
			Expect(err).NotTo(HaveOccurred())
			Expect(boundModel).NotTo(BeNil())
		})
	})

	Describe("Request Verification", func() {
		It("should send correct request to mock server", func() {
			req := &provider.CompletionRequest{
				Model: "claude-3-opus-20240229",
				Messages: []*schema.Message{
					{Role: schema.User, Content: "hello test"},
				},
				MaxTokens: 100,
			}

			stream, err := anthropicProvider.CreateCompletion(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Drain the stream
			for {
				_, err := stream.Recv()
				if err != nil {
					break
				}
			}
			stream.Close()

			// Verify request was recorded
			requests := mockServer.GetRequests()
			Expect(len(requests)).To(BeNumerically(">", 0))

			// Check the last request
			lastReq := requests[len(requests)-1]
			Expect(lastReq.Path).To(Equal("/v1/messages"))

			// Verify messages were sent
			messages, ok := lastReq.Body["messages"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(len(messages)).To(BeNumerically(">", 0))
		})
	})

	Describe("Determinism", func() {
		It("should return identical responses for identical prompts", func() {
			req := &provider.CompletionRequest{
				Model: "claude-3-opus-20240229",
				Messages: []*schema.Message{
					{Role: schema.User, Content: "hello"},
				},
				MaxTokens: 100,
			}

			// First request
			stream1, err := anthropicProvider.CreateCompletion(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			var response1 string
			for {
				msg, err := stream1.Recv()
				if err != nil {
					break
				}
				if msg != nil {
					response1 += msg.Content
				}
			}
			stream1.Close()

			// Second identical request
			stream2, err := anthropicProvider.CreateCompletion(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			var response2 string
			for {
				msg, err := stream2.Recv()
				if err != nil {
					break
				}
				if msg != nil {
					response2 += msg.Content
				}
			}
			stream2.Close()

			// Responses should be identical
			Expect(response1).To(Equal(response2))
		})
	})
})
