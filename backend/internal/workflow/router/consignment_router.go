package router

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/service"
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
	var req model.CreateConsignmentDTO

	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: Get trader ID from auth context
	// For now use a mock trader ID if not provided
	traderId := "TRADER-001"

	// TODO: Inital global context should be get from auth context or other sources. For now, we will use mock data.
	globalContext := map[string]any{
		"roc:br:br_no":   "PV 00234567",
		"ird:vat:vat_no": "114234222-7000",
		"ird:tin:tin_no": "114234222",
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

// HandleGetConsignmentsByTraderID handles GET /api/v1/consignments?traderId={traderId}
// Query params: traderId (required)
// Response: array of ConsignmentResponseDTO
func (c *ConsignmentRouter) HandleGetConsignmentsByTraderID(w http.ResponseWriter, r *http.Request) {
	// Get traderId from query params
	traderID := r.URL.Query().Get("traderId")
	if traderID == "" {
		http.Error(w, "traderId query parameter is required", http.StatusBadRequest)
		return
	}

	// Get consignments from service
	consignments, err := c.cs.GetConsignmentsByTraderID(r.Context(), traderID)
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
