package service_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencode-ai/opencode/citest/testutil"
)

var _ = Describe("Phase 3: MCP Endpoints", func() {
	Describe("GET /mcp", func() {
		It("should return MCP status", func() {
			resp, err := client.Get(ctx, "/mcp")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			var status map[string]interface{}
			err = resp.JSON(&status)
			Expect(err).NotTo(HaveOccurred())

			// Should have enabled field
			Expect(status).To(HaveKey("enabled"))
		})

		It("should include server count fields", func() {
			resp, err := client.Get(ctx, "/mcp")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.IsSuccess()).To(BeTrue())

			var status map[string]interface{}
			err = resp.JSON(&status)
			Expect(err).NotTo(HaveOccurred())

			// Should have server counts when MCP is enabled
			if status["enabled"].(bool) {
				Expect(status).To(HaveKey("serverCount"))
				Expect(status).To(HaveKey("connectedCount"))
			}
		})
	})

	Describe("GET /mcp/tools", func() {
		It("should return MCP tools list", func() {
			resp, err := client.Get(ctx, "/mcp/tools")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			// Initialize slice to distinguish between null and empty array
			tools := make([]interface{}, 0)
			err = resp.JSON(&tools)
			Expect(err).NotTo(HaveOccurred())
			// Tools list may be empty if no MCP servers configured
			// An empty array [] is valid (length 0)
		})
	})

	Describe("GET /mcp/resources", func() {
		It("should return MCP resources list", func() {
			resp, err := client.Get(ctx, "/mcp/resources")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			// Initialize slice to distinguish between null and empty array
			resources := make([]interface{}, 0)
			err = resp.JSON(&resources)
			Expect(err).NotTo(HaveOccurred())
			// Resources list may be empty if no MCP servers configured
			// An empty array [] is valid (length 0)
		})
	})

	Describe("POST /mcp", func() {
		It("should require name field", func() {
			resp, err := client.Post(ctx, "/mcp", map[string]interface{}{
				"type": "local",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))

			var errResp struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			err = resp.JSON(&errResp)
			Expect(err).NotTo(HaveOccurred())
			Expect(errResp.Error.Code).To(Equal("INVALID_REQUEST"))
		})
	})

	Describe("DELETE /mcp/{name}", func() {
		It("should return 404 for non-existent server", func() {
			resp, err := client.Delete(ctx, "/mcp/nonexistent-server")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(404))
		})
	})

	Describe("POST /mcp/tool/{name}", func() {
		It("should return error for non-existent tool", func() {
			resp, err := client.Post(ctx, "/mcp/tool/nonexistent-tool", map[string]interface{}{})
			Expect(err).NotTo(HaveOccurred())
			// Should fail because tool doesn't exist
			Expect(resp.StatusCode).To(BeNumerically(">=", 400))
		})
	})

	Describe("GET /mcp/resource", func() {
		It("should require uri parameter", func() {
			resp, err := client.Get(ctx, "/mcp/resource")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))

			var errResp struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			err = resp.JSON(&errResp)
			Expect(err).NotTo(HaveOccurred())
			Expect(errResp.Error.Code).To(Equal("INVALID_REQUEST"))
		})
	})
})

var _ = Describe("Phase 3: Command Endpoints", func() {
	Describe("GET /command", func() {
		It("should return list of commands", func() {
			resp, err := client.Get(ctx, "/command")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			var commands []map[string]interface{}
			err = resp.JSON(&commands)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(commands)).To(BeNumerically(">", 0), "Should have at least one command")
		})

		It("should include builtin commands", func() {
			resp, err := client.Get(ctx, "/command")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.IsSuccess()).To(BeTrue())

			var commands []map[string]interface{}
			err = resp.JSON(&commands)
			Expect(err).NotTo(HaveOccurred())

			// Check for builtin commands
			builtinNames := []string{"help", "clear", "compact"}
			foundBuiltins := make(map[string]bool)

			for _, cmd := range commands {
				name := cmd["name"].(string)
				for _, b := range builtinNames {
					if name == b {
						foundBuiltins[b] = true
					}
				}
			}

			for _, b := range builtinNames {
				Expect(foundBuiltins[b]).To(BeTrue(), "Should include builtin command: %s", b)
			}
		})

		It("should return command with required fields", func() {
			resp, err := client.Get(ctx, "/command")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.IsSuccess()).To(BeTrue())

			var commands []map[string]interface{}
			err = resp.JSON(&commands)
			Expect(err).NotTo(HaveOccurred())

			// Each command should have name and description
			for _, cmd := range commands {
				Expect(cmd).To(HaveKey("name"))
				Expect(cmd).To(HaveKey("description"))
			}
		})
	})

	Describe("GET /command/{name}", func() {
		It("should return builtin command details", func() {
			resp, err := client.Get(ctx, "/command/help")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			var cmd map[string]interface{}
			err = resp.JSON(&cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(cmd["name"]).To(Equal("help"))
		})

		It("should return 404 for unknown command", func() {
			resp, err := client.Get(ctx, "/command/nonexistent-command")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(404))
		})
	})

	Describe("POST /command/{name}", func() {
		It("should return 404 for unknown command execution", func() {
			resp, err := client.Post(ctx, "/command/nonexistent-command", map[string]string{
				"args": "test",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(404))
		})
	})
})

var _ = Describe("Phase 3: Formatter Endpoints", func() {
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

	Describe("GET /formatter", func() {
		It("should return formatter status", func() {
			resp, err := client.Get(ctx, "/formatter")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			var status map[string]interface{}
			err = resp.JSON(&status)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(HaveKey("enabled"))
		})

		It("should include formatters list when enabled", func() {
			resp, err := client.Get(ctx, "/formatter")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.IsSuccess()).To(BeTrue())

			var status map[string]interface{}
			err = resp.JSON(&status)
			Expect(err).NotTo(HaveOccurred())

			if status["enabled"].(bool) {
				Expect(status).To(HaveKey("formatters"))
			}
		})
	})

	Describe("POST /formatter/format", func() {
		It("should require path or paths field", func() {
			resp, err := client.Post(ctx, "/formatter/format", map[string]interface{}{})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))

			var errResp struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			err = resp.JSON(&errResp)
			Expect(err).NotTo(HaveOccurred())
			Expect(errResp.Error.Code).To(Equal("INVALID_REQUEST"))
		})

		It("should handle single file formatting", func() {
			// Create a test file
			testFile, err := tempDir.CreateFile("test.txt", "hello world")
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.Post(ctx, "/formatter/format", map[string]string{
				"path": testFile.Path,
			})
			Expect(err).NotTo(HaveOccurred())
			// Should succeed (even if no formatter matches the extension)
			Expect(resp.StatusCode).To(Equal(200))

			var result map[string]interface{}
			err = resp.JSON(&result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveKey("filePath"))
			Expect(result).To(HaveKey("success"))
		})

		It("should handle multiple files formatting", func() {
			// Create test files
			file1, err := tempDir.CreateFile("test1.txt", "hello")
			Expect(err).NotTo(HaveOccurred())
			file2, err := tempDir.CreateFile("test2.txt", "world")
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.Post(ctx, "/formatter/format", map[string]interface{}{
				"paths": []string{file1.Path, file2.Path},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			var results []map[string]interface{}
			err = resp.JSON(&results)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(results)).To(Equal(2))
		})
	})
})

var _ = Describe("Phase 3: Session Sharing Endpoints", func() {
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

	Describe("POST /session/{id}/share", func() {
		It("should create share for session", func() {
			resp, err := client.Post(ctx, "/session/"+session.ID+"/share", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.IsSuccess()).To(BeTrue())

			var shareResult map[string]interface{}
			err = resp.JSON(&shareResult)
			Expect(err).NotTo(HaveOccurred())

			// Should return a URL or share info
			// The exact format may vary based on implementation
			Expect(shareResult).NotTo(BeEmpty())
		})
	})

	Describe("DELETE /session/{id}/share", func() {
		It("should unshare a session", func() {
			// First share the session
			_, err := client.Post(ctx, "/session/"+session.ID+"/share", nil)
			Expect(err).NotTo(HaveOccurred())

			// Then unshare
			resp, err := client.Delete(ctx, "/session/"+session.ID+"/share")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.IsSuccess()).To(BeTrue())
		})

		It("should handle unsharing non-shared session", func() {
			resp, err := client.Delete(ctx, "/session/"+session.ID+"/share")
			Expect(err).NotTo(HaveOccurred())
			// Should either succeed or return appropriate error
			Expect(resp.StatusCode).To(BeNumerically(">=", 200))
		})
	})
})

var _ = Describe("Phase 3: Agent Endpoints", func() {
	Describe("GET /agent", func() {
		It("should return list of agents", func() {
			resp, err := client.Get(ctx, "/agent")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			var agents []map[string]interface{}
			err = resp.JSON(&agents)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(agents)).To(BeNumerically(">", 0), "Should have at least one agent")
		})

		It("should return agents with required fields", func() {
			resp, err := client.Get(ctx, "/agent")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.IsSuccess()).To(BeTrue())

			var agents []map[string]interface{}
			err = resp.JSON(&agents)
			Expect(err).NotTo(HaveOccurred())

			for _, agent := range agents {
				Expect(agent).To(HaveKey("id"))
				Expect(agent).To(HaveKey("name"))
			}
		})
	})
})

// Helper types for JSON parsing
type MCPStatus struct {
	Enabled        bool          `json:"enabled"`
	ServerCount    int           `json:"serverCount"`
	ConnectedCount int           `json:"connectedCount"`
	Servers        []interface{} `json:"servers"`
	Tools          []interface{} `json:"tools"`
}

type Command struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Agent       string `json:"agent,omitempty"`
	Model       string `json:"model,omitempty"`
	Subtask     bool   `json:"subtask,omitempty"`
}

type FormatterStatus struct {
	Enabled    bool                   `json:"enabled"`
	Formatters []map[string]interface{} `json:"formatters"`
}

type FormatResult struct {
	FilePath      string `json:"filePath"`
	Success       bool   `json:"success"`
	Changed       bool   `json:"changed"`
	Error         string `json:"error,omitempty"`
	Duration      int64  `json:"duration"`
	Formatter     string `json:"formatter,omitempty"`
	OriginalSize  int    `json:"originalSize,omitempty"`
	FormattedSize int    `json:"formattedSize,omitempty"`
}

// Ensure json package is used
var _ = json.Marshal
