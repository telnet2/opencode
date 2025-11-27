# Implementation Plan: Provider OAuth Endpoints

## Overview

**Endpoints:**
- `POST /provider/:id/oauth/authorize` - Initiate OAuth flow
- `POST /provider/:id/oauth/callback` - Handle OAuth callback

**Current Status:** Returns 501 Not Implemented
**Priority:** Medium
**Effort:** High (requires state management, token storage)

---

## 1. Current State Analysis

### TypeScript Reference Implementation

**Location:** `packages/opencode/src/provider/auth.ts`

#### OAuth State Management

```typescript
// packages/opencode/src/provider/auth.ts:10-19
const state = Instance.state(async () => {
  const methods = pipe(
    await Plugin.list(),
    filter((x) => x.auth?.provider !== undefined),
    map((x) => [x.auth!.provider, x.auth!] as const),
    fromEntries(),
  )
  return { methods, pending: {} as Record<string, AuthOuathResult> }
})
```

#### Authorize Endpoint

```typescript
// packages/opencode/src/provider/auth.ts:54-72
export const authorize = fn(
  z.object({
    providerID: z.string(),
    method: z.number(),
  }),
  async (input): Promise<Authorization | undefined> => {
    const auth = await state().then((s) => s.methods[input.providerID])
    const method = auth.methods[input.method]
    if (method.type === "oauth") {
      const result = await method.authorize()
      await state().then((s) => (s.pending[input.providerID] = result))
      return {
        url: result.url,
        method: result.method,
        instructions: result.instructions,
      }
    }
  },
)
```

#### Callback Endpoint

```typescript
// packages/opencode/src/provider/auth.ts:74-114
export const callback = fn(
  z.object({
    providerID: z.string(),
    method: z.number(),
    code: z.string().optional(),
  }),
  async (input) => {
    const match = await state().then((s) => s.pending[input.providerID])
    if (!match) throw new OauthMissing({ providerID: input.providerID })

    let result
    if (match.method === "code") {
      if (!input.code) throw new OauthCodeMissing({ providerID: input.providerID })
      result = await match.callback(input.code)
    }
    if (match.method === "auto") {
      result = await match.callback()
    }

    if (result?.type === "success") {
      if ("key" in result) {
        await Auth.set(input.providerID, { type: "api", key: result.key })
      }
      if ("refresh" in result) {
        await Auth.set(input.providerID, {
          type: "oauth",
          access: result.access,
          refresh: result.refresh,
          expires: result.expires,
        })
      }
      return
    }

    throw new OauthCallbackFailed({})
  },
)
```

### Plugin System (TypeScript)

Providers with OAuth support are loaded via the plugin system:

```typescript
// Plugin interface for auth
interface AuthPlugin {
  provider: string
  methods: AuthMethod[]
}

interface AuthMethod {
  type: "oauth" | "api"
  label: string
  authorize?: () => Promise<AuthOuathResult>
}

interface AuthOuathResult {
  url: string
  method: "auto" | "code"
  instructions: string
  callback: (code?: string) => Promise<AuthResult>
}
```

### Go Current State

**Handlers:** `go-opencode/internal/server/handlers_config.go:85-93`

```go
func (s *Server) oauthAuthorize(w http.ResponseWriter, r *http.Request) {
    notImplemented(w)
}

func (s *Server) oauthCallback(w http.ResponseWriter, r *http.Request) {
    notImplemented(w)
}
```

---

## 2. Implementation Tasks

### Task 1: Create OAuth Package

**New File:** `go-opencode/internal/oauth/oauth.go`

```go
package oauth

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "errors"
    "fmt"
    "net/http"
    "net/url"
    "sync"
    "time"

    "golang.org/x/oauth2"
)

// Method represents an OAuth method type
type Method string

const (
    MethodAuto Method = "auto"
    MethodCode Method = "code"
)

// Authorization represents the response from authorize endpoint
type Authorization struct {
    URL          string `json:"url"`
    Method       Method `json:"method"`
    Instructions string `json:"instructions"`
}

// TokenResult represents successful OAuth token exchange
type TokenResult struct {
    Type    string `json:"type"` // "oauth" or "api"
    Access  string `json:"access,omitempty"`
    Refresh string `json:"refresh,omitempty"`
    Expires int64  `json:"expires,omitempty"`
    Key     string `json:"key,omitempty"`
}

// PendingAuth tracks an in-progress OAuth flow
type PendingAuth struct {
    ProviderID string
    Method     Method
    Config     *oauth2.Config
    State      string
    Verifier   string // PKCE verifier
    CreatedAt  time.Time
}

// ProviderConfig holds OAuth configuration for a provider
type ProviderConfig struct {
    ClientID     string
    ClientSecret string
    AuthURL      string
    TokenURL     string
    Scopes       []string
    RedirectURL  string
}

// Manager handles OAuth flows
type Manager struct {
    mu      sync.RWMutex
    pending map[string]*PendingAuth
    configs map[string]ProviderConfig
}

// NewManager creates a new OAuth manager
func NewManager() *Manager {
    return &Manager{
        pending: make(map[string]*PendingAuth),
        configs: make(map[string]ProviderConfig),
    }
}

// Global manager instance
var globalManager = NewManager()

// RegisterProvider registers OAuth configuration for a provider
func RegisterProvider(providerID string, config ProviderConfig) {
    globalManager.RegisterProvider(providerID, config)
}

func (m *Manager) RegisterProvider(providerID string, config ProviderConfig) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.configs[providerID] = config
}

// Authorize initiates an OAuth flow
func Authorize(providerID string, methodIndex int) (*Authorization, error) {
    return globalManager.Authorize(providerID, methodIndex)
}

func (m *Manager) Authorize(providerID string, methodIndex int) (*Authorization, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    config, ok := m.configs[providerID]
    if !ok {
        return nil, fmt.Errorf("unknown provider: %s", providerID)
    }

    // Generate state and PKCE verifier
    state, err := generateRandomString(32)
    if err != nil {
        return nil, fmt.Errorf("failed to generate state: %w", err)
    }

    verifier, err := generateRandomString(64)
    if err != nil {
        return nil, fmt.Errorf("failed to generate verifier: %w", err)
    }

    oauth2Config := &oauth2.Config{
        ClientID:     config.ClientID,
        ClientSecret: config.ClientSecret,
        Endpoint: oauth2.Endpoint{
            AuthURL:  config.AuthURL,
            TokenURL: config.TokenURL,
        },
        Scopes:      config.Scopes,
        RedirectURL: config.RedirectURL,
    }

    // Determine method based on provider capabilities
    method := MethodCode
    if config.ClientSecret == "" {
        // PKCE flow for public clients
        method = MethodAuto
    }

    // Build authorization URL
    authURL := oauth2Config.AuthCodeURL(state,
        oauth2.SetAuthURLParam("code_challenge", generateCodeChallenge(verifier)),
        oauth2.SetAuthURLParam("code_challenge_method", "S256"),
    )

    // Store pending auth
    m.pending[providerID] = &PendingAuth{
        ProviderID: providerID,
        Method:     method,
        Config:     oauth2Config,
        State:      state,
        Verifier:   verifier,
        CreatedAt:  time.Now(),
    }

    instructions := "Open the URL in your browser to authorize."
    if method == MethodCode {
        instructions = "Open the URL in your browser, authorize, and enter the code."
    }

    return &Authorization{
        URL:          authURL,
        Method:       method,
        Instructions: instructions,
    }, nil
}

// Callback handles the OAuth callback
func Callback(ctx context.Context, providerID string, code string) (*TokenResult, error) {
    return globalManager.Callback(ctx, providerID, code)
}

func (m *Manager) Callback(ctx context.Context, providerID string, code string) (*TokenResult, error) {
    m.mu.Lock()
    pending, ok := m.pending[providerID]
    if !ok {
        m.mu.Unlock()
        return nil, ErrNoPendingAuth
    }
    delete(m.pending, providerID)
    m.mu.Unlock()

    // Check if pending auth is expired (10 minutes)
    if time.Since(pending.CreatedAt) > 10*time.Minute {
        return nil, ErrAuthExpired
    }

    // Exchange code for token
    token, err := pending.Config.Exchange(ctx, code,
        oauth2.SetAuthURLParam("code_verifier", pending.Verifier),
    )
    if err != nil {
        return nil, fmt.Errorf("token exchange failed: %w", err)
    }

    return &TokenResult{
        Type:    "oauth",
        Access:  token.AccessToken,
        Refresh: token.RefreshToken,
        Expires: token.Expiry.Unix(),
    }, nil
}

// GetPending returns pending auth for a provider (for testing)
func GetPending(providerID string) *PendingAuth {
    return globalManager.GetPending(providerID)
}

func (m *Manager) GetPending(providerID string) *PendingAuth {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.pending[providerID]
}

// ClearExpired removes expired pending authentications
func (m *Manager) ClearExpired() {
    m.mu.Lock()
    defer m.mu.Unlock()

    for id, pending := range m.pending {
        if time.Since(pending.CreatedAt) > 10*time.Minute {
            delete(m.pending, id)
        }
    }
}

// Errors
var (
    ErrNoPendingAuth = errors.New("no pending authentication for provider")
    ErrAuthExpired   = errors.New("authentication request expired")
    ErrCodeRequired  = errors.New("authorization code required")
)

// Helper functions
func generateRandomString(length int) (string, error) {
    bytes := make([]byte, length)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return base64.RawURLEncoding.EncodeToString(bytes)[:length], nil
}

func generateCodeChallenge(verifier string) string {
    // SHA256 hash of verifier, base64url encoded
    h := sha256.Sum256([]byte(verifier))
    return base64.RawURLEncoding.EncodeToString(h[:])
}
```

### Task 2: Create Auth Storage

**New File:** `go-opencode/internal/auth/storage.go`

```go
package auth

import (
    "encoding/json"
    "os"
    "path/filepath"
    "sync"
)

// Info represents stored authentication
type Info struct {
    Type    string `json:"type"` // "oauth", "api", "wellknown"
    Key     string `json:"key,omitempty"`
    Access  string `json:"access,omitempty"`
    Refresh string `json:"refresh,omitempty"`
    Expires int64  `json:"expires,omitempty"`
    Token   string `json:"token,omitempty"`
}

// Storage manages authentication credentials
type Storage struct {
    mu       sync.RWMutex
    path     string
    auths    map[string]Info
}

var globalStorage *Storage

// Init initializes the auth storage
func Init(configDir string) error {
    path := filepath.Join(configDir, "auth.json")
    globalStorage = &Storage{
        path:  path,
        auths: make(map[string]Info),
    }
    return globalStorage.load()
}

func (s *Storage) load() error {
    data, err := os.ReadFile(s.path)
    if os.IsNotExist(err) {
        return nil
    }
    if err != nil {
        return err
    }
    return json.Unmarshal(data, &s.auths)
}

func (s *Storage) save() error {
    s.mu.RLock()
    data, err := json.MarshalIndent(s.auths, "", "  ")
    s.mu.RUnlock()
    if err != nil {
        return err
    }

    dir := filepath.Dir(s.path)
    if err := os.MkdirAll(dir, 0700); err != nil {
        return err
    }

    return os.WriteFile(s.path, data, 0600)
}

// Set stores authentication for a provider
func Set(providerID string, info Info) error {
    return globalStorage.Set(providerID, info)
}

func (s *Storage) Set(providerID string, info Info) error {
    s.mu.Lock()
    s.auths[providerID] = info
    s.mu.Unlock()
    return s.save()
}

// Get retrieves authentication for a provider
func Get(providerID string) (Info, bool) {
    return globalStorage.Get(providerID)
}

func (s *Storage) Get(providerID string) (Info, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    info, ok := s.auths[providerID]
    return info, ok
}

// Delete removes authentication for a provider
func Delete(providerID string) error {
    return globalStorage.Delete(providerID)
}

func (s *Storage) Delete(providerID string) error {
    s.mu.Lock()
    delete(s.auths, providerID)
    s.mu.Unlock()
    return s.save()
}
```

### Task 3: Update Handlers

**File:** `go-opencode/internal/server/handlers_config.go`

Replace the stub implementations:

```go
import (
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/opencode-ai/opencode/internal/auth"
    "github.com/opencode-ai/opencode/internal/oauth"
)

// oauthAuthorize handles POST /provider/{providerID}/oauth/authorize
func (s *Server) oauthAuthorize(w http.ResponseWriter, r *http.Request) {
    providerID := chi.URLParam(r, "providerID")
    if providerID == "" {
        writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "providerID required")
        return
    }

    var req struct {
        Method int `json:"method"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "invalid JSON body")
        return
    }

    authorization, err := oauth.Authorize(providerID, req.Method)
    if err != nil {
        writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
        return
    }

    if authorization == nil {
        // API key auth, no OAuth needed
        writeJSON(w, http.StatusOK, nil)
        return
    }

    writeJSON(w, http.StatusOK, authorization)
}

// oauthCallback handles POST /provider/{providerID}/oauth/callback
func (s *Server) oauthCallback(w http.ResponseWriter, r *http.Request) {
    providerID := chi.URLParam(r, "providerID")
    if providerID == "" {
        writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "providerID required")
        return
    }

    var req struct {
        Method int    `json:"method"`
        Code   string `json:"code,omitempty"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "invalid JSON body")
        return
    }

    // Check for pending auth
    pending := oauth.GetPending(providerID)
    if pending == nil {
        writeError(w, http.StatusBadRequest, "OAUTH_MISSING",
            "no pending authentication for provider")
        return
    }

    // For code method, code is required
    if pending.Method == oauth.MethodCode && req.Code == "" {
        writeError(w, http.StatusBadRequest, "OAUTH_CODE_MISSING",
            "authorization code required")
        return
    }

    // Exchange code for token
    result, err := oauth.Callback(r.Context(), providerID, req.Code)
    if err != nil {
        writeError(w, http.StatusBadRequest, "OAUTH_CALLBACK_FAILED", err.Error())
        return
    }

    // Store the authentication
    authInfo := auth.Info{
        Type: result.Type,
    }
    if result.Type == "oauth" {
        authInfo.Access = result.Access
        authInfo.Refresh = result.Refresh
        authInfo.Expires = result.Expires
    } else if result.Type == "api" {
        authInfo.Key = result.Key
    }

    if err := auth.Set(providerID, authInfo); err != nil {
        writeError(w, http.StatusInternalServerError, ErrCodeInternalError,
            "failed to store authentication")
        return
    }

    writeJSON(w, http.StatusOK, true)
}
```

### Task 4: Configure Provider OAuth Settings

**File:** `go-opencode/internal/oauth/providers.go`

```go
package oauth

// Predefined OAuth configurations for known providers
var KnownProviders = map[string]ProviderConfig{
    "github-copilot": {
        ClientID:    "", // Set via environment or config
        AuthURL:     "https://github.com/login/oauth/authorize",
        TokenURL:    "https://github.com/login/oauth/access_token",
        Scopes:      []string{"copilot"},
        RedirectURL: "http://localhost:8766/callback",
    },
    "amazon-q": {
        ClientID:    "",
        AuthURL:     "https://oidc.us-east-1.amazonaws.com/authorize",
        TokenURL:    "https://oidc.us-east-1.amazonaws.com/token",
        Scopes:      []string{"codewhisperer:completions"},
        RedirectURL: "http://localhost:8766/callback",
    },
}

// LoadFromConfig loads OAuth configurations from app config
func LoadFromConfig(config map[string]any) {
    for providerID, cfg := range config {
        if cfgMap, ok := cfg.(map[string]any); ok {
            providerCfg := ProviderConfig{}

            if clientID, ok := cfgMap["clientId"].(string); ok {
                providerCfg.ClientID = clientID
            }
            if clientSecret, ok := cfgMap["clientSecret"].(string); ok {
                providerCfg.ClientSecret = clientSecret
            }
            if authURL, ok := cfgMap["authUrl"].(string); ok {
                providerCfg.AuthURL = authURL
            }
            if tokenURL, ok := cfgMap["tokenUrl"].(string); ok {
                providerCfg.TokenURL = tokenURL
            }
            if scopes, ok := cfgMap["scopes"].([]any); ok {
                for _, s := range scopes {
                    if str, ok := s.(string); ok {
                        providerCfg.Scopes = append(providerCfg.Scopes, str)
                    }
                }
            }
            if redirectURL, ok := cfgMap["redirectUrl"].(string); ok {
                providerCfg.RedirectURL = redirectURL
            }

            RegisterProvider(providerID, providerCfg)
        }
    }
}
```

---

## 3. External Configuration

### Provider OAuth Configuration

**File:** `~/.config/opencode/config.json`

```json
{
  "provider": {
    "github-copilot": {
      "oauth": {
        "clientId": "YOUR_CLIENT_ID",
        "clientSecret": "YOUR_CLIENT_SECRET",
        "authUrl": "https://github.com/login/oauth/authorize",
        "tokenUrl": "https://github.com/login/oauth/access_token",
        "scopes": ["copilot"],
        "redirectUrl": "http://localhost:8766/callback"
      }
    },
    "amazon-q": {
      "oauth": {
        "clientId": "YOUR_CLIENT_ID",
        "authUrl": "https://oidc.us-east-1.amazonaws.com/authorize",
        "tokenUrl": "https://oidc.us-east-1.amazonaws.com/token",
        "scopes": ["codewhisperer:completions"],
        "redirectUrl": "http://localhost:8766/callback"
      }
    }
  }
}
```

### Environment Variables

```bash
# GitHub Copilot
GITHUB_COPILOT_CLIENT_ID=xxx
GITHUB_COPILOT_CLIENT_SECRET=xxx

# Amazon Q (AWS)
AWS_ACCESS_KEY_ID=xxx
AWS_SECRET_ACCESS_KEY=xxx
AWS_REGION=us-east-1
```

### Auth Storage Location

```
~/.config/opencode/auth.json
```

**Format:**
```json
{
  "github-copilot": {
    "type": "oauth",
    "access": "gho_xxxx",
    "refresh": "ghr_xxxx",
    "expires": 1704067200
  },
  "openai": {
    "type": "api",
    "key": "sk-xxxx"
  }
}
```

---

## 4. Integration Test Plan

### Test File Location

`go-opencode/citest/service/oauth_test.go`

### Test Cases

```go
package service_test

import (
    "net/http"
    "net/http/httptest"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"

    "github.com/opencode-ai/opencode/internal/oauth"
)

var _ = Describe("OAuth Endpoints", func() {
    BeforeEach(func() {
        // Register a test provider
        oauth.RegisterProvider("test-provider", oauth.ProviderConfig{
            ClientID:    "test-client-id",
            AuthURL:     "https://example.com/oauth/authorize",
            TokenURL:    "https://example.com/oauth/token",
            Scopes:      []string{"read", "write"},
            RedirectURL: "http://localhost:8766/callback",
        })
    })

    Describe("POST /provider/:id/oauth/authorize", func() {
        It("should return authorization URL for valid provider", func() {
            resp, err := client.Post(ctx, "/provider/test-provider/oauth/authorize",
                map[string]int{"method": 0})
            Expect(err).NotTo(HaveOccurred())
            Expect(resp.StatusCode).To(Equal(200))

            var auth map[string]any
            Expect(resp.JSON(&auth)).To(Succeed())
            Expect(auth).To(HaveKey("url"))
            Expect(auth).To(HaveKey("method"))
            Expect(auth).To(HaveKey("instructions"))

            url := auth["url"].(string)
            Expect(url).To(ContainSubstring("example.com/oauth/authorize"))
            Expect(url).To(ContainSubstring("client_id=test-client-id"))
        })

        It("should return 400 for unknown provider", func() {
            resp, err := client.Post(ctx, "/provider/unknown-provider/oauth/authorize",
                map[string]int{"method": 0})
            Expect(err).NotTo(HaveOccurred())
            Expect(resp.StatusCode).To(Equal(400))

            var errResp struct {
                Error struct {
                    Code string `json:"code"`
                } `json:"error"`
            }
            Expect(resp.JSON(&errResp)).To(Succeed())
            Expect(errResp.Error.Code).To(Equal("INVALID_REQUEST"))
        })

        It("should store pending auth state", func() {
            _, err := client.Post(ctx, "/provider/test-provider/oauth/authorize",
                map[string]int{"method": 0})
            Expect(err).NotTo(HaveOccurred())

            pending := oauth.GetPending("test-provider")
            Expect(pending).NotTo(BeNil())
            Expect(pending.ProviderID).To(Equal("test-provider"))
            Expect(pending.State).NotTo(BeEmpty())
        })
    })

    Describe("POST /provider/:id/oauth/callback", func() {
        It("should return error when no pending auth", func() {
            resp, err := client.Post(ctx, "/provider/test-provider/oauth/callback",
                map[string]any{"method": 0, "code": "test-code"})
            Expect(err).NotTo(HaveOccurred())
            Expect(resp.StatusCode).To(Equal(400))

            var errResp struct {
                Error struct {
                    Code string `json:"code"`
                } `json:"error"`
            }
            Expect(resp.JSON(&errResp)).To(Succeed())
            Expect(errResp.Error.Code).To(Equal("OAUTH_MISSING"))
        })

        It("should require code for code method", func() {
            // First authorize
            client.Post(ctx, "/provider/test-provider/oauth/authorize",
                map[string]int{"method": 0})

            // Then callback without code
            resp, err := client.Post(ctx, "/provider/test-provider/oauth/callback",
                map[string]any{"method": 0})
            Expect(err).NotTo(HaveOccurred())
            Expect(resp.StatusCode).To(Equal(400))

            var errResp struct {
                Error struct {
                    Code string `json:"code"`
                } `json:"error"`
            }
            Expect(resp.JSON(&errResp)).To(Succeed())
            Expect(errResp.Error.Code).To(Equal("OAUTH_CODE_MISSING"))
        })

        // Integration test with mock OAuth server
        Context("with mock OAuth server", func() {
            var mockServer *httptest.Server

            BeforeEach(func() {
                mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                    if r.URL.Path == "/oauth/token" {
                        w.Header().Set("Content-Type", "application/json")
                        w.Write([]byte(`{
                            "access_token": "test-access-token",
                            "refresh_token": "test-refresh-token",
                            "expires_in": 3600,
                            "token_type": "Bearer"
                        }`))
                    }
                }))

                oauth.RegisterProvider("mock-provider", oauth.ProviderConfig{
                    ClientID:    "mock-client",
                    AuthURL:     mockServer.URL + "/oauth/authorize",
                    TokenURL:    mockServer.URL + "/oauth/token",
                    RedirectURL: "http://localhost/callback",
                })
            })

            AfterEach(func() {
                mockServer.Close()
            })

            It("should exchange code for token", func() {
                // Authorize
                client.Post(ctx, "/provider/mock-provider/oauth/authorize",
                    map[string]int{"method": 0})

                // Callback with code
                resp, err := client.Post(ctx, "/provider/mock-provider/oauth/callback",
                    map[string]any{"method": 0, "code": "test-code"})
                Expect(err).NotTo(HaveOccurred())
                Expect(resp.StatusCode).To(Equal(200))

                // Verify token stored
                authInfo, ok := auth.Get("mock-provider")
                Expect(ok).To(BeTrue())
                Expect(authInfo.Type).To(Equal("oauth"))
                Expect(authInfo.Access).To(Equal("test-access-token"))
            })
        })
    })
})
```

### Comparative Test

`go-opencode/citest/comparative/oauth_test.go`

```go
func TestOAuth_Comparative(t *testing.T) {
    harness := StartComparativeHarness(t)
    defer harness.Stop()

    // Both should fail for unknown provider
    t.Run("unknown provider returns error", func(t *testing.T) {
        resp := harness.Client().Post(ctx,
            "/provider/unknown/oauth/authorize",
            map[string]int{"method": 0})

        assert.Equal(t, resp.TS.StatusCode, resp.Go.StatusCode)
        assert.Equal(t, 400, resp.TS.StatusCode)
    })

    // Both should require method field
    t.Run("method field required", func(t *testing.T) {
        resp := harness.Client().Post(ctx,
            "/provider/test/oauth/authorize",
            map[string]any{})

        assert.Equal(t, resp.TS.StatusCode, resp.Go.StatusCode)
    })

    // Response structure should match
    t.Run("authorization response structure", func(t *testing.T) {
        // Register same provider on both
        // ... setup code ...

        resp := harness.Client().Post(ctx,
            "/provider/test/oauth/authorize",
            map[string]int{"method": 0})

        if resp.TS.StatusCode == 200 && resp.Go.StatusCode == 200 {
            var tsAuth, goAuth map[string]any
            json.Unmarshal(resp.TS.Body, &tsAuth)
            json.Unmarshal(resp.Go.Body, &goAuth)

            // Both should have same keys
            assert.Contains(t, tsAuth, "url")
            assert.Contains(t, goAuth, "url")
            assert.Contains(t, tsAuth, "method")
            assert.Contains(t, goAuth, "method")
            assert.Contains(t, tsAuth, "instructions")
            assert.Contains(t, goAuth, "instructions")
        }
    })
}
```

---

## 5. Implementation Checklist

- [ ] Create `oauth/oauth.go` with core OAuth logic
- [ ] Create `oauth/providers.go` with provider configurations
- [ ] Create `auth/storage.go` for credential storage
- [ ] Update `handlers_config.go` with real implementations
- [ ] Add PKCE support for public clients
- [ ] Add state parameter validation
- [ ] Add pending auth expiration (10 min)
- [ ] Initialize auth storage on server start
- [ ] Load OAuth configs from app config
- [ ] Add environment variable support
- [ ] Write unit tests for OAuth manager
- [ ] Write unit tests for auth storage
- [ ] Write integration tests with mock server
- [ ] Write comparative tests
- [ ] Update OpenAPI spec with proper schemas

---

## 6. Security Considerations

1. **State Parameter:** Always validate state to prevent CSRF
2. **PKCE:** Use PKCE for all flows (especially public clients)
3. **Token Storage:** Store tokens with restricted file permissions (0600)
4. **Token Refresh:** Implement automatic token refresh before expiry
5. **Secrets:** Never log tokens or client secrets
6. **Expiration:** Clear pending auths after 10 minutes

---

## 7. Rollout

1. **Week 1:** Core OAuth manager and auth storage
2. **Week 2:** Handler implementation and provider configs
3. **Week 3:** Testing with mock OAuth servers
4. **Week 4:** Integration with real providers (GitHub Copilot, etc.)

---

## References

- TypeScript Auth: `packages/opencode/src/provider/auth.ts`
- TypeScript Plugin: `packages/opencode/src/plugin/index.ts`
- Go Auth Storage: `packages/opencode/src/auth/index.ts`
- Go Handler Stubs: `go-opencode/internal/server/handlers_config.go:85-93`
- OAuth2 Go Package: `golang.org/x/oauth2`
- PKCE RFC: https://datatracker.ietf.org/doc/html/rfc7636
