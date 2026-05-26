package profile

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenNSW/nsw/backend/internal/auth"
	"github.com/OpenNSW/nsw/backend/internal/profile/company"
	"github.com/OpenNSW/nsw/backend/internal/profile/user"
)

type mockUserService struct {
	getUserFn func(id string) (*user.Record, error)
}

func (m *mockUserService) GetUser(id string) (*user.Record, error) {
	if m.getUserFn != nil {
		return m.getUserFn(id)
	}
	return nil, nil
}
func (m *mockUserService) GetOrCreateUser(_, _, _, _, _ string) (*string, error) { return nil, nil }
func (m *mockUserService) UpdateUserData(_ string, _ []byte) error               { return nil }
func (m *mockUserService) Health() error                                         { return nil }

type mockCompanyService struct {
	getCompanyByOUHandleFn func(ctx context.Context, ouHandle string) (*company.Record, error)
}

func (m *mockCompanyService) GetCompanyByID(_ context.Context, _ string) (*company.Record, error) {
	return nil, nil
}
func (m *mockCompanyService) GetCompanyByOUHandle(ctx context.Context, ouHandle string) (*company.Record, error) {
	if m.getCompanyByOUHandleFn != nil {
		return m.getCompanyByOUHandleFn(ctx, ouHandle)
	}
	return nil, nil
}
func (m *mockCompanyService) ListCompanies(_ context.Context, _ company.ListFilter) ([]company.Record, error) {
	return nil, nil
}
func (m *mockCompanyService) UpdateCompany(_ context.Context, _ string, _ map[string]any) error {
	return nil
}
func (m *mockCompanyService) Health(_ context.Context) error { return nil }

func TestHandler_HandleGetProfile_Unauthorized(t *testing.T) {
	uSvc := &mockUserService{}
	cSvc := &mockCompanyService{}
	h := NewHandler(uSvc, cSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	w := httptest.NewRecorder()
	h.HandleGetProfile(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", w.Code)
	}
}

func TestHandler_HandleGetProfile_UserNotFound(t *testing.T) {
	uSvc := &mockUserService{
		getUserFn: func(id string) (*user.Record, error) {
			return nil, user.ErrUserNotFound
		},
	}
	cSvc := &mockCompanyService{}
	h := NewHandler(uSvc, cSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	ctx := context.WithValue(req.Context(), auth.AuthContextKey, &auth.AuthContext{
		User: &auth.UserContext{
			ID: "test-user-id",
		},
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.HandleGetProfile(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found, got %d", w.Code)
	}
}

func TestHandler_HandleGetProfile_Success(t *testing.T) {
	uRecord := &user.Record{
		ID:          "user-1",
		IDPUserID:   "idp-user-1",
		Email:       "user@example.com",
		PhoneNumber: "123456",
		OUID:        "ou-1",
		OUHandle:    "ou-handle-1",
		Data:        []byte(`{}`),
	}
	cRecord := &company.Record{
		ID:       "company-1",
		Name:     "Test Company",
		OUHandle: "ou-handle-1",
		HasCHA:   false,
		Data:     []byte(`{}`),
	}

	uSvc := &mockUserService{
		getUserFn: func(id string) (*user.Record, error) {
			return uRecord, nil
		},
	}
	cSvc := &mockCompanyService{
		getCompanyByOUHandleFn: func(ctx context.Context, ouHandle string) (*company.Record, error) {
			if ouHandle != "ou-handle-1" {
				t.Errorf("expected ouHandle 'ou-handle-1', got '%s'", ouHandle)
			}
			return cRecord, nil
		},
	}
	h := NewHandler(uSvc, cSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	ctx := context.WithValue(req.Context(), auth.AuthContextKey, &auth.AuthContext{
		User: &auth.UserContext{
			ID:       "user-1",
			OUHandle: "ou-handle-1",
		},
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.HandleGetProfile(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp UserProfile
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ID != "user-1" {
		t.Errorf("expected User ID 'user-1', got %s", resp.ID)
	}
	if resp.Company == nil || resp.Company.ID != "company-1" {
		t.Errorf("expected Company ID 'company-1', got %+v", resp.Company)
	}
}

func TestHandler_HandleGetProfile_CompanyNotFound(t *testing.T) {
	uRecord := &user.Record{
		ID:       "user-1",
		OUHandle: "non-existent-ou",
	}

	uSvc := &mockUserService{
		getUserFn: func(id string) (*user.Record, error) {
			return uRecord, nil
		},
	}
	cSvc := &mockCompanyService{
		getCompanyByOUHandleFn: func(ctx context.Context, ouHandle string) (*company.Record, error) {
			return nil, company.ErrCompanyNotFound
		},
	}
	h := NewHandler(uSvc, cSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	ctx := context.WithValue(req.Context(), auth.AuthContextKey, &auth.AuthContext{
		User: &auth.UserContext{
			ID:       "user-1",
			OUHandle: "non-existent-ou",
		},
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.HandleGetProfile(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK even if company not found, got %d", w.Code)
	}

	var resp UserProfile
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ID != "user-1" {
		t.Errorf("expected User ID 'user-1', got %s", resp.ID)
	}
	if resp.Company != nil {
		t.Errorf("expected Company to be nil, got %+v", resp.Company)
	}
}
