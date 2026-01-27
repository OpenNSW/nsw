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

	"github.com/OpenNSW/nsw/oga"
)

func main() {
	// Load configuration from environment variables
	dbPath := os.Getenv("OGA_DB_PATH")
	if dbPath == "" {
		dbPath = "./oga_applications.db"
	}

	port := os.Getenv("OGA_PORT")
	if port == "" {
		port = "8081"
	}

	slog.Info("OGA service configuration",
		"db_path", dbPath,
		"port", port,
	)

	// Initialize database store
	store, err := oga.NewApplicationStore(dbPath)
	if err != nil {
		log.Fatalf("failed to create application store: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			slog.Error("failed to close database", "error", err)
		}
	}()

	// Initialize OGA service
	service := oga.NewOGAService(store)
	defer func() {
		if err := service.Close(); err != nil {
			slog.Error("failed to close service", "error", err)
		}
	}()

	// Initialize handler
	handler := oga.NewOGAHandler(service)

	// Set up HTTP routes
	mux := http.NewServeMux()
	// Health check
	mux.HandleFunc("GET /health", handler.HandleHealth)
	// Endpoint for services to inject data
	mux.HandleFunc("POST /api/oga/inject", handler.HandleInjectData)
	// Endpoints for UI to fetch and manage applications
	mux.HandleFunc("GET /api/oga/applications", handler.HandleGetApplications)
	mux.HandleFunc("GET /api/oga/applications/{taskId}", handler.HandleGetApplication)
	mux.HandleFunc("POST /api/oga/applications/{taskId}/review", handler.HandleReviewApplication)

	// Set up graceful shutdown
	serverAddr := fmt.Sprintf(":%s", port)

	// CORS middleware for development
	corsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		mux.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:    serverAddr,
		Handler: corsHandler,
	}

	// Channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		slog.Info("starting OGA service", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to start server", "error", err)
			quit <- syscall.SIGTERM
		}
	}()

	// Wait for interrupt signal
	<-quit
	slog.Info("shutting down OGA service...")

	// Create a context with timeout for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Attempt graceful shutdown of HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	} else {
		slog.Info("server gracefully stopped")
	}

	slog.Info("OGA service stopped")
}
