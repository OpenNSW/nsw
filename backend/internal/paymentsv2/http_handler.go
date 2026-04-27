package paymentsv2

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

// HTTPHandler handles public HTTP requests for the Payment Service.
type HTTPHandler struct {
	service PaymentService
}

// NewHTTPHandler creates a new handler.
func NewHTTPHandler(service PaymentService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

// HandleValidateReference handles POST /api/v1/payments/:providerId/validate
// Called by gateways to query if a reference number is valid and payable.
func (h *HTTPHandler) HandleValidateReference(w http.ResponseWriter, r *http.Request) {
	// TODO: Extract providerID from URL parameters
	providerID := r.PathValue("providerId")
	if providerID == "" {
		http.Error(w, "provider ID is required in URL", http.StatusBadRequest)
		return
	}

	var req ValidateReferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	resp, err := h.service.ValidateReference(r.Context(), providerID, req)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to validate reference", "provider", providerID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// HandleWebhook handles POST /api/v1/payments/:providerID/webhook
// Called by payment gateways to notify about payment successes and failures.
func (h *HTTPHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// TODO: Extract providerID from URL parameters
	providerID := r.PathValue("providerId")
	if providerID == "" {
		http.Error(w, "provider ID is required in URL", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "request body too large or unreadable", http.StatusBadRequest)
		return
	}

	err = h.service.ProcessWebhook(r.Context(), providerID, body, r.Header)
	if err != nil {
		slog.ErrorContext(r.Context(), "webhook processing failed", "provider", providerID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status": "accepted"}`))
}
