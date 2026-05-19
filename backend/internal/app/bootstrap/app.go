package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/OpenNSW/nsw/internal/auth"
	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/consignment"
	"github.com/OpenNSW/nsw/internal/database"
	"github.com/OpenNSW/nsw/internal/hscode"
	"github.com/OpenNSW/nsw/internal/middleware"
	"github.com/OpenNSW/nsw/internal/payments"
	"github.com/OpenNSW/nsw/internal/profile/cha"
	"github.com/OpenNSW/nsw/internal/profile/user"
	taskmanager "github.com/OpenNSW/nsw/internal/task/manager"
	"github.com/OpenNSW/nsw/internal/task/plugin"
	"github.com/OpenNSW/nsw/internal/temporal"
	workflowruntime "github.com/OpenNSW/nsw/internal/workflow/runtime"
	"github.com/OpenNSW/nsw/internal/workflow/service"
	"github.com/OpenNSW/nsw/pkg/storage"
	"github.com/OpenNSW/nsw/pkg/storage/drivers"

	"github.com/OpenNSW/nsw/pkg/notification"
	"github.com/OpenNSW/nsw/pkg/notification/channels"
)

// App contains an initialized HTTP server and cleanup hooks.
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

	factory := plugin.NewTaskFactory(cfg, db, paymentService)
	tm, err := taskmanager.NewTaskManager(db, factory)
	if err != nil {
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to create task manager: %w", err)
	}

	templateService := service.NewTemplateService(db)
	chaService := cha.NewService(db)
	hsCodeService := hscode.NewService(db)

	temporalClient, err := temporal.NewClient(cfg.Temporal)
	if err != nil {
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to create temporal client: %w", err)
	}

	consignmentService := consignment.NewService(db, templateService, chaService, hsCodeService)
	consignmentRouter := consignment.NewRouter(consignmentService, chaService)

	workflowRuntime, err := workflowruntime.NewRuntime(temporalClient, tm, templateService, consignmentService)
	if err != nil {
		temporalClient.Close()
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to create workflow runtime: %w", err)
	}

	registererr := consignmentService.RegisterWorkflowManager(workflowRuntime.Manager())
	if registererr != nil {
		_ = workflowRuntime.Close()
		temporalClient.Close()
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to register workflow manager with consignment service: %w", registererr)
	}

	hsCodeRouter := hscode.NewRouter(hsCodeService)
	chaHandler := cha.NewHandler(chaService)

	storageDriver, err := storage.NewStorageFromConfig(ctx, cfg.Storage)
	if err != nil {
		_ = workflowRuntime.Close()
		temporalClient.Close()
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}
	storageService := storage.NewService(storageDriver)
	storageHandler := storage.NewHTTPHandler(storageService)

	paymentHandler := payments.NewHTTPHandler(paymentService)

	userProfileService := user.NewService(db)

	authManager, err := auth.NewManager(userProfileService, cfg.Auth)
	if err != nil {
		_ = workflowRuntime.Close()
		temporalClient.Close()
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to create auth manager: %w", err)
	}

	if err := authManager.Health(); err != nil {
		_ = workflowRuntime.Close()
		temporalClient.Close()
		_ = authManager.Close()
		_ = database.Close(db)
		return nil, fmt.Errorf("auth system health check failed: %w", err)
	}

	// Initialize notification manager
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

	// TODO: Add SMS channel if needed
	// smsChannel := channels.NewSMSChannel(...)
	// notificationManager.RegisterSMSChannel(smsChannel)

	tmHandler := taskmanager.NewHTTPHandler(tm)

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

	// API v1 routes. Each handler is individually wrapped with auth,
	// so public or differently-authenticated routes can be added
	// alongside these without restructuring the mux.
	mux.Handle("POST /api/v1/tasks", withAuth(http.HandlerFunc(tmHandler.HandleExecuteTask)))
	mux.Handle("GET /api/v1/tasks/{id}", withAuth(http.HandlerFunc(tmHandler.HandleGetTask)))
	mux.Handle("GET /api/v1/hscodes", withAuth(http.HandlerFunc(hsCodeRouter.HandleGetAll)))
	mux.Handle("GET /api/v1/chas", withAuth(http.HandlerFunc(chaHandler.HandleGetCHAs)))
	mux.Handle("POST /api/v1/consignments", withAuth(http.HandlerFunc(consignmentRouter.HandleCreateConsignment)))
	mux.Handle("GET /api/v1/consignments/{id}", withAuth(http.HandlerFunc(consignmentRouter.HandleGetConsignmentByID)))
	mux.Handle("PUT /api/v1/consignments/{id}", withAuth(http.HandlerFunc(consignmentRouter.HandleInitializeConsignment)))
	mux.Handle("GET /api/v1/consignments", withAuth(http.HandlerFunc(consignmentRouter.HandleGetConsignments)))
	// TODO: Add pre-consignment routes once migrated to Temporal.
	// mux.Handle("POST /api/v1/pre-consignments", withAuth(http.HandlerFunc(preConsignmentRouter.HandleCreatePreConsignment)))
	// mux.Handle("GET /api/v1/pre-consignments/{preConsignmentId}", withAuth(http.HandlerFunc(preConsignmentRouter.HandleGetPreConsignmentByID)))
	// mux.Handle("GET /api/v1/pre-consignments", withAuth(http.HandlerFunc(preConsignmentRouter.HandleGetTraderPreConsignments)))
	mux.Handle("POST /api/v1/storage", withAuth(http.HandlerFunc(storageHandler.Upload)))
	mux.Handle("GET /api/v1/storage/{key}", withAuth(http.HandlerFunc(storageHandler.Download)))
	mux.Handle("DELETE /api/v1/storage/{key}", withAuth(http.HandlerFunc(storageHandler.Delete)))

	// External Webhooks bypass standard JWT auth.
	// They should use webhook signatures, implemented in the handler directly or via specialized middleware.
	mux.Handle("POST /api/v1/payments/webhook", http.HandlerFunc(paymentHandler.HandleWebhook))
	mux.Handle("POST /api/v1/payments/validate", http.HandlerFunc(paymentHandler.HandleValidateReference))

	// When using local storage, these endpoints serve as mocks for S3.
	if _, ok := storageDriver.(*drivers.LocalFSDriver); ok {
		mux.HandleFunc("PUT /api/v1/storage/{key}/content", storageHandler.UploadContentLocal)
		mux.HandleFunc("GET /api/v1/storage/{key}/content", storageHandler.DownloadContent)
	}

	handler := middleware.CORS(&cfg.CORS)(mux)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: handler,
	}

	closeFn := func() error {
		var closeErrs []error

		if err := workflowRuntime.Close(); err != nil {
			closeErrs = append(closeErrs, fmt.Errorf("failed to close workflow runtime: %w", err))
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
