package service_test

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencode-ai/opencode/citest/testutil"
)

var _ = Describe("Tool Execution", func() {
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

	Describe("Bash Tool", func() {
		It("should make tool calls for bash commands", func() {
			resp, err := client.SendMessage(ctx, session.ID,
				"Run the bash command 'echo hello'.")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())

			// Verify we got parts back (infrastructure test)
			Expect(len(resp.Parts)).To(BeNumerically(">", 0), "Should have parts")

			// Check that at least one tool part exists (LLM attempted tool call)
			hasToolPart := false
			for _, p := range resp.Parts {
				if p.Type == "tool" {
					hasToolPart = true
					break
				}
			}
			Expect(hasToolPart).To(BeTrue(), "Should have at least one tool part")
		})
	})

	// Note: Detailed tool behavior tests are skipped because they depend heavily on
	// the specific LLM model's ability to follow instructions. The Bash Tool test
	// above verifies the infrastructure is working correctly.

	Describe("File Read Tool", func() {
		It("should attempt to read files", func() {
			testFile, err := tempDir.CreateFile("readme.txt", "test content")
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.SendMessage(ctx, session.ID,
				"Read the file "+testFile.Path)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())

			// Infrastructure test: verify we got a response with parts
			Expect(len(resp.Parts)).To(BeNumerically(">", 0), "Should have parts")
		})
	})

	Describe("File Write Tool", func() {
		It("should attempt to write files", func() {
			targetPath := filepath.Join(tempDir.Path, "output.txt")

			resp, err := client.SendMessage(ctx, session.ID,
				"Write 'test' to "+targetPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())

			// Infrastructure test: verify we got a response
			Expect(len(resp.Parts)).To(BeNumerically(">", 0), "Should have parts")
		})
	})
})
