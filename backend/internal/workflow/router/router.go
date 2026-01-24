package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/service"
	"github.com/google/uuid"
)

type WorkflowRouter struct {
	cs               *service.ConsignmentService
	onTasksReadyFunc func(tasks []*model.Task) // Callback to register ready tasks
}

func NewWorkflowRouter(cs *service.ConsignmentService, onTasksReadyFunc func(tasks []*model.Task)) *WorkflowRouter {
	return &WorkflowRouter{
		cs:               cs,
		onTasksReadyFunc: onTasksReadyFunc,
	}
}

// HandleGetHSCodes handles GET /api/hscodes requests
// Optional Query Filters: offset, limit, hsCode
func (wr *WorkflowRouter) HandleGetHSCodes(w http.ResponseWriter, r *http.Request) {
	var filter model.HSCodeFilter

	if hsCodeStartsWith := r.URL.Query().Get("hsCodeStartsWith"); hsCodeStartsWith != "" {
		filter.HSCodeStartsWith = &hsCodeStartsWith
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = &limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = &offset
		}
	}

	hsCodes, err := wr.cs.GetAllHSCodes(r.Context(), filter)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get HS codes: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(hsCodes); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// HandleGetWorkflowTemplate handles GET /api/workflow-template requests
// Query params: (hsCode or hsCodeId), type
func (wr *WorkflowRouter) HandleGetWorkflowTemplate(w http.ResponseWriter, r *http.Request) {
	hsCode := r.URL.Query().Get("hsCode")
	hsCodeID := r.URL.Query().Get("hsCodeId")
	tradeFlow := model.TradeFlow(r.URL.Query().Get("tradeFlow"))

	var hsCodeIDPtr *uuid.UUID
	if hsCodeID != "" {
		parsedID, err := uuid.Parse(hsCodeID)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid hscodeId: %v", err), http.StatusBadRequest)
			return
		}
		hsCodeIDPtr = &parsedID
	}

	if hsCode == "" && hsCodeIDPtr == nil {
		http.Error(w, "missing required query parameter: hsCode or hsCodeId", http.StatusBadRequest)
		return
	}
	if tradeFlow == "" {
		http.Error(w, "missing required query parameter: tradeFlow", http.StatusBadRequest)
		return
	}

	template, err := wr.cs.GetWorkFlowTemplate(r.Context(), &hsCode, hsCodeIDPtr, tradeFlow)
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

	consignment, readyTasks, err := wr.cs.InitializeConsignment(r.Context(), &createReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create consignment: %v", err), http.StatusInternalServerError)
		return
	}

	// Push ready tasks to Task Manager
	if wr.onTasksReadyFunc != nil && len(readyTasks) > 0 {
		wr.onTasksReadyFunc(readyTasks)
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
