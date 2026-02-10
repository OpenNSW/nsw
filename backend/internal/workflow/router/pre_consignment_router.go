package router

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/service"
)

// PreConsignmentRouter handles HTTP routing for pre-consignment endpoints.
type PreConsignmentRouter struct {
	pcs *service.PreConsignmentService
}

// NewPreConsignmentRouter creates a new PreConsignmentRouter.
func NewPreConsignmentRouter(pcs *service.PreConsignmentService) *PreConsignmentRouter {
	return &PreConsignmentRouter{
		pcs: pcs,
	}
}

// HandleGetTraderPreConsignments handles GET /api/v1/pre-consignments?traderId={traderId}
func (r *PreConsignmentRouter) HandleGetTraderPreConsignments(w http.ResponseWriter, req *http.Request) {
	traderID := req.URL.Query().Get("traderId")
	if traderID == "" {
		http.Error(w, "traderId query parameter is required", http.StatusBadRequest)
		return
	}

	templates, err := r.pcs.GetTraderPreConsignments(req.Context(), traderID, nil, nil)
	if err != nil {
		http.Error(w, "failed to retrieve pre-consignment templates: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(templates); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HandleCreatePreConsignment handles POST /api/v1/pre-consignments
func (r *PreConsignmentRouter) HandleCreatePreConsignment(w http.ResponseWriter, req *http.Request) {
	var createReq model.CreatePreConsignmentDTO
	if err := json.NewDecoder(req.Body).Decode(&createReq); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	preConsignment, _, err := r.pcs.InitializePreConsignment(req.Context(), &createReq)
	if err != nil {
		http.Error(w, "failed to create pre-consignment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(preConsignment); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HandleGetPreConsignmentsByTraderID handles GET /api/v1/pre-consignments?traderId={traderId}
func (r *PreConsignmentRouter) HandleGetPreConsignmentsByTraderID(w http.ResponseWriter, req *http.Request) {
	traderID := req.URL.Query().Get("traderId")
	if traderID == "" {
		http.Error(w, "traderId query parameter is required", http.StatusBadRequest)
		return
	}

	preConsignments, err := r.pcs.GetPreConsignmentsByTraderID(req.Context(), traderID)
	if err != nil {
		http.Error(w, "failed to retrieve pre-consignments: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(preConsignments); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HandleGetPreConsignmentByID handles GET /api/v1/pre-consignments/{preConsignmentId}
func (r *PreConsignmentRouter) HandleGetPreConsignmentByID(w http.ResponseWriter, req *http.Request) {
	preConsignmentIDStr := req.PathValue("preConsignmentId")
	if preConsignmentIDStr == "" {
		http.Error(w, "pre-consignment ID is required", http.StatusBadRequest)
		return
	}

	preConsignmentID, err := uuid.Parse(preConsignmentIDStr)
	if err != nil {
		http.Error(w, "invalid pre-consignment ID format: "+err.Error(), http.StatusBadRequest)
		return
	}

	preConsignment, err := r.pcs.GetPreConsignmentByID(req.Context(), preConsignmentID)
	if err != nil {
		http.Error(w, "failed to retrieve pre-consignment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(preConsignment); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
