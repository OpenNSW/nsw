package company

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// stubService is a minimal Service stub for handler tests. It captures the filter passed
// into ListCompanies so tests can assert query-param parsing.
type stubService struct {
	listRecords []Record
	listErr     error
	lastFilter  ListFilter
}

func (s *stubService) GetCompanyByID(_ context.Context, _ string) (*Record, error) { return nil, nil }
func (s *stubService) GetCompanyByOUHandle(_ context.Context, _ string) (*Record, error) {
	return nil, nil
}
func (s *stubService) UpdateCompany(_ context.Context, _ string, _ map[string]any) error { return nil }
func (s *stubService) Health(_ context.Context) error                                    { return nil }
func (s *stubService) ListCompanies(_ context.Context, filter ListFilter) ([]Record, error) {
	s.lastFilter = filter
	return s.listRecords, s.listErr
}

func TestHandler_HandleGetCompanies_Success(t *testing.T) {
	records := []Record{
		// Include fields the Summary should drop to ensure they don't leak into the response.
		{ID: "adam-pvt-ltd", Name: "ADAM PVT LTD", OUHandle: "adam-pvt-ltd", HasCHA: true, Data: []byte(`{"br_no":"BR-1"}`)},
		{ID: "edward-pvt-ltd", Name: "EDWARD PVT LTD", OUHandle: "edward-pvt-ltd", HasCHA: true, Data: []byte(`{"br_no":"BR-2"}`)},
	}
	stub := &stubService{listRecords: records}
	h := NewHandler(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies", nil)
	w := httptest.NewRecorder()
	h.HandleGetCompanies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}
	var got ListResult
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.Total != 2 || got.Offset != 0 || got.Limit != 2 {
		t.Fatalf("unexpected envelope: %+v", got)
	}
	if len(got.Items) != 2 || got.Items[0].ID != "adam-pvt-ltd" || got.Items[1].ID != "edward-pvt-ltd" {
		t.Fatalf("unexpected items: %+v", got.Items)
	}
	if !got.Items[0].HasCHA || got.Items[0].Name != "ADAM PVT LTD" {
		t.Fatalf("unexpected first item: %+v", got.Items[0])
	}
	if stub.lastFilter.HasCHA != nil || stub.lastFilter.Name != nil {
		t.Fatalf("expected empty filter, got %+v", stub.lastFilter)
	}

	// Verify the response shape carries only Summary fields (ouHandle/data/createdAt/updatedAt
	// dropped). Decode into a generic map to assert key presence.
	w2 := httptest.NewRecorder()
	h.HandleGetCompanies(w2, httptest.NewRequest(http.MethodGet, "/api/v1/companies", nil))
	var rawEnv struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(w2.Body).Decode(&rawEnv); err != nil {
		t.Fatalf("failed to decode raw envelope: %v", err)
	}
	if len(rawEnv.Items) != 2 {
		t.Fatalf("expected 2 items in raw envelope, got %d", len(rawEnv.Items))
	}
	for i, item := range rawEnv.Items {
		for _, leaked := range []string{"ouHandle", "data", "createdAt", "updatedAt"} {
			if _, ok := item[leaked]; ok {
				t.Errorf("item %d leaks %q: %+v", i, leaked, item)
			}
		}
		for _, required := range []string{"id", "name", "hasCha"} {
			if _, ok := item[required]; !ok {
				t.Errorf("item %d missing %q: %+v", i, required, item)
			}
		}
	}
}

func TestHandler_HandleGetCompanies_HasCHATrue(t *testing.T) {
	stub := &stubService{listRecords: []Record{}}
	h := NewHandler(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies?has_cha=true", nil)
	w := httptest.NewRecorder()
	h.HandleGetCompanies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if stub.lastFilter.HasCHA == nil || !*stub.lastFilter.HasCHA {
		t.Fatalf("expected HasCHA=true, got %+v", stub.lastFilter.HasCHA)
	}
}

func TestHandler_HandleGetCompanies_HasCHAFalse(t *testing.T) {
	stub := &stubService{listRecords: []Record{}}
	h := NewHandler(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies?has_cha=false", nil)
	w := httptest.NewRecorder()
	h.HandleGetCompanies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if stub.lastFilter.HasCHA == nil || *stub.lastFilter.HasCHA {
		t.Fatalf("expected HasCHA=false, got %+v", stub.lastFilter.HasCHA)
	}
}

func TestHandler_HandleGetCompanies_HasCHAInvalid(t *testing.T) {
	stub := &stubService{listRecords: []Record{}}
	h := NewHandler(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies?has_cha=notabool", nil)
	w := httptest.NewRecorder()
	h.HandleGetCompanies(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandler_HandleGetCompanies_NameFilter(t *testing.T) {
	stub := &stubService{listRecords: []Record{}}
	h := NewHandler(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies?name=adam", nil)
	w := httptest.NewRecorder()
	h.HandleGetCompanies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if stub.lastFilter.Name == nil || *stub.lastFilter.Name != "adam" {
		t.Fatalf("expected Name=adam, got %+v", stub.lastFilter.Name)
	}
}

func TestHandler_HandleGetCompanies_CombinedFilter(t *testing.T) {
	stub := &stubService{listRecords: []Record{}}
	h := NewHandler(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies?has_cha=true&name=adam", nil)
	w := httptest.NewRecorder()
	h.HandleGetCompanies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if stub.lastFilter.HasCHA == nil || !*stub.lastFilter.HasCHA {
		t.Fatalf("expected HasCHA=true, got %+v", stub.lastFilter.HasCHA)
	}
	if stub.lastFilter.Name == nil || *stub.lastFilter.Name != "adam" {
		t.Fatalf("expected Name=adam, got %+v", stub.lastFilter.Name)
	}
}

func TestHandler_HandleGetCompanies_ServiceError(t *testing.T) {
	h := NewHandler(&stubService{listErr: errors.New("database down")})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies", nil)
	w := httptest.NewRecorder()
	h.HandleGetCompanies(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
