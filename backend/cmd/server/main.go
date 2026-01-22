package main

import (
	"log"
	"net/http"

	"github.com/OpenNSW/nsw/internal/task"
)

func main() {

	ch := make(chan task.TaskCompletionNotification)

	tm := task.NewTaskManager(ch)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/tasks", tm.HandleExecuteTask)

	err := http.ListenAndServe(":8080", mux)
	if err != nil {

		log.Fatalf("failed to start server: %v", err)
	}
}
