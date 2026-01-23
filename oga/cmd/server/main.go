package main

import (
	"context"
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
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081" // Default OGA port
	}

	// Create OGA service and handler
	ogaService := oga.NewOGAService() // In-memory service for MVP
	ogaHandler := oga.NewOGAHandler(ogaService)

	// Set up HTTP routes
	mux := http.NewServeMux()
	
	// Notifications from Task Manager
	mux.HandleFunc("POST /api/oga/notifications", ogaHandler.HandleNotification)
	mux.HandleFunc("POST /api/oga/tasks/{taskId}/completed", ogaHandler.HandleTaskCompleted)
	
	// Endpoints for OGA Portal (frontend)
	mux.HandleFunc("GET /api/oga/applications", ogaHandler.HandleGetApplications)
	mux.HandleFunc("GET /api/oga/applications/{taskId}", ogaHandler.HandleGetApplication)

	serverAddr := ":" + port
	
	// Simple CORS middleware for development
	corsHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	server := &http.Server{
		Addr:    serverAddr,
		Handler: corsHandler(mux),
	}

	// Channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		slog.Info("OGA server starting", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start OGA server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	slog.Info("shutting down OGA server...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("OGA server forced to shutdown", "error", err)
	} else {
		slog.Info("OGA server gracefully stopped")
	}
}
