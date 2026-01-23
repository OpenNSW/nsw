package router

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/service"
	"github.com/google/uuid"
)

type WorkflowRouter struct {
	cs *service.ConsignmentService
}

func NewWorkflowRouter(cs *service.ConsignmentService) *WorkflowRouter {
	return &WorkflowRouter{
		cs: cs,
	}
}

// HandleGetWorkflowTemplate handles GET /api/workflow-template requests
// Query params: hscode, type
func (wr *WorkflowRouter) HandleGetWorkflowTemplate(w http.ResponseWriter, r *http.Request) {
	hscode := r.URL.Query().Get("hscode")
	consignmentType := model.ConsignmentType(r.URL.Query().Get("type"))

	if hscode == "" {
		http.Error(w, "missing required query parameter: hscode", http.StatusBadRequest)
		return
	}
	if consignmentType == "" {
		http.Error(w, "missing required query parameter: type", http.StatusBadRequest)
		return
	}

	template, err := wr.cs.GetWorkFlowTemplate(r.Context(), hscode, consignmentType)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get workflow template: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(template); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// HandleCreateConsignment handles POST /api/consignments requests
func (wr *WorkflowRouter) HandleCreateConsignment(w http.ResponseWriter, r *http.Request) {
	var createReq model.CreateConsignmentDTO
	if err := json.NewDecoder(r.Body).Decode(&createReq); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	consignment, err := wr.cs.InitializeConsignment(r.Context(), &createReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create consignment: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(consignment); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// HandleGetConsignment handles GET /api/consignments/{consignmentID} requests
func (wr *WorkflowRouter) HandleGetConsignment(w http.ResponseWriter, r *http.Request) {
	consignmentIDStr := r.PathValue("consignmentID")
	if consignmentIDStr == "" {
		http.Error(w, "missing consignmentID in path", http.StatusBadRequest)
		return
	}

	consignmentID, err := uuid.Parse(consignmentIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid consignmentID: %v", err), http.StatusBadRequest)
		return
	}

	consignment, err := wr.cs.GetConsignmentByID(r.Context(), consignmentID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get consignment: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(consignment); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}
