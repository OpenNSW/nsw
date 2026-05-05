package paymentsv2

import (
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

// HandleValidateReference handles POST /api/v1/payments/:gatewayId/validate
// Called by gateways to query if a reference number is valid and payable.
func (h *HTTPHandler) HandleValidateReference(w http.ResponseWriter, r *http.Request) {
	gatewayID := r.PathValue("gatewayId")
	if gatewayID == "" {
		http.Error(w, "gateway ID is required in URL", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "request body too large or unreadable", http.StatusBadRequest)
		return
	}

	resp, err := h.service.ValidateReference(r.Context(), gatewayID, body)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to validate reference", "gateway", gatewayID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.HTTPStatus)
	if _, err := w.Write(resp.Payload); err != nil {
		slog.ErrorContext(r.Context(), "failed to write response", "error", err)
	}
}

// HandleWebhook handles POST /api/v1/payments/:gatewayID/webhook
// Called by payment gateways to notify about payment successes and failures.
func (h *HTTPHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	gatewayID := r.PathValue("gatewayId")
	if gatewayID == "" {
		http.Error(w, "gateway ID is required in URL", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "request body too large or unreadable", http.StatusBadRequest)
		return
	}

	err = h.service.ProcessWebhook(r.Context(), gatewayID, body, r.Header)
	if err != nil {
		slog.ErrorContext(r.Context(), "webhook processing failed", "gateway", gatewayID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status": "accepted"}`))
}
