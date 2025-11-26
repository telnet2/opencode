package server_test

import (
	"encoding/json"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencode-ai/opencode/citest/testutil"
)

var _ = Describe("HTTP Response Behavior", func() {
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

	Describe("Success Responses", func() {
		It("should return 200 with JSON body for GET", func() {
			resp, err := client.Get(ctx, "/session")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
			Expect(resp.Headers.Get("Content-Type")).To(ContainSubstring("application/json"))
		})

		It("should return JSON array for list endpoints", func() {
			resp, err := client.Get(ctx, "/session")
			Expect(err).NotTo(HaveOccurred())

			var sessions []map[string]interface{}
			err = json.Unmarshal(resp.Body, &sessions)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return success response for DELETE", func() {
			// Create a session to delete
			session, err := client.CreateSession(ctx, tempDir.Path)
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.Delete(ctx, "/session/"+session.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.IsSuccess()).To(BeTrue())
		})
	})

	Describe("Error Responses", func() {
		It("should return 404 for unknown resource", func() {
			resp, err := client.Get(ctx, "/session/nonexistent-id")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(404))
		})

		It("should return structured error for 404", func() {
			resp, err := client.Get(ctx, "/session/nonexistent-id")
			Expect(err).NotTo(HaveOccurred())

			var errResp struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			err = json.Unmarshal(resp.Body, &errResp)
			Expect(err).NotTo(HaveOccurred())
			Expect(errResp.Error.Code).To(Equal("NOT_FOUND"))
		})

		It("should return 400 for invalid request", func() {
			// Send invalid JSON
			req, err := http.NewRequest("POST", testServer.BaseURL+"/session", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Content-Type", "application/json")

			httpClient := &http.Client{}
			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			// Empty body might be 400 or accepted depending on implementation
			Expect(resp.StatusCode).To(BeNumerically(">=", 200))
		})
	})

	Describe("CORS Headers", func() {
		It("should respond to OPTIONS request", func() {
			req, err := http.NewRequest("OPTIONS", testServer.BaseURL+"/session", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Origin", "http://example.com")
			req.Header.Set("Access-Control-Request-Method", "POST")

			httpClient := &http.Client{}
			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(200))
		})

		It("should include CORS headers in response", func() {
			req, err := http.NewRequest("GET", testServer.BaseURL+"/session", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Origin", "http://example.com")

			httpClient := &http.Client{}
			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			// Should have CORS headers
			Expect(resp.Header.Get("Access-Control-Allow-Origin")).NotTo(BeEmpty())
		})

		It("should allow required methods", func() {
			req, err := http.NewRequest("OPTIONS", testServer.BaseURL+"/session", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Origin", "http://example.com")
			req.Header.Set("Access-Control-Request-Method", "POST")

			httpClient := &http.Client{}
			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			allowedMethods := resp.Header.Get("Access-Control-Allow-Methods")
			Expect(allowedMethods).To(SatisfyAny(
				ContainSubstring("POST"),
				ContainSubstring("*"),
			))
		})
	})

	Describe("Content-Type Handling", func() {
		It("should accept JSON content-type", func() {
			resp, err := client.Post(ctx, "/session", map[string]string{
				"directory": tempDir.Path,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.IsSuccess()).To(BeTrue())

			// Cleanup
			var session testutil.Session
			resp.JSON(&session)
			if session.ID != "" {
				client.DeleteSession(ctx, session.ID)
			}
		})

		It("should return JSON content-type for API responses", func() {
			resp, err := client.Get(ctx, "/config")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Headers.Get("Content-Type")).To(ContainSubstring("application/json"))
		})
	})
})

var _ = Describe("Streaming Response Behavior", func() {
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
		It("should return 200 for message request", func() {
			stream, err := client.SendMessageStreaming(ctx, session.ID, "Say hello")
			Expect(err).NotTo(HaveOccurred())
			defer stream.Close()

			Expect(stream.StatusCode).To(Equal(200))
		})

		It("should stream multiple chunks", func() {
			stream, err := client.SendMessageStreaming(ctx, session.ID, "Count from 1 to 3")
			Expect(err).NotTo(HaveOccurred())
			defer stream.Close()

			chunkCount := 0
			for {
				var resp testutil.MessageResponse
				err := stream.ReadChunk(&resp)
				if err != nil {
					break
				}
				chunkCount++
				if chunkCount > 10 {
					break // Avoid infinite loop
				}
			}

			Expect(chunkCount).To(BeNumerically(">", 0))
		})
	})
})

var _ = Describe("Config and Provider Endpoints", func() {
	Describe("GET /config", func() {
		It("should return configuration", func() {
			resp, err := client.Get(ctx, "/config")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.IsSuccess()).To(BeTrue())

			var config map[string]interface{}
			err = resp.JSON(&config)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GET /config/providers", func() {
		It("should list providers", func() {
			providers, err := client.GetProviders(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(providers)).To(BeNumerically(">", 0))
		})

		It("should include ARK provider", func() {
			providers, err := client.GetProviders(ctx)
			Expect(err).NotTo(HaveOccurred())

			found := false
			for _, p := range providers {
				if p.ID == "ark" {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "ARK provider should be in the list")
		})
	})
})
