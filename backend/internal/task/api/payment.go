package api

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/OpenNSW/nsw/internal/config"
	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	"github.com/OpenNSW/nsw/internal/task/persistence"
	"github.com/OpenNSW/nsw/internal/task/plugin"
	"github.com/OpenNSW/nsw/internal/task/plugin/gateway"
	"github.com/OpenNSW/nsw/internal/task/plugin/payment_types"
	"gorm.io/gorm"
)

type PaymentHandler struct {
	tm       taskManager.TaskManager
	cfg      *config.Config
	db       *gorm.DB
	repo     payment_types.PaymentRepository
	gateways *gateway.Registry
}

func NewPaymentHandler(tm taskManager.TaskManager, cfg *config.Config, db *gorm.DB, gateways *gateway.Registry) *PaymentHandler {
	return &PaymentHandler{
		tm:       tm,
		cfg:      cfg,
		db:       db,
		repo:     persistence.NewPaymentRepository(db),
		gateways: gateways,
	}
}

func (h *PaymentHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	provider := r.PathValue("provider")
	gw, err := h.gateways.Get(provider)
	if err != nil {
		slog.Error("unsupported payment provider", "provider", provider, "error", err)
		h.writeError(w, http.StatusBadRequest, "Unsupported payment provider")
		return
	}

	result, err := gw.VerifyCallback(r)
	if err != nil {
		slog.Error("callback verification failed", "provider", provider, "error", err)
		h.writeError(w, http.StatusUnauthorized, "Invalid payload or signature")
		return
	}

	h.processPayment(w, r, result)
}

func (h *PaymentHandler) HandleMockCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.cfg.Payment.MockMode {
		h.writeError(w, http.StatusForbidden, "Mock mode disabled")
		return
	}

	gw, err := h.gateways.Get("mock")
	if err != nil {
		slog.Error("mock payment gateway not registered", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Mock gateway not configured")
		return
	}
	result, err := gw.VerifyCallback(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid mock payload")
		return
	}

	h.processPayment(w, r, result)
}

func (h *PaymentHandler) processPayment(w http.ResponseWriter, r *http.Request, req gateway.CallbackResult) {
	ctx := r.Context()
	slog.InfoContext(ctx, "received payment result", "reference", req.ReferenceNumber, "status", req.Status)

	// Validate status explicitly to avoid incorrect success processing
	var action string
	if req.Status == "SUCCESS" {
		action = plugin.PaymentActionSuccess
	} else {
		action = plugin.PaymentActionFailed
	}

	// Use a transaction to ensure atomicity between DB update and TaskManager execution
	err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		transaction, err := h.repo.GetTransactionByReference(ctx, req.ReferenceNumber, true)
		if err != nil {
			return err
		}

		// Security: Match provider to prevent spoofing
		if transaction.ProviderID != req.ProviderID {
			return fmt.Errorf("provider mismatch: expected %q, got %q", transaction.ProviderID, req.ProviderID)
		}

		if transaction.Status != "PENDING" {
			slog.InfoContext(ctx, "transaction already processed", "reference", req.ReferenceNumber, "status", transaction.Status)
			return nil // Already processed, return success to gateway
		}

		// Update DB status
		dbStatus := "COMPLETED"
		if req.Status == "FAILED" {
			dbStatus = "FAILED"
		}
		if err := tx.Model(&transaction).Update("status", dbStatus).Error; err != nil {
			return err
		}

		// Execute Task event
		execReq := taskManager.ExecuteTaskRequest{
			TaskID: transaction.TaskID,
			Payload: &plugin.ExecutionRequest{
				Action: action,
			},
		}

		_, err = h.tm.ExecuteTask(ctx, execReq)
		if err != nil {
			return fmt.Errorf("failed to transition task: %w", err)
		}

		return nil
	})

	if err != nil {
		slog.ErrorContext(ctx, "failed to process payment", "error", err)
		// Mask internal errors from public endpoint
		h.writeError(w, http.StatusInternalServerError, "An internal error occurred while processing the payment")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *PaymentHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func (h *PaymentHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

func (h *PaymentHandler) HandleTransactionInquiry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	provider := r.PathValue("provider")
	gw, err := h.gateways.Get(provider)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Unsupported provider: %s", provider))
		return
	}

	// Basic shared secret check
	apiKey := r.Header.Get("X-API-Key")
	if subtle.ConstantTimeCompare([]byte(apiKey), []byte(h.cfg.Payment.InquiryAPIKey)) != 1 {
		h.writeError(w, http.StatusUnauthorized, "Invalid API Key")
		return
	}

	reference := r.PathValue("reference")
	if reference == "" {
		h.writeError(w, http.StatusBadRequest, "Missing reference parameter")
		return
	}

	trx, err := h.repo.GetTransactionByReference(r.Context(), reference, false)
	if err != nil {
		slog.ErrorContext(r.Context(), "inquiry failed", "ref", reference, "error", err)
		h.writeError(w, http.StatusNotFound, "Transaction not found")
		return
	}

	resp, err := gw.FormatInquiryResponse(trx)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to format inquiry response", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to format inquiry response")
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}
