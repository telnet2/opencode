package service_test

import (
	"encoding/json"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencode-ai/opencode/citest/testutil"
)

var _ = Describe("Debug Response", func() {
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

	It("should show streaming response chunks", func() {
		stream, err := client.SendMessageStreaming(ctx, session.ID, "Say 'Hello'")
		Expect(err).NotTo(HaveOccurred())
		defer stream.Close()

		GinkgoWriter.Println("Reading streaming response chunks:")
		chunkNum := 0
		for {
			var resp testutil.MessageResponse
			err := stream.ReadChunk(&resp)
			if err == io.EOF {
				break
			}
			if err != nil {
				GinkgoWriter.Printf("Read error: %v\n", err)
				break
			}
			chunkNum++
			data, _ := json.MarshalIndent(resp, "", "  ")
			GinkgoWriter.Printf("Chunk %d:\n%s\n", chunkNum, string(data))
			GinkgoWriter.Printf("  Info present: %v, Parts count: %d\n", resp.Info != nil, len(resp.Parts))
			for i, part := range resp.Parts {
				GinkgoWriter.Printf("  Part %d: Type=%q, Text=%q\n", i, part.Type, part.Text)
			}
		}
		GinkgoWriter.Printf("Total chunks: %d\n", chunkNum)

		// We expect at least some chunks
		Expect(chunkNum).To(BeNumerically(">", 0), "Should have received at least one chunk")
	})
})
