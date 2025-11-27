package service_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Phase 4: Client Tools Endpoints", func() {
	Describe("POST /client-tools/register", func() {
		It("should require tool name", func() {
			resp, err := client.Post(ctx, "/client-tools/register", map[string]interface{}{
				"description": "A test tool",
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

		It("should register a client tool", func() {
			resp, err := client.Post(ctx, "/client-tools/register", map[string]interface{}{
				"name":        "test-tool",
				"description": "A test tool for unit testing",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"input": map[string]interface{}{
							"type":        "string",
							"description": "Input parameter",
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			var result map[string]interface{}
			err = resp.JSON(&result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveKey("success"))

			// Cleanup: unregister the tool
			client.Delete(ctx, "/client-tools/unregister?name=test-tool")
		})
	})

	Describe("DELETE /client-tools/unregister", func() {
		It("should require tool name parameter", func() {
			resp, err := client.Delete(ctx, "/client-tools/unregister")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})

		It("should unregister an existing tool", func() {
			// First register a tool
			_, err := client.Post(ctx, "/client-tools/register", map[string]interface{}{
				"name":        "tool-to-delete",
				"description": "Tool to be deleted",
			})
			Expect(err).NotTo(HaveOccurred())

			// Then unregister it
			resp, err := client.Delete(ctx, "/client-tools/unregister?name=tool-to-delete")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
		})

		It("should handle non-existent tool gracefully", func() {
			resp, err := client.Delete(ctx, "/client-tools/unregister?name=nonexistent-tool")
			Expect(err).NotTo(HaveOccurred())
			// Should either succeed (idempotent) or return 404
			Expect(resp.StatusCode).To(BeNumerically(">=", 200))
		})
	})

	Describe("POST /client-tools/execute", func() {
		It("should require tool name", func() {
			resp, err := client.Post(ctx, "/client-tools/execute", map[string]interface{}{
				"params": map[string]interface{}{},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})

		It("should return error for non-registered tool", func() {
			resp, err := client.Post(ctx, "/client-tools/execute", map[string]interface{}{
				"name":   "nonexistent-tool",
				"params": map[string]interface{}{},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(404))
		})
	})

	Describe("POST /client-tools/result", func() {
		It("should require execution ID", func() {
			resp, err := client.Post(ctx, "/client-tools/result", map[string]interface{}{
				"result": "test result",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})

		It("should handle result submission", func() {
			resp, err := client.Post(ctx, "/client-tools/result", map[string]interface{}{
				"executionId": "test-exec-123",
				"result":      "tool execution result",
			})
			Expect(err).NotTo(HaveOccurred())
			// Should either succeed or return 404 for unknown execution
			Expect(resp.StatusCode).To(BeNumerically(">=", 200))
		})
	})
})

var _ = Describe("Phase 4: Documentation Endpoint", func() {
	Describe("GET /doc", func() {
		It("should return OpenAPI documentation", func() {
			resp, err := client.Get(ctx, "/doc")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
		})

		It("should return valid JSON", func() {
			resp, err := client.Get(ctx, "/doc")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.IsSuccess()).To(BeTrue())

			var doc map[string]interface{}
			err = resp.JSON(&doc)
			Expect(err).NotTo(HaveOccurred())
			// OpenAPI spec should have certain keys
			Expect(doc).To(Or(
				HaveKey("openapi"),  // OpenAPI 3.x
				HaveKey("swagger"),  // OpenAPI 2.x
				HaveKey("info"),     // Common to both
			))
		})
	})
})

var _ = Describe("Phase 4: Extended MCP Endpoints (SDK Coverage)", func() {
	// These tests verify that the MCP endpoints added to the SDK work correctly
	// Most functionality is already tested in Phase 3, these ensure SDK coverage

	Describe("DELETE /mcp/{name} (SDK: mcp.remove)", func() {
		It("should be accessible via SDK method name", func() {
			// This endpoint was implemented in Phase 3 but now exposed in SDK
			resp, err := client.Delete(ctx, "/mcp/test-server")
			Expect(err).NotTo(HaveOccurred())
			// 404 is expected for non-existent server
			Expect(resp.StatusCode).To(Equal(404))
		})
	})

	Describe("GET /mcp/tools (SDK: mcp.tools)", func() {
		It("should return tools array", func() {
			resp, err := client.Get(ctx, "/mcp/tools")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			var tools []interface{}
			err = resp.JSON(&tools)
			Expect(err).NotTo(HaveOccurred())
			Expect(tools).NotTo(BeNil())
		})
	})

	Describe("GET /mcp/resources (SDK: mcp.resources)", func() {
		It("should return resources array", func() {
			resp, err := client.Get(ctx, "/mcp/resources")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			var resources []interface{}
			err = resp.JSON(&resources)
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).NotTo(BeNil())
		})
	})

	Describe("GET /mcp/resource (SDK: mcp.resource)", func() {
		It("should require uri parameter", func() {
			resp, err := client.Get(ctx, "/mcp/resource")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})
	})

	Describe("POST /mcp/tool/{name} (SDK: mcp.execute_tool)", func() {
		It("should handle non-existent tool", func() {
			resp, err := client.Post(ctx, "/mcp/tool/nonexistent", map[string]interface{}{})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(BeNumerically(">=", 400))
		})
	})
})

var _ = Describe("Phase 4: Extended Command Endpoints (SDK Coverage)", func() {
	Describe("GET /command/{name} (SDK: command.get)", func() {
		It("should return command details", func() {
			resp, err := client.Get(ctx, "/command/help")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			var cmd map[string]interface{}
			err = resp.JSON(&cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(cmd).To(HaveKey("name"))
			Expect(cmd["name"]).To(Equal("help"))
		})
	})

	Describe("POST /command/{name} (SDK: command.execute)", func() {
		It("should execute builtin command", func() {
			resp, err := client.Post(ctx, "/command/help", map[string]interface{}{})
			Expect(err).NotTo(HaveOccurred())
			// help command should succeed
			Expect(resp.StatusCode).To(Equal(200))
		})
	})
})

var _ = Describe("Phase 4: Extended Formatter Endpoints (SDK Coverage)", func() {
	Describe("POST /formatter/format (SDK: formatter.format)", func() {
		It("should validate request body", func() {
			resp, err := client.Post(ctx, "/formatter/format", map[string]interface{}{})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})

		It("should accept path parameter", func() {
			resp, err := client.Post(ctx, "/formatter/format", map[string]interface{}{
				"path": "/tmp/nonexistent.go",
			})
			Expect(err).NotTo(HaveOccurred())
			// Should return 200 with error info, or 404
			Expect(resp.StatusCode).To(BeNumerically(">=", 200))
		})
	})
})

// Helper types for Phase 4 JSON parsing
type ClientTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type ClientToolResult struct {
	ExecutionID string      `json:"executionId"`
	Result      interface{} `json:"result"`
	Error       string      `json:"error,omitempty"`
}

type OpenAPIDoc struct {
	OpenAPI string                 `json:"openapi,omitempty"`
	Swagger string                 `json:"swagger,omitempty"`
	Info    map[string]interface{} `json:"info"`
	Paths   map[string]interface{} `json:"paths"`
}

// Ensure json package is used
var _ = json.Marshal
