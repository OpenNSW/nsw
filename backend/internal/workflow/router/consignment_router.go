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

type ConsignmentRouter struct {
	cs *service.ConsignmentService
}

func NewConsignmentRouter(cs *service.ConsignmentService, _ interface{}) *ConsignmentRouter {
	return &ConsignmentRouter{
		cs: cs,
	}
}

// HandleCreateConsignment handles POST /api/v1/consignments
// Request body: CreateConsignmentDTO
// Response: ConsignmentResponseDTO
func (c *ConsignmentRouter) HandleCreateConsignment(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	authCtx := auth.GetAuthContext(r.Context())
	if authCtx == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req model.CreateConsignmentDTO

	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Extract traderId from auth context
	traderId := authCtx.TraderID

	// Extract globalContext from auth context
	globalContext, err := authCtx.GetTraderContextMap()
	if err != nil {
		http.Error(w, "failed to parse trader context", http.StatusInternalServerError)
		return
	}

	// Create consignment through service
	// Task registration happens within the transaction via pre-commit callback
	consignment, _, err := c.cs.InitializeConsignment(r.Context(), &req, traderId, globalContext)
	if err != nil {
		http.Error(w, "failed to create consignment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response - all operations completed successfully within transaction
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(consignment); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HandleGetConsignmentsByTraderID handles GET /api/v1/consignments
// No query params required for traderId - uses traderId from auth context
// Pagination query params: offset (optional), limit (optional)
// Response: ConsignmentListResult
func (c *ConsignmentRouter) HandleGetConsignmentsByTraderID(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	authCtx := auth.GetAuthContext(r.Context())
	if authCtx == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Use traderId from auth context
	traderID := authCtx.TraderID

	offset, limit, err := utils.ParsePaginationParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Parse optional filters
	var filter model.ConsignmentFilter
	if stateStr := r.URL.Query().Get("state"); stateStr != "" {
		state := model.ConsignmentState(stateStr)
		filter.State = &state
	}
	if flowStr := r.URL.Query().Get("flow"); flowStr != "" {
		flow := model.ConsignmentFlow(flowStr)
		filter.Flow = &flow
	}

	// Get consignments from service
	consignments, err := c.cs.GetConsignmentsByTraderID(r.Context(), traderID, offset, limit, filter)
	if err != nil {
		http.Error(w, "failed to retrieve consignments: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(consignments); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HandleGetConsignmentByID handles GET /api/v1/consignments/{id}
// Path param: id (required)
// Response: ConsignmentResponseDTO
func (c *ConsignmentRouter) HandleGetConsignmentByID(w http.ResponseWriter, r *http.Request) {
	// Extract consignment ID from path
	consignmentIDStr := r.PathValue("id")
	if consignmentIDStr == "" {
		http.Error(w, "consignment ID is required", http.StatusBadRequest)
		return
	}

	// Parse UUID
	consignmentID, err := uuid.Parse(consignmentIDStr)
	if err != nil {
		http.Error(w, "invalid consignment ID format: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get consignment from service
	consignment, err := c.cs.GetConsignmentByID(r.Context(), consignmentID)
	if err != nil {
		http.Error(w, "failed to retrieve consignment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(consignment); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
