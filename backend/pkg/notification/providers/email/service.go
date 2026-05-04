package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/OpenNSW/nsw/pkg/notification/internal/core"
)

type emailConfig struct {
	BaseURL string `json:"baseURL"`
	Token   string `json:"token"`
}

type emailProvider struct {
	cfg    emailConfig
	client *http.Client
}

func NewProvider(client *http.Client) core.Provider {
	return &emailProvider{client: client}
}

func (p *emailProvider) Type() core.ChannelType {
	return core.ChannelEmail
}

func (p *emailProvider) Configure(raw json.RawMessage) error {
	if err := json.Unmarshal(raw, &p.cfg); err != nil {
		return fmt.Errorf("parse email config: %w", err)
	}
	if p.cfg.BaseURL == "" {
		return fmt.Errorf("baseURL is required")
	}
	if !strings.HasPrefix(strings.ToLower(p.cfg.BaseURL), "https://") {
		return fmt.Errorf("baseURL must use HTTPS")
	}
	return nil
}

type emailPayload struct {
	To       string `json:"to"`
	Subject  string `json:"subject"`
	TextBody string `json:"text_body"`
	HTMLBody string `json:"html_body,omitempty"`
}

func (p *emailProvider) Send(ctx context.Context, req core.Request) error {
	payload := emailPayload{
		To:       req.To,
		Subject:  req.Subject,
		TextBody: req.Body,
		HTMLBody: req.HTMLBody,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal email payload: %w", err)
	}

	endpoint, err := url.JoinPath(p.cfg.BaseURL, "emails")
	if err != nil {
		return fmt.Errorf("construct email URL: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create email request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.cfg.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.cfg.Token)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("email service request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.WarnContext(ctx, "close email response body", "error", closeErr)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		return fmt.Errorf("email service returned status %d: %s", resp.StatusCode, string(respBody))
	}

	slog.InfoContext(ctx, "email sent", "recipient", req.To)
	return nil
}
