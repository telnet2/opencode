package service_test

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

func TestService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Suite")
}

var _ = BeforeSuite(func() {
	// Load environment variables from .env file first
	_ = godotenv.Load("../../.env")

	// Skip env var check for mockllm provider
	testProvider := os.Getenv("TEST_PROVIDER")
	if testProvider != "mockllm" {
		// Skip if required env vars are missing (only for real providers)
		if testutil.SkipIfMissingEnv("ARK_API_KEY", "ARK_MODEL_ID") {
			Skip("ARK environment variables not set")
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
