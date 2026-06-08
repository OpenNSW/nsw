package hscode

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/OpenNSW/nsw/backend/pkg/pagination"
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

	if hsCodeStartsWith := r.URL.Query().Get("hsCodeStartsWith"); hsCodeStartsWith != "" {
		filter.HSCodeStartsWith = &hsCodeStartsWith
	}

	offset, limit, err := pagination.ParsePaginationParams(r)
	if err != nil {
		http.Error(w, "invalid pagination parameters", http.StatusBadRequest)
		slog.Error("invalid pagination parameters", "error", err)
		return
	}
	filter.Offset = offset
	filter.Limit = limit

	hsCodes, err := h.service.GetAll(r.Context(), filter)
	if err != nil {
		http.Error(w, "failed to retrieve HS Codes", http.StatusInternalServerError)
		slog.Error("failed to retrieve HS Codes", "error", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(hsCodes); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		slog.Error("failed to encode response", "error", err)
		return
	}
}
