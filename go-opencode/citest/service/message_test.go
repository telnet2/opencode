package service_test

import (
	"io"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencode-ai/opencode/citest/testutil"
)

var _ = Describe("Message Flow", func() {
	var tempDir *testutil.TempDir
	var session *testutil.Session

	BeforeEach(func() {
		var err error
		tempDir, err = testutil.NewTempDir()
		Expect(err).NotTo(HaveOccurred())

		session, err = client.CreateSession(ctx, tempDir.Path)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if session != nil {
			client.DeleteSession(ctx, session.ID)
		}
		if tempDir != nil {
			tempDir.Cleanup()
		}
	})

	Describe("POST /session/{id}/message", func() {
		It("should send message and receive response", func() {
			resp, err := client.SendMessage(ctx, session.ID, "Say 'Hello, World!' and nothing else.")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
			Expect(resp.Info).NotTo(BeNil())
			Expect(resp.Info.Content).To(ContainSubstring("Hello"))
		})

		It("should handle simple question", func() {
			resp, err := client.SendMessage(ctx, session.ID, "What is 2+2? Answer with just the number.")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Info.Content).To(ContainSubstring("4"))
		})

		It("should stream response chunks", func() {
			stream, err := client.SendMessageStreaming(ctx, session.ID, "Count from 1 to 5, one number per line.")
			Expect(err).NotTo(HaveOccurred())
			defer stream.Close()

			Expect(stream.StatusCode).To(Equal(200))

			// Count chunks received
			chunkCount := 0
			for {
				var resp testutil.MessageResponse
				err := stream.ReadChunk(&resp)
				if err == io.EOF {
					break
				}
				if err != nil {
					// Some errors are expected at end of stream
					if !strings.Contains(err.Error(), "unexpected end") {
						Fail("Unexpected error: " + err.Error())
					}
					break
				}
				chunkCount++
			}

			// Should have received multiple chunks
			Expect(chunkCount).To(BeNumerically(">", 1))
		})

		It("should maintain conversation context", func() {
			// First message - establish context
			_, err := client.SendMessage(ctx, session.ID, "Remember this number: 42. Just say 'OK' to confirm.")
			Expect(err).NotTo(HaveOccurred())

			// Second message - reference context
			resp, err := client.SendMessage(ctx, session.ID, "What number did I ask you to remember? Just say the number.")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Info.Content).To(ContainSubstring("42"))
		})
	})

	Describe("GET /session/{id}/message", func() {
		BeforeEach(func() {
			// Send a message to populate the session
			_, err := client.SendMessage(ctx, session.ID, "Hello")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return all messages in session", func() {
			messages, err := client.GetMessages(ctx, session.ID)
			Expect(err).NotTo(HaveOccurred())
			// Should have at least user message and assistant response
			Expect(len(messages)).To(BeNumerically(">=", 2))
		})

		It("should include both user and assistant messages", func() {
			messages, err := client.GetMessages(ctx, session.ID)
			Expect(err).NotTo(HaveOccurred())

			roles := make(map[string]bool)
			for _, m := range messages {
				roles[m.Role] = true
			}
			Expect(roles["user"]).To(BeTrue(), "Should have user message")
			Expect(roles["assistant"]).To(BeTrue(), "Should have assistant message")
		})
	})

	Describe("Multi-turn Conversation", func() {
		It("should handle multiple exchanges", func() {
			// Exchange 1
			_, err := client.SendMessage(ctx, session.ID, "My name is Alice. Just say 'Nice to meet you, Alice'.")
			Expect(err).NotTo(HaveOccurred())

			// Exchange 2
			resp, err := client.SendMessage(ctx, session.ID, "What is my name? Just say the name.")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Info.Content).To(ContainSubstring("Alice"))
		})

		It("should handle rapid consecutive messages", func() {
			for i := 0; i < 3; i++ {
				resp, err := client.SendMessage(ctx, session.ID, "Say 'OK'")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Info).NotTo(BeNil())
			}

			messages, err := client.GetMessages(ctx, session.ID)
			Expect(err).NotTo(HaveOccurred())
			// 3 user messages + 3 assistant responses
			Expect(len(messages)).To(BeNumerically(">=", 6))
		})
	})

	Describe("Response Timing", func() {
		It("should respond within reasonable time", func() {
			start := time.Now()
			_, err := client.SendMessage(ctx, session.ID, "Say 'Hello'")
			elapsed := time.Since(start)

			Expect(err).NotTo(HaveOccurred())
			Expect(elapsed).To(BeNumerically("<", 30*time.Second))
		})
	})
})
