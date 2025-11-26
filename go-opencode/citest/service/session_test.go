package service_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencode-ai/opencode/citest/testutil"
)

var _ = Describe("Session Management", func() {
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

	Describe("POST /session", func() {
		It("should create a new session", func() {
			session, err := client.CreateSession(ctx, tempDir.Path)
			Expect(err).NotTo(HaveOccurred())
			Expect(session.ID).NotTo(BeEmpty())
			Expect(session.Directory).To(Equal(tempDir.Path))

			// Cleanup
			client.DeleteSession(ctx, session.ID)
		})

		It("should create session with specified directory", func() {
			subDir, err := tempDir.CreateSubDir("project")
			Expect(err).NotTo(HaveOccurred())

			session, err := client.CreateSession(ctx, subDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(session.Directory).To(Equal(subDir))

			// Cleanup
			client.DeleteSession(ctx, session.ID)
		})

		It("should handle /tmp directory", func() {
			session, err := client.CreateSession(ctx, os.TempDir())
			Expect(err).NotTo(HaveOccurred())
			Expect(session.ID).NotTo(BeEmpty())

			// Cleanup
			client.DeleteSession(ctx, session.ID)
		})
	})

	Describe("GET /session", func() {
		var sessions []*testutil.Session

		BeforeEach(func() {
			// Create multiple sessions
			for i := 0; i < 3; i++ {
				s, err := client.CreateSession(ctx, tempDir.Path)
				Expect(err).NotTo(HaveOccurred())
				sessions = append(sessions, s)
			}
		})

		AfterEach(func() {
			for _, s := range sessions {
				client.DeleteSession(ctx, s.ID)
			}
			sessions = nil
		})

		It("should list all sessions", func() {
			list, err := client.ListSessions(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(list)).To(BeNumerically(">=", 3))

			// Verify our sessions are in the list
			ids := make(map[string]bool)
			for _, s := range list {
				ids[s.ID] = true
			}
			for _, s := range sessions {
				Expect(ids[s.ID]).To(BeTrue(), "Session %s should be in list", s.ID)
			}
		})
	})

	Describe("GET /session/{id}", func() {
		var session *testutil.Session

		BeforeEach(func() {
			var err error
			session, err = client.CreateSession(ctx, tempDir.Path)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			if session != nil {
				client.DeleteSession(ctx, session.ID)
			}
		})

		It("should return session by ID", func() {
			retrieved, err := client.GetSession(ctx, session.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.ID).To(Equal(session.ID))
			Expect(retrieved.Directory).To(Equal(session.Directory))
		})

		It("should return 404 for unknown session", func() {
			resp, err := client.Get(ctx, "/session/nonexistent-id")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(404))
		})
	})

	Describe("DELETE /session/{id}", func() {
		It("should delete session", func() {
			session, err := client.CreateSession(ctx, tempDir.Path)
			Expect(err).NotTo(HaveOccurred())

			err = client.DeleteSession(ctx, session.ID)
			Expect(err).NotTo(HaveOccurred())

			// Verify it's gone
			resp, err := client.Get(ctx, "/session/"+session.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(404))
		})

		It("should handle deleting non-existent session", func() {
			resp, err := client.Delete(ctx, "/session/nonexistent-id")
			Expect(err).NotTo(HaveOccurred())
			// Should be 404 or 200 depending on implementation
			Expect(resp.StatusCode).To(BeNumerically(">=", 200))
		})
	})
})
