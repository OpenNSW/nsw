package api

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/payment"
)

type PaymentHandler struct {
	cfg     *config.Config
	service payment.Service
}

func NewPaymentHandler(cfg *config.Config, service payment.Service) *PaymentHandler {
	return &PaymentHandler{
		cfg:     cfg,
		service: service,
	}
}

func (h *PaymentHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	provider := r.PathValue("provider")
	if err := h.service.ProcessCallback(r.Context(), provider, r); err != nil {
		slog.ErrorContext(r.Context(), "callback processing failed", "provider", provider, "error", err)
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

	resp, err := h.service.GetTransactionInquiry(r.Context(), provider, reference)
	if err != nil {
		slog.ErrorContext(r.Context(), "inquiry failed", "ref", reference, "error", err)
		h.writeError(w, http.StatusNotFound, "Transaction not found or inquiry failed")
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}
