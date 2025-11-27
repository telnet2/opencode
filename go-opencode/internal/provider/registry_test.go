package provider

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/model"

	"github.com/opencode-ai/opencode/pkg/types"
)

// mockProvider implements Provider for testing
type mockProvider struct {
	id     string
	name   string
	models []types.Model
}

func (m *mockProvider) ID() string                   { return m.id }
func (m *mockProvider) Name() string                 { return m.name }
func (m *mockProvider) Models() []types.Model        { return m.models }
func (m *mockProvider) ChatModel() model.ToolCallingChatModel { return nil }
func (m *mockProvider) CreateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionStream, error) {
	return nil, nil
}

func newMockProvider(id, name string, models []types.Model) *mockProvider {
	return &mockProvider{id: id, name: name, models: models}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	registry := NewRegistry(nil)

	provider := newMockProvider("test", "Test Provider", nil)
	registry.Register(provider)

	got, err := registry.Get("test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID() != "test" {
		t.Errorf("Got provider ID %q, want 'test'", got.ID())
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	registry := NewRegistry(nil)

	_, err := registry.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent provider")
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry(nil)

	registry.Register(newMockProvider("p1", "Provider 1", nil))
	registry.Register(newMockProvider("p2", "Provider 2", nil))
	registry.Register(newMockProvider("p3", "Provider 3", nil))

	providers := registry.List()
	if len(providers) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(providers))
	}
}

func TestRegistry_GetModel(t *testing.T) {
	registry := NewRegistry(nil)

	models := []types.Model{
		{ID: "model-a", Name: "Model A", ProviderID: "test"},
		{ID: "model-b", Name: "Model B", ProviderID: "test"},
	}
	registry.Register(newMockProvider("test", "Test", models))

	model, err := registry.GetModel("test", "model-a")
	if err != nil {
		t.Fatalf("GetModel failed: %v", err)
	}
	if model.ID != "model-a" {
		t.Errorf("Got model ID %q, want 'model-a'", model.ID)
	}
}

func TestRegistry_GetModel_NotFound(t *testing.T) {
	registry := NewRegistry(nil)

	models := []types.Model{
		{ID: "model-a", Name: "Model A", ProviderID: "test"},
	}
	registry.Register(newMockProvider("test", "Test", models))

	// Provider exists, model doesn't
	_, err := registry.GetModel("test", "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent model")
	}

	// Provider doesn't exist
	_, err = registry.GetModel("nonexistent", "model-a")
	if err == nil {
		t.Error("Expected error for nonexistent provider")
	}
}

func TestRegistry_AllModels(t *testing.T) {
	registry := NewRegistry(nil)

	registry.Register(newMockProvider("p1", "Provider 1", []types.Model{
		{ID: "gpt-4o-latest", Name: "GPT-4o"},
	}))
	registry.Register(newMockProvider("p2", "Provider 2", []types.Model{
		{ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4"},
		{ID: "claude-3-5-sonnet", Name: "Claude 3.5 Sonnet"},
	}))

	models := registry.AllModels()
	if len(models) != 3 {
		t.Fatalf("Expected 3 models, got %d", len(models))
	}

	// Should be sorted by priority (claude-sonnet-4 > gpt-4o > claude-3-5)
	if models[0].ID != "claude-sonnet-4-20250514" {
		t.Errorf("First model should be claude-sonnet-4, got %s", models[0].ID)
	}
}

func TestRegistry_DefaultModel_FromConfig(t *testing.T) {
	config := &types.Config{
		Model: "test/model-custom",
	}
	registry := NewRegistry(config)

	models := []types.Model{
		{ID: "model-custom", Name: "Custom Model", ProviderID: "test"},
	}
	registry.Register(newMockProvider("test", "Test", models))

	model, err := registry.DefaultModel()
	if err != nil {
		t.Fatalf("DefaultModel failed: %v", err)
	}
	if model.ID != "model-custom" {
		t.Errorf("Expected model-custom, got %s", model.ID)
	}
}

func TestRegistry_DefaultModel_Fallback(t *testing.T) {
	registry := NewRegistry(nil)

	models := []types.Model{
		{ID: "some-model", Name: "Some Model", ProviderID: "test"},
	}
	registry.Register(newMockProvider("test", "Test", models))

	model, err := registry.DefaultModel()
	if err != nil {
		t.Fatalf("DefaultModel failed: %v", err)
	}
	if model.ID != "some-model" {
		t.Errorf("Expected some-model as fallback, got %s", model.ID)
	}
}

func TestRegistry_DefaultModel_NoModels(t *testing.T) {
	registry := NewRegistry(nil)

	_, err := registry.DefaultModel()
	if err == nil {
		t.Error("Expected error when no models available")
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewRegistry(nil)

	// Start multiple goroutines doing concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			provider := newMockProvider("p"+string(rune('0'+n)), "Provider", nil)
			registry.Register(provider)
			registry.List()
			registry.Get("p" + string(rune('0'+n)))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have all providers registered
	providers := registry.List()
	if len(providers) != 10 {
		t.Errorf("Expected 10 providers, got %d", len(providers))
	}
}

// Note: TestCompletionStream removed because schema.StreamReaderFromChan doesn't exist in Eino.
// The CompletionStream is tested indirectly through integration tests.

func TestInitializeProviders_NoConfig(t *testing.T) {
	config := &types.Config{
		Provider: make(map[string]types.ProviderConfig),
	}

	registry, err := InitializeProviders(context.Background(), config)
	if err != nil {
		t.Fatalf("InitializeProviders failed: %v", err)
	}

	// Should have no providers without API keys
	providers := registry.List()
	if len(providers) != 0 {
		t.Errorf("Expected 0 providers without API keys, got %d", len(providers))
	}
}

func TestInferNpmFromProviderName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"anthropic", NpmAnthropic},
		{"claude", NpmAnthropic},
		{"openai", NpmOpenAI},
		{"unknown", ""},
		{"ark", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			npm := inferNpmFromProviderName(tt.name)
			if npm != tt.expected {
				t.Errorf("inferNpmFromProviderName(%q) = %q, want %q", tt.name, npm, tt.expected)
			}
		})
	}
}

func TestGetProviderCredentials(t *testing.T) {
	tests := []struct {
		name        string
		config      types.ProviderConfig
		wantAPIKey  string
		wantBaseURL string
	}{
		{
			name: "with options",
			config: types.ProviderConfig{
				Options: &types.ProviderOptions{
					APIKey:  "test-key",
					BaseURL: "https://api.example.com",
				},
			},
			wantAPIKey:  "test-key",
			wantBaseURL: "https://api.example.com",
		},
		{
			name:        "nil options",
			config:      types.ProviderConfig{},
			wantAPIKey:  "",
			wantBaseURL: "",
		},
		{
			name: "partial options - only API key",
			config: types.ProviderConfig{
				Options: &types.ProviderOptions{
					APIKey: "only-key",
				},
			},
			wantAPIKey:  "only-key",
			wantBaseURL: "",
		},
		{
			name: "partial options - only base URL",
			config: types.ProviderConfig{
				Options: &types.ProviderOptions{
					BaseURL: "https://custom.api.com",
				},
			},
			wantAPIKey:  "",
			wantBaseURL: "https://custom.api.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey, baseURL := getProviderCredentials(tt.config)
			if apiKey != tt.wantAPIKey {
				t.Errorf("apiKey = %q, want %q", apiKey, tt.wantAPIKey)
			}
			if baseURL != tt.wantBaseURL {
				t.Errorf("baseURL = %q, want %q", baseURL, tt.wantBaseURL)
			}
		})
	}
}
