package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/OpenNSW/nsw/pkg/notification"
)

// GovSMSConfig holds the credentials and settings for the Gov of Sri Lanka SMS provider.
type GovSMSConfig struct {
	UserName     string
	Password     string
	SIDCode      string
	BaseURL      string
	TemplateRoot string
	HTTPClient   *http.Client
}

// GovSMSChannel implements the notification.SMSChannel interface for Gov SL.
type GovSMSChannel struct {
	config GovSMSConfig
}

func NewGovSMSChannel(config GovSMSConfig) *GovSMSChannel {
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &GovSMSChannel{config: config}
}

type govSMSRequestPayload struct {
	Data        string `json:"data"`
	PhoneNumber string `json:"phoneNumber"`
	SIDCode     string `json:"sIDCode"`
	UserName    string `json:"userName"`
	Password    string `json:"password"`
}

func (s *GovSMSChannel) Send(ctx context.Context, payload notification.SMSPayload) map[string]error {
	results := make(map[string]error)

	// Security Check: Ensure BaseURL uses HTTPS to protect credentials sent in the body.
	if !strings.HasPrefix(strings.ToLower(s.config.BaseURL), "https://") {
		err := fmt.Errorf("insecure GovSMS BaseURL: HTTPS is required to protect credentials")
		for _, recipient := range payload.Recipients {
			results[recipient] = err
		}
		return results
	}

	body := payload.Body
	if payload.TemplateID != "" {
		var err error
		body, err = s.renderTemplate(payload.TemplateID, payload.TemplateData)
		if err != nil {
			renderErr := fmt.Errorf("failed to render GovSMS template: %w", err)
			for _, recipient := range payload.Recipients {
				results[recipient] = renderErr
			}
			return results
		}
	}

	for _, recipient := range payload.Recipients {
		results[recipient] = s.sendToRecipient(ctx, recipient, body)
	}

	return results
}

func (s *GovSMSChannel) sendToRecipient(ctx context.Context, recipient string, body string) error {
	reqPayload := govSMSRequestPayload{
		Data:        body,
		PhoneNumber: recipient,
		SIDCode:     s.config.SIDCode,
		UserName:    s.config.UserName,
		Password:    s.config.Password,
	}

	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal GovSMS payload: %w", err)
	}

	slog.InfoContext(ctx, "sending GovSMS", "recipient", recipient, "url", s.config.BaseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", s.config.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create GovSMS request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.config.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send GovSMS request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			slog.ErrorContext(ctx, "failed to close GovSMS response body", "error", err)
		}
	}(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GovSMS provider returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}

func (s *GovSMSChannel) renderTemplate(templateID string, data map[string]interface{}) (string, error) {
	tmplPath := filepath.Join(s.config.TemplateRoot, templateID+".txt")
	tmplContent, err := os.ReadFile(tmplPath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file %s: %w", tmplPath, err)
	}

	tmpl, err := template.New(templateID).Parse(string(tmplContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
