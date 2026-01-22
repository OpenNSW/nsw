package main

import (
	"net/http"

	"github.com/OpenNSW/nsw/internal/task"
)

func main() {

	tm := task.NewTaskManager()

	wm := workflow.NewWorkflowManager(tm)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/tasks", tm.HandleExecuteTask)

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		return
	}
}
