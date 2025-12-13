package server_test

import (
	"context"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/joho/godotenv"

	"github.com/opencode-ai/opencode/citest/testutil"
)

var (
	testServer *testutil.TestServer
	client     *testutil.TestClient
	ctx        context.Context
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = BeforeSuite(func() {
	// Load environment variables from .env file first
	_ = godotenv.Load("../../.env")

	// Skip env var check for mockllm provider
	testProvider := os.Getenv("TEST_PROVIDER")
	if testProvider != "mockllm" {
		// Skip if required env vars are missing (only for real providers)
		switch testProvider {
		case "ark":
			if testutil.SkipIfMissingEnv("ARK_API_KEY", "ARK_MODEL_ID") {
				Skip("ARK environment variables not set")
			}
		case "anthropic":
			if testutil.SkipIfMissingEnv("ANTHROPIC_API_KEY") {
				Skip("ANTHROPIC_API_KEY not set")
			}
		case "openai":
			if testutil.SkipIfMissingEnv("OPENAI_API_KEY") {
				Skip("OPENAI_API_KEY not set")
			}
		default:
			// Default to ARK for backwards compatibility
			if testutil.SkipIfMissingEnv("ARK_API_KEY", "ARK_MODEL_ID") {
				Skip("ARK environment variables not set")
			}
		}
	}

	var err error
	testServer, err = testutil.StartTestServer()
	Expect(err).NotTo(HaveOccurred(), "Failed to start test server")

	client = testServer.Client()
	ctx = context.Background()
})

var _ = AfterSuite(func() {
	if testServer != nil {
		testServer.Stop()
	}
})
