package router

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/OpenNSW/nsw/internal/auth"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/service"
	"github.com/OpenNSW/nsw/utils"
)

type ConsignmentRouter struct {
	cs  *service.ConsignmentService
	cha *service.CHAService
}

func NewConsignmentRouter(cs *service.ConsignmentService, cha *service.CHAService) *ConsignmentRouter {
	return &ConsignmentRouter{cs: cs, cha: cha}
}

// HandleCreateConsignment handles POST /api/v1/consignments
// Stage 1 (two-stage): body { flow, chaId } → creates shell (INITIALIZED)
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

	traderID := authCtx.UserID
	globalContext, err := authCtx.GetUserContextMap()
	if err != nil {
		http.Error(w, "failed to parse user context", http.StatusInternalServerError)
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
		if err := json.NewEncoder(w).Encode(consignment); err != nil {
			slog.Error("failed to encode response for consignment", "error", err)
		}
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
	if err := json.NewEncoder(w).Encode(consignment); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HandleGetConsignments handles GET /api/v1/consignments
// Query params: role=trader | role=cha (defaults to trader).
// When role=cha the CHA is resolved from the authenticated user's email.
// Pagination: offset, limit. Optional filters: state, flow.
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

	userID := authCtx.UserID

	if role != "cha" {
		filter.TraderID = &userID
	} else {
		// Handle CHA role: URL param takes priority
		if chaIDStr := r.URL.Query().Get("cha_id"); chaIDStr != "" {
			chaID, err := uuid.Parse(chaIDStr)
			if err != nil {
				http.Error(w, "invalid cha_id format", http.StatusBadRequest)
				return
			}
			filter.ChaID = &chaID
		} else {
			// Fallback: Resolve the specific CHA Entity from the DB
			cha, err := c.cha.GetCHAByEmail(r.Context(), userID)
			if err != nil {
				http.Error(w, "failed to resolve CHA", http.StatusForbidden)
				return
			}
			filter.ChaID = &cha.ID
		}
	}

	consignments, err := c.cs.ListConsignments(r.Context(), filter)
	if err != nil {
		http.Error(w, "failed to retrieve consignments: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(consignments); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HandleInitializeConsignment handles PUT /api/v1/consignments/{id} (Stage 2: CHA selects HS Codes).
// Body: InitializeConsignmentDTO { hsCodeIds: []uuid }. Response: ConsignmentDetailDTO.
func (c *ConsignmentRouter) HandleInitializeConsignment(w http.ResponseWriter, r *http.Request) {
	authCtx := auth.GetAuthContext(r.Context())
	if authCtx == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

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
	// TODO: Need to Call GetConsignmentByID and check whether the Consignment.ChaID and authContext.UserID are equal
	// Otherwise Forbidden
	var req model.InitializeConsignmentDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.HSCodeIDs) == 0 {
		http.Error(w, "hsCodeIds must contain at least one ID", http.StatusBadRequest)
		return
	}

	consignment, _, err := c.cs.InitializeConsignmentByID(r.Context(), consignmentID, req.HSCodeIDs)
	if err != nil {
		http.Error(w, "failed to initialize consignment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(consignment); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
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
