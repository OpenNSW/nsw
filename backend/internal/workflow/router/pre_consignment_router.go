package router

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/OpenNSW/nsw/internal/auth"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/service"
	"github.com/OpenNSW/nsw/utils"
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

// HandleGetTraderPreConsignments handles GET /api/v1/pre-consignments
// No query params required for traderId - uses traderId from auth context
// Pagination query params: offset (optional), limit (optional)
// Response: TraderPreConsignmentsResponseDTO
func (r *PreConsignmentRouter) HandleGetTraderPreConsignments(w http.ResponseWriter, req *http.Request) {
	// Require authentication
	authCtx := auth.GetAuthContext(req.Context())
	if authCtx == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Use traderId from auth context
	traderID := authCtx.TraderID

	offset, limit, err := utils.ParsePaginationParams(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	templates, err := r.pcs.GetTraderPreConsignments(req.Context(), traderID, offset, limit)
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
	// Require authentication
	authCtx := auth.GetAuthContext(req.Context())
	if authCtx == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var createReq model.CreatePreConsignmentDTO
	if err := json.NewDecoder(req.Body).Decode(&createReq); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Extract traderId from auth context
	traderId := authCtx.TraderID

	// Extract traderContext from auth
	traderContext, err := authCtx.GetTraderContextMap()
	if err != nil {
		http.Error(w, "failed to parse trader context", http.StatusInternalServerError)
		return
	}

	preConsignment, _, err := r.pcs.InitializePreConsignment(req.Context(), &createReq, traderId, traderContext)
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

// HandleGetPreConsignmentsByTraderID handles GET /api/v1/pre-consignments
// Returns all pre-consignment instances for authenticated trader
func (r *PreConsignmentRouter) HandleGetPreConsignmentsByTraderID(w http.ResponseWriter, req *http.Request) {
	// Require authentication
	authCtx := auth.GetAuthContext(req.Context())
	if authCtx == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Use traderId from auth context
	traderID := authCtx.TraderID

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
