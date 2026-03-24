package notification_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenNSW/nsw/pkg/notification"
	"github.com/OpenNSW/nsw/pkg/notification/channels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationManager(t *testing.T) {
	// Setup mock SMS server
	smsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer smsServer.Close()

	// Setup temporary template directory
	tmpDir, err := os.MkdirTemp("", "notification-templates-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	smsTmplDir := filepath.Join(tmpDir, "sms")
	waTmplDir := filepath.Join(tmpDir, "whatsapp")

	require.NoError(t, os.MkdirAll(smsTmplDir, 0755))
	require.NoError(t, os.MkdirAll(waTmplDir, 0755))

	manager := notification.NewManager()

	// Register channels
	smsChannel := channels.NewGovSMSChannel(channels.GovSMSConfig{
		TemplateRoot: smsTmplDir,
		UserName:     "testuser",
		Password:     "testpass",
		SIDCode:      "TESTSID",
		BaseURL:      smsServer.URL,
	})

	manager.RegisterSMSChannel(smsChannel)

	ctx := context.Background()

	t.Run("SendSMS", func(t *testing.T) {
		payload := notification.SMSPayload{
			Recipients: []string{"+1234567890"},
			Body:       "Hello SMS!",
		}
		manager.SendSMS(ctx, payload)
		time.Sleep(10 * time.Millisecond)
	})
}
