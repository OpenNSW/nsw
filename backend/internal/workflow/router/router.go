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
	onTasksReadyFunc func(tasks []*model.Task, consignmentGlobalContext map[string]interface{}) // Callback to register ready tasks
}

func NewWorkflowRouter(cs *service.ConsignmentService, onTasksReadyFunc func(tasks []*model.Task, consignmentGlobalContext map[string]interface{})) *WorkflowRouter {
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
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "invalid 'limit' query parameter, must be an integer", http.StatusBadRequest)
			return
		}
		filter.Limit = &limit
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			http.Error(w, "invalid 'offset' query parameter, must be an integer", http.StatusBadRequest)
			return
		}
		filter.Offset = &offset
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

// HandleGetHSCodeID handles GET /api/hscodes/{hsCodeId} requests
func (wr *WorkflowRouter) HandleGetHSCodeID(w http.ResponseWriter, r *http.Request) {
	hsCodeIDStr := r.PathValue("hsCodeId")
	if hsCodeIDStr == "" {
		http.Error(w, "missing hsCodeId in path", http.StatusBadRequest)
		return
	}

	hsCodeID, err := uuid.Parse(hsCodeIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid hsCodeId: %v", err), http.StatusBadRequest)
		return
	}

	hsCode, err := wr.cs.GetHSCodeByID(r.Context(), hsCodeID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get HS code: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(hsCode); err != nil {
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

	// TODO: TraderID should be obtained from authenticated user context
	// For now, set it to a default value
	defaultTraderID := "trader-123"
	createReq.TraderID = &defaultTraderID

	consignment, readyTasks, err := wr.cs.InitializeConsignment(r.Context(), &createReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create consignment: %v", err), http.StatusInternalServerError)
		return
	}

	// Push ready tasks to Task Manager
	if wr.onTasksReadyFunc != nil && len(readyTasks) > 0 {
		wr.onTasksReadyFunc(readyTasks, consignment.GlobalContext)
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

// HandleGetConsignments handles GET /api/consignments?traderId={traderId}&offset={offset}&limit={limit} requests
// TradeID is required
// Optional Query Params: offset, limit
func (wr *WorkflowRouter) HandleGetConsignments(w http.ResponseWriter, r *http.Request) {
	traderID := r.URL.Query().Get("traderId")

	// If traderID is provided, should validate the Authenticated user has access to it
	// TODO: Implement authentication and authorization

	// If TraderID is not provided, return as bad request for now
	if traderID == "" {
		http.Error(w, "missing traderId query parameter", http.StatusBadRequest)
		return
	}

	var filter model.ConsignmentTraderFilter
	filter.TraderID = traderID

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "invalid 'limit' query parameter, must be an integer", http.StatusBadRequest)
			return
		}
		filter.Limit = &limit
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			http.Error(w, "invalid 'offset' query parameter, must be an integer", http.StatusBadRequest)
			return
		}
		filter.Offset = &offset
	}

	consignments, err := wr.cs.GetConsignmentsByTraderID(r.Context(), filter)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get consignments: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(consignments); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}
