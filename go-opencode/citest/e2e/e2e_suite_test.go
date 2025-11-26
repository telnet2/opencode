package e2e_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	opencode "github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"

	"github.com/opencode-ai/opencode/citest/testutil"
)

var (
	testServer *testutil.TestServer
	client     *opencode.Client
	ctx        context.Context
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func() {
	// Skip if required env vars are missing
	if testutil.SkipIfMissingEnv("ARK_API_KEY", "ARK_MODEL_ID") {
		Skip("ARK environment variables not set")
	}

	var err error
	testServer, err = testutil.StartTestServer()
	Expect(err).NotTo(HaveOccurred(), "Failed to start test server")

	// Create SDK client pointing to test server
	client = opencode.NewClient(
		option.WithBaseURL(testServer.BaseURL),
	)
	ctx = context.Background()
})

var _ = AfterSuite(func() {
	if testServer != nil {
		testServer.Stop()
	}
})
