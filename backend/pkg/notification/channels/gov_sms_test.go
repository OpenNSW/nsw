package channels

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenNSW/nsw/pkg/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGovSMSChannel_New(t *testing.T) {
	cfg := GovSMSConfig{}
	ch := NewGovSMSChannel(cfg)
	assert.NotNil(t, ch.config.HTTPClient)
}

func TestGovSMSChannel_Send(t *testing.T) {
	ctx := context.Background()

	// Create temporary template directory
	tmpDir, err := os.MkdirTemp("", "govsms-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a dummy template
	tmplPath := filepath.Join(tmpDir, "test_tmpl.txt")
	require.NoError(t, os.WriteFile(tmplPath, []byte("Hello {{.Name}}!"), 0644))

	t.Run("Successful Send with Body", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req govSMSRequestPayload
			_ = json.Unmarshal(body, &req)

			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "Direct Body", req.Data)
			assert.Equal(t, "+12345", req.PhoneNumber)
			assert.Equal(t, "user1", req.UserName)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ch := NewGovSMSChannel(GovSMSConfig{
			UserName:   "user1",
			BaseURL:    server.URL,
			HTTPClient: server.Client(),
		})

		payload := notification.SMSPayload{
			Recipients: []string{"+12345"},
			Body:       "Direct Body",
		}

		results := ch.Send(ctx, payload)
		assert.NoError(t, results["+12345"])
	})

	t.Run("Successful Send with Template", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req govSMSRequestPayload
			_ = json.Unmarshal(body, &req)

			assert.Equal(t, "Hello World!", req.Data)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ch := NewGovSMSChannel(GovSMSConfig{
			BaseURL:      server.URL,
			TemplateRoot: tmpDir,
			HTTPClient:   server.Client(),
		})

		payload := notification.SMSPayload{
			Recipients: []string{"+12345"},
		}
		payload.TemplateID = "test_tmpl"
		payload.TemplateData = map[string]interface{}{"Name": "World"}

		results := ch.Send(ctx, payload)
		assert.NoError(t, results["+12345"])
	})

	t.Run("Failure - Provider Error 500", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		ch := NewGovSMSChannel(GovSMSConfig{
			BaseURL:    server.URL,
			HTTPClient: server.Client(),
		})
		payload := notification.SMSPayload{Recipients: []string{"+12345"}, Body: "test"}

		results := ch.Send(ctx, payload)
		assert.Error(t, results["+12345"])
		assert.Contains(t, results["+12345"].Error(), "500")
	})

	t.Run("Failure - Insecure BaseURL", func(t *testing.T) {
		ch := NewGovSMSChannel(GovSMSConfig{
			BaseURL: "http://api.sms.gov.lk",
		})
		payload := notification.SMSPayload{Recipients: []string{"+12345"}, Body: "test"}

		results := ch.Send(ctx, payload)
		assert.Error(t, results["+12345"])
		assert.Contains(t, results["+12345"].Error(), "insecure GovSMS BaseURL")
	})

	t.Run("Failure - Template Not Found", func(t *testing.T) {
		ch := NewGovSMSChannel(GovSMSConfig{
			BaseURL:      "https://api.sms.gov.lk",
			TemplateRoot: tmpDir,
		})
		payload := notification.SMSPayload{Recipients: []string{"+12345"}}
		payload.TemplateID = "non_existent"

		results := ch.Send(ctx, payload)
		assert.Error(t, results["+12345"])
		assert.Contains(t, results["+12345"].Error(), "failed to read template file")
	})

	t.Run("Multiple Recipients", func(t *testing.T) {
		callCount := 0
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ch := NewGovSMSChannel(GovSMSConfig{
			BaseURL:    server.URL,
			HTTPClient: server.Client(),
		})
		payload := notification.SMSPayload{
			Recipients: []string{"+1", "+2", "+3"},
			Body:       "test",
		}

		results := ch.Send(ctx, payload)
		assert.Len(t, results, 3)
		assert.Equal(t, 3, callCount)
		for _, err := range results {
			assert.NoError(t, err)
		}
	})
}
