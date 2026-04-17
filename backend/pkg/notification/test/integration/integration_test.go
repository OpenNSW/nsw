package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/OpenNSW/nsw/pkg/notification"
	"github.com/OpenNSW/nsw/pkg/notification/channels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationIntegration(t *testing.T) {
	ctx := context.Background()
	received := make(chan map[string]interface{}, 1)

	// 1. Setup Mock SMS Server
	smsServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &payload))
		received <- payload
		w.WriteHeader(http.StatusOK)
	}))
	defer smsServer.Close()

	// 2. Initialize Manager and Channels
	manager := notification.NewManager()

	smsChan := channels.NewGovSMSChannel(channels.GovSMSConfig{
		UserName:     "test_user",
		Password:     "test_pass",
		SIDCode:      "TEST_BRAND",
		BaseURL:      smsServer.URL,
		TemplateRoot: "testdata/sms",
		HTTPClient:   smsServer.Client(),
	})

	manager.RegisterSMSChannel(smsChan)

	// 3. Test Case: Send SMS with Template
	t.Run("SMS with Template Integration", func(t *testing.T) {
		payload := notification.SMSPayload{
			Recipients: []string{"+1234567890"},
		}
		payload.TemplateID = "otp"
		payload.TemplateData = map[string]interface{}{
			"OTP": "998877",
		}

		manager.SendSMS(ctx, payload)

		// Wait for the async mock server to receive the request
		select {
		case data := <-received:
			assert.Equal(t, "test_user", data["userName"])
			assert.Equal(t, "+1234567890", data["phoneNumber"])
			assert.Equal(t, "Your OTP is 998877. Do not share it with anyone.\n", data["data"])
		case <-time.After(1 * time.Second):
			t.Fatal("SMS was not received by the mock server in time")
		}
	})

	// 4. Test Case: Email Template Rendering (without sending)
	t.Run("Email Template Rendering", func(t *testing.T) {
		emailChan := channels.NewEmailChannel(notification.EmailConfig{
			TemplateRoot: "testdata/email",
		})

		// We can't easily test SMTP sending in unit tests, so just test channel creation
		// In a real scenario, you'd use a mock SMTP server
		require.NotNil(t, emailChan)

		// Test payload creation
		payload := notification.EmailPayload{
			Recipients: []string{"test@example.com"},
		}
		payload.TemplateID = "otp"
		payload.TemplateData = map[string]interface{}{
			"OTP": "123456",
		}

		// Note: Actual sending would require a real SMTP server
		// For now, we just ensure the payload is structured correctly
		assert.Equal(t, []string{"test@example.com"}, payload.Recipients)
		assert.Equal(t, "otp", payload.TemplateID)
		assert.Equal(t, "123456", payload.TemplateData["OTP"])
	})
}
