package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/OpenNSW/nsw/pkg/notification/internal/core"
)

type smsConfig struct {
	BaseURL  string `json:"baseURL"`
	UserName string `json:"userName"`
	Password string `json:"password"`
	SIDCode  string `json:"sidCode"`
}

type smsProvider struct {
	cfg    smsConfig
	client *http.Client
}

func NewProvider(client *http.Client) core.Provider {
	return &smsProvider{client: client}
}

func (p *smsProvider) Type() core.ChannelType {
	return core.ChannelSMS
}

func (p *smsProvider) Configure(raw json.RawMessage) error {
	if err := json.Unmarshal(raw, &p.cfg); err != nil {
		return fmt.Errorf("parse sms config: %w", err)
	}
	if p.cfg.BaseURL == "" {
		return fmt.Errorf("baseURL is required")
	}
	if !strings.HasPrefix(strings.ToLower(p.cfg.BaseURL), "https://") {
		return fmt.Errorf("baseURL must use HTTPS")
	}
	if p.cfg.UserName == "" {
		return fmt.Errorf("userName is required")
	}
	return nil
}

type smsPayload struct {
	Data        string `json:"data"`
	PhoneNumber string `json:"phoneNumber"`
	SIDCode     string `json:"sIDCode"`
	UserName    string `json:"userName"`
	Password    string `json:"password"`
}

func (p *smsProvider) Send(ctx context.Context, req core.Request) error {
	payload := smsPayload{
		Data:        req.Body,
		PhoneNumber: req.To,
		SIDCode:     p.cfg.SIDCode,
		UserName:    p.cfg.UserName,
		Password:    p.cfg.Password,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal sms payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create sms request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("sms request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.WarnContext(ctx, "close sms response body", "error", closeErr)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		return fmt.Errorf("sms service returned status %d: %s", resp.StatusCode, string(respBody))
	}

	slog.InfoContext(ctx, "sms sent", "recipient", req.To)
	return nil
}
