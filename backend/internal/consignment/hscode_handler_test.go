package consignment

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleGetAllHSCodes_InvalidLimit(t *testing.T) {
	h := NewHSCodeHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hscodes?limit=bad", nil)
	w := httptest.NewRecorder()

	h.HandleGetAllHSCodes(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleGetAllHSCodes_InvalidOffset(t *testing.T) {
	h := NewHSCodeHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hscodes?offset=bad", nil)
	w := httptest.NewRecorder()

	h.HandleGetAllHSCodes(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
