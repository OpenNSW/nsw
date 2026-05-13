package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/OpenNSW/nsw-task-flow/orchestrator"
	"github.com/OpenNSW/nsw/internal/profile/user"

	"github.com/OpenNSW/nsw/internal/auth"
	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/database"
	"github.com/OpenNSW/nsw/internal/middleware"
	"github.com/OpenNSW/nsw/internal/payments"
	taskv2router "github.com/OpenNSW/nsw/internal/taskv2/router"
	taskv2store "github.com/OpenNSW/nsw/internal/taskv2/store"
	"github.com/OpenNSW/nsw/internal/template"
	"github.com/OpenNSW/nsw/internal/temporal"
	"github.com/OpenNSW/nsw/internal/uploads"
	"github.com/OpenNSW/nsw/internal/uploads/drivers"
	"github.com/OpenNSW/nsw/internal/workflow/router"
	"github.com/OpenNSW/nsw/internal/workflow/service"

	"github.com/OpenNSW/nsw/pkg/notification"
	"github.com/OpenNSW/nsw/pkg/notification/channels"
	"github.com/OpenNSW/nsw/pkg/remote"
)

// App contains initialized HTTP server and cleanup hooks.
type App struct {
	Server              *http.Server
	NotificationManager *notification.Manager
	close               func() error
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
	db, err := database.New(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := database.HealthCheck(db); err != nil {
		_ = database.Close(db)
		return nil, fmt.Errorf("database health check failed: %w", err)
	}

	paymentRepo := payments.NewPaymentRepository(db)
	paymentService := payments.NewPaymentService(paymentRepo)

	// Workflow / consignment domain services (still used by /api/v1/consignments).
	templateService := service.NewTemplateService(db)
	chaService := service.NewCHAService(db)
	hsCodeService := service.NewHSCodeService(db)

	// Temporal client — shared by both nsw-task-flow Temporal managers.
	temporalClient, err := temporal.NewClient(cfg.Temporal)
	if err != nil {
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to create temporal client: %w", err)
	}

	consignmentService := service.NewConsignmentService(db, templateService)
	consignmentRouter := router.NewConsignmentRouter(consignmentService, chaService)

	// nsw-task-flow registry, store, runtime.
	registry := orchestrator.NewTaskTemplateRegistry()
	if err := template.Load(registry, cfg.Server.TemplatesDir); err != nil {
		temporalClient.Close()
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to load task templates from %q: %w", cfg.Server.TemplatesDir, err)
	}

	remoteManager := remote.NewManager()
	if err := remoteManager.LoadServices(cfg.Server.ServicesConfigPath); err != nil {
		slog.Warn("app: failed to load external services configuration",
			"path", cfg.Server.ServicesConfigPath, "error", err)
	} else {
		slog.Info("app: external services configuration loaded",
			"services", remoteManager.ListServices())
	}

	taskStore := taskv2store.NewGormTaskStore(db)
	runtime, err := NewRuntime(Config{
		TemporalClient:  temporalClient,
		Store:           taskStore,
		Registry:        registry,
		RemoteManager:   remoteManager,
		BackendBaseURL:  cfg.Server.ServiceURL,
		DevMode:         cfg.Server.TaskFlowDevMode,
		UpstreamService: consignmentService,
	})
	if err != nil {
		temporalClient.Close()
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to start taskv2 runtime: %w", err)
	}

	registererr := consignmentService.RegisterWorkflowManager(runtime.WorkflowManager())
	if registererr != nil {
		_ = runtime.Close()
		temporalClient.Close()
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to register workflow manager with consignment service: %w", registererr)
	}
	// TODO: Pre-consignment wiring is intentionally disabled until it is migrated to Temporal.
	// preConsignmentService := service.NewPreConsignmentService(db, templateService, wm)
	// preConsignmentRouter := router.NewPreConsignmentRouter(preConsignmentService)

	hsCodeRouter := router.NewHSCodeRouter(hsCodeService)
	chaRouter := router.NewCHARouter(chaService)

	// Storage / uploads.
	storageDriver, err := uploads.NewStorageFromConfig(ctx, cfg.Storage)
	if err != nil {
		_ = runtime.Close()
		temporalClient.Close()
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}
	uploadService := uploads.NewUploadService(storageDriver)
	uploadHandler := uploads.NewHTTPHandler(uploadService)

	paymentHandler := payments.NewHTTPHandler(paymentService)

	userProfileService := user.NewService(db)

	authManager, err := auth.NewManager(userProfileService, cfg.Auth)
	if err != nil {
		_ = runtime.Close()
		temporalClient.Close()
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to create auth manager: %w", err)
	}
	if err := authManager.Health(); err != nil {
		_ = runtime.Close()
		temporalClient.Close()
		_ = authManager.Close()
		_ = database.Close(db)
		return nil, fmt.Errorf("auth system health check failed: %w", err)
	}

	notificationManager := notification.NewManager()
	emailChannel := channels.NewEmailChannel(notification.EmailConfig{
		SMTPHost:     cfg.Notification.SMTPHost,
		SMTPPort:     cfg.Notification.SMTPPort,
		SMTPUsername: cfg.Notification.SMTPUsername,
		SMTPPassword: cfg.Notification.SMTPPassword,
		SMTPSender:   cfg.Notification.SMTPSender,
		TemplateRoot: cfg.Notification.TemplateRoot,
	})
	notificationManager.RegisterEmailChannel(emailChannel)

	// nsw-task-flow HTTP routers — trader-facing and consignment-task discovery.
	// The /api/oga/* surface lives in the standalone OGA service (oga/), not here.
	tfRouter := taskv2router.New(runtime.TaskManager(), runtime.WorkflowManager(), registry)
	consignmentTasksRouter := taskv2router.NewConsignmentTasksRouter(runtime.TaskManager(), registry)

	withAuth := authManager.Middleware()

	mux := http.NewServeMux()

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
				Status: "error", Service: "nsw-backend", UnhealthyComponents: unhealthy,
			})
			return
		}
		writeJSON(w, http.StatusOK, healthResponse{Status: "ok", Service: "nsw-backend"})
	})

	// ── nsw-task-flow surface ────────────────────────────────────────────────
	// New endpoints (parent workflow start, list/get/complete tasks).
	mux.Handle("POST /api/v1/start-workflow", withAuth(http.HandlerFunc(tfRouter.HandleStartWorkflow)))
	mux.Handle("GET /api/v1/tasks", withAuth(http.HandlerFunc(tfRouter.HandleListTasks)))
	mux.Handle("GET /api/v1/tasks/{id}", withAuth(http.HandlerFunc(tfRouter.HandleGetTask)))
	// Modern path: POST /api/v1/tasks/{id}
	mux.Handle("POST /api/v1/tasks/{id}", withAuth(http.HandlerFunc(tfRouter.HandleCompleteTaskStep)))
	// Legacy path: POST /api/v1/tasks (body has task_id, payload.action, payload.content) —
	// kept so the existing trader-app screens still resume tasks. The action field is logged
	// and ignored (nsw-task-flow has no FSM).
	mux.Handle("POST /api/v1/tasks", withAuth(http.HandlerFunc(tfRouter.HandleCompleteTaskStep)))

	// ── Domain routes (consignments, HS codes, CHAs) ─────────────────────────
	mux.Handle("GET /api/v1/hscodes", withAuth(http.HandlerFunc(hsCodeRouter.HandleGetAllHSCodes)))
	mux.Handle("GET /api/v1/chas", withAuth(http.HandlerFunc(chaRouter.HandleGetCHAs)))
	mux.Handle("POST /api/v1/consignments", withAuth(http.HandlerFunc(consignmentRouter.HandleCreateConsignment)))
	mux.Handle("GET /api/v1/consignments/{id}", withAuth(http.HandlerFunc(consignmentRouter.HandleGetConsignmentByID)))
	mux.Handle("PUT /api/v1/consignments/{id}", withAuth(http.HandlerFunc(consignmentRouter.HandleInitializeConsignment)))
	mux.Handle("GET /api/v1/consignments", withAuth(http.HandlerFunc(consignmentRouter.HandleGetConsignments)))
	// Discover the dynamic nsw-task-flow task IDs that belong to a consignment
	// — used by trader-app to pivot from consignment detail to active task.
	mux.Handle("GET /api/v1/consignments/{id}/tasks", withAuth(http.HandlerFunc(consignmentTasksRouter.HandleListConsignmentTasks)))

	// ── Uploads, payments, webhooks ──────────────────────────────────────────
	mux.Handle("POST /api/v1/uploads", withAuth(http.HandlerFunc(uploadHandler.Upload)))
	mux.Handle("GET /api/v1/uploads/{key}", withAuth(http.HandlerFunc(uploadHandler.Download)))
	mux.Handle("DELETE /api/v1/uploads/{key}", withAuth(http.HandlerFunc(uploadHandler.Delete)))

	mux.Handle("POST /api/v1/payments/webhook", http.HandlerFunc(paymentHandler.HandleWebhook))
	mux.Handle("POST /api/v1/payments/validate", http.HandlerFunc(paymentHandler.HandleValidateReference))

	if _, ok := storageDriver.(*drivers.LocalFSDriver); ok {
		mux.HandleFunc("PUT /api/v1/uploads/{key}/content", uploadHandler.UploadContentLocal)
		mux.HandleFunc("GET /api/v1/uploads/{key}/content", uploadHandler.DownloadContent)
	}

	handler := middleware.CORS(&cfg.CORS)(mux)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: handler,
	}

	closeFn := func() error {
		var closeErrs []error
		if err := runtime.Close(); err != nil {
			closeErrs = append(closeErrs, fmt.Errorf("failed to close taskv2 runtime: %w", err))
		}
		temporalClient.Close()
		if err := authManager.Close(); err != nil {
			closeErrs = append(closeErrs, fmt.Errorf("failed to close auth manager: %w", err))
		}
		if err := database.Close(db); err != nil {
			closeErrs = append(closeErrs, fmt.Errorf("failed to close database: %w", err))
		}
		return errors.Join(closeErrs...)
	}

	return &App{
		Server:              server,
		NotificationManager: notificationManager,
		close:               closeFn,
	}, nil
}
