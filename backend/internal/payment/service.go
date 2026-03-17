package payment

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	"github.com/OpenNSW/nsw/internal/task/plugin"
	"github.com/OpenNSW/nsw/internal/task/plugin/gateway"
	"github.com/OpenNSW/nsw/internal/task/plugin/payment_types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service handles all business logic routing for payment webhooks and inquiries.
// It acts as the bridge between external payment gateways and the internal TaskManager.
type Service interface {
	// ProcessCallback handles incoming webhooks and return-redirects.
	// It uses the PaymentGateway to extract the reference, performs an outbound fetch to verify the status,
	// updates the PaymentTransactionDB, and calls tm.ExecuteTask to transition the FSM.
	ProcessCallback(ctx context.Context, provider string, r *http.Request) error

	// GetTransactionInquiry handles synchronous GET requests from gateways (e.g., LankaPay).
	// It securely fetches the transaction from the DB and uses the PaymentGateway to format
	// the response payload exactly as the provider expects.
	GetTransactionInquiry(ctx context.Context, provider string, reference string) (any, error)
}

type service struct {
	gateways *gateway.Registry
	repo     payment_types.PaymentRepository
	tm       taskManager.TaskManager
	db       *gorm.DB
}

func NewService(gateways *gateway.Registry, repo payment_types.PaymentRepository, tm taskManager.TaskManager, db *gorm.DB) Service {
	return &service{
		gateways: gateways,
		repo:     repo,
		tm:       tm,
		db:       db,
	}
}

func (s *service) GetTransactionInquiry(ctx context.Context, provider string, reference string) (any, error) {
	gw, err := s.gateways.Get(provider)
	if err != nil {
		return nil, fmt.Errorf("unsupported provider: %w", err)
	}

	trx, err := s.repo.GetTransactionByReference(ctx, reference, false)
	if err != nil {
		return nil, fmt.Errorf("transaction not found: %w", err)
	}

	resp, err := gw.FormatInquiryResponse(trx)
	if err != nil {
		return nil, fmt.Errorf("failed to format inquiry response: %w", err)
	}

	return resp, nil
}

func (s *service) ProcessCallback(ctx context.Context, provider string, r *http.Request) error {
	gw, err := s.gateways.Get(provider)
	if err != nil {
		return fmt.Errorf("unsupported provider: %w", err)
	}

	// Unpack the unverified reference number from the incoming payload
	refNo, err := gw.ExtractReference(r)
	if err != nil {
		return fmt.Errorf("failed to extract reference from webhook: %w", err)
	}

	// Outbound Fetch: Ask the gateway for the absolute truth
	result, err := gw.GetPaymentInfo(ctx, refNo)
	if err != nil {
		return fmt.Errorf("failed to fetch payment info from provider: %w", err)
	}

	// Determine the required FSM action based on the verified result
	action := plugin.PaymentActionFailed
	if result.Status == "SUCCESS" {
		action = plugin.PaymentActionSuccess
	}

	// Commit the database update before executing the task FSM
	// to avoid side-effect risks (like workflow notifications) during rollbacks.
	var taskID uuid.UUID
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		transaction, err := s.repo.GetTransactionByReference(ctx, result.ReferenceNumber, true)
		if err != nil {
			return err
		}
		taskID = transaction.TaskID

		if transaction.Status != "PENDING" {
			slog.InfoContext(ctx, "transaction already processed", "reference", result.ReferenceNumber)
			return nil
		}

		dbStatus := "COMPLETED"
		if result.Status == "FAILED" {
			dbStatus = "FAILED"
		}
		if err := tx.Model(&transaction).Update("status", dbStatus).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	if taskID == uuid.Nil {
		return nil // Already processed or not found
	}

	// Now that our local state is durable, notify the Task Manager
	_, err = s.tm.ExecuteTask(ctx, taskManager.ExecuteTaskRequest{
		TaskID:  taskID,
		Payload: &plugin.ExecutionRequest{Action: action},
	})
	return err
}
