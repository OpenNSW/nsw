package main

import (
	"log"
	"net/http"

	"github.com/OpenNSW/nsw/internal/task"
	"github.com/OpenNSW/nsw/internal/workflow"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

const ChannelSize = 100
const DbPath = "./taskmanager.db"

func main() {

	ch := make(chan model.TaskCompletionNotification, ChannelSize)

	tm, err := task.NewTaskManager(DbPath, ch)

	if err != nil {
		log.Fatalf("failed to create task manager: %v", err)
	}

	defer func(tm task.TaskManager) {
		err := tm.Close()
		if err != nil {
			log.Fatalf("failed to close task manager: %v", err)
		}
	}(tm)

	wm := workflow.NewManager(tm, ch, nil)
	log.Println("Starting task update listener...")
	wm.StartTaskUpdateListener()
	log.Println("Task update listener started")

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/tasks", tm.HandleExecuteTask)
	mux.HandleFunc("GET /api/workflow-template", wm.HandleGetWorkflowTemplate)
	mux.HandleFunc("POST /api/consignments", wm.HandleCreateConsignment)
	mux.HandleFunc("GET /api/consignments/{consignmentID}", wm.HandleGetConsignment)

	err = http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
