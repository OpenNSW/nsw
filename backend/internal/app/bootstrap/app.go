package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	workflowManagerV2 "github.com/OpenNSW/go-temporal-workflow"

	"github.com/OpenNSW/nsw/internal/auth"
	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/database"
	"github.com/OpenNSW/nsw/internal/middleware"
	"github.com/OpenNSW/nsw/internal/payments"
	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	"github.com/OpenNSW/nsw/internal/task/plugin"
	"github.com/OpenNSW/nsw/internal/uploads"
	"github.com/OpenNSW/nsw/internal/uploads/drivers"
	workflowManagerV1 "github.com/OpenNSW/nsw/internal/workflow/manager"
	"github.com/OpenNSW/nsw/internal/workflow/router"
	"github.com/OpenNSW/nsw/internal/workflow/service"

	"github.com/OpenNSW/nsw/pkg/notification"
	"github.com/OpenNSW/nsw/pkg/notification/channels"

	"go.temporal.io/sdk/client"
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

func setupWorkflowManagerV2(
	ctx context.Context,
	cfg *config.Config,
	tm taskManager.TaskManager,
	templateService *service.TemplateService,
) (workflowManagerV2.TemporalManager, error) {
	// 1. Connect to the local Temporal Server (Needed for Workflow Manager V2)
	c, err := client.Dial(client.Options{})
	if err != nil {
		return nil, fmt.Errorf("error creating temporal client: %w", err)
	}
	// ***************
	// Note: You may need to manage closing the client gracefully elsewhere
	// defer c.Close()

	// 4. Define Handlers for Manager Bridge
	activationHandler := func(payload workflowManagerV2.TaskPayload) error {
		template, err := templateService.GetWorkflowNodeTemplateByID(ctx, payload.TaskTemplateID)
		if err != nil {
			return fmt.Errorf("error getting workflow node template: %w", err)
		}

		// TODO: We need to pass the TaskPayload.RunID in the future to avoid issues with
		// task retries. For example, when retrying a task instance, a stale version might
		// send a completion that will trigger the new version.
		tmRequest := taskManager.InitTaskRequest{
			TaskID:                 payload.NodeID,
			WorkflowID:             payload.WorkflowID,
			WorkflowNodeTemplateID: template.ID,
			GlobalState:            payload.Inputs,
			Type:                   template.Type,
			Config:                 template.Config,
		}
		_, err = tm.InitTask(ctx, tmRequest)
		if err != nil {
			return fmt.Errorf("error initializing task manager: %w", err)
		}
		return nil
	}

	completionHandler := func(workflowID string, finalContext map[string]any) error {
		slog.Info("Workflow logically completed", "workflowID", workflowID, "finalContext", finalContext)
		// TODO: If consignment, need to call OnWorkflowStatusChanged
		//       If pre-consignment, need to call OnPreWorkflowStatusChanged
		return nil
	}

	// 5. Initialize Manager
	workflowManager := workflowManagerV2.NewTemporalManager(c, "INTERPRETER_TASK_QUEUE", activationHandler, completionHandler)

	taskDoneWrapper := func(ctx context.Context, workflowID string, taskID string, outputs map[string]any) {
		err := workflowManager.TaskDone(ctx, workflowID, "", taskID, outputs)
		if err != nil {
			slog.Error("error completing task", "error", err)
		}
	}

	tm.RegisterUpstreamDoneCallback(taskDoneWrapper)

	// Start the workers.
	if err := workflowManager.StartWorker(); err != nil {
		return nil, fmt.Errorf("failed to start workflow manager worker: %w", err)
	}

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

	paymentRepo := payments.NewPaymentRepository(db)
	paymentService := payments.NewPaymentService(paymentRepo)

	factory := plugin.NewTaskFactory(cfg, db, paymentService)
	tm, err := taskManager.NewTaskManager(db, factory)
	if err != nil {
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to create task manager: %w", err)
	}

	templateService := service.NewTemplateService(db)
	chaService := service.NewCHAService(db)
	hsCodeService := service.NewHSCodeService(db)

	var consignmentRouter *router.ConsignmentRouter
	var preConsignmentRouter *router.PreConsignmentRouter

	var wmV2 workflowManagerV2.TemporalManager
	if cfg.UseWorkflowManagerV2 {
		// --- NEW WORKFLOW MANAGER CODE ---
		var err error
		wmV2, err = setupWorkflowManagerV2(ctx, cfg, tm, templateService)
		if err != nil {
			_ = database.Close(db)
			return nil, fmt.Errorf("failed to create workflow manager v2: %w", err)
		}

		consignmentService := service.NewConsignmentService(db, templateService, nil, wmV2)
		consignmentRouter = router.NewConsignmentRouter(consignmentService, chaService)

		// TODO: Pre-Consignment is commented out in new workflow for now
		// preConsignmentService := service.NewPreConsignmentService(db, templateService, wmV2)
		// preConsignmentRouter = router.NewPreConsignmentRouter(preConsignmentService)
		preConsignmentRouter = nil
	} else {
		// --- OLD WORKFLOW MANAGER CODE ---
		nodeService := service.NewWorkflowNodeService(db)
		wm := workflowManagerV1.NewManager(db, nodeService, templateService)

		if err := WireManagers(wm, tm); err != nil {
			_ = database.Close(db)
			return nil, fmt.Errorf("failed to wire managers: %w", err)
		}

		// Note: The signature of NewConsignmentService might have changed to accept both managers
		// We pass 'wm' in place of the v1 manager, and 'nil' for the v2 one.
		consignmentService := service.NewConsignmentService(db, templateService, wm, nil)
		preConsignmentService := service.NewPreConsignmentService(db, templateService, wm)

		consignmentRouter = router.NewConsignmentRouter(consignmentService, chaService)
		preConsignmentRouter = router.NewPreConsignmentRouter(preConsignmentService)
	}

	hsCodeRouter := router.NewHSCodeRouter(hsCodeService)
	chaRouter := router.NewCHARouter(chaService)

	storageDriver, err := uploads.NewStorageFromConfig(ctx, cfg.Storage)
	if err != nil {
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}
	uploadService := uploads.NewUploadService(storageDriver)
	uploadHandler := uploads.NewHTTPHandler(uploadService)

	paymentHandler := payments.NewHTTPHandler(paymentService)

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
	if !cfg.UseWorkflowManagerV2 {
		mux.Handle("POST /api/v1/pre-consignments", withAuth(http.HandlerFunc(preConsignmentRouter.HandleCreatePreConsignment)))
		mux.Handle("GET /api/v1/pre-consignments/{preConsignmentId}", withAuth(http.HandlerFunc(preConsignmentRouter.HandleGetPreConsignmentByID)))
		mux.Handle("GET /api/v1/pre-consignments", withAuth(http.HandlerFunc(preConsignmentRouter.HandleGetTraderPreConsignments)))
	}
	mux.Handle("POST /api/v1/uploads", withAuth(http.HandlerFunc(uploadHandler.Upload)))
	mux.Handle("GET /api/v1/uploads/{key}", withAuth(http.HandlerFunc(uploadHandler.Download)))
	mux.Handle("DELETE /api/v1/uploads/{key}", withAuth(http.HandlerFunc(uploadHandler.Delete)))

	// External Webhooks bypass standard JWT auth.
	// They should use webhook signatures, implemented in the handler directly or via specialized middleware.
	mux.Handle("POST /api/v1/payments/webhook", http.HandlerFunc(paymentHandler.HandleWebhook))
	mux.Handle("POST /api/v1/payments/validate", http.HandlerFunc(paymentHandler.HandleValidateReference))

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

		if cfg.UseWorkflowManagerV2 {
			wmV2.StopWorker()
		}
		return nil
	}

	return &App{
		Server:              server,
		NotificationManager: notificationManager,
		close:               closeFn,
	}, nil
}
