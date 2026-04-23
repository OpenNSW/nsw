package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupAuthTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	return gormDB, mock
}

func TestConfigValidate(t *testing.T) {
	valid := Config{
		JWKSURL:  "https://localhost/jwks",
		Issuer:   "https://localhost/token",
		Audience: "TRADER_PORTAL_APP",
		ClientIDs: []string{
			"TRADER_PORTAL_APP",
		},
	}

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{name: "valid config", config: valid},
		{name: "missing jwks url", config: Config{Issuer: valid.Issuer, Audience: valid.Audience, ClientIDs: valid.ClientIDs}, wantErr: true},
		{name: "missing issuer", config: Config{JWKSURL: valid.JWKSURL, Audience: valid.Audience, ClientIDs: valid.ClientIDs}, wantErr: true},
		{name: "missing audience", config: Config{JWKSURL: valid.JWKSURL, Issuer: valid.Issuer, ClientIDs: valid.ClientIDs}, wantErr: true},
		{name: "missing client ids", config: Config{JWKSURL: valid.JWKSURL, Issuer: valid.Issuer, Audience: valid.Audience}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestGetUserContext(t *testing.T) {
	t.Run("empty user id", func(t *testing.T) {
		service := NewAuthService(nil)
		if _, err := service.GetUserContext(""); err == nil {
			t.Fatalf("expected error for empty user id")
		}
	})

	t.Run("found", func(t *testing.T) {
		db, mock := setupAuthTestDB(t)
		service := NewAuthService(db)

		rows := sqlmock.NewRows([]string{"user_id", "email", "phone_number", "ou_id", "nsw_data"}).
			AddRow("TRADER-001", "trader@example.com", "+61400111222", "OU-001", []byte(`{"company":"Acme"}`))

		mock.ExpectQuery(`SELECT .* FROM "user_records" WHERE user_id = \$1 ORDER BY "user_records"\."user_id" LIMIT \$2`).
			WithArgs("TRADER-001", 1).
			WillReturnRows(rows)

		userCtx, err := service.GetUserContext("TRADER-001")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if userCtx == nil || userCtx.UserID != "TRADER-001" || userCtx.PhoneNumber != "+61400111222" {
			t.Fatalf("unexpected user context: %#v", userCtx)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations not met: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := setupAuthTestDB(t)
		service := NewAuthService(db)

		mock.ExpectQuery(`SELECT .* FROM "user_records" WHERE user_id = \$1 ORDER BY "user_records"\."user_id" LIMIT \$2`).
			WithArgs("UNKNOWN", 1).
			WillReturnError(gorm.ErrRecordNotFound)

		if _, err := service.GetUserContext("UNKNOWN"); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("expected record not found error, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations not met: %v", err)
		}
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := setupAuthTestDB(t)
		service := NewAuthService(db)

		mock.ExpectQuery(`SELECT .* FROM "user_records" WHERE user_id = \$1 ORDER BY "user_records"\."user_id" LIMIT \$2`).
			WithArgs("TRADER-001", 1).
			WillReturnError(errors.New("db down"))

		if _, err := service.GetUserContext("TRADER-001"); err == nil {
			t.Fatalf("expected error")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations not met: %v", err)
		}
	})
}

func TestUpdateUserContext(t *testing.T) {
	t.Run("validation errors", func(t *testing.T) {
		service := NewAuthService(nil)
		if err := service.UpdateUserContext("", json.RawMessage(`{}`)); err == nil {
			t.Fatalf("expected user id validation error")
		}
		if err := service.UpdateUserContext("TRADER-001", nil); err == nil {
			t.Fatalf("expected empty context validation error")
		}
		if err := service.UpdateUserContext("TRADER-001", json.RawMessage(`not-json`)); err == nil {
			t.Fatalf("expected invalid json error")
		}
	})

	t.Run("success", func(t *testing.T) {
		db, mock := setupAuthTestDB(t)
		service := NewAuthService(db)
		ctx := json.RawMessage(`{"company":"Acme"}`)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_records" SET "nsw_data"=\$1 WHERE user_id = \$2`).
			WithArgs(ctx, "TRADER-001").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		if err := service.UpdateUserContext("TRADER-001", ctx); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations not met: %v", err)
		}
	})

	t.Run("missing row", func(t *testing.T) {
		db, mock := setupAuthTestDB(t)
		service := NewAuthService(db)
		ctx := json.RawMessage(`{"company":"Acme"}`)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_records" SET "nsw_data"=\$1 WHERE user_id = \$2`).
			WithArgs(ctx, "TRADER-001").
			WillReturnResult(sqlmock.NewResult(1, 0))
		mock.ExpectCommit()

		if err := service.UpdateUserContext("TRADER-001", ctx); err == nil {
			t.Fatalf("expected not found error")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations not met: %v", err)
		}
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := setupAuthTestDB(t)
		service := NewAuthService(db)
		ctx := json.RawMessage(`{"company":"Acme"}`)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_records" SET "nsw_data"=\$1 WHERE user_id = \$2`).
			WithArgs(ctx, "TRADER-001").
			WillReturnError(errors.New("db down"))
		mock.ExpectRollback()

		if err := service.UpdateUserContext("TRADER-001", ctx); err == nil {
			t.Fatalf("expected error")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations not met: %v", err)
		}
	})
}

func TestUpsertUserContext(t *testing.T) {
	t.Run("create new record", func(t *testing.T) {
		db, mock := setupAuthTestDB(t)
		service := NewAuthService(db)
		email := "trader@example.com"
		phone := "+61400111222"
		ouID := "OU-001"

		mock.ExpectQuery(`SELECT .* FROM "user_records" WHERE user_id = \$1 ORDER BY "user_records"\."user_id" LIMIT \$2`).
			WithArgs("TRADER-001", 1).
			WillReturnError(gorm.ErrRecordNotFound)
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO "user_records" .* ON CONFLICT \("user_id"\) DO UPDATE SET .*`).
			WithArgs("TRADER-001", email, phone, ouID, []byte(`{"company":"Acme"}`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		if err := service.UpsertUserContext("TRADER-001", UpsertUserContextPayload{
			Email:       &email,
			PhoneNumber: &phone,
			OUID:        &ouID,
			NSWData:     json.RawMessage(`{"company":"Acme"}`),
		}); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations not met: %v", err)
		}
	})

	t.Run("update existing record", func(t *testing.T) {
		db, mock := setupAuthTestDB(t)
		service := NewAuthService(db)
		phone := "+61400111222"
		ouID := "OU-001"
		rows := sqlmock.NewRows([]string{"user_id", "email", "phone_number", "ou_id", "nsw_data"}).
			AddRow("TRADER-001", "old@example.com", "", "OLD-OU", []byte(`{"old":true}`))

		mock.ExpectQuery(`SELECT .* FROM "user_records" WHERE user_id = \$1 ORDER BY "user_records"\."user_id" LIMIT \$2`).
			WithArgs("TRADER-001", 1).
			WillReturnRows(rows)
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO "user_records" .* ON CONFLICT \("user_id"\) DO UPDATE SET .*`).
			WithArgs("TRADER-001", "old@example.com", phone, ouID, []byte(`{"new":true}`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		if err := service.UpsertUserContext("TRADER-001", UpsertUserContextPayload{
			PhoneNumber: &phone,
			OUID:        &ouID,
			NSWData:     json.RawMessage(`{"new":true}`),
		}); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations not met: %v", err)
		}
	})

	t.Run("wraps db errors", func(t *testing.T) {
		db, mock := setupAuthTestDB(t)
		service := NewAuthService(db)

		mock.ExpectQuery(`SELECT .* FROM "user_records" WHERE user_id = \$1 ORDER BY "user_records"\."user_id" LIMIT \$2`).
			WithArgs("TRADER-001", 1).
			WillReturnError(errors.New("db down"))

		if err := service.UpsertUserContext("TRADER-001", UpsertUserContextPayload{}); err == nil {
			t.Fatalf("expected error")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations not met: %v", err)
		}
	})
}

func TestGetUserContextMap(t *testing.T) {
	t.Run("nil auth context", func(t *testing.T) {
		var authCtx *AuthContext
		data, err := authCtx.GetUserContextMap()
		if err != nil || len(data) != 0 {
			t.Fatalf("expected empty map and no error, got %v %v", data, err)
		}
	})

	t.Run("bad json", func(t *testing.T) {
		authCtx := &AuthContext{User: &UserContext{NSWData: json.RawMessage(`not-json`)}}
		if _, err := authCtx.GetUserContextMap(); err == nil {
			t.Fatalf("expected json error")
		}
	})

	t.Run("success", func(t *testing.T) {
		authCtx := &AuthContext{User: &UserContext{NSWData: json.RawMessage(`{"company":"Acme"}`)}}
		data, err := authCtx.GetUserContextMap()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if data["company"] != "Acme" {
			t.Fatalf("unexpected map: %#v", data)
		}
	})
}

func TestUserContextTableName(t *testing.T) {
	if got := (&UserContext{}).TableName(); got != "user_contexts" {
		t.Fatalf("TableName() got = %v, want %v", got, "user_contexts")
	}
}

func TestBuildAuthContextAdditionalBranches(t *testing.T) {
	t.Run("nil principal", func(t *testing.T) {
		if ctx := buildAuthContext(nil); ctx == nil || ctx.User != nil || ctx.Client != nil {
			t.Fatalf("unexpected context: %#v", ctx)
		}
	})

	t.Run("default branch client principal", func(t *testing.T) {
		ctx := buildAuthContext(&Principal{ClientPrincipal: &ClientPrincipal{ClientID: "CLIENT-001"}})
		if ctx.Client == nil || ctx.Client.ClientID != "CLIENT-001" {
			t.Fatalf("unexpected client context: %#v", ctx)
		}
	})

	t.Run("user principal with phone", func(t *testing.T) {
		phone := "+61400111222"
		ctx := buildAuthContext(&Principal{
			Type: UserPrincipalType,
			UserPrincipal: &UserPrincipal{
				UserID:      "TRADER-001",
				Email:       "trader@example.com",
				PhoneNumber: &phone,
				OUID:        "OU-001",
			},
		})
		if ctx.User == nil || ctx.User.PhoneNumber != phone {
			t.Fatalf("unexpected user context: %#v", ctx.User)
		}
	})
}

func TestManagerWrappers(t *testing.T) {
	db, mock := setupAuthTestDB(t)
	service := NewAuthService(db)
	manager := &Manager{
		service:        service,
		tokenExtractor: &TokenExtractor{},
		middleware:     Middleware(service, &TokenExtractor{}),
	}

	if manager.Service() != service {
		t.Fatalf("expected service to be returned")
	}
	if manager.Middleware() == nil || manager.OptionalAuthMiddleware() == nil || manager.RequireAuthMiddleware() == nil {
		t.Fatalf("expected middleware constructors to be non-nil")
	}

	mock.ExpectQuery(`SELECT 1 FROM user_records LIMIT 1`).WillReturnRows(sqlmock.NewRows([]string{"?column?"}).AddRow(1))
	if err := manager.Health(); err != nil {
		t.Fatalf("expected health check to pass, got %v", err)
	}

	if err := manager.Close(); err != nil {
		t.Fatalf("expected close to be nil, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestManagerGetUserContext(t *testing.T) {
	db, mock := setupAuthTestDB(t)
	manager := &Manager{service: NewAuthService(db)}

	rows := sqlmock.NewRows([]string{"user_id", "email", "phone_number", "ou_id", "nsw_data"}).
		AddRow("TRADER-001", "trader@example.com", "+61400111222", "OU-001", []byte(`{}`))
	mock.ExpectQuery(`SELECT .* FROM "user_records" WHERE user_id = \$1 ORDER BY "user_records"\."user_id" LIMIT \$2`).
		WithArgs("TRADER-001", 1).
		WillReturnRows(rows)

	userCtx, err := manager.GetUserContext("TRADER-001")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if userCtx == nil || userCtx.UserID != "TRADER-001" {
		t.Fatalf("unexpected user context: %#v", userCtx)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestManagerUpdateUserContext(t *testing.T) {
	db, mock := setupAuthTestDB(t)
	service := NewAuthService(db)
	manager := &Manager{service: service}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "user_records" SET "nsw_data"=\$1 WHERE user_id = \$2`).
		WithArgs([]byte(`{"status":"verified"}`), "TRADER-001").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := manager.UpdateUserContext("TRADER-001", map[string]any{"status": "verified"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestManagerUpdateUserContextBytes(t *testing.T) {
	db, mock := setupAuthTestDB(t)
	manager := &Manager{service: NewAuthService(db)}
	ctx := []byte(`{"status":"verified"}`)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "user_records" SET "nsw_data"=\$1 WHERE user_id = \$2`).
		WithArgs(ctx, "TRADER-001").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := manager.UpdateUserContext("TRADER-001", ctx); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestNewManagerAndTokenExtractorWithClient(t *testing.T) {
	db, mock := setupAuthTestDB(t)
	defer func() {
		_ = db
	}()

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	defer jwksServer.Close()

	config := Config{
		JWKSURL:   jwksServer.URL,
		Issuer:    "https://localhost:8090/oauth2/token",
		Audience:  "TRADER_PORTAL_APP",
		ClientIDs: []string{"TRADER_PORTAL_APP"},
	}

	manager, err := NewManager(db, config)
	if err != nil {
		t.Fatalf("expected manager to be created, got %v", err)
	}
	if manager == nil || manager.Service() == nil || manager.Middleware() == nil {
		t.Fatalf("expected initialized manager, got %#v", manager)
	}

	if _, err := NewTokenExtractorWithClient(config.JWKSURL, config.Issuer, config.Audience, config.ClientIDs, nil); err != nil {
		t.Fatalf("expected nil-client constructor path to succeed, got %v", err)
	}

	if _, err := NewTokenExtractorWithClient(config.JWKSURL, config.Issuer, config.Audience, config.ClientIDs, &http.Client{Timeout: time.Second}); err != nil {
		t.Fatalf("expected custom-client constructor path to succeed, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestNewManager_InvalidConfig(t *testing.T) {
	db, _ := setupAuthTestDB(t)
	_, err := NewManager(db, Config{})
	if err == nil {
		t.Fatalf("expected manager initialization to fail for invalid config")
	}
}

func TestClientPrincipalFromClaimsAndParserHelpers(t *testing.T) {
	if _, err := (&TokenExtractor{}).clientPrincipalFromClaims(&tokenClaims{}); err == nil {
		t.Fatalf("expected client principal validation error")
	}

	principal, err := (&TokenExtractor{}).clientPrincipalFromClaims(&tokenClaims{ClientID: "CLIENT-001"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if principal.ClientID != "CLIENT-001" {
		t.Fatalf("unexpected client principal: %#v", principal)
	}

	if _, err := parseRSAPublicKey(jwk{}); err == nil {
		t.Fatalf("expected parseRSAPublicKey error")
	}

	if _, err := (&TokenExtractor{}).keyFunc(jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "TRADER-001"})); err == nil {
		t.Fatalf("expected signing method error")
	}
}
