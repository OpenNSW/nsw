package auth

import "testing"

func TestNewManager_AllowsNilUserProfileService(t *testing.T) {
	cfg := Config{
		JWKSURL:  "https://localhost/jwks",
		Issuer:   "https://localhost/token",
		Audience: "TRADER_PORTAL_APP",
		ClientIDs: []string{
			"TRADER_PORTAL_APP",
		},
	}

	manager, err := NewManager(nil, cfg)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if manager == nil {
		t.Fatalf("expected manager, got nil")
	}
	if manager.userProfileService != nil {
		t.Fatalf("expected nil userProfileService, got %T", manager.userProfileService)
	}
	if manager.tokenExtractor == nil {
		t.Fatal("expected tokenExtractor to be initialized")
	}
}
