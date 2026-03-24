package preconsignment

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenNSW/nsw/internal/auth"
)

func withAuth(req *http.Request, userID string) *http.Request {
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthContextKey, &auth.AuthContext{UserID: userID}))
	return req
}

func TestHandleGetTraderPreConsignments_Unauthorized(t *testing.T) {
	h := NewPreConsignmentHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pre-consignments", nil)
	w := httptest.NewRecorder()

	h.HandleGetTraderPreConsignments(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleGetTraderPreConsignments_InvalidPagination(t *testing.T) {
	h := NewPreConsignmentHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pre-consignments?limit=bad", nil)
	req = withAuth(req, "trader-1")
	w := httptest.NewRecorder()

	h.HandleGetTraderPreConsignments(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleCreatePreConsignment_Unauthorized(t *testing.T) {
	h := NewPreConsignmentHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pre-consignments", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()

	h.HandleCreatePreConsignment(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleCreatePreConsignment_InvalidBody(t *testing.T) {
	h := NewPreConsignmentHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pre-consignments", bytes.NewBufferString("{"))
	req = withAuth(req, "trader-1")
	w := httptest.NewRecorder()

	h.HandleCreatePreConsignment(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleGetPreConsignmentsByTraderID_Unauthorized(t *testing.T) {
	h := NewPreConsignmentHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pre-consignments", nil)
	w := httptest.NewRecorder()

	h.HandleGetPreConsignmentsByTraderID(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleGetPreConsignmentByID_MissingID(t *testing.T) {
	h := NewPreConsignmentHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pre-consignments/", nil)
	w := httptest.NewRecorder()

	h.HandleGetPreConsignmentByID(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
