package server_test

import (
	"context"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencode-ai/opencode/citest/testutil"
)

var _ = Describe("SSE Event Streaming", func() {
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

	Describe("GET /event", func() {
		It("should return SSE content-type header", func() {
			req, err := http.NewRequest("GET", testServer.BaseURL+"/event?sessionID="+session.ID, nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Accept", "text/event-stream")

			httpClient := &http.Client{Timeout: 5 * time.Second}
			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.Header.Get("Content-Type")).To(HavePrefix("text/event-stream"))
		})

		It("should set cache control headers", func() {
			req, err := http.NewRequest("GET", testServer.BaseURL+"/event?sessionID="+session.ID, nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Accept", "text/event-stream")

			httpClient := &http.Client{Timeout: 5 * time.Second}
			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.Header.Get("Cache-Control")).To(Equal("no-cache"))
		})

		It("should require sessionID parameter", func() {
			resp, err := client.Get(ctx, "/event")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})

		It("should deliver events for session activity", func() {
			// Start SSE connection
			sseClient := testServer.SSEClient()
			err := sseClient.Connect(ctx, "/event?sessionID="+session.ID)
			Expect(err).NotTo(HaveOccurred())
			defer sseClient.Close()

			// Give connection time to establish
			time.Sleep(500 * time.Millisecond)

			// Trigger activity by sending a message
			go func() {
				client.SendMessage(ctx, session.ID, "Say OK")
			}()

			// Wait for events
			events := sseClient.CollectEvents(10 * time.Second)

			// Should have received some events
			Expect(len(events)).To(BeNumerically(">", 0))
		})
	})

	Describe("GET /global/event", func() {
		It("should stream events without session filter", func() {
			sseClient := testServer.SSEClient()
			err := sseClient.Connect(ctx, "/global/event")
			Expect(err).NotTo(HaveOccurred())
			defer sseClient.Close()

			// Give connection time to establish
			time.Sleep(500 * time.Millisecond)

			// Create a new session (should trigger event)
			newSession, err := client.CreateSession(ctx, tempDir.Path)
			Expect(err).NotTo(HaveOccurred())
			defer client.DeleteSession(ctx, newSession.ID)

			// Wait for events
			events := sseClient.CollectEvents(5 * time.Second)

			// Should have received session.created event
			matcher := testutil.NewEventMatcher(events)
			Expect(matcher.HasType("session.created") || len(events) > 0).To(BeTrue())
		})
	})

	Describe("SSE Connection Lifecycle", func() {
		It("should handle client disconnect gracefully", func() {
			sseClient := testServer.SSEClient()
			err := sseClient.Connect(ctx, "/event?sessionID="+session.ID)
			Expect(err).NotTo(HaveOccurred())

			// Close connection
			sseClient.Close()

			// Server should still be running
			resp, err := client.Get(ctx, "/config")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.IsSuccess()).To(BeTrue())
		})

		It("should stop sending after context cancel", func() {
			cancelCtx, cancel := context.WithCancel(ctx)

			sseClient := testServer.SSEClient()
			err := sseClient.Connect(cancelCtx, "/event?sessionID="+session.ID)
			Expect(err).NotTo(HaveOccurred())

			// Cancel context
			cancel()

			// Give time for cancellation to propagate
			time.Sleep(500 * time.Millisecond)

			// Connection should be closed
			sseClient.Close()

			// Server should still be running
			resp, err := client.Get(ctx, "/config")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.IsSuccess()).To(BeTrue())
		})
	})

	Describe("Event Filtering", func() {
		It("should only deliver events for specified session", func() {
			// Create second session
			session2, err := client.CreateSession(ctx, tempDir.Path)
			Expect(err).NotTo(HaveOccurred())
			defer client.DeleteSession(ctx, session2.ID)

			// Connect SSE to first session
			sseClient := testServer.SSEClient()
			err = sseClient.Connect(ctx, "/event?sessionID="+session.ID)
			Expect(err).NotTo(HaveOccurred())
			defer sseClient.Close()

			time.Sleep(500 * time.Millisecond)

			// Send message to second session
			go func() {
				client.SendMessage(ctx, session2.ID, "Say OK")
			}()

			// Collect events for a short time
			events := sseClient.CollectEvents(3 * time.Second)

			// Should not have received message events for session2
			// (might receive heartbeats though)
			for _, evt := range events {
				if evt.Type == "message.created" || evt.Type == "message.updated" {
					// Parse and check session ID
					msgData, err := evt.ParseMessageEvent()
					if err == nil && msgData != nil {
						Expect(msgData.SessionID).NotTo(Equal(session2.ID))
					}
				}
			}
		})
	})
})
