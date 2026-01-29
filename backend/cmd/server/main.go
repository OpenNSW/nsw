package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/database"
	"github.com/OpenNSW/nsw/internal/form"
	"github.com/OpenNSW/nsw/internal/middleware"
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
	)

	slog.Info("CORS configuration",
		"allowed_origins", cfg.CORS.AllowedOrigins,
		"allowed_methods", cfg.CORS.AllowedMethods,
		"allowed_headers", cfg.CORS.AllowedHeaders,
		"allow_credentials", cfg.CORS.AllowCredentials,
		"max_age", cfg.CORS.MaxAge,
	)

	slog.Info("server configuration",
		"port", cfg.Server.Port,
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

	// Initialize form service
	formService := form.NewFormService(db)

	// Initialize task manager with database connection
	tm, err := task.NewTaskManager(db, ch, cfg, formService)
	if err != nil {
		log.Fatalf("failed to create task manager: %v", err)
	}

	// Initialize workflow manager with database connection
	wm := workflow.NewManager(tm, ch, db)
	slog.Info("starting task update listener...")
	wm.StartTaskUpdateListener()
	slog.Info("task update listener started")

	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/tasks", tm.HandleExecuteTask)
	mux.HandleFunc("GET /api/hscodes", wm.HandleGetHSCodes)
	mux.HandleFunc("GET /api/hscodes/{hsCodeId}", wm.HandleGetHSCodeID)
	mux.HandleFunc("GET /api/workflows/templates", wm.HandleGetWorkflowTemplate)
	mux.HandleFunc("POST /api/consignments", wm.HandleCreateConsignment)
	mux.HandleFunc("GET /api/consignments", wm.HandleGetConsignments)
	mux.HandleFunc("GET /api/consignments/{consignmentID}", wm.HandleGetConsignment)

	// Set up graceful shutdown
	serverAddr := fmt.Sprintf(":%d", cfg.Server.Port)

	// Wrap handler with CORS middleware
	handler := middleware.CORS(&cfg.CORS)(mux)

	server := &http.Server{
		Addr:    serverAddr,
		Handler: handler,
	}

	// Channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		slog.Info("starting server", "port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to start server", "error", err)
			quit <- syscall.SIGTERM
		}
	}()

	// Wait for interrupt signal
	<-quit
	slog.Info("shutting down server...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown of HTTP server
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	} else {
		slog.Info("server gracefully stopped")
	}

	// Stop the workflow manager's task update listener
	slog.Info("stopping task update listener...")
	wm.StopTaskUpdateListener()

	slog.Info("server stopped")
}
