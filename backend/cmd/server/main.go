package main

import (
	"log"
	"net/http"

	"github.com/OpenNSW/nsw/internal/task"
	"github.com/OpenNSW/nsw/internal/workflow"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

func main() {

	CHANNEL_SIZE := 100

	ch := make(chan model.TaskCompletionNotification, CHANNEL_SIZE)

	tm := task.NewTaskManager(ch)

	wm := workflow.NewManager(tm, ch, nil) // Pass actual *gorm.DB instance here
	wm.StartTaskUpdateListener()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/tasks", tm.HandleExecuteTask)
	mux.HandleFunc("GET /api/workflow-template", wm.HandleGetWorkflowTemplate)
	mux.HandleFunc("POST /api/consignments", wm.HandleCreateConsignment)
	mux.HandleFunc("GET /api/consignments/{consignmentID}", wm.HandleGetConsignment)

	err := http.ListenAndServe(":8080", mux)
	if err != nil {

		log.Fatalf("failed to start server: %v", err)
	}
}
