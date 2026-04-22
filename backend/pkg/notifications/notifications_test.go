package notifications_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/OpenNSW/nsw/pkg/notifications"
)

type mockProvider struct {
	channelType notifications.ChannelType
	sendFn      func(ctx context.Context, req notifications.Request) error
}

func (m *mockProvider) Type() notifications.ChannelType { return m.channelType }
func (m *mockProvider) Send(ctx context.Context, req notifications.Request) error {
	return m.sendFn(ctx, req)
}

func noop(_ context.Context, _ notifications.Request) error { return nil }

func noConfig() notifications.Config { return notifications.Config{} }

func TestNew_RegistersProviders(t *testing.T) {
	smsCalled := false
	emailCalled := false

	n := notifications.New(noConfig(),
		&mockProvider{channelType: notifications.ChannelSMS, sendFn: func(_ context.Context, _ notifications.Request) error {
			smsCalled = true
			return nil
		}},
		&mockProvider{channelType: notifications.ChannelEmail, sendFn: func(_ context.Context, _ notifications.Request) error {
			emailCalled = true
			return nil
		}},
	)

	ctx := context.Background()
	if err := n.Send(ctx, notifications.Request{Channel: notifications.ChannelSMS}); err != nil {
		t.Fatalf("SMS send: unexpected error: %v", err)
	}
	if err := n.Send(ctx, notifications.Request{Channel: notifications.ChannelEmail}); err != nil {
		t.Fatalf("email send: unexpected error: %v", err)
	}
	if !smsCalled {
		t.Error("expected SMS provider to be called")
	}
	if !emailCalled {
		t.Error("expected email provider to be called")
	}

	if err := n.Send(ctx, notifications.Request{Channel: "push"}); err == nil {
		t.Error("expected error for unregistered channel, got nil")
	}
}

func TestNew_LastProviderWins(t *testing.T) {
	firstCalled := false
	secondCalled := false

	n := notifications.New(noConfig(),
		&mockProvider{channelType: notifications.ChannelSMS, sendFn: func(_ context.Context, _ notifications.Request) error {
			firstCalled = true
			return nil
		}},
		&mockProvider{channelType: notifications.ChannelSMS, sendFn: func(_ context.Context, _ notifications.Request) error {
			secondCalled = true
			return nil
		}},
	)

	if err := n.Send(context.Background(), notifications.Request{Channel: notifications.ChannelSMS}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if firstCalled {
		t.Error("expected first provider to be replaced, but it was called")
	}
	if !secondCalled {
		t.Error("expected second provider to be called")
	}
}

func TestManager_Send(t *testing.T) {
	sentinelErr := errors.New("gateway down")

	tests := []struct {
		name        string
		providers   []notifications.Provider
		req         notifications.Request
		wantErr     bool
		errContains string
	}{
		{
			name:      "routes to sms provider",
			providers: []notifications.Provider{&mockProvider{channelType: notifications.ChannelSMS, sendFn: noop}},
			req:       notifications.Request{Channel: notifications.ChannelSMS},
		},
		{
			name:      "routes to email provider",
			providers: []notifications.Provider{&mockProvider{channelType: notifications.ChannelEmail, sendFn: noop}},
			req:       notifications.Request{Channel: notifications.ChannelEmail},
		},
		{
			name: "propagates provider error",
			providers: []notifications.Provider{
				&mockProvider{channelType: notifications.ChannelSMS, sendFn: func(_ context.Context, _ notifications.Request) error {
					return sentinelErr
				}},
			},
			req:         notifications.Request{Channel: notifications.ChannelSMS},
			wantErr:     true,
			errContains: "gateway down",
		},
		{
			name:        "no provider registered",
			providers:   nil,
			req:         notifications.Request{Channel: notifications.ChannelSMS},
			wantErr:     true,
			errContains: "no provider registered for channel: sms",
		},
		{
			name: "unknown channel constant",
			providers: []notifications.Provider{
				&mockProvider{channelType: notifications.ChannelSMS, sendFn: noop},
				&mockProvider{channelType: notifications.ChannelEmail, sendFn: noop},
			},
			req:         notifications.Request{Channel: "push"},
			wantErr:     true,
			errContains: "no provider registered for channel: push",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := notifications.New(noConfig(), tt.providers...)
			err := n.Send(context.Background(), tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestManager_Send_ForwardsRequest(t *testing.T) {
	want := notifications.Request{
		Channel:  notifications.ChannelEmail,
		To:       "user@example.com",
		Subject:  "Hello",
		Body:     "World",
		HTMLBody: "<p>World</p>",
	}

	var got notifications.Request
	n := notifications.New(noConfig(), &mockProvider{
		channelType: notifications.ChannelEmail,
		sendFn: func(_ context.Context, req notifications.Request) error {
			got = req
			return nil
		},
	})

	if err := n.Send(context.Background(), want); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("forwarded request = %+v, want %+v", got, want)
	}
}

func TestManager_SendEmail_RawContent(t *testing.T) {
	var got notifications.Request
	n := notifications.New(noConfig(), &mockProvider{
		channelType: notifications.ChannelEmail,
		sendFn: func(_ context.Context, req notifications.Request) error {
			got = req
			return nil
		},
	})

	req := notifications.EmailRequest{
		To:       "user@example.com",
		Subject:  "Hello",
		Body:     "Plain text",
		HTMLBody: "<p>HTML</p>",
	}
	if err := n.SendEmail(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Channel != notifications.ChannelEmail {
		t.Errorf("Channel = %q, want %q", got.Channel, notifications.ChannelEmail)
	}
	if got.To != req.To {
		t.Errorf("To = %q, want %q", got.To, req.To)
	}
	if got.Subject != req.Subject {
		t.Errorf("Subject = %q, want %q", got.Subject, req.Subject)
	}
	if got.Body != req.Body {
		t.Errorf("Body = %q, want %q", got.Body, req.Body)
	}
	if got.HTMLBody != req.HTMLBody {
		t.Errorf("HTMLBody = %q, want %q", got.HTMLBody, req.HTMLBody)
	}
}

func TestManager_SendEmail_WithTemplate(t *testing.T) {
	var got notifications.Request
	n := notifications.New(
		notifications.Config{EmailTemplateRoot: "testdata/email"},
		&mockProvider{
			channelType: notifications.ChannelEmail,
			sendFn: func(_ context.Context, req notifications.Request) error {
				got = req
				return nil
			},
		},
	)

	err := n.SendEmail(context.Background(), notifications.EmailRequest{
		To:           "user@example.com",
		TemplateID:   "otp",
		TemplateData: map[string]any{"OTP": "123456"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Subject != "Your OTP Code" {
		t.Errorf("Subject = %q, want %q", got.Subject, "Your OTP Code")
	}
	if !strings.Contains(got.Body, "123456") {
		t.Errorf("plain Body %q does not contain OTP", got.Body)
	}
	if !strings.Contains(got.HTMLBody, "123456") {
		t.Errorf("HTMLBody %q does not contain OTP", got.HTMLBody)
	}
	if !strings.Contains(got.HTMLBody, "<strong>") {
		t.Errorf("HTMLBody %q expected HTML markup", got.HTMLBody)
	}
}

func TestManager_SendEmail_MissingTemplate(t *testing.T) {
	n := notifications.New(
		notifications.Config{EmailTemplateRoot: "testdata/email"},
		&mockProvider{channelType: notifications.ChannelEmail, sendFn: noop},
	)

	err := n.SendEmail(context.Background(), notifications.EmailRequest{
		To:         "user@example.com",
		TemplateID: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for missing template, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error %q does not reference template id", err.Error())
	}
}

func TestManager_SendSMS_RawContent(t *testing.T) {
	var got notifications.Request
	n := notifications.New(noConfig(), &mockProvider{
		channelType: notifications.ChannelSMS,
		sendFn: func(_ context.Context, req notifications.Request) error {
			got = req
			return nil
		},
	})

	req := notifications.SMSRequest{
		To:   "+61400000000",
		Body: "Your code is 999",
	}
	if err := n.SendSMS(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Channel != notifications.ChannelSMS {
		t.Errorf("Channel = %q, want %q", got.Channel, notifications.ChannelSMS)
	}
	if got.To != req.To {
		t.Errorf("To = %q, want %q", got.To, req.To)
	}
	if got.Body != req.Body {
		t.Errorf("Body = %q, want %q", got.Body, req.Body)
	}
}

func TestManager_SendSMS_WithTemplate(t *testing.T) {
	var got notifications.Request
	n := notifications.New(
		notifications.Config{SMSTemplateRoot: "testdata/sms"},
		&mockProvider{
			channelType: notifications.ChannelSMS,
			sendFn: func(_ context.Context, req notifications.Request) error {
				got = req
				return nil
			},
		},
	)

	err := n.SendSMS(context.Background(), notifications.SMSRequest{
		To:           "+61400000000",
		TemplateID:   "otp",
		TemplateData: map[string]any{"OTP": "654321"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got.Body, "654321") {
		t.Errorf("Body %q does not contain OTP", got.Body)
	}
}

func TestManager_SendSMS_MissingTemplate(t *testing.T) {
	n := notifications.New(
		notifications.Config{SMSTemplateRoot: "testdata/sms"},
		&mockProvider{channelType: notifications.ChannelSMS, sendFn: noop},
	)

	err := n.SendSMS(context.Background(), notifications.SMSRequest{
		To:         "+61400000000",
		TemplateID: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for missing template, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error %q does not reference template id", err.Error())
	}
}
