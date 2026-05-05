package gateways

import (
	"context"
	"encoding/json"
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

func NewGovPayGateway(cfg json.RawMessage) (PaymentGateway, error) {

	var config Config
	if err := json.Unmarshal(cfg, &config); err != nil {
		return nil, err
	}

	return &GovPayGateway{
		cfg: config,
	}, nil
}

func (g *GovPayGateway) ApplyConfig(config json.RawMessage) error {
	if err := json.Unmarshal(config, &g.cfg); err != nil {
		return err
	}
	return nil
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
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func (g *GovPayGateway) ExtractReferenceNumber(ctx context.Context, referenceData json.RawMessage) (string, error) {
	var req GovPayReq

	if err := json.Unmarshal(referenceData, &req); err != nil {
		return "", err
	}

	return req.TransactionID, nil
}

func (g *GovPayGateway) HandleValidateReference(ctx context.Context, tx ValidationTransaction, reqData json.RawMessage) (*ValidationResponse, error) {
	var req GovPayReq
	if err := json.Unmarshal(reqData, &req); err != nil {
		return nil, err
	}

	resp := GovPayValidateResponse{
		TransactionID: req.TransactionID,
		SubInstID:     req.SubInstID,
		ServiceID:     req.ServiceID,
		ServiceName:   req.ServiceName,
		Message:       "Success",
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
