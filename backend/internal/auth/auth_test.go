package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestTokenExtractor_ExtractPrincipalFromHeader(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{{
				"kid": "test-kid",
				"kty": "RSA",
				"alg": "RS256",
				"use": "sig",
				"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
			}},
		})
	}))
	defer jwksServer.Close()

	extractor, err := NewTokenExtractor(jwksServer.URL, "https://localhost:8090/oauth2/token", "TRADER_PORTAL_APP", []string{"TRADER_PORTAL_APP"})
	if err != nil {
		t.Fatalf("failed to create token extractor: %v", err)
	}

	mintToken := func(claims map[string]any) string {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims(claims))
		token.Header["kid"] = "test-kid"
		signedToken, signErr := token.SignedString(privateKey)
		if signErr != nil {
			t.Fatalf("failed to sign token: %v", signErr)
		}
		return signedToken
	}

	validUserToken := mintToken(map[string]any{
		"sub":          "TRADER-001",
		"iss":          "https://localhost:8090/oauth2/token",
		"aud":          "TRADER_PORTAL_APP",
		"client_id":    "TRADER_PORTAL_APP",
		"grant_type":   "authorization_code",
		"email":        "trader@example.com",
		"phone_number": "+61400111222",
		"ouId":         "OU-001",
		"roles":        []string{"exporter", "trader"},
		"iat":          time.Now().Add(-1 * time.Minute).Unix(),
		"nbf":          time.Now().Add(-1 * time.Minute).Unix(),
		"exp":          time.Now().Add(10 * time.Minute).Unix(),
	})

	validClientToken := mintToken(map[string]any{
		"sub":        "FCAU_TO_NSW",
		"iss":        "https://localhost:8090/oauth2/token",
		"aud":        "TRADER_PORTAL_APP",
		"client_id":  "TRADER_PORTAL_APP",
		"grant_type": "client_credentials",
		"iat":        time.Now().Add(-1 * time.Minute).Unix(),
		"nbf":        time.Now().Add(-1 * time.Minute).Unix(),
		"exp":        time.Now().Add(10 * time.Minute).Unix(),
	})

	tests := []struct {
		name       string
		authHeader string
		want       string
		wantErr    bool
	}{
		{name: "valid user token", authHeader: "Bearer " + validUserToken, want: "TRADER-001"},
		{name: "valid client token", authHeader: "Bearer " + validClientToken, want: "TRADER_PORTAL_APP"},
		{name: "missing header", authHeader: "", wantErr: true},
		{name: "invalid token", authHeader: "Bearer invalid.jwt.token", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			principal, err := extractor.ExtractPrincipalFromHeader(tt.authHeader)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ExtractPrincipalFromHeader() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			got := ""
			if principal != nil && principal.UserPrincipal != nil {
				got = principal.UserPrincipal.UserID
			}
			if principal != nil && principal.ClientPrincipal != nil {
				got = principal.ClientPrincipal.ClientID
			}
			if got != tt.want {
				t.Fatalf("unexpected principal id: got %q want %q", got, tt.want)
			}
		})
	}
}

func TestUserPrincipalFromClaims(t *testing.T) {
	phone := "+61400111222"
	ouID := "OU-001"
	claims := &tokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "TRADER-001"},
		Email:            strPtr("trader@example.com"),
		PhoneNumber:      &phone,
		OUID:             &ouID,
		Roles:            []string{"exporter"},
	}

	principal, err := (&TokenExtractor{}).userPrincipalFromClaims(claims)
	if err != nil {
		t.Fatalf("userPrincipalFromClaims() error = %v", err)
	}
	if principal.UserID != "TRADER-001" || principal.Email != "trader@example.com" {
		t.Fatalf("unexpected principal: %#v", principal)
	}
	if principal.PhoneNumber == nil || *principal.PhoneNumber != phone {
		t.Fatalf("unexpected phone number: %#v", principal.PhoneNumber)
	}
	if principal.OUID != ouID || len(principal.Roles) != 1 || principal.Roles[0] != "exporter" {
		t.Fatalf("unexpected claims mapping: %#v", principal)
	}

	missingClaims := []struct {
		name   string
		claims *tokenClaims
	}{
		{name: "missing email", claims: &tokenClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "TRADER-001"}, OUID: &ouID}},
		{name: "missing ou id", claims: &tokenClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "TRADER-001"}, Email: strPtr("trader@example.com")}},
	}

	for _, tt := range missingClaims {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := (&TokenExtractor{}).userPrincipalFromClaims(tt.claims); err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
		})
	}
}

func TestUserRecordModel(t *testing.T) {
	if got := (&UserRecord{}).TableName(); got != "user_records" {
		t.Fatalf("TableName() got = %v, want %v", got, "user_records")
	}
}

func TestNewTokenExtractor_InvalidConfig(t *testing.T) {
	tests := []struct {
		name              string
		jwksURL           string
		issuer            string
		audience          string
		expectedClientIDs []string
	}{
		{name: "missing jwks url", issuer: "iss", audience: "aud", expectedClientIDs: []string{"client"}},
		{name: "missing issuer", jwksURL: "https://localhost/jwks", audience: "aud", expectedClientIDs: []string{"client"}},
		{name: "missing audience", jwksURL: "https://localhost/jwks", issuer: "iss", expectedClientIDs: []string{"client"}},
		{name: "missing client ids", jwksURL: "https://localhost/jwks", issuer: "iss", audience: "aud", expectedClientIDs: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if extractor, err := NewTokenExtractor(tt.jwksURL, tt.issuer, tt.audience, tt.expectedClientIDs); err == nil {
				t.Fatalf("expected constructor error, got extractor: %#v", extractor)
			}
		})
	}
}

func TestTokenExtractor_JWKSIsCached(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}

	var fetchCount int32
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&fetchCount, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{{
				"kid": "cache-kid",
				"kty": "RSA",
				"alg": "RS256",
				"use": "sig",
				"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
			}},
		})
	}))
	defer jwksServer.Close()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub":        "TRADER-001",
		"iss":        "https://localhost:8090/oauth2/token",
		"aud":        "TRADER_PORTAL_APP",
		"client_id":  "TRADER_PORTAL_APP",
		"grant_type": "authorization_code",
		"email":      "trader@example.com",
		"ouId":       "OU-001",
		"roles":      []string{"exporter"},
		"iat":        time.Now().Add(-1 * time.Minute).Unix(),
		"nbf":        time.Now().Add(-1 * time.Minute).Unix(),
		"exp":        time.Now().Add(10 * time.Minute).Unix(),
	})
	token.Header["kid"] = "cache-kid"
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	extractor, err := NewTokenExtractor(jwksServer.URL, "https://localhost:8090/oauth2/token", "TRADER_PORTAL_APP", []string{"TRADER_PORTAL_APP"})
	if err != nil {
		t.Fatalf("failed to create token extractor: %v", err)
	}

	if _, err := extractor.ExtractPrincipalFromHeader("Bearer " + signedToken); err != nil {
		t.Fatalf("first extract failed: %v", err)
	}
	if _, err := extractor.ExtractPrincipalFromHeader("Bearer " + signedToken); err != nil {
		t.Fatalf("second extract failed: %v", err)
	}

	if got := atomic.LoadInt32(&fetchCount); got != 1 {
		t.Fatalf("expected JWKS to be fetched once, got %d", got)
	}
}

func TestTokenExtractor_RefreshesJWKSOnUnknownKid(t *testing.T) {
	privateKeyA, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate first rsa key: %v", err)
	}
	privateKeyB, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate second rsa key: %v", err)
	}

	var fetchCount int32
	var serveNewKey int32
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&fetchCount, 1)
		w.Header().Set("Content-Type", "application/json")

		key := privateKeyA.PublicKey
		kid := "old-kid"
		if atomic.LoadInt32(&serveNewKey) == 1 {
			key = privateKeyB.PublicKey
			kid = "new-kid"
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{{
				"kid": kid,
				"kty": "RSA",
				"alg": "RS256",
				"use": "sig",
				"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
			}},
		})
	}))
	defer jwksServer.Close()

	oldToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub":        "TRADER-001",
		"iss":        "https://localhost:8090/oauth2/token",
		"aud":        "TRADER_PORTAL_APP",
		"client_id":  "TRADER_PORTAL_APP",
		"grant_type": "authorization_code",
		"email":      "trader@example.com",
		"ouId":       "OU-001",
		"roles":      []string{"exporter"},
		"iat":        time.Now().Add(-1 * time.Minute).Unix(),
		"nbf":        time.Now().Add(-1 * time.Minute).Unix(),
		"exp":        time.Now().Add(10 * time.Minute).Unix(),
	})
	oldToken.Header["kid"] = "old-kid"
	oldSignedToken, err := oldToken.SignedString(privateKeyA)
	if err != nil {
		t.Fatalf("failed to sign old token: %v", err)
	}

	newToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub":        "TRADER-001",
		"iss":        "https://localhost:8090/oauth2/token",
		"aud":        "TRADER_PORTAL_APP",
		"client_id":  "TRADER_PORTAL_APP",
		"grant_type": "authorization_code",
		"email":      "trader@example.com",
		"ouId":       "OU-001",
		"roles":      []string{"exporter"},
		"iat":        time.Now().Add(-1 * time.Minute).Unix(),
		"nbf":        time.Now().Add(-1 * time.Minute).Unix(),
		"exp":        time.Now().Add(10 * time.Minute).Unix(),
	})
	newToken.Header["kid"] = "new-kid"
	newSignedToken, err := newToken.SignedString(privateKeyB)
	if err != nil {
		t.Fatalf("failed to sign new token: %v", err)
	}

	extractor, err := NewTokenExtractor(jwksServer.URL, "https://localhost:8090/oauth2/token", "TRADER_PORTAL_APP", []string{"TRADER_PORTAL_APP"})
	if err != nil {
		t.Fatalf("failed to create token extractor: %v", err)
	}
	if _, err := extractor.ExtractPrincipalFromHeader("Bearer " + oldSignedToken); err != nil {
		t.Fatalf("old token extract failed: %v", err)
	}

	atomic.StoreInt32(&serveNewKey, 1)
	if _, err := extractor.ExtractPrincipalFromHeader("Bearer " + newSignedToken); err != nil {
		t.Fatalf("new token extract failed after refresh: %v", err)
	}

	if got := atomic.LoadInt32(&fetchCount); got != 2 {
		t.Fatalf("expected JWKS fetches to be 2 (initial + refresh on unknown kid), got %d", got)
	}
}

func strPtr(value string) *string { return &value }
