package router

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/OpenNSW/nsw-task-flow/orchestrator"
	"github.com/OpenNSW/nsw-task-flow/store"
)

// ConsignmentTaskRow is a thin projection of a TaskRecord, suitable for the
// trader-app's consignment-detail view. The UI can use these to navigate from
// a consignment (parent workflow) to the dynamic task ID nsw-task-flow generated
// for the currently active sub-task step.
type ConsignmentTaskRow struct {
	TaskID                string `json:"taskId"`
	ParentWorkflowID      string `json:"parentWorkflowId"`
	ParentNodeID          string `json:"parentNodeId"`
	TaskWorkflowID        string `json:"taskWorkflowId"`
	SubTaskNodeID         string `json:"subtaskNodeId"`
	ActiveTaskTemplateID  string `json:"activeTaskTemplateId"`
	PluginName            string `json:"pluginName"`
	Status                string `json:"status"`      // raw nsw-task-flow status
	UIType                string `json:"uiType"`      // SIMPLE_FORM | WAIT_FOR_EVENT | PAYMENT
	State                 string `json:"state"`       // adapted task state
	PluginState           string `json:"pluginState"` // adapted FSM-style plugin state
	IsCurrentlyActionable bool   `json:"isCurrentlyActionable"`
	CreatedAt             string `json:"createdAt"`
}

// ConsignmentTasksRouter exposes GET /api/v1/consignments/{id}/tasks.
type ConsignmentTasksRouter struct {
	tm       *orchestrator.TaskManager
	registry *orchestrator.TaskTemplateRegistry
}

func NewConsignmentTasksRouter(tm *orchestrator.TaskManager, registry *orchestrator.TaskTemplateRegistry) *ConsignmentTasksRouter {
	return &ConsignmentTasksRouter{tm: tm, registry: registry}
}

// HandleListConsignmentTasks lists every TaskRecord whose parent_workflow_id
// matches the URL's consignment ID. Returns the same TaskRow shape regardless
// of plugin so the UI can pick the currently-actionable one.
//
// Optional `?activeOnly=1` filters out completed tasks so the UI only sees
// what still needs trader interaction.
func (c *ConsignmentTasksRouter) HandleListConsignmentTasks(w http.ResponseWriter, r *http.Request) {
	consignmentID := r.PathValue("id")
	if consignmentID == "" {
		writeError(w, http.StatusBadRequest, "consignment id is required")
		return
	}
	activeOnly := r.URL.Query().Get("activeOnly") == "1"

	all := c.tm.GetAllTasks()
	out := make([]ConsignmentTaskRow, 0)
	for _, rec := range all {
		if rec.ParentWorkflowID != consignmentID {
			continue
		}
		if activeOnly && (rec.Status == "COMPLETED" || rec.Status == "DISPATCHED") {
			continue
		}
		out = append(out, c.buildRow(rec))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })

	writeJSON(w, http.StatusOK, map[string]any{
		"consignmentId": consignmentID,
		"items":         out,
		"total":         len(out),
	})
}

func (c *ConsignmentTasksRouter) buildRow(rec store.TaskRecord) ConsignmentTaskRow {
	regEntry, _ := c.registry.Get(rec.ActiveTaskTemplateID)
	state, pluginState := mapStatus(rec.Status, regEntry.PluginName)
	return ConsignmentTaskRow{
		TaskID:                rec.TaskID,
		ParentWorkflowID:      rec.ParentWorkflowID,
		ParentNodeID:          rec.ParentNodeID,
		TaskWorkflowID:        rec.TaskWorkflowID,
		SubTaskNodeID:         rec.SubTaskNodeID,
		ActiveTaskTemplateID:  rec.ActiveTaskTemplateID,
		PluginName:            regEntry.PluginName,
		Status:                rec.Status,
		UIType:                pluginToUIType(regEntry.PluginName),
		State:                 state,
		PluginState:           pluginState,
		IsCurrentlyActionable: isActionable(rec.Status, regEntry.PluginName),
		CreatedAt:             rec.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// pluginToUIType matches what adaptToTraderShape returns; centralised here so
// the consignment-tasks endpoint and the GET /tasks/{id} endpoint agree.
func pluginToUIType(plugin string) string {
	switch plugin {
	case "generic_user_input", "generic_officer_input":
		return "SIMPLE_FORM"
	case "generic_external_review", "register_task_and_wait":
		return "WAIT_FOR_EVENT"
	case "generic_payment":
		return "PAYMENT"
	case "generic_api_call":
		return "FIRE_AND_FORGET"
	default:
		return strings.ToUpper(plugin)
	}
}

// isActionable returns true when the trader is the one expected to act next.
// (QUEUED_EXTERNALLY tasks belong to the OGA officer, not the trader.)
func isActionable(status, plugin string) bool {
	if plugin == "generic_external_review" || plugin == "generic_officer_input" {
		// These steps belong to OGA; trader can't progress them.
		return false
	}
	switch status {
	case "PENDING_USER", "PENDING_PAYMENT":
		return true
	default:
		return false
	}
}
