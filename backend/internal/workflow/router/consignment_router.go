package router

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/OpenNSW/nsw/internal/auth"
	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/service"
	"github.com/OpenNSW/nsw/utils"
)

type ConsignmentRouter struct {
	cs  *service.ConsignmentService
	cha *service.CHAService
	cfg *config.AuthConfig
}

func NewConsignmentRouter(cs *service.ConsignmentService, cha *service.CHAService, cfg *config.AuthConfig) *ConsignmentRouter {
	return &ConsignmentRouter{cs: cs, cha: cha, cfg: cfg}
}

// HandleCreateConsignment handles POST /api/v1/consignments
// Stage 1 (two-stage): body { flow, chaId } → creates shell (INITIALIZED)
// Legacy: body { flow, items } → creates and initializes workflow
func (c *ConsignmentRouter) HandleCreateConsignment(w http.ResponseWriter, r *http.Request) {
	authCtx := auth.GetAuthContext(r.Context())
	if authCtx == nil || authCtx.IsM2M || authCtx.UserID == nil {
		http.Error(w, "Unauthorized: Human context required", http.StatusUnauthorized)
		return
	}

	// RBAC: Only Traders can create consignments
	if c.cfg != nil && !authCtx.HasGroup(c.cfg.TraderGroup) {
		http.Error(w, "Forbidden: Only Traders can create consignments", http.StatusForbidden)
		return
	}

	var req model.CreateConsignmentDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	traderID := *authCtx.UserID
	// Stage 1: create shell only
	consignment, err := c.cs.CreateConsignmentShell(r.Context(), req.Flow, req.ChaID, traderID)
	if err != nil {
		http.Error(w, "failed to create consignment: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(consignment); err != nil {
		slog.Error("failed to encode response for consignment", "error", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// HandleGetConsignments handles GET /api/v1/consignments
// Query params: role=trader | role=cha (defaults to trader).
// When role=cha the CHA is resolved from the authenticated user's email.
// Pagination: offset, limit. Optional filters: state, flow.
func (c *ConsignmentRouter) HandleGetConsignments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := auth.GetAuthContext(ctx)
	if authCtx == nil || authCtx.IsM2M || authCtx.UserID == nil {
		http.Error(w, "Unauthorized: Human context required", http.StatusUnauthorized)
		return
	}


	role := r.URL.Query().Get("role")
	if role == "" {
		role = "trader"
	}

	// RBAC: Authorize the requested role against the token's groups
	isAuthorized := false
	if c.cfg != nil {
		if role == "trader" && authCtx.HasGroup(c.cfg.TraderGroup) {
			isAuthorized = true
		} else if role == "cha" && authCtx.HasGroup(c.cfg.CHAGroup) {
			isAuthorized = true
		}
	} else {
		// If config is missing, default to lenient for now (or strict if preferred)
		isAuthorized = true
	}

	if !isAuthorized {
		http.Error(w, "Forbidden: Insufficient privileges for requested role", http.StatusForbidden)
		return
	}
	offset, limit, err := utils.ParsePaginationParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	filter := model.ConsignmentFilter{
		Offset: offset,
		Limit:  limit,
	}

	// Optional Filters
	if stateStr := r.URL.Query().Get("state"); stateStr != "" {
		state := model.ConsignmentState(stateStr)
		filter.State = &state
	}
	if flowStr := r.URL.Query().Get("flow"); flowStr != "" {
		flow := model.ConsignmentFlow(flowStr)
		filter.Flow = &flow
	}

	// Role-Based Identity Resolution
	switch role {
	case "cha":
		// Check if UI sent a specific ID to filter by
		if chaIDStr := r.URL.Query().Get("cha_id"); chaIDStr != "" {
			filter.ChaID = &chaIDStr
		} else {
			// Fallback: Default to the user's own profile if no specific ID requested
			cha, err := c.cha.GetCHAByEmail(ctx, authCtx.Email)
			if err != nil {
				http.Error(w, "failed to resolve default CHA profile", http.StatusForbidden)
				return
			}
			filter.ChaID = &cha.ID
		}
	case "trader":
		filter.TraderID = authCtx.UserID
	default:
		http.Error(w, "query param role must be trader or cha", http.StatusBadRequest)
		return
	}
	consignments, err := c.cs.ListConsignments(ctx, filter)
	if err != nil {
		http.Error(w, "failed to retrieve consignments", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(consignments); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// HandleInitializeConsignment handles PUT /api/v1/consignments/{id} (Stage 2: CHA selects HS Codes).
// Body: InitializeConsignmentDTO { hsCodeIds: []uuid }. Response: ConsignmentDetailDTO.
func (c *ConsignmentRouter) HandleInitializeConsignment(w http.ResponseWriter, r *http.Request) {
	authCtx := auth.GetAuthContext(r.Context())
	if authCtx == nil || authCtx.IsM2M || authCtx.UserID == nil {
		http.Error(w, "Unauthorized: Human context required", http.StatusUnauthorized)
		return
	}

	// RBAC: Only CHAs can initialize consignments
	if c.cfg != nil && !authCtx.HasGroup(c.cfg.CHAGroup) {
		http.Error(w, "Forbidden: Only CHAs can initialize consignments", http.StatusForbidden)
		return
	}

	consignmentIDStr := r.PathValue("id")
	if consignmentIDStr == "" {
		http.Error(w, "consignment ID is required", http.StatusBadRequest)
		return
	}
	consignmentID := consignmentIDStr

	// Ownership check: Fetch consignment and verify CHA ownership
	existing, err := c.cs.GetConsignmentByID(r.Context(), consignmentID)
	if err != nil {
		http.Error(w, "consignment not found", http.StatusNotFound)
		return
	}

	// Resolve the current user's CHA profile
	userCHA, err := c.cha.GetCHAByEmail(r.Context(), *authCtx.UserID)
	if err != nil {
		http.Error(w, "failed to resolve CHA profile", http.StatusForbidden)
		return
	}

	if existing.ChaID != userCHA.ID {
		http.Error(w, "Forbidden: You do not own this consignment", http.StatusForbidden)
		return
	}
	var req model.InitializeConsignmentDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.HSCodeIDs) == 0 {
		http.Error(w, "hsCodeIds must contain at least one ID", http.StatusBadRequest)
		return
	}

	globalContext, err := authCtx.GetUserContextMap()
	if err != nil {
		http.Error(w, "failed to parse user context", http.StatusInternalServerError)
		return
	}

	consignment, err := c.cs.InitializeConsignmentByID(r.Context(), consignmentID, req.HSCodeIDs, globalContext)
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
	consignmentID := consignmentIDStr

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
