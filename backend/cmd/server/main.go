package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/database"
	"github.com/OpenNSW/nsw/internal/task"
	"github.com/OpenNSW/nsw/internal/workflow"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

const ChannelSize = 100

func main() {
	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	slog.Info("configuration loaded successfully",
		"db_host", cfg.Database.Host,
		"db_port", cfg.Database.Port,
		"db_name", cfg.Database.Name,
		"db_sslmode", cfg.Database.SSLMode,
		"server_port", cfg.Server.Port,
	)

	// Initialize database connection
	db, err := database.New(&cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer func() {
		if err := database.Close(db); err != nil {
			slog.Error("failed to close database", "error", err)
		}
	}()

	// Perform health check
	if err := database.HealthCheck(db); err != nil {
		log.Fatalf("database health check failed: %v", err)
	}

	// Create task completion notification channel
	ch := make(chan model.TaskCompletionNotification, ChannelSize)

	// Initialize task manager (still using SQLite for now)
	// TODO: Migrate task manager to use PostgreSQL
	tm, err := task.NewTaskManager("./taskmanager.db", ch)
	if err != nil {
		log.Fatalf("failed to create task manager: %v", err)
	}
	defer func() {
		if err := tm.Close(); err != nil {
			slog.Error("failed to close task manager", "error", err)
		}
	}()

	// Initialize workflow manager with database connection
	wm := workflow.NewManager(tm, ch, db)
	slog.Info("starting task update listener...")
	wm.StartTaskUpdateListener()
	slog.Info("task update listener started")

	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/tasks", tm.HandleExecuteTask)
	mux.HandleFunc("GET /api/hscodes", wm.HandleGetHSCodes)
	mux.HandleFunc("GET /api/hscodes/", wm.HandleGetHSCodes)
	mux.HandleFunc("GET /api/workflow-template", wm.HandleGetWorkflowTemplate)
	mux.HandleFunc("POST /api/consignments", wm.HandleCreateConsignment)
	mux.HandleFunc("GET /api/consignments/{consignmentID}", wm.HandleGetConsignment)

	// Set up graceful shutdown
	serverAddr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := &http.Server{
		Addr:    serverAddr,
		Handler: mux,
	}

	// Channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		slog.Info("starting server", "port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	slog.Info("shutting down server...")

	// Graceful shutdown would go here if needed
	slog.Info("server stopped")
}
