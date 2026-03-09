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
// Stage 1 (two-stage): body { flow, chaId } → creates shell (AWAITING_INITIATION)
// Legacy: body { flow, items } → creates and initializes workflow
func (c *ConsignmentRouter) HandleCreateConsignment(w http.ResponseWriter, r *http.Request) {
	authCtx := auth.GetAuthContext(r.Context())
	if authCtx == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req model.CreateConsignmentDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	traderID := authCtx.TraderID
	globalContext, err := authCtx.GetTraderContextMap()
	if err != nil {
		http.Error(w, "failed to parse trader context", http.StatusInternalServerError)
		return
	}

	if req.ChaID != nil {
		// Stage 1: create shell only
		consignment, err := c.cs.CreateConsignmentShell(r.Context(), req.Flow, *req.ChaID, traderID, globalContext)
		if err != nil {
			http.Error(w, "failed to create consignment: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(consignment)
		return
	}

	// Legacy: full init with items
	consignment, _, err := c.cs.InitializeConsignment(r.Context(), &req, traderID, globalContext)
	if err != nil {
		http.Error(w, "failed to create consignment: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(consignment)
}

// HandleGetConsignments handles GET /api/v1/consignments
// Query params: role=trader | role=cha (required). When role=cha, cha_id (UUID) is required
// Pagination: offset, limit. Optional filters: state, flow
// Response: ConsignmentListResult (containing ConsignmentSummaryDTO)
func (c *ConsignmentRouter) HandleGetConsignments(w http.ResponseWriter, r *http.Request) {
	authCtx := auth.GetAuthContext(r.Context())
	if authCtx == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	role := r.URL.Query().Get("role")
	if role == "" {
		role = "trader"
	}
	if role != "trader" && role != "cha" {
		http.Error(w, "query param role must be trader or cha", http.StatusBadRequest)
		return
	}

	offset, limit, err := utils.ParsePaginationParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var filter model.ConsignmentFilter
	filter.Offset = offset
	filter.Limit = limit
	if stateStr := r.URL.Query().Get("state"); stateStr != "" {
		state := model.ConsignmentState(stateStr)
		filter.State = &state
	}
	if flowStr := r.URL.Query().Get("flow"); flowStr != "" {
		flow := model.ConsignmentFlow(flowStr)
		filter.Flow = &flow
	}

	if role == "cha" {
		chaIDStr := r.URL.Query().Get("cha_id")
		if chaIDStr == "" {
			http.Error(w, "cha_id query param is required when role=cha", http.StatusBadRequest)
			return
		}
		chaID, err := uuid.Parse(chaIDStr)
		if err != nil {
			http.Error(w, "invalid cha_id format", http.StatusBadRequest)
			return
		}
		filter.ChaID = &chaID
	} else {
		traderID := authCtx.TraderID
		filter.TraderID = &traderID
	}

	consignments, err := c.cs.ListConsignments(r.Context(), filter)
	if err != nil {
		http.Error(w, "failed to retrieve consignments: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(consignments)
}

// HandleInitializeConsignment handles PUT /api/v1/consignments/{id}/initialize (Stage 2: CHA selects HS Code).
// Body: InitializeConsignmentDTO { hsCodeId }. Response: ConsignmentDetailDTO.
func (c *ConsignmentRouter) HandleInitializeConsignment(w http.ResponseWriter, r *http.Request) {
	consignmentIDStr := r.PathValue("id")
	if consignmentIDStr == "" {
		http.Error(w, "consignment ID is required", http.StatusBadRequest)
		return
	}
	consignmentID, err := uuid.Parse(consignmentIDStr)
	if err != nil {
		http.Error(w, "invalid consignment ID format", http.StatusBadRequest)
		return
	}

	var req model.InitializeConsignmentDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	consignment, _, err := c.cs.InitializeConsignmentByID(r.Context(), consignmentID, req.HSCodeID)
	if err != nil {
		http.Error(w, "failed to initialize consignment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(consignment)
}

// HandleGetConsignmentByID handles GET /api/v1/consignments/{id}
// Path param: id (required)
// Response: ConsignmentDetailDTO
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
