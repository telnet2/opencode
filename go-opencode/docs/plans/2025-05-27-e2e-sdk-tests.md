# Plan: End-to-End SDK Tests

## Overview

This plan covers end-to-end tests using the OpenCode Go SDK (`github.com/sst/opencode-sdk-go`). These tests validate complete workflows from the client perspective, using the SDK's high-level abstractions.

### Location
`citest/e2e/`

### Focus Areas
1. **SDK Client Integration** - Verify SDK works with our server
2. **Complete Workflows** - Session creation through tool execution
3. **Event Streaming** - Real-time event subscription via SDK
4. **Error Handling** - SDK error types and recovery

---

## Part 1: SDK Client Setup (`e2e_suite_test.go`)

```ginkgo
package e2e_test

import (
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    opencode "github.com/sst/opencode-sdk-go"
    "github.com/sst/opencode-sdk-go/option"
)

var (
    testServer *testutil.TestServer
    client     *opencode.Client
)

var _ = BeforeSuite(func() {
    var err error
    testServer, err = testutil.StartTestServer()
    Expect(err).NotTo(HaveOccurred())

    client = opencode.NewClient(
        option.WithBaseURL(testServer.BaseURL),
    )
})

var _ = AfterSuite(func() {
    if testServer != nil {
        testServer.Stop()
    }
})

func TestE2E(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "E2E Suite")
}
```

---

## Part 2: Session Workflows (`workflow_test.go`)

```ginkgo
var _ = Describe("Session Workflows", func() {

    Describe("Basic Session Lifecycle", func() {
        var session *opencode.Session

        It("should create a new session", func() {
            var err error
            session, err = client.Session.New(ctx, opencode.SessionNewParams{
                Directory: opencode.F("/tmp"),
                Title:     opencode.F("Test Session"),
            })
            Expect(err).NotTo(HaveOccurred())
            Expect(session.ID).NotTo(BeEmpty())
            Expect(session.Title).To(Equal("Test Session"))
        })

        It("should retrieve session by ID", func() {
            retrieved, err := client.Session.Get(ctx, session.ID)
            Expect(err).NotTo(HaveOccurred())
            Expect(retrieved.ID).To(Equal(session.ID))
        })

        It("should list sessions", func() {
            sessions, err := client.Session.List(ctx)
            Expect(err).NotTo(HaveOccurred())
            Expect(sessions).NotTo(BeEmpty())

            found := false
            for _, s := range sessions {
                if s.ID == session.ID {
                    found = true
                    break
                }
            }
            Expect(found).To(BeTrue())
        })

        It("should delete session", func() {
            err := client.Session.Delete(ctx, session.ID)
            Expect(err).NotTo(HaveOccurred())

            _, err = client.Session.Get(ctx, session.ID)
            Expect(err).To(HaveOccurred())
        })
    })
})
```

---

## Part 3: Message Workflows (`message_test.go`)

```ginkgo
var _ = Describe("Message Workflows", func() {
    var session *opencode.Session

    BeforeEach(func() {
        var err error
        session, err = client.Session.New(ctx, opencode.SessionNewParams{
            Directory: opencode.F("/tmp"),
        })
        Expect(err).NotTo(HaveOccurred())
    })

    AfterEach(func() {
        if session != nil {
            client.Session.Delete(ctx, session.ID)
        }
    })

    Describe("Simple Message Exchange", func() {
        It("should send message and receive response", func() {
            response, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
                Message: opencode.F("Say 'Hello, World!' and nothing else."),
            })
            Expect(err).NotTo(HaveOccurred())
            Expect(response.Info).NotTo(BeNil())
            Expect(response.Info.Content).To(ContainSubstring("Hello"))
        })

        It("should maintain conversation context", func() {
            // First message
            _, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
                Message: opencode.F("Remember the number 42."),
            })
            Expect(err).NotTo(HaveOccurred())

            // Second message referencing first
            response, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
                Message: opencode.F("What number did I ask you to remember?"),
            })
            Expect(err).NotTo(HaveOccurred())
            Expect(response.Info.Content).To(ContainSubstring("42"))
        })
    })

    Describe("Message Retrieval", func() {
        It("should retrieve all messages in session", func() {
            // Send a message
            _, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
                Message: opencode.F("Hello"),
            })
            Expect(err).NotTo(HaveOccurred())

            // Get messages
            messages, err := client.Session.Messages(ctx, session.ID)
            Expect(err).NotTo(HaveOccurred())
            Expect(len(messages)).To(BeNumerically(">=", 2)) // user + assistant
        })

        It("should retrieve specific message by ID", func() {
            response, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
                Message: opencode.F("Test message"),
            })
            Expect(err).NotTo(HaveOccurred())

            message, err := client.Session.Message(ctx, session.ID, response.Info.ID)
            Expect(err).NotTo(HaveOccurred())
            Expect(message.ID).To(Equal(response.Info.ID))
        })
    })
})
```

---

## Part 4: Tool Execution Workflows (`tools_test.go`)

```ginkgo
var _ = Describe("Tool Execution Workflows", func() {
    var session *opencode.Session

    BeforeEach(func() {
        var err error
        session, err = client.Session.New(ctx, opencode.SessionNewParams{
            Directory: opencode.F("/tmp"),
        })
        Expect(err).NotTo(HaveOccurred())
    })

    AfterEach(func() {
        if session != nil {
            client.Session.Delete(ctx, session.ID)
        }
    })

    Describe("Bash Tool Execution", func() {
        It("should execute bash command via prompt", func() {
            response, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
                Message: opencode.F("Run the command 'echo hello world' in bash and tell me the output."),
            })
            Expect(err).NotTo(HaveOccurred())
            Expect(response.Info.Content).To(ContainSubstring("hello world"))
        })

        It("should capture command exit status", func() {
            response, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
                Message: opencode.F("Run 'ls /tmp' and tell me if it succeeded."),
            })
            Expect(err).NotTo(HaveOccurred())
            // Should indicate success
            Expect(response.Info.Content).To(MatchRegexp(`(?i)(success|succeeded|worked)`))
        })
    })

    Describe("File Operations", func() {
        var testFile string

        BeforeEach(func() {
            testFile = "/tmp/opencode-test-" + randomString(8) + ".txt"
        })

        AfterEach(func() {
            os.Remove(testFile)
        })

        It("should read file content", func() {
            // Create test file
            err := os.WriteFile(testFile, []byte("test content for reading"), 0644)
            Expect(err).NotTo(HaveOccurred())

            response, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
                Message: opencode.F(fmt.Sprintf("Read the file %s and tell me what it contains.", testFile)),
            })
            Expect(err).NotTo(HaveOccurred())
            Expect(response.Info.Content).To(ContainSubstring("test content for reading"))
        })

        It("should write file content", func() {
            response, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
                Message: opencode.F(fmt.Sprintf("Write the text 'hello from opencode' to the file %s", testFile)),
            })
            Expect(err).NotTo(HaveOccurred())

            // Verify file was created
            content, err := os.ReadFile(testFile)
            Expect(err).NotTo(HaveOccurred())
            Expect(string(content)).To(ContainSubstring("hello from opencode"))
        })

        It("should handle file not found", func() {
            response, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
                Message: opencode.F("Read the file /nonexistent/path/file.txt"),
            })
            Expect(err).NotTo(HaveOccurred())
            // Should indicate error or file not found
            Expect(response.Info.Content).To(MatchRegexp(`(?i)(not found|doesn't exist|error|cannot)`))
        })
    })

    Describe("Multi-Tool Workflow", func() {
        It("should chain multiple tool calls", func() {
            testFile := "/tmp/opencode-chain-" + randomString(8) + ".txt"
            defer os.Remove(testFile)

            response, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
                Message: opencode.F(fmt.Sprintf(
                    "Please do the following: 1) Write 'step one complete' to %s, 2) Read the file back, 3) Tell me what you read.",
                    testFile,
                )),
            })
            Expect(err).NotTo(HaveOccurred())
            Expect(response.Info.Content).To(ContainSubstring("step one complete"))
        })
    })
})
```

---

## Part 5: Event Streaming (`events_test.go`)

```ginkgo
var _ = Describe("Event Streaming", func() {
    var session *opencode.Session

    BeforeEach(func() {
        var err error
        session, err = client.Session.New(ctx, opencode.SessionNewParams{
            Directory: opencode.F("/tmp"),
        })
        Expect(err).NotTo(HaveOccurred())
    })

    AfterEach(func() {
        if session != nil {
            client.Session.Delete(ctx, session.ID)
        }
    })

    Describe("Session Event Streaming", func() {
        It("should receive events for session activity", func() {
            ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
            defer cancel()

            // Start event stream
            stream := client.Event.ListStreaming(ctx, opencode.EventListParams{
                SessionID: opencode.F(session.ID),
            })

            // Collect events in goroutine
            events := make(chan opencode.Event, 10)
            go func() {
                for stream.Next() {
                    events <- stream.Current()
                }
                close(events)
            }()

            // Trigger activity
            _, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
                Message: opencode.F("Hello"),
            })
            Expect(err).NotTo(HaveOccurred())

            // Verify we received events
            receivedEvents := []opencode.Event{}
            timeout := time.After(5 * time.Second)
        loop:
            for {
                select {
                case evt, ok := <-events:
                    if !ok {
                        break loop
                    }
                    receivedEvents = append(receivedEvents, evt)
                    if len(receivedEvents) >= 3 {
                        break loop
                    }
                case <-timeout:
                    break loop
                }
            }

            Expect(len(receivedEvents)).To(BeNumerically(">", 0))
        })
    })

    Describe("Global Event Streaming", func() {
        It("should receive events from all sessions", func() {
            // Similar to above but without session filter
            // Create multiple sessions, verify events from all
        })
    })
})
```

---

## Part 6: Error Handling (`errors_test.go`)

```ginkgo
var _ = Describe("SDK Error Handling", func() {

    Describe("Not Found Errors", func() {
        It("should return error for non-existent session", func() {
            _, err := client.Session.Get(ctx, "nonexistent-session-id")
            Expect(err).To(HaveOccurred())

            var apiErr *opencode.Error
            Expect(errors.As(err, &apiErr)).To(BeTrue())
            Expect(apiErr.StatusCode).To(Equal(404))
        })
    })

    Describe("Invalid Request Errors", func() {
        It("should return error for invalid parameters", func() {
            _, err := client.Session.New(ctx, opencode.SessionNewParams{
                Directory: opencode.F(""), // Empty directory
            })
            // May succeed or fail depending on server validation
            // Test the error structure if it fails
            if err != nil {
                var apiErr *opencode.Error
                if errors.As(err, &apiErr) {
                    Expect(apiErr.StatusCode).To(BeNumerically(">=", 400))
                }
            }
        })
    })

    Describe("Request Timeout", func() {
        It("should respect context timeout", func() {
            ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
            defer cancel()

            _, err := client.Session.List(ctx)
            Expect(err).To(HaveOccurred())
            Expect(errors.Is(err, context.DeadlineExceeded) ||
                strings.Contains(err.Error(), "deadline")).To(BeTrue())
        })
    })

    Describe("Error Details", func() {
        It("should include request ID in error", func() {
            _, err := client.Session.Get(ctx, "nonexistent")
            Expect(err).To(HaveOccurred())

            var apiErr *opencode.Error
            if errors.As(err, &apiErr) {
                // Check if request details are available
                dump := apiErr.DumpRequest(false)
                Expect(dump).NotTo(BeEmpty())
            }
        })
    })
})
```

---

## Part 7: Configuration and Providers (`config_test.go`)

```ginkgo
var _ = Describe("Configuration", func() {

    Describe("GET /config", func() {
        It("should retrieve configuration", func() {
            config, err := client.Config.Get(ctx)
            Expect(err).NotTo(HaveOccurred())
            Expect(config).NotTo(BeNil())
        })
    })

    Describe("Provider Listing", func() {
        It("should list available providers", func() {
            providers, err := client.App.Providers(ctx)
            Expect(err).NotTo(HaveOccurred())
            Expect(providers).NotTo(BeEmpty())

            // Verify ARK provider is available
            found := false
            for _, p := range providers {
                if p.ID == "ark" {
                    found = true
                    break
                }
            }
            Expect(found).To(BeTrue())
        })
    })
})
```

---

## Test Utilities

### Random String Helper

```go
func randomString(n int) string {
    const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
    b := make([]byte, n)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
    return string(b)
}
```

---

## Running Tests

```bash
# Run all E2E tests
cd citest && ginkgo -v ./e2e/

# Run specific workflow
cd citest && ginkgo -v --focus="Tool Execution" ./e2e/

# Run with timeout (for CI)
cd citest && ginkgo -v --timeout=5m ./e2e/

# Run sequentially (avoid session conflicts)
cd citest && ginkgo -v -p=1 ./e2e/
```

---

## CI Integration

```yaml
# .github/workflows/integration-tests.yml
name: Integration Tests

on:
  push:
    branches: [main]
  pull_request:

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Install Ginkgo
        run: go install github.com/onsi/ginkgo/v2/ginkgo@latest

      - name: Run E2E Tests
        env:
          ARK_API_KEY: ${{ secrets.ARK_API_KEY }}
          ARK_MODEL_ID: ${{ secrets.ARK_MODEL_ID }}
          ARK_BASE_URL: ${{ secrets.ARK_BASE_URL }}
        run: |
          cd citest
          ginkgo -v --timeout=10m ./e2e/
```

---

## Success Criteria

1. SDK client connects to test server successfully
2. Session CRUD operations work via SDK
3. Message prompts receive LLM responses
4. Tool execution (bash, file) works through SDK
5. Event streaming delivers real-time updates
6. Error handling provides actionable information
7. All tests pass within reasonable timeout (5min)
