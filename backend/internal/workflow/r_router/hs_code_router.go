package r_router

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/OpenNSW/nsw/internal/workflow/r_model"
	"github.com/OpenNSW/nsw/internal/workflow/r_service"
)

type HSCodeRouter struct {
	hscs *r_service.HSCodeService
}

func NewHSCodeRouter(hscs *r_service.HSCodeService) *HSCodeRouter {
	return &HSCodeRouter{
		hscs: hscs,
	}
}

// HandleGetAllHSCodes handles GET /api/v1/hscodes
// Optional Query Params: hsCodeStartsWith, offset, limit
func (h *HSCodeRouter) HandleGetAllHSCodes(w http.ResponseWriter, r *http.Request) {
	var filter r_model.HSCodeFilter

	// Parse query parameters
	if hsCodeStartsWith := r.URL.Query().Get("hsCodeStartsWith"); hsCodeStartsWith != "" {
		filter.HSCodeStartsWith = &hsCodeStartsWith
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "invalid 'limit' query parameter, must be an integer", http.StatusBadRequest)
			return
		}
		filter.Limit = &limit
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			http.Error(w, "invalid 'offset' query parameter, must be an integer", http.StatusBadRequest)
			return
		}
		filter.Offset = &offset
	}

	// Get HS codes from service
	hsCodes, err := h.hscs.GetAllHSCodes(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(hsCodes); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
