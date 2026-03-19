package preconsignment

import (
	"encoding/json"
	"net/http"

	"github.com/OpenNSW/nsw/internal/auth"
	"github.com/OpenNSW/nsw/utils"
)

// PreConsignmentHandler handles HTTP routing for pre-consignment endpoints.
type PreConsignmentHandler struct {
	pcs *PreConsignmentService
}

// NewPreConsignmentHandler creates a new PreConsignmentHandler.
func NewPreConsignmentHandler(pcs *PreConsignmentService) *PreConsignmentHandler {
	return &PreConsignmentHandler{
		pcs: pcs,
	}
}

// HandleGetTraderPreConsignments handles GET /api/v1/pre-consignments
// No query params required for traderId - uses traderId from auth context
// Pagination query params: offset (optional), limit (optional)
// Response: TraderPreConsignmentsResponseDTO
func (r *PreConsignmentHandler) HandleGetTraderPreConsignments(w http.ResponseWriter, req *http.Request) {
	// Require authentication
	authCtx := auth.GetAuthContext(req.Context())
	if authCtx == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	traderID := authCtx.UserID

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
func (r *PreConsignmentHandler) HandleCreatePreConsignment(w http.ResponseWriter, req *http.Request) {
	// Require authentication
	authCtx := auth.GetAuthContext(req.Context())
	if authCtx == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var createReq CreatePreConsignmentDTO
	if err := json.NewDecoder(req.Body).Decode(&createReq); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	traderId := authCtx.UserID
	traderContext, err := authCtx.GetUserContextMap()
	if err != nil {
		http.Error(w, "failed to parse trader context", http.StatusInternalServerError)
		return
	}

	preConsignment, err := r.pcs.InitializePreConsignment(req.Context(), &createReq, traderId, traderContext)
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
func (r *PreConsignmentHandler) HandleGetPreConsignmentsByTraderID(w http.ResponseWriter, req *http.Request) {
	// Require authentication
	authCtx := auth.GetAuthContext(req.Context())
	if authCtx == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	traderID := authCtx.UserID

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
func (r *PreConsignmentHandler) HandleGetPreConsignmentByID(w http.ResponseWriter, req *http.Request) {
	preConsignmentIDStr := req.PathValue("preConsignmentId")
	if preConsignmentIDStr == "" {
		http.Error(w, "pre-consignment ID is required", http.StatusBadRequest)
		return
	}

	preConsignmentID := preConsignmentIDStr

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
