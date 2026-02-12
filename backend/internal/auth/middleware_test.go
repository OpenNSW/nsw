package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Note: These are integration test examples. To run them, you need:
// 1. A test database instance (PostgreSQL)
// 2. Mock database setup/teardown
// 3. Proper test data initialization

// TestGetAuthContextFromRequest tests context retrieval
func TestGetAuthContext_FromRequest(t *testing.T) {
	// Create a context with auth
	tc := &TraderContext{
		TraderID:      "TRADER-001",
		TraderContext: json.RawMessage(`{"test": "data"}`),
	}
	authCtx := &AuthContext{TraderContext: tc}
	ctx := context.WithValue(context.Background(), AuthContextKey, authCtx)

	// Retrieve context
	retrieved := GetAuthContext(ctx)
	if retrieved == nil {
		t.Error("expected to retrieve auth context")
		return
	}
	if retrieved.TraderID != "TRADER-001" {
		t.Errorf("got trader id %s, want TRADER-001", retrieved.TraderID)
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

// TestContextJSONUnmarshaling tests unmarshaling trader context JSON
func TestTraderContext_JSONUnmarshaling(t *testing.T) {
	contextJSON := json.RawMessage(`{
		"company": "Acme Inc",
		"trading_type": "exporter",
		"verified": true
	}`)

	tc := &TraderContext{
		TraderID:      "TRADER-001",
		TraderContext: contextJSON,
	}

	// Verify TraderID is set
	if tc.TraderID != "TRADER-001" {
		t.Errorf("got trader id %s, want TRADER-001", tc.TraderID)
	}

	// Unmarshal the JSON data
	var data map[string]interface{}
	err := json.Unmarshal(tc.TraderContext, &data)
	if err != nil {
		t.Errorf("failed to unmarshal trader context: %v", err)
	}

	if data["company"] != "Acme Inc" {
		t.Errorf("got company %v, want Acme Inc", data["company"])
	}
}

// Example: How to test in CI/CD with Docker
//
// func setupTestDB(t *testing.T) *gorm.DB {
// 	// Use testcontainers to spin up PostgreSQL
// 	// req := testcontainers.ContainerRequest{
// 	//     Image:        "postgres:14",
// 	//     ExposedPorts: []string{"5432/tcp"},
// 	//     Env: map[string]string{
// 	//         "POSTGRES_PASSWORD": "password",
// 	//         "POSTGRES_DB":      "test_nsw",
// 	//     },
// 	// }
//
// 	// container, err := testcontainers.GenericContainer(context.Background(), ...)
// 	// // Create connection string and connect with GORM
// 	// db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
// 	// return db
// }

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
	// won't attempt to use AuthService or TokenExtractor
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
