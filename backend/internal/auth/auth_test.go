package auth

import (
	"encoding/json"
	"testing"
)

// TestTokenExtractor tests the token extraction logic
func TestTokenExtractor_ExtractTraderIDFromHeader(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		want       string
		wantErr    bool
	}{
		{
			name:       "valid trader token",
			authHeader: "TRADER-001",
			want:       "TRADER-001",
			wantErr:    false,
		},
		{
			name:       "valid trader token with spaces",
			authHeader: "  TRADER-002  ",
			want:       "TRADER-002",
			wantErr:    false,
		},
		{
			name:       "empty auth header",
			authHeader: "",
			want:       "",
			wantErr:    true,
		},
		{
			name:       "invalid format - missing TRADER- prefix",
			authHeader: "INVALID-001",
			want:       "",
			wantErr:    true,
		},
		{
			name:       "invalid format - only TRADER-",
			authHeader: "TRADER-",
			want:       "",
			wantErr:    true,
		},
		{
			name:       "bearer token format (future JWT)",
			authHeader: "Bearer eyJhbGciOiJIUzI1NiJ9",
			want:       "",
			wantErr:    true, // Currently not supported
		},
	}

	extractor := NewTokenExtractor()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractor.ExtractTraderIDFromHeader(tt.authHeader)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractTraderIDFromHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractTraderIDFromHeader() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestTraderContextModel tests the TraderContext model structure
func TestTraderContextModel(t *testing.T) {
	tests := []struct {
		name      string
		traderID  string
		context   map[string]interface{}
		wantTable string
	}{
		{
			name:     "valid trader context",
			traderID: "TRADER-001",
			context: map[string]interface{}{
				"company": "Acme Inc",
				"role":    "exporter",
			},
			wantTable: "trader_contexts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contextJSON, err := json.Marshal(tt.context)
			if err != nil {
				t.Fatalf("failed to marshal test context: %v", err)
			}
			traderCtx := &TraderContext{
				TraderID:      tt.traderID,
				TraderContext: contextJSON,
			}

			got := traderCtx.TableName()
			if got != tt.wantTable {
				t.Errorf("TableName() got = %v, want %v", got, tt.wantTable)
			}
		})
	}
}

// TestAuthContextCreation tests AuthContext creation
func TestAuthContextCreation(t *testing.T) {
	contextJSON := json.RawMessage(`{"company": "Test Corp"}`)
	tc := &TraderContext{
		TraderID:      "TRADER-TEST",
		TraderContext: contextJSON,
	}

	authCtx := &AuthContext{
		TraderContext: tc,
	}

	if authCtx.TraderID != "TRADER-TEST" {
		t.Errorf("AuthContext.TraderID got = %v, want TRADER-TEST", authCtx.TraderID)
	}

	if string(authCtx.TraderContext.TraderContext) != `{"company": "Test Corp"}` {
		t.Errorf("AuthContext.TraderContext not preserved")
	}
}

// Example benchmark for token extraction
func BenchmarkTokenExtraction(b *testing.B) {
	extractor := NewTokenExtractor()
	authHeader := "TRADER-001"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor.ExtractTraderIDFromHeader(authHeader)
	}
}
