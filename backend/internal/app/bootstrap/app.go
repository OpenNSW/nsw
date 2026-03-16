package bootstrap

import (
	"context"
	"fmt"
	"net/http"

	"github.com/OpenNSW/nsw/internal/auth"
	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/database"
	"github.com/OpenNSW/nsw/internal/form"
	"github.com/OpenNSW/nsw/internal/middleware"
	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	"github.com/OpenNSW/nsw/internal/uploads"
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

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/tasks", tmHandler.HandleExecuteTask)
	mux.HandleFunc("GET /api/v1/tasks/{id}", tmHandler.HandleGetTask)
	mux.HandleFunc("GET /api/v1/hscodes", hsCodeRouter.HandleGetAllHSCodes)
	mux.HandleFunc("GET /api/v1/chas", chaRouter.HandleGetCHAs)
	mux.HandleFunc("POST /api/v1/consignments", consignmentRouter.HandleCreateConsignment)
	mux.HandleFunc("GET /api/v1/consignments/{id}", consignmentRouter.HandleGetConsignmentByID)
	mux.HandleFunc("PUT /api/v1/consignments/{id}", consignmentRouter.HandleInitializeConsignment)
	mux.HandleFunc("GET /api/v1/consignments", consignmentRouter.HandleGetConsignments)
	mux.HandleFunc("POST /api/v1/pre-consignments", preConsignmentRouter.HandleCreatePreConsignment)
	mux.HandleFunc("GET /api/v1/pre-consignments/{preConsignmentId}", preConsignmentRouter.HandleGetPreConsignmentByID)
	mux.HandleFunc("GET /api/v1/pre-consignments", preConsignmentRouter.HandleGetTraderPreConsignments)
	mux.HandleFunc("POST /api/v1/uploads", uploadHandler.Upload)
	mux.HandleFunc("GET /api/v1/uploads/{key}/content", uploadHandler.DownloadContent)
	mux.HandleFunc("GET /api/v1/uploads/{key}", uploadHandler.Download)
	mux.HandleFunc("DELETE /api/v1/uploads/{key}", uploadHandler.Delete)

	handler := middleware.CORS(&cfg.CORS)(authManager.Middleware()(mux))
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
