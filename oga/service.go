package oga

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// ErrApplicationNotFound is returned when an application is not found
var ErrApplicationNotFound = errors.New("application not found")

// OGAService handles OGA portal operations
// OGA service maintains its own database and syncs with backend via polling
type OGAService interface {
	// SyncApplications syncs applications from backend by polling GET /api/tasks
	SyncApplications(ctx context.Context) error

	// GetApplications returns all applications ready for review
	GetApplications(ctx context.Context) ([]Application, error)

	// GetApplication returns a specific application by task ID
	GetApplication(ctx context.Context, taskID uuid.UUID) (*Application, error)

	// StartSyncWorker starts a background worker that periodically syncs applications
	StartSyncWorker(ctx context.Context, interval time.Duration)

	// Close closes the service and releases resources
	Close() error
}

// Application represents an application ready for OGA review
type Application struct {
	TaskID        uuid.UUID `json:"taskId"`
	ConsignmentID uuid.UUID `json:"consignmentId"`
	StepID        string    `json:"stepId"`
	FormID        string    `json:"formId"`
	Status        string    `json:"status"`
}

type ogaService struct {
	store        *ApplicationStore
	backendClient BackendClient
}

// NewOGAService creates a new OGA service instance with database storage
func NewOGAService(store *ApplicationStore, backendClient BackendClient) OGAService {
	return &ogaService{
		store:        store,
		backendClient: backendClient,
	}
}

// SyncApplications syncs applications from backend by polling GET /api/tasks
// It fetches all OGA_FORM tasks with READY status and stores them in the database
func (s *ogaService) SyncApplications(ctx context.Context) error {
	// Fetch OGA_FORM tasks with READY status from backend
	// READY means the task dependencies are satisfied and it's waiting for OGA officer action
	tasks, err := s.backendClient.GetTasks(ctx, "OGA_FORM", "READY")
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "syncing applications from backend",
		"count", len(tasks))

	// Process each task and store/update in database
	for _, task := range tasks {
		// Extract formID from config
		var config struct {
			FormID string `json:"formId"`
		}
		if err := json.Unmarshal(task.Config, &config); err != nil {
			slog.WarnContext(ctx, "failed to parse task config",
				"taskID", task.ID,
				"error", err)
			continue
		}

		appRecord := &ApplicationRecord{
			TaskID:        task.ID,
			ConsignmentID: task.ConsignmentID,
			StepID:        task.StepID,
			FormID:        config.FormID,
			Status:        task.Status,
		}

		if err := s.store.CreateOrUpdate(appRecord); err != nil {
			slog.ErrorContext(ctx, "failed to store application",
				"taskID", task.ID,
				"error", err)
			continue
		}

		slog.DebugContext(ctx, "synced application",
			"taskID", task.ID,
			"consignmentID", task.ConsignmentID)
	}

	return nil
}

// GetApplications returns all applications ready for review
func (s *ogaService) GetApplications(ctx context.Context) ([]Application, error) {
	records, err := s.store.GetAll()
	if err != nil {
		return nil, err
	}

	applications := make([]Application, len(records))
	for i, record := range records {
		applications[i] = Application{
			TaskID:        record.TaskID,
			ConsignmentID: record.ConsignmentID,
			StepID:        record.StepID,
			FormID:        record.FormID,
			Status:        record.Status,
		}
	}

	return applications, nil
}

// GetApplication returns a specific application by task ID
func (s *ogaService) GetApplication(ctx context.Context, taskID uuid.UUID) (*Application, error) {
	record, err := s.store.GetByTaskID(taskID)
	if err != nil {
		return nil, ErrApplicationNotFound
	}

	return &Application{
		TaskID:        record.TaskID,
		ConsignmentID: record.ConsignmentID,
		FormID:        record.FormID,
		Status:        record.Status,
	}, nil
}

// StartSyncWorker starts a background worker that periodically syncs applications from backend
func (s *ogaService) StartSyncWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial sync
	if err := s.SyncApplications(ctx); err != nil {
		slog.ErrorContext(ctx, "initial sync failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("sync worker stopped")
			return
		case <-ticker.C:
			if err := s.SyncApplications(ctx); err != nil {
				slog.ErrorContext(ctx, "sync failed", "error", err)
			}
		}
	}
}

// Close closes the service and releases resources
func (s *ogaService) Close() error {
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}
