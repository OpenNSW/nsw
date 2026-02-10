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
	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	"github.com/OpenNSW/nsw/internal/workflow"
)

// ChannelSize defines the buffer size for workflow node update notifications.
// A larger buffer (1000) prevents notification drops during high load scenarios.
const ChannelSize = 1000

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
	ch := make(chan taskManager.WorkflowManagerNotification, ChannelSize)

	// Initialize form service
	formService := form.NewFormService(db)

	// Initialize task manager with database connection
	tm, err := taskManager.NewTaskManager(db, ch, cfg, formService)
	if err != nil {
		log.Fatalf("failed to create task manager: %v", err)
	}

	// Initialize workflow manager with database connection
	wm := workflow.NewManager(tm, ch, db)

	// Set up HTTP routes
	mux := http.NewServeMux()

	// V1 API routes (new refactored architecture)
	mux.HandleFunc("POST /api/v1/tasks", tm.HandleExecuteTask)
	mux.HandleFunc("GET /api/v1/hscodes", wm.HandleGetAllHSCodes)
	mux.HandleFunc("POST /api/v1/consignments", wm.HandleCreateConsignment)
	mux.HandleFunc("GET /api/v1/consignments/{id}", wm.HandleGetConsignmentByID)
	mux.HandleFunc("GET /api/v1/consignments", wm.HandleGetConsignmentsByTraderID)

	// Pre-consignment routes
	mux.HandleFunc("POST /api/v1/pre-consignments", wm.HandleCreatePreConsignment)
	mux.HandleFunc("GET /api/v1/pre-consignments/{preConsignmentId}", wm.HandleGetPreConsignmentByID)
	mux.HandleFunc("GET /api/v1/pre-consignments", wm.HandleGetPreConsignmentsByTraderID)

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
	wm.StopWorkflowNodeUpdateListener()

	slog.Info("server stopped")
}
