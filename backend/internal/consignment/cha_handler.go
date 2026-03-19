package consignment

import (
	"encoding/json"
	"net/http"
)

type CHAHandler struct {
	chaService *CHAService
}

func NewCHAHandler(chaService *CHAService) *CHAHandler {
	return &CHAHandler{chaService: chaService}
}

// HandleGetCHAs handles GET /api/v1/chas
func (cr *CHAHandler) HandleGetCHAs(w http.ResponseWriter, r *http.Request) {
	chas, err := cr.chaService.ListCHAs(r.Context())
	if err != nil {
		http.Error(w, "failed to retrieve CHAs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(chas); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
