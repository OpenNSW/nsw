package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// MockUserService is a mock implementation of UserProfileService for testing.
type MockUserService struct {
	createUserErr error
	getUserRecord any
	getUserErr    error
}

func (m *MockUserService) CreateUser(userID, email, phone, ouID string) error {
	return m.createUserErr

}

func (m *MockUserService) UserExists(userID string) (bool, error) {
	return m.getUserRecord != nil, m.getUserErr
}

// TestGetAuthContextFromRequest tests context retrieval
func TestGetAuthContext_FromRequest(t *testing.T) {
	// Create a context with auth
	uc := &UserContext{
		UserID: "TRADER-001",
		Email:  "trader@example.com",
		OUID:   "OU-001",
		Roles:  []string{"exporter"},
	}
	authCtx := &AuthContext{User: uc}
	ctx := context.WithValue(context.Background(), AuthContextKey, authCtx)

	// Retrieve context
	retrieved := GetAuthContext(ctx)
	if retrieved == nil {
		t.Error("expected to retrieve auth context")
		return
	}
	if retrieved.User == nil || retrieved.User.UserID != "TRADER-001" {
		t.Errorf("got trader id %v, want TRADER-001", retrieved.User)
	}
}

// TestGetAuthContextFromRequest_NoContext tests when context not present
func TestGetAuthContext_NoContext(t *testing.T) {
	// Create a context without auth
	ctx := context.Background()

	// Retrieve context
	retrieved := GetAuthContext(ctx)
	if retrieved != nil {
		t.Error("expected nil auth context")
	}
}

// TestUserContext_JSONUnmarshaling tests UserContext structure
func TestUserContext_Structure(t *testing.T) {
	uc := &UserContext{
		UserID:      "TRADER-001",
		Email:       "trader@example.com",
		PhoneNumber: "+61400111222",
		OUID:        "OU-001",
		Roles:       []string{"exporter"},
	}

	// Verify fields are set
	if uc.UserID != "TRADER-001" {
		t.Errorf("got trader id %s, want TRADER-001", uc.UserID)
	}
	if uc.Email != "trader@example.com" {
		t.Errorf("got email %s, want trader@example.com", uc.Email)
	}
	if uc.PhoneNumber != "+61400111222" {
		t.Errorf("got phone number %s, want +61400111222", uc.PhoneNumber)
	}
	if uc.OUID != "OU-001" {
		t.Errorf("got ou id %s, want OU-001", uc.OUID)
	}
	if len(uc.Roles) != 1 || uc.Roles[0] != "exporter" {
		t.Errorf("got roles %v, want [exporter]", uc.Roles)
	}
}

// TestAuthMiddleware_NoToken tests middleware when no auth header provided
func TestAuthMiddleware_NoToken(t *testing.T) {
	// Create a test handler that checks for auth context
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r.Context())
		if authCtx != nil {
			t.Error("expected no auth context when no token provided")
		}
		w.WriteHeader(http.StatusOK)
	})

	// Create middleware with nil dependencies
	// This is acceptable for this test case since no token means the middleware
	// won't attempt to use user service or TokenExtractor
	middleware := Middleware(nil, nil)
	handlerWithMiddleware := middleware(testHandler)

	// Make a test request without Authorization header
	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	recorder := httptest.NewRecorder()

	handlerWithMiddleware.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", recorder.Code)
	}
}

// TestAuthMiddleware_UninitializedTokenExtractor tests middleware returns 500 when tokenExtractor is nil
func TestAuthMiddleware_UninitializedDependencies(t *testing.T) {
	testHandlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testHandlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// With nil tokenExtractor, middleware should return 500
	middleware := Middleware(nil, nil)
	handlerWithMiddleware := middleware(testHandler)

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	recorder := httptest.NewRecorder()

	handlerWithMiddleware.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", recorder.Code)
	}
	if testHandlerCalled {
		t.Error("expected handler not to be called when tokenExtractor is nil")
	}
}

// TestAuthMiddleware_InvalidToken tests middleware returns 401 for invalid auth token
func TestAuthMiddleware_InvalidToken(t *testing.T) {
	testHandlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testHandlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	tokenExtractor, err := NewTokenExtractor("https://localhost:8090/oauth2/jwks", "https://localhost:8090/oauth2/token", "TRADER_PORTAL_APP", []string{"TRADER_PORTAL_APP"})
	if err != nil {
		t.Fatalf("failed to create token extractor: %v", err)
	}
	// Use mock user service to ensure this test validates token behavior
	mockUserService := &MockUserService{}
	middleware := Middleware(mockUserService, tokenExtractor)
	handlerWithMiddleware := middleware(testHandler)

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	recorder := httptest.NewRecorder()

	handlerWithMiddleware.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", recorder.Code)
	}
	if testHandlerCalled {
		t.Error("expected handler not to be called for invalid token")
	}
}

func TestBuildAuthContext_UserPrincipalOnly(t *testing.T) {
	principal := &Principal{
		Type: UserPrincipalType,
		UserPrincipal: &UserPrincipal{
			UserID: "TRADER-001",
			Email:  "trader@example.com",
			OUID:   "ou-id",
			Roles:  []string{"exporter"},
		},
	}

	authCtx := buildAuthContext(principal)

	if authCtx.User == nil || authCtx.User.UserID != "TRADER-001" {
		t.Fatalf("expected user id to be set from user principal")
	}
	if authCtx.Client != nil {
		t.Fatalf("expected client id to be nil when client principal is absent")
	}
	if len(authCtx.User.Roles) != 1 || authCtx.User.Roles[0] != "exporter" {
		t.Fatalf("expected roles to be set, got %v", authCtx.User.Roles)
	}
}

func TestBuildAuthContext_ClientPrincipalOnly(t *testing.T) {
	principal := &Principal{
		Type:            ClientPrincipalType,
		ClientPrincipal: &ClientPrincipal{ClientID: "CLIENT-001"},
	}

	authCtx := buildAuthContext(principal)

	if authCtx.Client == nil || authCtx.Client.ClientID != "CLIENT-001" {
		t.Fatalf("expected client id to be set from client principal")
	}
	if authCtx.User != nil {
		t.Fatalf("expected user fields to be nil when user principal is absent")
	}
}

func TestAuthMiddleware_ValidClientCredentialsToken(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kid": "test-kid",
					"kty": "RSA",
					"alg": "RS256",
					"use": "sig",
					"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
				},
			},
		})
	}))
	defer jwksServer.Close()

	tokenExtractor, err := NewTokenExtractor(jwksServer.URL, "https://localhost:8090/oauth2/token", "TRADER_PORTAL_APP", []string{"TRADER_PORTAL_APP"})
	if err != nil {
		t.Fatalf("failed to create token extractor: %v", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub":        "FCAU_TO_NSW",
		"iss":        "https://localhost:8090/oauth2/token",
		"aud":        "TRADER_PORTAL_APP",
		"client_id":  "TRADER_PORTAL_APP",
		"grant_type": "client_credentials",
		"iat":        time.Now().Add(-1 * time.Minute).Unix(),
		"nbf":        time.Now().Add(-1 * time.Minute).Unix(),
		"exp":        time.Now().Add(10 * time.Minute).Unix(),
	})
	token.Header["kid"] = "test-kid"
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	testHandlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testHandlerCalled = true
		authCtx := GetAuthContext(r.Context())
		if authCtx == nil {
			t.Fatalf("expected auth context")
		}
		if authCtx.Client == nil || authCtx.Client.ClientID != "TRADER_PORTAL_APP" {
			t.Fatalf("expected client id TRADER_PORTAL_APP, got %v", authCtx.Client)
		}
		if authCtx.User != nil {
			t.Fatalf("expected user id to be nil for client principal")
		}
		w.WriteHeader(http.StatusOK)
	})

	mockUserService := &MockUserService{}
	handlerWithMiddleware := Middleware(mockUserService, tokenExtractor)(testHandler)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	req.Header.Set("Authorization", "Bearer "+signedToken)
	recorder := httptest.NewRecorder()

	handlerWithMiddleware.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !testHandlerCalled {
		t.Fatalf("expected handler to be called for valid token")
	}
}

func TestRequireAuth_UnauthenticatedRequest(t *testing.T) {
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	tokenExtractor, err := NewTokenExtractor("https://localhost:8090/oauth2/jwks", "https://localhost:8090/oauth2/token", "TRADER_PORTAL_APP", []string{"TRADER_PORTAL_APP"})
	if err != nil {
		t.Fatalf("failed to create token extractor: %v", err)
	}
	mockUserService := &MockUserService{}
	protected := RequireAuth(mockUserService, tokenExtractor)(testHandler)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/protected", nil)
	recorder := httptest.NewRecorder()

	protected.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", recorder.Code)
	}
	if handlerCalled {
		t.Fatalf("expected protected handler not to be called")
	}
}

func TestRequireAuth_ValidClientCredentialsToken(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kid": "requireauth-kid",
					"kty": "RSA",
					"alg": "RS256",
					"use": "sig",
					"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
				},
			},
		})
	}))
	defer jwksServer.Close()

	tokenExtractor, err := NewTokenExtractor(jwksServer.URL, "https://localhost:8090/oauth2/token", "TRADER_PORTAL_APP", []string{"TRADER_PORTAL_APP"})
	if err != nil {
		t.Fatalf("failed to create token extractor: %v", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub":        "NPQS_TO_NSW",
		"iss":        "https://localhost:8090/oauth2/token",
		"aud":        "TRADER_PORTAL_APP",
		"client_id":  "TRADER_PORTAL_APP",
		"grant_type": "client_credentials",
		"iat":        time.Now().Add(-1 * time.Minute).Unix(),
		"nbf":        time.Now().Add(-1 * time.Minute).Unix(),
		"exp":        time.Now().Add(10 * time.Minute).Unix(),
	})
	token.Header["kid"] = "requireauth-kid"
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	mockUserService := &MockUserService{}
	protected := RequireAuth(mockUserService, tokenExtractor)(testHandler)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signedToken)
	recorder := httptest.NewRecorder()

	protected.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !handlerCalled {
		t.Fatalf("expected protected handler to be called")
	}
}
