package service_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencode-ai/opencode/citest/testutil"
)

var _ = Describe("GET /find/symbol", func() {
	Describe("Query Parameter Validation", func() {
		It("should return 400 when query is missing", func() {
			resp, err := client.Get(ctx, "/find/symbol")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))

			var errResp struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			Expect(resp.JSON(&errResp)).To(Succeed())
			Expect(errResp.Error.Code).To(Equal("INVALID_REQUEST"))
			Expect(errResp.Error.Message).To(Equal("query parameter required"))
		})

		It("should return 400 when query is empty", func() {
			resp, err := client.Get(ctx, "/find/symbol",
				testutil.WithQuery(map[string]string{"query": ""}))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})
	})

	Describe("Basic Functionality", func() {
		It("should return 200 with array for valid query", func() {
			resp, err := client.Get(ctx, "/find/symbol",
				testutil.WithQuery(map[string]string{"query": "nonexistent_xyz_123"}))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			var symbols []any
			Expect(resp.JSON(&symbols)).To(Succeed())
			// Should return an array (might be empty if no LSP or no matches)
			Expect(symbols).NotTo(BeNil())
		})

		It("should return empty array when no symbols match", func() {
			resp, err := client.Get(ctx, "/find/symbol",
				testutil.WithQuery(map[string]string{"query": "unlikely_nonexistent_symbol_xyz_999"}))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			var symbols []any
			Expect(resp.JSON(&symbols)).To(Succeed())
			// For non-matching queries or when LSP is disabled, should return empty array
			Expect(symbols).To(BeEmpty())
		})
	})

	Describe("Response Format", func() {
		It("should return proper JSON array structure", func() {
			resp, err := client.Get(ctx, "/find/symbol",
				testutil.WithQuery(map[string]string{"query": "Test"}))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			// Verify response is valid JSON array
			var symbols []map[string]any
			Expect(resp.JSON(&symbols)).To(Succeed())

			// If any symbols are returned, verify structure
			if len(symbols) > 0 {
				symbol := symbols[0]
				Expect(symbol).To(HaveKey("name"))
				Expect(symbol).To(HaveKey("kind"))
				Expect(symbol).To(HaveKey("location"))

				// Verify location structure
				location, ok := symbol["location"].(map[string]any)
				Expect(ok).To(BeTrue(), "location should be an object")
				Expect(location).To(HaveKey("uri"))
				Expect(location).To(HaveKey("range"))

				// Verify range structure
				rangeObj, ok := location["range"].(map[string]any)
				Expect(ok).To(BeTrue(), "range should be an object")
				Expect(rangeObj).To(HaveKey("start"))
				Expect(rangeObj).To(HaveKey("end"))
			}
		})

		It("should limit results to at most 10 symbols", func() {
			// Query for something common that might return many results
			resp, err := client.Get(ctx, "/find/symbol",
				testutil.WithQuery(map[string]string{"query": ""}))
			Expect(err).NotTo(HaveOccurred())
			// Empty query should return 400
			Expect(resp.StatusCode).To(Equal(400))

			// Query with single character (if LSP is active, might return many results)
			resp, err = client.Get(ctx, "/find/symbol",
				testutil.WithQuery(map[string]string{"query": "a"}))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			var symbols []any
			Expect(resp.JSON(&symbols)).To(Succeed())
			// Should never return more than 10 results
			Expect(len(symbols)).To(BeNumerically("<=", 10))
		})
	})

	Describe("Symbol Kind Filtering", func() {
		It("should only return allowed symbol kinds if symbols are returned", func() {
			resp, err := client.Get(ctx, "/find/symbol",
				testutil.WithQuery(map[string]string{"query": "main"}))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			var symbols []map[string]any
			Expect(resp.JSON(&symbols)).To(Succeed())

			// Allowed kinds: Class(5), Method(6), Enum(10), Interface(11),
			// Function(12), Variable(13), Constant(14), Struct(23)
			allowedKinds := []float64{5, 6, 10, 11, 12, 13, 14, 23}

			for _, sym := range symbols {
				kind, ok := sym["kind"].(float64)
				if ok {
					Expect(allowedKinds).To(ContainElement(kind),
						"Symbol kind %v should be in allowed kinds", kind)
				}
			}
		})
	})
})
