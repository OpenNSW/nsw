package hscode

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type Router struct {
	service *Service
}

func NewRouter(service *Service) *Router {
	return &Router{
		service: service,
	}
}

// HandleGetAll handles GET /api/v1/hscodes
// Optional Query Params: hsCodeStartsWith, offset, limit
func (h *Router) HandleGetAll(w http.ResponseWriter, r *http.Request) {
	var filter Filter

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
	hsCodes, err := h.service.GetAll(r.Context(), filter)
	if err != nil {
		http.Error(w, "failed to retrieve HS Codes", http.StatusInternalServerError)
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
