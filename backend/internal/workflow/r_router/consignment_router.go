package r_router

import (
	"encoding/json"
	"net/http"

	"github.com/OpenNSW/nsw/internal/workflow/r_model"
	"github.com/OpenNSW/nsw/internal/workflow/r_service"
)

// WorkflowNodeCallback is a callback function to register workflow nodes with the manager
type WorkflowNodeCallback func(workflowNodes []r_model.WorkflowNode, consignmentGlobalContext map[string]interface{})

type ConsignmentRouter struct {
	cs                      *r_service.ConsignmentService
	registerWorkflowNodesCb WorkflowNodeCallback
}

func NewConsignmentRouter(cs *r_service.ConsignmentService, registerWorkflowNodesCb WorkflowNodeCallback) *ConsignmentRouter {
	return &ConsignmentRouter{
		cs:                      cs,
		registerWorkflowNodesCb: registerWorkflowNodesCb,
	}
}

// HandleCreateConsignment handles POST /api/v1/consignments
// Request body: CreateConsignmentDTO
// Response: ConsignmentResponseDTO
func (c *ConsignmentRouter) HandleCreateConsignment(w http.ResponseWriter, r *http.Request) {
	var req r_model.CreateConsignmentDTO

	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: Get trader ID from auth context
	// For now, if traderId is not provided in the request, use a default
	if req.TraderID == nil {
		defaultTraderID := "trader-123"
		req.TraderID = &defaultTraderID
	}

	// Initialize global context if nil
	if req.GlobalContext == nil {
		req.GlobalContext = make(map[string]interface{})
	}

	// Create consignment through service
	consignment, newReadyNodes, err := c.cs.InitializeConsignment(r.Context(), &req)
	if err != nil {
		http.Error(w, "failed to create consignment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Register newly ready workflow nodes with the manager (via callback)
	if len(newReadyNodes) > 0 && c.registerWorkflowNodesCb != nil {
		c.registerWorkflowNodesCb(newReadyNodes, consignment.GlobalContext)
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(consignment); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
