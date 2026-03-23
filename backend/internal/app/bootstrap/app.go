package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/OpenNSW/nsw/internal/auth"
	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/database"
	"github.com/OpenNSW/nsw/internal/form"
	"github.com/OpenNSW/nsw/internal/middleware"
	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	"github.com/OpenNSW/nsw/internal/uploads"
	"github.com/OpenNSW/nsw/internal/uploads/drivers"
	workflowmanager "github.com/OpenNSW/nsw/internal/workflow/manager"
	"github.com/OpenNSW/nsw/internal/workflow/router"
	"github.com/OpenNSW/nsw/internal/workflow/service"
)

// App contains initialized HTTP server and cleanup hooks.
type App struct {
	Server *http.Server
	close  func() error
}

// Close releases resources initialized during bootstrap.
func (a *App) Close() error {
	if a == nil || a.close == nil {
		return nil
	}
	return a.close()
}

// healthResponse is the JSON shape returned by the health endpoint in all cases.
// UnhealthyComponents is omitted on success and populated with the names of all
// failing subsystems on failure.
type healthResponse struct {
	Status              string   `json:"status"`
	Service             string   `json:"service"`
	UnhealthyComponents []string `json:"unhealthy_components,omitempty"`
}

// writeJSON sets the Content-Type header, writes the status code, and encodes v as JSON.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Build initializes dependencies and returns a fully wired application server.
func Build(ctx context.Context, cfg *config.Config) (*App, error) {
	db, err := database.New(&cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := database.HealthCheck(db); err != nil {
		_ = database.Close(db)
		return nil, fmt.Errorf("database health check failed: %w", err)
	}

	formService := form.NewFormService(db)
	tm, err := taskManager.NewTaskManager(db, cfg, formService)
	if err != nil {
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to create task manager: %w", err)
	}

	nodeService := service.NewWorkflowNodeService(db)
	templateService := service.NewTemplateService(db)
	wm := workflowmanager.NewManager(db, nodeService, templateService)

	chaService := service.NewCHAService(db)
	hsCodeService := service.NewHSCodeService(db)
	consignmentService := service.NewConsignmentService(db, templateService, wm)
	preConsignmentService := service.NewPreConsignmentService(db, templateService, wm)

	if err := WireManagers(wm, tm); err != nil {
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to wire managers: %w", err)
	}

	hsCodeRouter := router.NewHSCodeRouter(hsCodeService)
	chaRouter := router.NewCHARouter(chaService)
	consignmentRouter := router.NewConsignmentRouter(consignmentService, chaService)
	preConsignmentRouter := router.NewPreConsignmentRouter(preConsignmentService)

	storageDriver, err := uploads.NewStorageFromConfig(ctx, cfg.Storage)
	if err != nil {
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}
	uploadService := uploads.NewUploadService(storageDriver)
	uploadHandler := uploads.NewHTTPHandler(uploadService)

	authManager, err := auth.NewManager(db, cfg.Auth)
	if err != nil {
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to create auth manager: %w", err)
	}

	if err := authManager.Health(); err != nil {
		_ = authManager.Close()
		_ = database.Close(db)
		return nil, fmt.Errorf("auth system health check failed: %w", err)
	}

	tmHandler := taskManager.NewHTTPHandler(tm)

	// withAuth wraps an individual handler with the authentication middleware.
	withAuth := authManager.Middleware()

	mux := http.NewServeMux()

	// Health check is public and returns JSON in all cases.
	// On failure, the component field identifies which subsystem is unhealthy
	// without exposing internal error details.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		var unhealthy []string

		if err := database.HealthCheck(db); err != nil {
			unhealthy = append(unhealthy, "database")
		}
		if err := authManager.Health(); err != nil {
			unhealthy = append(unhealthy, "auth")
		}

		if len(unhealthy) > 0 {
			writeJSON(w, http.StatusServiceUnavailable, healthResponse{
				Status:              "error",
				Service:             "nsw-backend",
				UnhealthyComponents: unhealthy,
			})
			return
		}

		writeJSON(w, http.StatusOK, healthResponse{
			Status:  "ok",
			Service: "nsw-backend",
		})
	})

	// v1 routes. Each handler is individually wrapped with auth,
	// so public or differently-authenticated routes can be added
	// alongside these without restructuring the mux.
	mux.Handle("POST /api/v1/tasks", withAuth(http.HandlerFunc(tmHandler.HandleExecuteTask)))
	mux.Handle("GET /api/v1/tasks/{id}", withAuth(http.HandlerFunc(tmHandler.HandleGetTask)))
	mux.Handle("GET /api/v1/hscodes", withAuth(http.HandlerFunc(hsCodeRouter.HandleGetAllHSCodes)))
	mux.Handle("GET /api/v1/chas", withAuth(http.HandlerFunc(chaRouter.HandleGetCHAs)))
	mux.Handle("POST /api/v1/consignments", withAuth(http.HandlerFunc(consignmentRouter.HandleCreateConsignment)))
	mux.Handle("GET /api/v1/consignments/{id}", withAuth(http.HandlerFunc(consignmentRouter.HandleGetConsignmentByID)))
	mux.Handle("PUT /api/v1/consignments/{id}", withAuth(http.HandlerFunc(consignmentRouter.HandleInitializeConsignment)))
	mux.Handle("GET /api/v1/consignments", withAuth(http.HandlerFunc(consignmentRouter.HandleGetConsignments)))
	mux.Handle("POST /api/v1/pre-consignments", withAuth(http.HandlerFunc(preConsignmentRouter.HandleCreatePreConsignment)))
	mux.Handle("GET /api/v1/pre-consignments/{preConsignmentId}", withAuth(http.HandlerFunc(preConsignmentRouter.HandleGetPreConsignmentByID)))
	mux.Handle("GET /api/v1/pre-consignments", withAuth(http.HandlerFunc(preConsignmentRouter.HandleGetTraderPreConsignments)))
	mux.Handle("POST /api/v1/uploads", withAuth(http.HandlerFunc(uploadHandler.Upload)))
	mux.Handle("GET /api/v1/uploads/{key}", withAuth(http.HandlerFunc(uploadHandler.Download)))
	mux.Handle("DELETE /api/v1/uploads/{key}", withAuth(http.HandlerFunc(uploadHandler.Delete)))

	// When using local storage, this endpoint serves the actual file bytes.
	// It's made public since it's the equivalent of a presigned URL when using S3.
	if _, ok := storageDriver.(*drivers.LocalFSDriver); ok {
		mux.HandleFunc("GET /api/v1/uploads/{key}/content", uploadHandler.DownloadContent)
	}

	handler := middleware.CORS(&cfg.CORS)(mux)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: handler,
	}

	closeFn := func() error {
		authErr := authManager.Close()
		dbErr := database.Close(db)
		if authErr != nil {
			if dbErr != nil {
				return fmt.Errorf("failed to close auth manager: %v; failed to close database: %v", authErr, dbErr)
			}
			return fmt.Errorf("failed to close auth manager: %w", authErr)
		}
		if dbErr != nil {
			return fmt.Errorf("failed to close database: %w", dbErr)
		}
		return nil
	}

	return &App{Server: server, close: closeFn}, nil
}
