package e2e_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	opencode "github.com/sst/opencode-sdk-go"

	"github.com/opencode-ai/opencode/citest/testutil"
)

var _ = Describe("Session Workflows", func() {
	var tempDir *testutil.TempDir

	BeforeEach(func() {
		var err error
		tempDir, err = testutil.NewTempDir()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if tempDir != nil {
			tempDir.Cleanup()
		}
	})

	Describe("Basic Session Lifecycle", func() {
		It("should create a new session", func() {
			session, err := client.Session.New(ctx, opencode.SessionNewParams{
				Directory: opencode.F(tempDir.Path),
				Title:     opencode.F("Test Session"),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(session.ID).NotTo(BeEmpty())
			Expect(session.Title).To(Equal("Test Session"))

			// Cleanup
			client.Session.Delete(ctx, session.ID, opencode.SessionDeleteParams{})
		})

		It("should retrieve session by ID", func() {
			session, err := client.Session.New(ctx, opencode.SessionNewParams{
				Directory: opencode.F(tempDir.Path),
			})
			Expect(err).NotTo(HaveOccurred())
			defer client.Session.Delete(ctx, session.ID, opencode.SessionDeleteParams{})

			retrieved, err := client.Session.Get(ctx, session.ID, opencode.SessionGetParams{})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.ID).To(Equal(session.ID))
		})

		It("should list sessions", func() {
			session, err := client.Session.New(ctx, opencode.SessionNewParams{
				Directory: opencode.F(tempDir.Path),
			})
			Expect(err).NotTo(HaveOccurred())
			defer client.Session.Delete(ctx, session.ID, opencode.SessionDeleteParams{})

			sessions, err := client.Session.List(ctx, opencode.SessionListParams{})
			Expect(err).NotTo(HaveOccurred())
			Expect(sessions).NotTo(BeNil())
			Expect(len(*sessions)).To(BeNumerically(">", 0))

			// Check our session is in the list
			found := false
			for _, s := range *sessions {
				if s.ID == session.ID {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Created session should be in list")
		})

		It("should delete session", func() {
			session, err := client.Session.New(ctx, opencode.SessionNewParams{
				Directory: opencode.F(tempDir.Path),
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = client.Session.Delete(ctx, session.ID, opencode.SessionDeleteParams{})
			Expect(err).NotTo(HaveOccurred())

			// Verify it's gone - should return error
			_, err = client.Session.Get(ctx, session.ID, opencode.SessionGetParams{})
			Expect(err).To(HaveOccurred())
		})
	})
})
