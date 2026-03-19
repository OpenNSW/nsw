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
)

func TestNotificationIntegration(t *testing.T) {
	ctx := context.Background()
	received := make(chan map[string]interface{}, 1)

	// 1. Setup Mock SMS Server
	smsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		_ = json.Unmarshal(body, &payload)
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
}
