package auth

import (
	"fmt"
	"strings"
)

// TokenExtractor handles token extraction and parsing from HTTP headers.
// This component is intentionally separated to allow easy replacement with JWT parsing logic in the future.
// When migrating to JWT, only this file needs to be modified.
type TokenExtractor struct{}

// NewTokenExtractor creates a new token extractor
func NewTokenExtractor() *TokenExtractor {
	return &TokenExtractor{}
}

// ExtractTraderIDFromHeader extracts the trader ID from the Authorization header.
// Currently supports the simple token format: "TRADER-<ID>"
//
// Example:
//
//	Authorization: "TRADER-001" -> returns "TRADER-001"
//	Authorization: "InvalidFormat" -> returns error
//
// TODO_JWT_FUTURE: Replace this implementation with JWT parsing:
// 1. Parse the Authorization header as: "Bearer <jwt_token>"
// 2. Verify JWT signature using the configured public key
// 3. Extract trader_id from JWT claims (typically in 'sub' or 'trader_id' claim)
// 4. Return the trader_id from JWT claims
//
// Expected JWT format:
//
//	{
//	  "sub": "TRADER-001",
//	  "trader_id": "TRADER-001",
//	  "iat": 1234567890,
//	  "exp": 1234571490,
//	  ... other claims
//	}
func (te *TokenExtractor) ExtractTraderIDFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is empty")
	}

	// Current implementation: Simple token format "TRADER-<ID>"
	// In the future, this will be: "Bearer <jwt_token>"
	token := strings.TrimSpace(authHeader)

	// TODO_JWT_FUTURE: Change this to parse "Bearer <token>" format
	// if !strings.HasPrefix(token, "Bearer ") {
	//   return "", fmt.Errorf("invalid authorization header format")
	// }
	// tokenString := strings.TrimPrefix(token, "Bearer ")

	// Validate token format (currently expecting "TRADER-<ID>")
	if !strings.HasPrefix(token, "TRADER-") {
		return "", fmt.Errorf("invalid token format: expected 'TRADER-<ID>'")
	}

	// Ensure the trader ID part is not empty (more than just "TRADER-")
	if len(token) <= len("TRADER-") {
		return "", fmt.Errorf("invalid token format: trader ID cannot be empty")
	}

	// Extract trader ID (currently the entire token is the trader ID)
	traderID := token

	// TODO_JWT_FUTURE: After JWT signature verification and claim extraction:
	// traderID := claims["trader_id"].(string) // or claims["sub"]
	// Validate trader ID format (optional, could be more sophisticated)
	if len(traderID) == 0 {
		return "", fmt.Errorf("trader ID is empty")
	}

	return traderID, nil
}
