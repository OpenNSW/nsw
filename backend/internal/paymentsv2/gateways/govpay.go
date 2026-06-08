package gateways

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type Config struct {
	BaseURL string
}

type GovPayReq struct {
	TransactionID string `json:"transactionId"`
	SubInstID     string `json:"subinstId"`
	ServiceID     string `json:"serviceId"`
	ServiceName   string `json:"serviceName"`
	Data          []struct {
		Seq       string `json:"seq"`
		ParamName string `json:"paramName"`
		Value     string `json:"value"`
	} `json:"data"`
}

type GovPayValidateResponse struct {
	TransactionID string `json:"transactionId"`
	SubInstID     string `json:"subinstId"`
	ServiceID     string `json:"serviceId"`
	ServiceName   string `json:"serviceName"`
	Message       string `json:"message"`
}

type GovPayGateway struct {
	cfg Config
}

// NewGovPayGateway satisfies gateways.Factory: it constructs a fully configured
// GovPayGateway from its raw config.
func NewGovPayGateway(cfg json.RawMessage) (PaymentGateway, error) {
	var config Config
	if err := json.Unmarshal(cfg, &config); err != nil {
		return nil, err
	}

	return &GovPayGateway{
		cfg: config,
	}, nil
}

func (g *GovPayGateway) GetFlowType() InteractionType {
	return FlowTypeInstruction
}

func (g *GovPayGateway) CreateSession(ctx context.Context, req SessionRequest) (*SessionResponse, error) {
	return &SessionResponse{
		Type:         FlowTypeInstruction,
		Instructions: "Please pay using your bank application. Enter the provided reference number in the bill payment section of your app.",
	}, nil
}

func (g *GovPayGateway) ParseWebhook(ctx context.Context, body []byte, headers map[string][]string) (*WebhookPayload, error) {
	// Capture the raw status string (embedded field is shadowed for JSON decoding)
	// so we can normalize GovPay's vocabulary instead of casting it blindly.
	var raw struct {
		WebhookPayload
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	status, err := mapGovPayStatus(raw.Status)
	if err != nil {
		return nil, err
	}

	payload := raw.WebhookPayload
	payload.Status = status
	return &payload, nil
}

// mapGovPayStatus normalizes GovPay's status vocabulary into the canonical
// WebhookStatus. Unknown values are rejected rather than silently stored.
func mapGovPayStatus(raw string) (WebhookStatus, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "SUCCESS", "PAID", "COMPLETED":
		return WebhookStatusSuccess, nil
	case "FAILED", "DECLINED", "REJECTED":
		return WebhookStatusFailed, nil
	case "PENDING", "INITIATED":
		return WebhookStatusPending, nil
	default:
		return "", fmt.Errorf("govpay status %q: %w", raw, ErrUnsupportedWebhookStatus)
	}
}

func (g *GovPayGateway) ExtractReferenceNumber(ctx context.Context, referenceData json.RawMessage) (string, error) {
	var req GovPayReq

	if err := json.Unmarshal(referenceData, &req); err != nil {
		return "", err
	}

	if req.TransactionID == "" {
		return "", fmt.Errorf("transactionId is missing in validation request")
	}

	return req.TransactionID, nil
}

func (g *GovPayGateway) HandleValidateReference(ctx context.Context, tx *ValidationTransaction, isPayable bool, reqData json.RawMessage) (*ValidationResponse, error) {
	var req GovPayReq
	if err := json.Unmarshal(reqData, &req); err != nil {
		return nil, err
	}

	message := "Reference number is invalid, already settled, or expired"
	if isPayable {
		message = "Success"
	}

	resp := GovPayValidateResponse{
		TransactionID: req.TransactionID,
		SubInstID:     req.SubInstID,
		ServiceID:     req.ServiceID,
		ServiceName:   req.ServiceName,
		Message:       message,
	}

	payload, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	return &ValidationResponse{
		Payload:    payload,
		HTTPStatus: 200,
	}, nil
}
