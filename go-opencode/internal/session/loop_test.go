package session

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"github.com/opencode-ai/opencode/internal/permission"
	"github.com/opencode-ai/opencode/internal/provider"
	"github.com/opencode-ai/opencode/internal/storage"
	"github.com/opencode-ai/opencode/internal/tool"
	"github.com/opencode-ai/opencode/pkg/types"
)

func TestAgenticLoopWithRealLLM(t *testing.T) {
	// Load environment variables
	godotenv.Load("../../.env")

	apiKey := os.Getenv("ARK_API_KEY")
	modelID := os.Getenv("ARK_MODEL_ID")
	baseURL := os.Getenv("ARK_BASE_URL")

	if apiKey == "" || modelID == "" {
		t.Skip("ARK_API_KEY and ARK_MODEL_ID required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create config
	cfg := &types.Config{
		Model: "ark/" + modelID,
		Provider: map[string]types.ProviderConfig{
			"ark": {
				APIKey:  apiKey,
				BaseURL: baseURL,
				Model:   modelID,
			},
		},
	}

	// Initialize providers
	providerReg, err := provider.InitializeProviders(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to initialize providers: %v", err)
	}

	// Create temp storage
	tempDir, _ := os.MkdirTemp("", "test-session-*")
	defer os.RemoveAll(tempDir)
	store := storage.New(tempDir)

	// Create processor
	toolReg := tool.DefaultRegistry(tempDir)
	permChecker := permission.NewChecker()
	processor := NewProcessor(providerReg, toolReg, store, permChecker, "ark", modelID)

	// Create a session
	sessionID := "test-session"
	session := &types.Session{
		ID:        sessionID,
		Directory: tempDir,
	}
	store.Put(ctx, []string{"session", sessionID}, session)

	// Create user message
	userMsg := &types.Message{
		ID:        "user-msg-1",
		SessionID: sessionID,
		Role:      "user",
		Time:      types.MessageTime{Created: time.Now().UnixMilli()},
	}
	store.Put(ctx, []string{"message", sessionID, userMsg.ID}, userMsg)

	// Create user message part
	userPart := &types.TextPart{
		ID:   "user-part-1",
		Type: "text",
		Text: "Say hello in one word.",
	}
	store.Put(ctx, []string{"part", userMsg.ID, userPart.ID}, userPart)

	// Track what we receive
	var receivedParts []types.Part
	var receivedMsg *types.Message
	callbackCount := 0

	// Run the loop
	err = processor.Process(ctx, sessionID, DefaultAgent(), func(msg *types.Message, ps []types.Part) {
		receivedMsg = msg
		receivedParts = ps
		callbackCount++
		t.Logf("Callback #%d: msg=%+v, parts count=%d", callbackCount, msg.ID, len(ps))
		for i, p := range ps {
			switch pt := p.(type) {
			case *types.TextPart:
				t.Logf("  Part %d: TextPart text=%q", i, pt.Text)
			case *types.ToolPart:
				t.Logf("  Part %d: ToolPart tool=%s", i, pt.ToolName)
			default:
				t.Logf("  Part %d: Unknown type %T", i, p)
			}
		}
	})

	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	t.Logf("Final parts count: %d", len(receivedParts))
	t.Logf("Total callbacks: %d", callbackCount)

	// Verify callback was called
	if callbackCount == 0 {
		t.Fatal("Callback was not called")
	}

	if receivedMsg == nil {
		t.Fatal("Expected assistant message")
	}

	if len(receivedParts) == 0 {
		t.Fatal("Expected at least one part")
	}

	t.Logf("Test passed! Received %d parts", len(receivedParts))
}
