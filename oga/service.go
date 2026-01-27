package oga

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ErrApplicationNotFound is returned when an application is not found
var ErrApplicationNotFound = errors.New("application not found")

// OGAService handles OGA portal operations
type OGAService interface {
	// CreateApplication creates a new application from injected data
	CreateApplication(ctx context.Context, req *InjectRequest) error

	// GetApplications returns all applications (optionally filtered by status)
	GetApplications(ctx context.Context, status string) ([]Application, error)

	// GetApplication returns a specific application by task ID
	GetApplication(ctx context.Context, taskID uuid.UUID) (*Application, error)

	// ReviewApplication approves or rejects an application and sends response back to service
	ReviewApplication(ctx context.Context, taskID uuid.UUID, decision string, reviewerNotes string) error

	// Close closes the service and releases resources
	Close() error
}

// InjectRequest represents the incoming data from services
type InjectRequest struct {
	TaskID        uuid.UUID              `json:"taskId"`
	ConsignmentID uuid.UUID              `json:"consignmentId"`
	Data          map[string]interface{} `json:"data"`
	ServiceURL    string                 `json:"serviceUrl"` // URL to send response back to
}

// Application represents an application for display in the UI
type Application struct {
	TaskID        uuid.UUID              `json:"taskId"`
	ConsignmentID uuid.UUID              `json:"consignmentId"`
	ServiceURL    string                 `json:"serviceUrl"`
	Data          map[string]interface{} `json:"data"`
	Status        string                 `json:"status"`
	ReviewerNotes string                 `json:"reviewerNotes,omitempty"`
	ReviewedAt    *time.Time             `json:"reviewedAt,omitempty"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

// TaskResponse represents the response sent back to the service
type TaskResponse struct {
	TaskID        uuid.UUID   `json:"task_id"`
	ConsignmentID uuid.UUID   `json:"consignment_id"`
	Payload       interface{} `json:"payload"`
}

type ogaService struct {
	store      *ApplicationStore
	httpClient *http.Client
}

// NewOGAService creates a new OGA service instance with database storage
func NewOGAService(store *ApplicationStore) OGAService {
	return &ogaService{
		store: store,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateApplication creates a new application from injected data
func (s *ogaService) CreateApplication(ctx context.Context, req *InjectRequest) error {
	// Validate required fields
	if req.TaskID == uuid.Nil {
		return fmt.Errorf("taskId is required")
	}
	if req.ConsignmentID == uuid.Nil {
		return fmt.Errorf("consignmentId is required")
	}
	if req.ServiceURL == "" {
		return fmt.Errorf("serviceUrl is required")
	}

	appRecord := &ApplicationRecord{
		TaskID:        req.TaskID,
		ConsignmentID: req.ConsignmentID,
		ServiceURL:    req.ServiceURL,
		Data:          req.Data,
		Status:        "PENDING",
	}

	if err := s.store.CreateOrUpdate(appRecord); err != nil {
		return fmt.Errorf("failed to store application: %w", err)
	}

	slog.InfoContext(ctx, "application created",
		"taskID", req.TaskID,
		"consignmentID", req.ConsignmentID)

	return nil
}

// GetApplications returns all applications (optionally filtered by status)
func (s *ogaService) GetApplications(ctx context.Context, status string) ([]Application, error) {
	var records []ApplicationRecord
	var err error

	if status != "" {
		records, err = s.store.GetByStatus(status)
	} else {
		records, err = s.store.GetAll()
	}

	if err != nil {
		return nil, err
	}

	applications := make([]Application, len(records))
	for i, record := range records {
		applications[i] = Application{
			TaskID:        record.TaskID,
			ConsignmentID: record.ConsignmentID,
			ServiceURL:    record.ServiceURL,
			Data:          record.Data,
			Status:        record.Status,
			ReviewerNotes: record.ReviewerNotes,
			ReviewedAt:    record.ReviewedAt,
			CreatedAt:     record.CreatedAt,
			UpdatedAt:     record.UpdatedAt,
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
		ServiceURL:    record.ServiceURL,
		Data:          record.Data,
		Status:        record.Status,
		ReviewerNotes: record.ReviewerNotes,
		ReviewedAt:    record.ReviewedAt,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}, nil
}

// ReviewApplication approves or rejects an application and sends response back to service
func (s *ogaService) ReviewApplication(ctx context.Context, taskID uuid.UUID, decision string, reviewerNotes string) error {
	// Get the application to retrieve service URL and consignment ID
	app, err := s.GetApplication(ctx, taskID)
	if err != nil {
		return err
	}

	// Update status in database
	var status string
	switch decision {
	case "APPROVED", "APPROVE":
		status = "APPROVED"
	case "REJECTED", "REJECT":
		status = "REJECTED"
	default:
		return fmt.Errorf("invalid decision: %s (must be APPROVED or REJECTED)", decision)
	}

	if err := s.store.UpdateStatus(taskID, status, reviewerNotes); err != nil {
		return fmt.Errorf("failed to update application status: %w", err)
	}

	// Prepare response payload for the service
	response := TaskResponse{
		TaskID:        app.TaskID,
		ConsignmentID: app.ConsignmentID,
		Payload: map[string]interface{}{
			"action": "OGA_VERIFICATION",
			"content": map[string]interface{}{
				"decision":      decision,
				"reviewerNotes": reviewerNotes,
				"reviewedAt":    time.Now().Format(time.RFC3339),
			},
		},
	}

	// Send response back to the service
	if err := s.sendToService(ctx, app.ServiceURL, response); err != nil {
		slog.ErrorContext(ctx, "failed to send response to service",
			"taskID", taskID,
			"serviceURL", app.ServiceURL,
			"error", err)
		return fmt.Errorf("failed to send response to service: %w", err)
	}

	slog.InfoContext(ctx, "application reviewed and response sent",
		"taskID", taskID,
		"decision", decision,
		"serviceURL", app.ServiceURL)

	return nil
}

// sendToService sends the task response to the originating service
func (s *ogaService) sendToService(ctx context.Context, serviceURL string, response TaskResponse) error {
	jsonData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serviceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("service returned status code %d", resp.StatusCode)
	}

	return nil
}

// Close closes the service and releases resources
func (s *ogaService) Close() error {
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}
