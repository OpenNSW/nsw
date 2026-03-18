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
	"github.com/OpenNSW/nsw/internal/task/plugin"
	"github.com/OpenNSW/nsw/internal/uploads"
	"github.com/OpenNSW/nsw/internal/workflow/router"
	"github.com/OpenNSW/nsw/internal/workflow/service"
	engine "github.com/lokewate/go-temporal-workflow"
	workflowmanager "github.com/lokewate/go-temporal-workflow"

	"go.temporal.io/sdk/client"
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

func setupTemporalWorkflowManager(
	ctx context.Context,
	cfg *config.Config,
	tm taskManager.TaskManager) (workflowmanager.TemporalManager, error) {
	// 1. Connect to the local Temporal Server
	c, err := client.Dial(client.Options{})
	if err != nil {
		return nil, fmt.Errorf("Error creating Temporal client: %v\n", err)
	}
	defer c.Close()

	// 2. Break Circular Dependency: The Dispatcher needs the Manager's TaskDone method,
	// but the Manager needs the Dispatcher's HandleTask method during initialization.
	// var workflowManager workflowmanager.TemporalManager

	// 4. Define Handlers for Temporal Bridge
	activationHandler := func(payload engine.TaskPayload) error {
		tmRequest := taskManager.InitTaskRequest{
			TaskID: payload.NodeID,
			// WorkflowID is not used in taskManager
			WorkflowID:             "",
			WorkflowNodeTemplateID: payload.WorkflowID,
			GlobalState:            payload.Inputs,
		}
		_, err = tm.InitTask(ctx, tmRequest)
		if err != nil {
			return fmt.Errorf("Error initializing task manager: %v\n", err)
		}
		return nil
	}

	completionHandler := func(workflowID string, finalContext map[string]any) error {
		fmt.Printf("Temporal Workflow %s logically completed with final context!\n", workflowID)
		return nil
	}

	// 5. Initialize Manager
	workflowManager := engine.NewTemporalManager(c, "INTERPRETER_TASK_QUEUE", activationHandler, completionHandler)

	taskDoneWrapper := func(
		ctx context.Context,
		taskID string,
		state *plugin.State,
		extendedState *string,
		appendGlobalContext map[string]any,
		outcome *string) {
		workflowID := ""
		runID := ""
		nodeID := taskID
		err := workflowManager.TaskDone(ctx, workflowID, runID, nodeID, appendGlobalContext)
		if err != nil {
			fmt.Printf("Error completing task: %v\n", err)
		}
	}

	tm.RegisterUpstreamCallback(taskDoneWrapper)

	return workflowManager, nil
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

	wm := setupTemporalWorkflowManager(ctx, cfg, tm)

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
	mux.Handle("GET /api/v1/uploads/{key}/content", withAuth(http.HandlerFunc(uploadHandler.DownloadContent)))
	mux.Handle("GET /api/v1/uploads/{key}", withAuth(http.HandlerFunc(uploadHandler.Download)))
	mux.Handle("DELETE /api/v1/uploads/{key}", withAuth(http.HandlerFunc(uploadHandler.Delete)))

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
