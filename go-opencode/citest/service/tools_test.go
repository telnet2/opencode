package service_test

import (
	"os"
	"path/filepath"
	"strings"

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
		It("should execute simple bash command", func() {
			resp, err := client.SendMessage(ctx, session.ID,
				"Run the bash command 'echo hello world' and tell me the output.")
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.ToLower(resp.Info.Content)).To(
				SatisfyAny(
					ContainSubstring("hello world"),
					ContainSubstring("hello"),
				))
		})

		It("should capture command output", func() {
			resp, err := client.SendMessage(ctx, session.ID,
				"Use bash to run 'pwd' and tell me the directory path.")
			Expect(err).NotTo(HaveOccurred())
			// Should contain some path
			Expect(resp.Info.Content).To(MatchRegexp(`/[a-zA-Z0-9/_-]+`))
		})

		It("should handle command with arguments", func() {
			// Create a test file first
			testFile, err := tempDir.CreateFile("test.txt", "test content")
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.SendMessage(ctx, session.ID,
				"Run 'cat "+testFile.Path+"' and tell me what's in the file.")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Info.Content).To(ContainSubstring("test content"))
		})

		It("should handle ls command", func() {
			// Create some files
			_, err := tempDir.CreateFile("file1.txt", "content1")
			Expect(err).NotTo(HaveOccurred())
			_, err = tempDir.CreateFile("file2.txt", "content2")
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.SendMessage(ctx, session.ID,
				"Run 'ls "+tempDir.Path+"' and list the files you see.")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Info.Content).To(SatisfyAny(
				ContainSubstring("file1"),
				ContainSubstring("file2"),
			))
		})
	})

	Describe("File Read Tool", func() {
		It("should read file content", func() {
			testFile, err := tempDir.CreateFile("readme.txt", "This is the readme content for testing.")
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.SendMessage(ctx, session.ID,
				"Read the file "+testFile.Path+" and tell me what it says.")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Info.Content).To(ContainSubstring("readme content"))
		})

		It("should handle file with multiple lines", func() {
			content := "Line 1\nLine 2\nLine 3"
			testFile, err := tempDir.CreateFile("multiline.txt", content)
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.SendMessage(ctx, session.ID,
				"Read "+testFile.Path+" and count how many lines it has.")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Info.Content).To(SatisfyAny(
				ContainSubstring("3"),
				ContainSubstring("three"),
			))
		})

		It("should handle non-existent file gracefully", func() {
			resp, err := client.SendMessage(ctx, session.ID,
				"Try to read the file /nonexistent/path/file.txt and tell me if it exists.")
			Expect(err).NotTo(HaveOccurred())
			// Should indicate file doesn't exist or error
			Expect(strings.ToLower(resp.Info.Content)).To(SatisfyAny(
				ContainSubstring("not found"),
				ContainSubstring("doesn't exist"),
				ContainSubstring("does not exist"),
				ContainSubstring("error"),
				ContainSubstring("cannot"),
				ContainSubstring("no such"),
			))
		})
	})

	Describe("File Write Tool", func() {
		It("should write content to new file", func() {
			targetPath := filepath.Join(tempDir.Path, "output.txt")

			resp, err := client.SendMessage(ctx, session.ID,
				"Write the text 'Hello from OpenCode' to the file "+targetPath)
			Expect(err).NotTo(HaveOccurred())

			// Verify file was created
			content, err := os.ReadFile(targetPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("Hello from OpenCode"))

			// Response should indicate success
			Expect(strings.ToLower(resp.Info.Content)).To(SatisfyAny(
				ContainSubstring("written"),
				ContainSubstring("created"),
				ContainSubstring("saved"),
				ContainSubstring("done"),
				ContainSubstring("success"),
			))
		})

		It("should overwrite existing file", func() {
			testFile, err := tempDir.CreateFile("existing.txt", "old content")
			Expect(err).NotTo(HaveOccurred())

			_, err = client.SendMessage(ctx, session.ID,
				"Replace the content of "+testFile.Path+" with 'new content here'")
			Expect(err).NotTo(HaveOccurred())

			// Verify content was replaced
			content, err := os.ReadFile(testFile.Path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("new content"))
		})
	})

	Describe("Tool Chain", func() {
		It("should execute multiple tools in sequence", func() {
			targetPath := filepath.Join(tempDir.Path, "chain_test.txt")

			resp, err := client.SendMessage(ctx, session.ID,
				"Please do these steps: 1) Write 'step complete' to "+targetPath+
					", 2) Read it back, 3) Tell me what you read.")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Info.Content).To(ContainSubstring("step complete"))
		})

		It("should handle file create and read workflow", func() {
			targetPath := filepath.Join(tempDir.Path, "workflow.txt")

			// Create file
			_, err := client.SendMessage(ctx, session.ID,
				"Create a file at "+targetPath+" with content 'workflow test data'")
			Expect(err).NotTo(HaveOccurred())

			// Read it in a new message
			resp, err := client.SendMessage(ctx, session.ID,
				"Read the file "+targetPath+" and tell me its contents.")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Info.Content).To(ContainSubstring("workflow test"))
		})
	})

	Describe("Tool Error Handling", func() {
		It("should handle permission denied gracefully", func() {
			// Try to write to a protected location
			resp, err := client.SendMessage(ctx, session.ID,
				"Try to write 'test' to /etc/test_file.txt and tell me the result.")
			Expect(err).NotTo(HaveOccurred())
			// Should indicate some kind of error
			Expect(strings.ToLower(resp.Info.Content)).To(SatisfyAny(
				ContainSubstring("permission"),
				ContainSubstring("denied"),
				ContainSubstring("cannot"),
				ContainSubstring("error"),
				ContainSubstring("unable"),
				ContainSubstring("failed"),
			))
		})
	})
})
