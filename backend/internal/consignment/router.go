package consignment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/LSFLK/argus/pkg/audit",
	"github.com/OpenNSW/nsw/backend/internal/auth",
	"github.com/OpenNSW/nsw/backend/internal/cha",
	"github.com/OpenNSW/nsw/backend/internal/company",
	"github.com/OpenNSW/nsw/backend/internal/profile/cha",
	"github.com/OpenNSW/nsw/backend/internal/profile/company",
	"github.com/OpenNSW/nsw/utils"
)

type Router struct {
	cs          *Service
	cha         cha.Service
	company     company.Service
	auditClient *audit.Client
}

func NewRouter(cs *Service, chaService cha.Service, companyService company.Service, auditClient *audit.Client) *Router {
	return &Router{
		cs:          cs,
		cha:         chaService,
		company:     companyService,
		auditClient: auditClient,
	}
}

func (c *Router) extractActor(r *http.Request) (string, string) {
	actorID := "SYSTEM"
	actorType := "SYSTEM"
	if authCtx := auth.GetAuthContext(r.Context()); authCtx != nil {
		if authCtx.User != nil {
			actorID = authCtx.User.ID
			actorType = "USER"
		} else if authCtx.Client != nil {
			actorID = authCtx.Client.ClientID
			actorType = "SYSTEM"
		}
	}
	return actorID, actorType
}

// HandleCreateConsignment handles POST /api/v1/consignments
// Stage 1 (two-stage): body { flow, chaId } → creates shell (INITIALIZED)
// Legacy: body { flow, items } → creates and initializes workflow
func (c *Router) HandleCreateConsignment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := auth.GetAuthContext(ctx)
	if authCtx == nil || authCtx.User == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	defer func() { _ = r.Body.Close() }()

	var req CreateConsignmentDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	traderID := authCtx.User.ID
	// Stage 1: create shell only
	consignment, err := c.cs.CreateConsignmentShell(r.Context(), req.Flow, req.ChaCompanyID, traderID)

	// Fire the audit log asynchronously before returning the HTTP response.
	// Do not let any failure block or fail the actual API response to the user.
	if c.auditClient != nil {
		actorID, actorType := c.extractActor(r)

		var status string
		var msg string
		var targetID *string
		if err != nil {
			status = "FAILURE"
			msg = fmt.Sprintf("Failed to create consignment: %v", err)
		} else {
			status = "SUCCESS"
			msg = fmt.Sprintf("User created consignment shell %s", consignment.ID)
			targetID = &consignment.ID
		}

		metadata := map[string]any{
			"flow":           req.Flow,
			"cha_company_id": req.ChaCompanyID,
		}

		auditLog := audit.AuditLogRequest{
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			EventType:  "SYSTEM_EVENT",
			Action:     "CREATE_CONSIGNMENT",
			Status:     status,
			ActorID:    actorID,
			ActorType:  actorType,
			TargetID:   targetID,
			TargetType: "CONSIGNMENT",
			Message:    []byte(msg),
			Metadata:   metadata,
		}

		go c.auditClient.LogEvent(context.Background(), &auditLog)
	}

	if err != nil {
		if errors.Is(err, company.ErrCompanyNotFound) {
			http.Error(w, "CHA company not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrCompanyNotCHA) {
			http.Error(w, "selected company is not a CHA company", http.StatusBadRequest)
			return
		}
		slog.Error("failed to create consignment shell", "error", err)
		http.Error(w, "failed to create consignment: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(consignment); err != nil {
		slog.Error("failed to encode response for consignment", "error", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HandleGetConsignments handles GET /api/v1/consignments
// Query params: role=trader | role=cha (defaults to trader).
// When role=cha the CHA is resolved from the authenticated user's email.
// Pagination: offset, limit. Optional filters: state, flow.
func (c *Router) HandleGetConsignments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := auth.GetAuthContext(ctx)
	if authCtx == nil || authCtx.User == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// TODO: Proper AuthZ need to be implemented.
	role := r.URL.Query().Get("role")
	if role == "" {
		role = "trader"
	}
	offset, limit, err := utils.ParsePaginationParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	filter := Filter{
		Offset: offset,
		Limit:  limit,
	}

	// Optional Filters
	if stateStr := r.URL.Query().Get("state"); stateStr != "" {
		state := State(stateStr)
		filter.State = &state
	}
	if flowStr := r.URL.Query().Get("flow"); flowStr != "" {
		flow := Flow(flowStr)
		filter.Flow = &flow
	}

	// Role-Based Identity Resolution
	switch role {
	case "cha":
		userCompany, err := c.company.GetCompanyByOUHandle(ctx, authCtx.User.OUHandle)
		if err != nil {
			if errors.Is(err, company.ErrCompanyNotFound) {
				http.Error(w, "company profile not found for user", http.StatusForbidden)
				return
			}
			slog.Error("failed to resolve user company", "ouHandle", authCtx.User.OUHandle, "error", err)
			http.Error(w, "failed to resolve user company", http.StatusInternalServerError)
			return
		}
		filter.CHACompanyID = &userCompany.ID
	case "trader":
		userCompany, err := c.company.GetCompanyByOUHandle(ctx, authCtx.User.OUHandle)
		if err != nil {
			if errors.Is(err, company.ErrCompanyNotFound) {
				http.Error(w, "company profile not found for user", http.StatusForbidden)
				return
			}
			slog.Error("failed to resolve user company", "ouHandle", authCtx.User.OUHandle, "error", err)
			http.Error(w, "failed to resolve user company", http.StatusInternalServerError)
			return
		}
		filter.TraderCompanyID = &userCompany.ID
	default:
		http.Error(w, "query param role must be trader or cha", http.StatusBadRequest)
		return
	}
	consignments, err := c.cs.ListConsignments(ctx, filter)
	if err != nil {
		slog.Error("failed to retrieve consignments", "error", err)
		http.Error(w, "failed to retrieve consignments", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(consignments); err != nil {
		slog.Error("failed to encode response", "error", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HandleInitializeConsignment handles PUT /api/v1/consignments/{id} (Stage 2: CHA selects HS Codes).
// Body: InitializeConsignmentDTO { hsCodeIds: []uuid }. Response: DetailDTO.
func (c *Router) HandleInitializeConsignment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := auth.GetAuthContext(ctx)
	if authCtx == nil || authCtx.User == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	defer func() { _ = r.Body.Close() }()

	consignmentIDStr := r.PathValue("id")
	if consignmentIDStr == "" {
		http.Error(w, "consignment ID is required", http.StatusBadRequest)
		return
	}
	consignmentID := consignmentIDStr
	var req InitializeConsignmentDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.HSCodeIDs) == 0 {
		http.Error(w, "hsCodeIds must contain at least one ID", http.StatusBadRequest)
		return
	}

	// Resolve the CHA picking up the consignment from the authenticated user's email.
	chaRecord, err := c.cha.GetByEmail(ctx, authCtx.User.Email)
	if err != nil {
		if errors.Is(err, cha.ErrCHANotFound) {
			http.Error(w, "CHA profile not found for user", http.StatusForbidden)
			return
		}
		slog.Error("failed to resolve CHA profile", "email", authCtx.User.Email, "error", err)
		http.Error(w, "failed to resolve CHA profile", http.StatusInternalServerError)
		return
	}

	consignment, err := c.cs.InitializeConsignmentByID(r.Context(), consignmentID, req.HSCodeIDs, chaRecord.ID)
	if err != nil {
		if errors.Is(err, ErrCHACompanyMismatch) {
			http.Error(w, "CHA does not belong to the consignment's CHA company", http.StatusForbidden)
			return
		}
		slog.Error("failed to initialize consignment", "error", err)
		http.Error(w, "failed to initialize consignment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(consignment); err != nil {
		slog.Error("failed to encode response", "error", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HandleGetConsignmentByID handles GET /api/v1/consignments/{id}
// Path param: id (required)
// Response: DetailDTO
func (c *Router) HandleGetConsignmentByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := auth.GetAuthContext(ctx)
	if authCtx == nil || authCtx.User == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
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
		slog.Error("failed to retrieve consignment", "error", err)
		http.Error(w, "failed to retrieve consignment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(consignment); err != nil {
		slog.Error("failed to encode response", "error", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
