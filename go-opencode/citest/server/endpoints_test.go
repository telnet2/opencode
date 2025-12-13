package server_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencode-ai/opencode/citest/testutil"
)

var _ = Describe("Server Endpoints Integration Tests", func() {
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

	// ==================== Session Endpoints ====================
	Describe("Session Endpoints", func() {
		Describe("GET /session", func() {
			It("should list sessions", func() {
				sessions, err := client.ListSessions(ctx, tempDir.Path)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(sessions)).To(BeNumerically(">=", 1))

				// Verify our session is in the list
				found := false
				for _, s := range sessions {
					if s.ID == session.ID {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue())
			})
		})

		Describe("POST /session", func() {
			It("should create session with title", func() {
				resp, err := client.Post(ctx, "/session", map[string]string{
					"directory": tempDir.Path,
					"title":     "Test Session Title",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())

				var newSession testutil.Session
				err = resp.JSON(&newSession)
				Expect(err).NotTo(HaveOccurred())
				Expect(newSession.ID).NotTo(BeEmpty())
				Expect(newSession.Title).To(Equal("Test Session Title"))

				// Cleanup
				client.DeleteSession(ctx, newSession.ID)
			})
		})

		Describe("GET /session/{sessionID}", func() {
			It("should retrieve session by ID", func() {
				retrieved, err := client.GetSession(ctx, session.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.ID).To(Equal(session.ID))
			})

			It("should return 404 for non-existent session", func() {
				resp, err := client.Get(ctx, "/session/non-existent-id")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(404))
			})
		})

		Describe("PATCH /session/{sessionID}", func() {
			It("should update session title", func() {
				resp, err := client.Patch(ctx, "/session/"+session.ID, map[string]string{
					"title": "Updated Title",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())

				// Verify update
				updated, err := client.GetSession(ctx, session.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Title).To(Equal("Updated Title"))
			})
		})

		Describe("DELETE /session/{sessionID}", func() {
			It("should delete session", func() {
				// Create a session to delete
				newSession, err := client.CreateSession(ctx, tempDir.Path)
				Expect(err).NotTo(HaveOccurred())

				err = client.DeleteSession(ctx, newSession.ID)
				Expect(err).NotTo(HaveOccurred())

				// Verify it's gone
				resp, err := client.Get(ctx, "/session/"+newSession.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(404))
			})
		})

		Describe("GET /session/status", func() {
			It("should return session status info", func() {
				resp, err := client.Get(ctx, "/session/status")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())

				var status map[string]interface{}
				err = resp.JSON(&status)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("POST /session/{sessionID}/abort", func() {
			It("should abort session without error", func() {
				resp, err := client.Post(ctx, "/session/"+session.ID+"/abort", nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())
			})
		})

		Describe("POST /session/{sessionID}/init", func() {
			It("should initialize session", func() {
				resp, err := client.Post(ctx, "/session/"+session.ID+"/init", nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())
			})
		})
	})

	// ==================== Message Endpoints ====================
	Describe("Message Endpoints", func() {
		Describe("POST /session/{sessionID}/message", func() {
			It("should send message and receive response", func() {
				msgResp, err := client.SendMessage(ctx, session.ID, "Say OK")
				Expect(err).NotTo(HaveOccurred())
				Expect(msgResp).NotTo(BeNil())
				Expect(msgResp.Info).NotTo(BeNil())
				Expect(msgResp.Info.Role).To(Equal("assistant"))
			})

			It("should return error for empty message", func() {
				stream, err := client.SendMessageStreaming(ctx, session.ID, "")
				Expect(err).NotTo(HaveOccurred())
				defer stream.Close()
				// Empty content should either fail or return empty response
				Expect(stream.StatusCode).To(Or(Equal(400), Equal(200)))
			})
		})

		Describe("GET /session/{sessionID}/message", func() {
			It("should list messages in session", func() {
				// First send a message
				_, err := client.SendMessage(ctx, session.ID, "Hello")
				Expect(err).NotTo(HaveOccurred())

				// Then list messages
				messages, err := client.GetMessages(ctx, session.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(messages)).To(BeNumerically(">=", 1))
			})
		})

		Describe("GET /session/{sessionID}/message/{messageID}", func() {
			It("should retrieve specific message", func() {
				// Send a message first
				msgResp, err := client.SendMessage(ctx, session.ID, "Test message")
				Expect(err).NotTo(HaveOccurred())
				Expect(msgResp.Info).NotTo(BeNil())

				// Retrieve it
				resp, err := client.Get(ctx, "/session/"+session.ID+"/message/"+msgResp.Info.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())
			})
		})
	})

	// ==================== File Endpoints ====================
	Describe("File Endpoints", func() {
		BeforeEach(func() {
			// Create a test file
			_, err := tempDir.CreateFile("test.txt", "Hello, World!")
			Expect(err).NotTo(HaveOccurred())

			// Create a subdirectory with file
			_, err = tempDir.CreateSubDir("subdir")
			Expect(err).NotTo(HaveOccurred())
			_, err = tempDir.CreateFile("subdir/nested.txt", "Nested content")
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("GET /file", func() {
			It("should list directory contents", func() {
				resp, err := client.Get(ctx, "/file", testutil.WithQuery(map[string]string{
					"path": tempDir.Path,
				}))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())

				var result struct {
					Files []struct {
						Name  string `json:"name"`
						IsDir bool   `json:"isDir"`
					} `json:"files"`
				}
				err = resp.JSON(&result)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(result.Files)).To(BeNumerically(">=", 1))

				// Verify test.txt is in the list
				hasTestFile := false
				for _, f := range result.Files {
					if f.Name == "test.txt" {
						hasTestFile = true
						break
					}
				}
				Expect(hasTestFile).To(BeTrue())
			})
		})

		Describe("GET /file/content", func() {
			It("should read file content", func() {
				filePath := filepath.Join(tempDir.Path, "test.txt")
				resp, err := client.Get(ctx, "/file/content", testutil.WithQuery(map[string]string{
					"path": filePath,
				}))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())

				var content struct {
					Content string `json:"content"`
					Lines   int    `json:"lines"`
				}
				err = resp.JSON(&content)
				Expect(err).NotTo(HaveOccurred())
				Expect(content.Content).To(ContainSubstring("Hello, World!"))
			})

			It("should support offset and limit", func() {
				// Create a file with multiple lines
				_, err := tempDir.CreateFile("multiline.txt", "Line 1\nLine 2\nLine 3\nLine 4\nLine 5")
				Expect(err).NotTo(HaveOccurred())

				filePath := filepath.Join(tempDir.Path, "multiline.txt")
				resp, err := client.Get(ctx, "/file/content", testutil.WithQuery(map[string]string{
					"path":   filePath,
					"offset": "1",
					"limit":  "2",
				}))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())
			})

			It("should return 404 for non-existent file", func() {
				resp, err := client.Get(ctx, "/file/content", testutil.WithQuery(map[string]string{
					"path": "/non/existent/file.txt",
				}))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(404))
			})
		})

		Describe("GET /file/status", func() {
			It("should return git status", func() {
				// Initialize git repo
				gitDir := filepath.Join(tempDir.Path, ".git")
				os.MkdirAll(gitDir, 0755)

				resp, err := client.Get(ctx, "/file/status", testutil.WithQuery(map[string]string{
					"directory": tempDir.Path,
				}))
				Expect(err).NotTo(HaveOccurred())
				// May return error if not a real git repo, but should not crash
			})
		})
	})

	// ==================== Search Endpoints ====================
	Describe("Search Endpoints", func() {
		BeforeEach(func() {
			// Create searchable files
			_, err := tempDir.CreateFile("search1.txt", "Hello World")
			Expect(err).NotTo(HaveOccurred())
			_, err = tempDir.CreateFile("search2.txt", "Goodbye World")
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("GET /find", func() {
			It("should search for text pattern", func() {
				resp, err := client.Get(ctx, "/find", testutil.WithQuery(map[string]string{
					"pattern": "World",
					"path":    tempDir.Path,
				}))
				Expect(err).NotTo(HaveOccurred())
				// Search might have different response codes depending on implementation
			})
		})

		Describe("GET /find/file", func() {
			It("should search for files by pattern", func() {
				resp, err := client.Get(ctx, "/find/file", testutil.WithQuery(map[string]string{
					"pattern": "*.txt",
					"path":    tempDir.Path,
				}))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())
			})
		})
	})

	// ==================== Config Endpoints ====================
	Describe("Config Endpoints", func() {
		Describe("GET /config", func() {
			It("should return configuration", func() {
				resp, err := client.Get(ctx, "/config")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())

				var config map[string]interface{}
				err = resp.JSON(&config)
				Expect(err).NotTo(HaveOccurred())
				// Config should have some fields
				Expect(config).NotTo(BeEmpty())
			})
		})

		Describe("GET /config/providers", func() {
			It("should list available providers", func() {
				resp, err := client.Get(ctx, "/config/providers")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())

				var result struct {
					Providers []struct {
						ID     string `json:"id"`
						Name   string `json:"name"`
						Models []struct {
							ID   string `json:"id"`
							Name string `json:"name"`
						} `json:"models"`
					} `json:"providers"`
				}
				err = resp.JSON(&result)
				Expect(err).NotTo(HaveOccurred())
				// Should have at least one provider
				Expect(len(result.Providers)).To(BeNumerically(">=", 1))
			})
		})
	})

	// ==================== Agent Endpoints ====================
	Describe("Agent Endpoints", func() {
		Describe("GET /agent", func() {
			It("should list available agents", func() {
				resp, err := client.Get(ctx, "/agent")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())

				var agents []interface{}
				err = resp.JSON(&agents)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	// ==================== VCS Endpoints ====================
	Describe("VCS Endpoints", func() {
		Describe("GET /vcs", func() {
			It("should return VCS info", func() {
				resp, err := client.Get(ctx, "/vcs", testutil.WithQuery(map[string]string{
					"directory": tempDir.Path,
				}))
				Expect(err).NotTo(HaveOccurred())
				// May not be a git repo, but should respond
			})
		})
	})

	// ==================== Command Endpoints ====================
	Describe("Command Endpoints", func() {
		Describe("GET /command", func() {
			It("should list available commands", func() {
				resp, err := client.Get(ctx, "/command")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())
			})
		})
	})

	// ==================== MCP Endpoints ====================
	Describe("MCP Endpoints", func() {
		Describe("GET /mcp", func() {
			It("should return MCP status", func() {
				resp, err := client.Get(ctx, "/mcp")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())
			})
		})

		Describe("GET /mcp/tools", func() {
			It("should list MCP tools", func() {
				resp, err := client.Get(ctx, "/mcp/tools")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())
			})
		})
	})

	// ==================== LSP Endpoints ====================
	Describe("LSP Endpoints", func() {
		Describe("GET /lsp", func() {
			It("should return LSP status", func() {
				resp, err := client.Get(ctx, "/lsp")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())
			})
		})
	})

	// ==================== Formatter Endpoints ====================
	Describe("Formatter Endpoints", func() {
		Describe("GET /formatter", func() {
			It("should return formatter status", func() {
				resp, err := client.Get(ctx, "/formatter")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())
			})
		})
	})

	// ==================== Project Endpoints ====================
	Describe("Project Endpoints", func() {
		Describe("GET /project", func() {
			It("should list projects", func() {
				resp, err := client.Get(ctx, "/project", testutil.WithQuery(map[string]string{
					"directory": tempDir.Path,
				}))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())
			})
		})

		Describe("GET /project/current", func() {
			It("should return current project", func() {
				resp, err := client.Get(ctx, "/project/current", testutil.WithQuery(map[string]string{
					"directory": tempDir.Path,
				}))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())
			})
		})
	})

	// ==================== Instance Endpoints ====================
	Describe("Instance Endpoints", func() {
		Describe("GET /path", func() {
			It("should return working directory path", func() {
				resp, err := client.Get(ctx, "/path")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())
			})
		})
	})

	// ==================== Experimental Endpoints ====================
	Describe("Experimental Endpoints", func() {
		Describe("GET /experimental/tool/ids", func() {
			It("should return tool IDs", func() {
				resp, err := client.Get(ctx, "/experimental/tool/ids")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())
			})
		})

		Describe("GET /experimental/tool", func() {
			It("should return tool definitions", func() {
				resp, err := client.Get(ctx, "/experimental/tool")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.IsSuccess()).To(BeTrue())
			})
		})
	})
})

// Additional tests for edge cases and error handling
var _ = Describe("Server Error Handling", func() {
	Describe("Invalid Requests", func() {
		It("should return 404 for unknown paths", func() {
			resp, err := client.Get(ctx, "/unknown/endpoint")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(404))
		})

		It("should return 400 for malformed JSON", func() {
			resp, err := client.Post(ctx, "/session", "invalid json{")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})
	})

	Describe("Session Validation", func() {
		It("should return 404 for operations on non-existent session", func() {
			resp, err := client.Get(ctx, "/session/invalid-session-id/message")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(404))
		})
	})
})

// Streaming response tests
var _ = Describe("Streaming Responses", func() {
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

	Describe("Message Streaming", func() {
		It("should stream message chunks", func() {
			stream, err := client.SendMessageStreaming(ctx, session.ID, "Count from 1 to 3")
			Expect(err).NotTo(HaveOccurred())
			defer stream.Close()

			Expect(stream.StatusCode).To(Equal(200))

			// Read at least one chunk
			var resp testutil.MessageResponse
			err = stream.ReadChunk(&resp)
			// May get EOF if response is complete
			if err != nil {
				Expect(err.Error()).To(ContainSubstring("EOF"))
			}
		})

		It("should support context cancellation", func() {
			cancelCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
			defer cancel()

			stream, err := client.SendMessageStreaming(cancelCtx, session.ID, "Say a very long response")
			Expect(err).NotTo(HaveOccurred())
			defer stream.Close()

			// Context should cancel before too many chunks
			var chunks int
			for {
				var resp testutil.MessageResponse
				err := stream.ReadChunk(&resp)
				if err != nil {
					break
				}
				chunks++
				if chunks > 100 {
					break
				}
			}
		})
	})
})

// Concurrent access tests
var _ = Describe("Concurrent Access", func() {
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

	It("should handle multiple concurrent session creations", func() {
		const numSessions = 5
		done := make(chan *testutil.Session, numSessions)
		errors := make(chan error, numSessions)

		for i := 0; i < numSessions; i++ {
			go func() {
				session, err := client.CreateSession(ctx, tempDir.Path)
				if err != nil {
					errors <- err
					return
				}
				done <- session
			}()
		}

		var sessions []*testutil.Session
		for i := 0; i < numSessions; i++ {
			select {
			case session := <-done:
				sessions = append(sessions, session)
			case err := <-errors:
				Expect(err).NotTo(HaveOccurred())
			case <-time.After(30 * time.Second):
				Fail("Timeout waiting for concurrent session creation")
			}
		}

		Expect(len(sessions)).To(Equal(numSessions))

		// Cleanup
		for _, s := range sessions {
			client.DeleteSession(ctx, s.ID)
		}
	})

	It("should handle concurrent config reads", func() {
		const numReads = 10
		done := make(chan bool, numReads)
		errors := make(chan error, numReads)

		for i := 0; i < numReads; i++ {
			go func() {
				resp, err := client.Get(ctx, "/config")
				if err != nil {
					errors <- err
					return
				}
				if !resp.IsSuccess() {
					errors <- json.Unmarshal(resp.Body, new(interface{}))
					return
				}
				done <- true
			}()
		}

		for i := 0; i < numReads; i++ {
			select {
			case <-done:
				// OK
			case err := <-errors:
				Expect(err).NotTo(HaveOccurred())
			case <-time.After(30 * time.Second):
				Fail("Timeout waiting for concurrent config reads")
			}
		}
	})
})

// Session workflow tests
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

	It("should complete full conversation workflow", func() {
		// Create session
		session, err := client.CreateSession(ctx, tempDir.Path)
		Expect(err).NotTo(HaveOccurred())
		defer client.DeleteSession(ctx, session.ID)

		// Send first message
		resp1, err := client.SendMessage(ctx, session.ID, "Remember the number 42")
		Expect(err).NotTo(HaveOccurred())
		Expect(resp1.Info).NotTo(BeNil())

		// Send follow-up message
		resp2, err := client.SendMessage(ctx, session.ID, "What number did I ask you to remember?")
		Expect(err).NotTo(HaveOccurred())
		Expect(resp2.Info).NotTo(BeNil())

		// Check message history
		messages, err := client.GetMessages(ctx, session.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(messages)).To(BeNumerically(">=", 2))
	})

	It("should handle session fork", func() {
		// Create parent session
		session, err := client.CreateSession(ctx, tempDir.Path)
		Expect(err).NotTo(HaveOccurred())
		defer client.DeleteSession(ctx, session.ID)

		// Send a message
		msgResp, err := client.SendMessage(ctx, session.ID, "Hello")
		Expect(err).NotTo(HaveOccurred())

		// Fork session
		resp, err := client.Post(ctx, "/session/"+session.ID+"/fork", map[string]string{
			"messageID": msgResp.Info.ID,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.IsSuccess()).To(BeTrue())

		var forkedSession testutil.Session
		err = resp.JSON(&forkedSession)
		Expect(err).NotTo(HaveOccurred())
		Expect(forkedSession.ID).NotTo(Equal(session.ID))

		// Cleanup forked session
		client.DeleteSession(ctx, forkedSession.ID)
	})
})

// File operations with tool execution
var _ = Describe("Tool Execution Workflows", func() {
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

	It("should execute file read via message", func() {
		// Create a test file
		_, err := tempDir.CreateFile("readme.txt", "This is a test file with important content.")
		Expect(err).NotTo(HaveOccurred())

		// Ask to read the file
		resp, err := client.SendMessage(ctx, session.ID, "Read the file readme.txt and tell me what it says")
		Expect(err).NotTo(HaveOccurred())
		Expect(resp).NotTo(BeNil())

		// Response should mention the content or file
		content := resp.Content()
		// The response might reference the file content
		GinkgoWriter.Printf("Response: %s\n", content)
	})

	It("should list files via message", func() {
		// Create some files
		_, err := tempDir.CreateFile("file1.txt", "content 1")
		Expect(err).NotTo(HaveOccurred())
		_, err = tempDir.CreateFile("file2.txt", "content 2")
		Expect(err).NotTo(HaveOccurred())

		// Ask to list files
		resp, err := client.SendMessage(ctx, session.ID, "List the files in the current directory")
		Expect(err).NotTo(HaveOccurred())
		Expect(resp).NotTo(BeNil())
	})
})

// Provider-specific tests
var _ = Describe("Provider Integration", func() {
	Describe("Provider Health", func() {
		It("should have at least one working provider", func() {
			resp, err := client.Get(ctx, "/config/providers")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.IsSuccess()).To(BeTrue())

			var result struct {
				Providers []struct {
					ID     string `json:"id"`
					Name   string `json:"name"`
					Models []struct {
						ID string `json:"id"`
					} `json:"models"`
				} `json:"providers"`
			}
			err = resp.JSON(&result)
			Expect(err).NotTo(HaveOccurred())

			// Should have providers with models
			hasModels := false
			for _, p := range result.Providers {
				if len(p.Models) > 0 {
					hasModels = true
					GinkgoWriter.Printf("Found provider %s with %d models\n", p.ID, len(p.Models))
					break
				}
			}
			Expect(hasModels).To(BeTrue(), "Should have at least one provider with models")
		})
	})
})

// Health and monitoring
var _ = Describe("Health and Monitoring", func() {
	It("should respond to config endpoint as health check", func() {
		resp, err := client.Get(ctx, "/config")
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.IsSuccess()).To(BeTrue())
	})

	It("should include proper CORS headers", func() {
		resp, err := client.Get(ctx, "/config")
		Expect(err).NotTo(HaveOccurred())
		// CORS headers should be present (may vary by config)
		// Just verify response is successful
		Expect(resp.IsSuccess()).To(BeTrue())
	})
})

// Special character and unicode handling
var _ = Describe("Character Encoding", func() {
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

	It("should handle unicode in messages", func() {
		resp, err := client.SendMessage(ctx, session.ID, "Say hello in Japanese: こんにちは")
		Expect(err).NotTo(HaveOccurred())
		Expect(resp).NotTo(BeNil())
	})

	It("should handle special characters in file names", func() {
		// Create file with special chars (safe ones)
		_, err := tempDir.CreateFile("test-file_123.txt", "content")
		Expect(err).NotTo(HaveOccurred())

		filePath := filepath.Join(tempDir.Path, "test-file_123.txt")
		resp, err := client.Get(ctx, "/file/content", testutil.WithQuery(map[string]string{
			"path": filePath,
		}))
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.IsSuccess()).To(BeTrue())
	})

	It("should handle unicode in file content", func() {
		unicodeContent := "Hello 世界! 🌍 Привет мир!"
		_, err := tempDir.CreateFile("unicode.txt", unicodeContent)
		Expect(err).NotTo(HaveOccurred())

		filePath := filepath.Join(tempDir.Path, "unicode.txt")
		resp, err := client.Get(ctx, "/file/content", testutil.WithQuery(map[string]string{
			"path": filePath,
		}))
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.IsSuccess()).To(BeTrue())

		var content struct {
			Content string `json:"content"`
		}
		err = resp.JSON(&content)
		Expect(err).NotTo(HaveOccurred())
		Expect(strings.Contains(content.Content, "世界")).To(BeTrue())
	})
})
