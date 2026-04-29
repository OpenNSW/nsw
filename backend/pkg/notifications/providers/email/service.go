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
	"time"

	"github.com/OpenNSW/nsw/pkg/notifications"
)

// Config holds configuration for the external email HTTP service.
type Config struct {
	BaseURL    string       `json:"baseURL"`
	Token      string       `json:"token"`
	HTTPClient *http.Client `json:"-"`
}

// Provider sends email via an external HTTP email service.
type Provider struct {
	config Config
}

func New(config Config) *Provider {
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Provider{config: config}
}

func (p *Provider) Type() notifications.ChannelType {
	return notifications.ChannelEmail
}

type serviceEmailPayload struct {
	To       string `json:"to"`
	Subject  string `json:"subject"`
	TextBody string `json:"text_body"`
	HTMLBody string `json:"html_body,omitempty"`
}

func (p *Provider) Send(ctx context.Context, req notifications.Request) error {
	if !strings.HasPrefix(strings.ToLower(p.config.BaseURL), "https://") {
		return fmt.Errorf("insecure email service BaseURL: HTTPS is required to protect credentials")
	}

	payload := serviceEmailPayload{
		To:       req.To,
		Subject:  req.Subject,
		TextBody: req.Body,
		HTMLBody: req.HTMLBody,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	endpoint, err := url.JoinPath(p.config.BaseURL, "emails")
	if err != nil {
		return fmt.Errorf("failed to construct email service URL: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create email request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.config.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.config.Token)
	}

	resp, err := p.config.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("email service request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.WarnContext(ctx, "failed to close email response body", "error", err)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		return fmt.Errorf("email service returned status %d: %s", resp.StatusCode, string(body))
	}

	slog.InfoContext(ctx, "email sent", "recipient", req.To)
	return nil
}
