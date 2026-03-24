package consignment

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

func TestHandleCreateConsignment_Unauthorized(t *testing.T) {
	h := NewHandler(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/consignments", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()

	h.HandleCreateConsignment(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleCreateConsignment_InvalidBody(t *testing.T) {
	h := NewHandler(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/consignments", bytes.NewBufferString("{"))
	req = withAuth(req, "trader-1")
	w := httptest.NewRecorder()

	h.HandleCreateConsignment(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleGetConsignments_InvalidPagination(t *testing.T) {
	h := NewHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/consignments?limit=bad", nil)
	req = withAuth(req, "trader-1")
	w := httptest.NewRecorder()

	h.HandleGetConsignments(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleGetConsignments_InvalidRole(t *testing.T) {
	h := NewHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/consignments?role=invalid", nil)
	req = withAuth(req, "trader-1")
	w := httptest.NewRecorder()

	h.HandleGetConsignments(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleInitializeConsignment_MissingID(t *testing.T) {
	h := NewHandler(nil, nil)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/consignments/", bytes.NewBufferString(`{"hsCodeIds":["a"]}`))
	req = withAuth(req, "trader-1")
	w := httptest.NewRecorder()

	h.HandleInitializeConsignment(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleInitializeConsignment_InvalidBody(t *testing.T) {
	h := NewHandler(nil, nil)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/consignments/id-1", bytes.NewBufferString("{"))
	req = withAuth(req, "trader-1")
	req.SetPathValue("id", "id-1")
	w := httptest.NewRecorder()

	h.HandleInitializeConsignment(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleInitializeConsignment_EmptyHSCodeIDs(t *testing.T) {
	h := NewHandler(nil, nil)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/consignments/id-1", bytes.NewBufferString(`{"hsCodeIds":[]}`))
	req = withAuth(req, "trader-1")
	req.SetPathValue("id", "id-1")
	w := httptest.NewRecorder()

	h.HandleInitializeConsignment(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleGetConsignmentByID_MissingID(t *testing.T) {
	h := NewHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/consignments/", nil)
	w := httptest.NewRecorder()

	h.HandleGetConsignmentByID(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
