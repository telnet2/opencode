package sharing

import (
	"sync"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	manager := NewManager("")

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}
	if manager.baseURL != "https://opencode.ai/share" {
		t.Errorf("expected default base URL, got %s", manager.baseURL)
	}
}

func TestNewManagerWithCustomURL(t *testing.T) {
	customURL := "https://custom.example.com/share"
	manager := NewManager(customURL)

	if manager.baseURL != customURL {
		t.Errorf("expected %s, got %s", customURL, manager.baseURL)
	}
}

func TestShare(t *testing.T) {
	manager := NewManager("")

	info, err := manager.Share("session-1", nil)
	if err != nil {
		t.Fatalf("Share failed: %v", err)
	}

	if info.Token == "" {
		t.Error("expected non-empty token")
	}
	if info.SessionID != "session-1" {
		t.Errorf("expected session-1, got %s", info.SessionID)
	}
	if info.URL == "" {
		t.Error("expected non-empty URL")
	}
	if info.CreatedAt.IsZero() {
		t.Error("expected non-zero created time")
	}
	if !info.Public {
		t.Error("expected public to be true by default")
	}
	if info.Views != 0 {
		t.Errorf("expected 0 views, got %d", info.Views)
	}
}

func TestShareWithOptions(t *testing.T) {
	manager := NewManager("")

	opts := &ShareOptions{
		ExpiresIn: 24 * time.Hour,
		MaxViews:  100,
		Public:    false,
	}

	info, err := manager.Share("session-1", opts)
	if err != nil {
		t.Fatalf("Share failed: %v", err)
	}

	if info.ExpiresAt.IsZero() {
		t.Error("expected non-zero expires time")
	}
	if time.Until(info.ExpiresAt) > 24*time.Hour || time.Until(info.ExpiresAt) < 23*time.Hour {
		t.Errorf("unexpected expiration time: %v", info.ExpiresAt)
	}
	if info.MaxViews != 100 {
		t.Errorf("expected max views 100, got %d", info.MaxViews)
	}
	if info.Public {
		t.Error("expected public to be false")
	}
}

func TestShareUpdate(t *testing.T) {
	manager := NewManager("")

	// Create initial share
	info1, err := manager.Share("session-1", nil)
	if err != nil {
		t.Fatalf("Share failed: %v", err)
	}
	originalToken := info1.Token

	// Update with new options
	opts := &ShareOptions{
		ExpiresIn: 48 * time.Hour,
		MaxViews:  50,
		Public:    false,
	}

	info2, err := manager.Share("session-1", opts)
	if err != nil {
		t.Fatalf("Share update failed: %v", err)
	}

	// Should have same token
	if info2.Token != originalToken {
		t.Error("expected same token on update")
	}
	if info2.MaxViews != 50 {
		t.Errorf("expected max views 50, got %d", info2.MaxViews)
	}
	if info2.Public {
		t.Error("expected public to be false after update")
	}
}

func TestUnshare(t *testing.T) {
	manager := NewManager("")

	// Create share
	_, err := manager.Share("session-1", nil)
	if err != nil {
		t.Fatalf("Share failed: %v", err)
	}

	// Verify it's shared
	if !manager.IsShared("session-1") {
		t.Error("expected session to be shared")
	}

	// Unshare
	err = manager.Unshare("session-1")
	if err != nil {
		t.Fatalf("Unshare failed: %v", err)
	}

	// Verify it's not shared
	if manager.IsShared("session-1") {
		t.Error("expected session to not be shared after unshare")
	}
}

func TestUnshareNotShared(t *testing.T) {
	manager := NewManager("")

	err := manager.Unshare("nonexistent")
	if err == nil {
		t.Error("expected error for unsharing non-shared session")
	}
}

func TestGetByToken(t *testing.T) {
	manager := NewManager("")

	info, err := manager.Share("session-1", nil)
	if err != nil {
		t.Fatalf("Share failed: %v", err)
	}

	// Get by token
	retrieved, err := manager.GetByToken(info.Token)
	if err != nil {
		t.Fatalf("GetByToken failed: %v", err)
	}

	if retrieved.SessionID != "session-1" {
		t.Errorf("expected session-1, got %s", retrieved.SessionID)
	}
}

func TestGetByTokenNotFound(t *testing.T) {
	manager := NewManager("")

	_, err := manager.GetByToken("nonexistent-token")
	if err == nil {
		t.Error("expected error for nonexistent token")
	}
}

func TestGetByTokenExpired(t *testing.T) {
	manager := NewManager("")

	// Create a share first
	opts := &ShareOptions{
		ExpiresIn: 1 * time.Hour,
	}

	info, err := manager.Share("session-1", opts)
	if err != nil {
		t.Fatalf("Share failed: %v", err)
	}

	// Manually set expiration to past (simulating expired share)
	manager.mu.Lock()
	manager.shares[info.Token].ExpiresAt = time.Now().Add(-1 * time.Hour)
	manager.mu.Unlock()

	_, err = manager.GetByToken(info.Token)
	if err == nil {
		t.Error("expected error for expired share")
	}
}

func TestGetByTokenViewLimitExceeded(t *testing.T) {
	manager := NewManager("")

	opts := &ShareOptions{
		MaxViews: 1,
	}

	info, err := manager.Share("session-1", opts)
	if err != nil {
		t.Fatalf("Share failed: %v", err)
	}

	// Record one view
	err = manager.RecordView(info.Token)
	if err != nil {
		t.Fatalf("RecordView failed: %v", err)
	}

	// Should fail - view limit exceeded
	_, err = manager.GetByToken(info.Token)
	if err == nil {
		t.Error("expected error for exceeded view limit")
	}
}

func TestGetBySession(t *testing.T) {
	manager := NewManager("")

	_, err := manager.Share("session-1", nil)
	if err != nil {
		t.Fatalf("Share failed: %v", err)
	}

	retrieved, err := manager.GetBySession("session-1")
	if err != nil {
		t.Fatalf("GetBySession failed: %v", err)
	}

	if retrieved.SessionID != "session-1" {
		t.Errorf("expected session-1, got %s", retrieved.SessionID)
	}
}

func TestGetBySessionNotShared(t *testing.T) {
	manager := NewManager("")

	_, err := manager.GetBySession("nonexistent")
	if err == nil {
		t.Error("expected error for non-shared session")
	}
}

func TestRecordView(t *testing.T) {
	manager := NewManager("")

	info, err := manager.Share("session-1", nil)
	if err != nil {
		t.Fatalf("Share failed: %v", err)
	}

	if info.Views != 0 {
		t.Errorf("expected 0 views initially, got %d", info.Views)
	}

	// Record views
	for i := 0; i < 5; i++ {
		err = manager.RecordView(info.Token)
		if err != nil {
			t.Fatalf("RecordView failed: %v", err)
		}
	}

	// Check view count
	retrieved, err := manager.GetByToken(info.Token)
	if err != nil {
		t.Fatalf("GetByToken failed: %v", err)
	}

	if retrieved.Views != 5 {
		t.Errorf("expected 5 views, got %d", retrieved.Views)
	}
}

func TestRecordViewNotFound(t *testing.T) {
	manager := NewManager("")

	err := manager.RecordView("nonexistent-token")
	if err == nil {
		t.Error("expected error for nonexistent token")
	}
}

func TestIsShared(t *testing.T) {
	manager := NewManager("")

	// Initially not shared
	if manager.IsShared("session-1") {
		t.Error("expected session to not be shared initially")
	}

	// Share it
	_, err := manager.Share("session-1", nil)
	if err != nil {
		t.Fatalf("Share failed: %v", err)
	}

	// Now should be shared
	if !manager.IsShared("session-1") {
		t.Error("expected session to be shared after Share")
	}

	// Other sessions should not be shared
	if manager.IsShared("session-2") {
		t.Error("expected session-2 to not be shared")
	}
}

func TestListShares(t *testing.T) {
	manager := NewManager("")

	// Initially empty
	shares := manager.ListShares()
	if len(shares) != 0 {
		t.Errorf("expected 0 shares initially, got %d", len(shares))
	}

	// Create some shares
	for i := 1; i <= 3; i++ {
		_, err := manager.Share("session-"+string(rune('0'+i)), nil)
		if err != nil {
			t.Fatalf("Share failed: %v", err)
		}
	}

	shares = manager.ListShares()
	if len(shares) != 3 {
		t.Errorf("expected 3 shares, got %d", len(shares))
	}
}

func TestCleanExpired(t *testing.T) {
	manager := NewManager("")

	// Create a share that will be manually expired
	expiredInfo, err := manager.Share("expired", &ShareOptions{
		ExpiresIn: 1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("Share failed: %v", err)
	}

	// Manually set expiration to past
	manager.mu.Lock()
	manager.shares[expiredInfo.Token].ExpiresAt = time.Now().Add(-1 * time.Hour)
	manager.mu.Unlock()

	// Create a valid share
	_, err = manager.Share("valid", &ShareOptions{
		ExpiresIn: 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("Share failed: %v", err)
	}

	// Create a view-limited share that exceeded limit
	viewLimitInfo, err := manager.Share("viewlimit", &ShareOptions{
		MaxViews: 1,
	})
	if err != nil {
		t.Fatalf("Share failed: %v", err)
	}
	manager.RecordView(viewLimitInfo.Token) // Exceed limit

	// Should have 3 shares before cleanup
	if len(manager.ListShares()) != 3 {
		t.Errorf("expected 3 shares before cleanup, got %d", len(manager.ListShares()))
	}

	// Clean expired
	cleaned := manager.CleanExpired()
	if cleaned != 2 {
		t.Errorf("expected 2 shares cleaned, got %d", cleaned)
	}

	// Should have 1 share after cleanup
	if len(manager.ListShares()) != 1 {
		t.Errorf("expected 1 share after cleanup, got %d", len(manager.ListShares()))
	}

	// Valid share should still exist
	if !manager.IsShared("valid") {
		t.Error("expected valid share to still exist")
	}
}

func TestGenerateShortCode(t *testing.T) {
	code1, err := GenerateShortCode()
	if err != nil {
		t.Fatalf("GenerateShortCode failed: %v", err)
	}

	if len(code1) != 8 {
		t.Errorf("expected 8 character code, got %d", len(code1))
	}

	// Generate another - should be different
	code2, err := GenerateShortCode()
	if err != nil {
		t.Fatalf("GenerateShortCode failed: %v", err)
	}

	if code1 == code2 {
		t.Error("expected different codes")
	}
}

func TestTokenUniqueness(t *testing.T) {
	manager := NewManager("")

	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		info, err := manager.Share("session-"+string(rune(i)), nil)
		if err != nil {
			t.Fatalf("Share failed: %v", err)
		}

		if tokens[info.Token] {
			t.Errorf("duplicate token: %s", info.Token)
		}
		tokens[info.Token] = true
	}
}

func TestURLFormat(t *testing.T) {
	customURL := "https://example.com/s"
	manager := NewManager(customURL)

	info, err := manager.Share("session-1", nil)
	if err != nil {
		t.Fatalf("Share failed: %v", err)
	}

	expectedPrefix := customURL + "/"
	if len(info.URL) <= len(expectedPrefix) {
		t.Errorf("URL too short: %s", info.URL)
	}
	if info.URL[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("expected URL to start with %s, got %s", expectedPrefix, info.URL)
	}
}

func TestConcurrentAccess(t *testing.T) {
	manager := NewManager("")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			sessionID := "session-" + string(rune('a'+i%26))

			// Share
			info, _ := manager.Share(sessionID, nil)

			// Read operations
			manager.IsShared(sessionID)
			manager.ListShares()
			if info != nil {
				manager.GetByToken(info.Token)
				manager.RecordView(info.Token)
			}
			manager.GetBySession(sessionID)
		}(i)
	}

	wg.Wait()
}

func TestShareNoExpirationNoMaxViews(t *testing.T) {
	manager := NewManager("")

	info, err := manager.Share("session-1", &ShareOptions{
		Public: true,
	})
	if err != nil {
		t.Fatalf("Share failed: %v", err)
	}

	if !info.ExpiresAt.IsZero() {
		t.Error("expected zero expiration time")
	}
	if info.MaxViews != 0 {
		t.Errorf("expected 0 max views, got %d", info.MaxViews)
	}

	// Should still be accessible after many views
	for i := 0; i < 1000; i++ {
		manager.RecordView(info.Token)
	}

	_, err = manager.GetByToken(info.Token)
	if err != nil {
		t.Errorf("expected no error with unlimited views: %v", err)
	}
}

func TestCleanExpiredWithNeverExpiring(t *testing.T) {
	manager := NewManager("")

	// Create a share without expiration
	_, err := manager.Share("forever", nil)
	if err != nil {
		t.Fatalf("Share failed: %v", err)
	}

	// Clean should not remove it
	cleaned := manager.CleanExpired()
	if cleaned != 0 {
		t.Errorf("expected 0 shares cleaned, got %d", cleaned)
	}

	if !manager.IsShared("forever") {
		t.Error("expected forever share to still exist")
	}
}

func TestMultipleSessions(t *testing.T) {
	manager := NewManager("")

	sessions := []string{"session-a", "session-b", "session-c"}

	// Share all sessions
	for _, s := range sessions {
		_, err := manager.Share(s, nil)
		if err != nil {
			t.Fatalf("Share failed for %s: %v", s, err)
		}
	}

	// Verify all are shared
	for _, s := range sessions {
		if !manager.IsShared(s) {
			t.Errorf("expected %s to be shared", s)
		}
	}

	// Unshare middle one
	err := manager.Unshare("session-b")
	if err != nil {
		t.Fatalf("Unshare failed: %v", err)
	}

	// Verify correct state
	if !manager.IsShared("session-a") {
		t.Error("expected session-a to still be shared")
	}
	if manager.IsShared("session-b") {
		t.Error("expected session-b to not be shared")
	}
	if !manager.IsShared("session-c") {
		t.Error("expected session-c to still be shared")
	}
}
