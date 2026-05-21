package company

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// Handler exposes company profile endpoints.
type Handler struct {
	svc Service
}

// NewHandler creates a new company HTTP handler.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// HandleGetCompanies handles GET /api/v1/companies.
// Optional query params: has_cha (true|false), name (substring, case-insensitive).
func (h *Handler) HandleGetCompanies(w http.ResponseWriter, r *http.Request) {
	filter := ListFilter{}
	if v := r.URL.Query().Get("has_cha"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			http.Error(w, "invalid has_cha (expected true or false)", http.StatusBadRequest)
			return
		}
		filter.HasCHA = &parsed
	}
	if name := r.URL.Query().Get("name"); name != "" {
		filter.Name = &name
	}

	records, err := h.svc.ListCompanies(r.Context(), filter)
	if err != nil {
		http.Error(w, "failed to retrieve companies", http.StatusInternalServerError)
		return
	}

	items := make([]Summary, 0, len(records))
	for _, r := range records {
		items = append(items, Summary{ID: r.ID, Name: r.Name, HasCHA: r.HasCHA})
	}
	// TODO: implement pagination — parse offset/limit query params, push them into
	// Service.ListCompanies, and return real Total/Offset/Limit instead of the
	// full-page placeholders below. The envelope shape is here so the contract is
	// pagination-ready and adding the params later is non-breaking.
	result := ListResult{
		Items:  items,
		Total:  int64(len(items)),
		Offset: 0,
		Limit:  len(items),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
