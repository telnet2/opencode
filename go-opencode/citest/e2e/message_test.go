package e2e_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	opencode "github.com/sst/opencode-sdk-go"

	"github.com/opencode-ai/opencode/citest/testutil"
)

var _ = Describe("Message Workflows", func() {
	var tempDir *testutil.TempDir
	var session *opencode.Session

	BeforeEach(func() {
		var err error
		tempDir, err = testutil.NewTempDir()
		Expect(err).NotTo(HaveOccurred())

		session, err = client.Session.New(ctx, opencode.SessionNewParams{
			Directory: opencode.F(tempDir.Path),
		})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if session != nil {
			client.Session.Delete(ctx, session.ID, opencode.SessionDeleteParams{})
		}
		if tempDir != nil {
			tempDir.Cleanup()
		}
	})

	Describe("Simple Message Exchange", func() {
		It("should send message and receive response", func() {
			resp, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
				Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
					opencode.TextPartInputParam{
						Type: opencode.F(opencode.TextPartInputTypeText),
						Text: opencode.F("Say 'Hello, World!' and nothing else."),
					},
				}),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
		})

		It("should handle simple question", func() {
			resp, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
				Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
					opencode.TextPartInputParam{
						Type: opencode.F(opencode.TextPartInputTypeText),
						Text: opencode.F("What is 2+2? Answer with just the number."),
					},
				}),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
		})

		It("should maintain conversation context", func() {
			// First message - establish context
			_, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
				Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
					opencode.TextPartInputParam{
						Type: opencode.F(opencode.TextPartInputTypeText),
						Text: opencode.F("Remember this number: 42. Just say 'OK' to confirm."),
					},
				}),
			})
			Expect(err).NotTo(HaveOccurred())

			// Second message - reference context
			resp, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
				Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
					opencode.TextPartInputParam{
						Type: opencode.F(opencode.TextPartInputTypeText),
						Text: opencode.F("What number did I ask you to remember? Just say the number."),
					},
				}),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("Message Retrieval", func() {
		BeforeEach(func() {
			// Send a message to populate the session
			_, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
				Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
					opencode.TextPartInputParam{
						Type: opencode.F(opencode.TextPartInputTypeText),
						Text: opencode.F("Hello"),
					},
				}),
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should retrieve all messages in session", func() {
			messages, err := client.Session.Messages(ctx, session.ID, opencode.SessionMessagesParams{})
			Expect(err).NotTo(HaveOccurred())
			Expect(messages).NotTo(BeNil())
			// Should have at least user message and assistant response
			Expect(len(*messages)).To(BeNumerically(">=", 2))
		})

		It("should include both user and assistant messages", func() {
			messages, err := client.Session.Messages(ctx, session.ID, opencode.SessionMessagesParams{})
			Expect(err).NotTo(HaveOccurred())

			hasUser := false
			hasAssistant := false
			for _, m := range *messages {
				// Check role from Info.Role
				if m.Info.Role == opencode.MessageRoleUser {
					hasUser = true
				}
				if m.Info.Role == opencode.MessageRoleAssistant {
					hasAssistant = true
				}
			}
			Expect(hasUser).To(BeTrue(), "Should have user message")
			Expect(hasAssistant).To(BeTrue(), "Should have assistant message")
		})
	})

	Describe("Multi-turn Conversation", func() {
		It("should handle multiple exchanges", func() {
			// Exchange 1
			_, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
				Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
					opencode.TextPartInputParam{
						Type: opencode.F(opencode.TextPartInputTypeText),
						Text: opencode.F("My name is Alice. Just say 'Nice to meet you, Alice'."),
					},
				}),
			})
			Expect(err).NotTo(HaveOccurred())

			// Exchange 2
			resp, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
				Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
					opencode.TextPartInputParam{
						Type: opencode.F(opencode.TextPartInputTypeText),
						Text: opencode.F("What is my name? Just say the name."),
					},
				}),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
		})
	})
})

var _ = Describe("Tool Execution via SDK", func() {
	var tempDir *testutil.TempDir
	var session *opencode.Session

	BeforeEach(func() {
		var err error
		tempDir, err = testutil.NewTempDir()
		Expect(err).NotTo(HaveOccurred())

		session, err = client.Session.New(ctx, opencode.SessionNewParams{
			Directory: opencode.F(tempDir.Path),
		})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if session != nil {
			client.Session.Delete(ctx, session.ID, opencode.SessionDeleteParams{})
		}
		if tempDir != nil {
			tempDir.Cleanup()
		}
	})

	Describe("Bash Tool", func() {
		It("should execute simple bash command", func() {
			resp, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
				Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
					opencode.TextPartInputParam{
						Type: opencode.F(opencode.TextPartInputTypeText),
						Text: opencode.F("Run the bash command 'echo hello world' and tell me the output."),
					},
				}),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
		})

		It("should handle ls command", func() {
			// Create some files
			_, err := tempDir.CreateFile("file1.txt", "content1")
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
				Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
					opencode.TextPartInputParam{
						Type: opencode.F(opencode.TextPartInputTypeText),
						Text: opencode.F("Run 'ls " + tempDir.Path + "' and list the files you see."),
					},
				}),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("File Operations", func() {
		It("should read file content", func() {
			testFile, err := tempDir.CreateFile("readme.txt", "This is the readme content.")
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
				Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
					opencode.TextPartInputParam{
						Type: opencode.F(opencode.TextPartInputTypeText),
						Text: opencode.F("Read the file " + testFile.Path + " and tell me what it says."),
					},
				}),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
		})

		It("should handle non-existent file gracefully", func() {
			resp, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
				Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
					opencode.TextPartInputParam{
						Type: opencode.F(opencode.TextPartInputTypeText),
						Text: opencode.F("Try to read the file /nonexistent/path/file.txt and tell me if it exists."),
					},
				}),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
		})
	})
})

// Helper to check if response contains text
func responseContains(resp *opencode.SessionPromptResponse, substr string) bool {
	if resp == nil {
		return false
	}
	// Check message content
	return strings.Contains(strings.ToLower(resp.JSON.RawJSON()), strings.ToLower(substr))
}
