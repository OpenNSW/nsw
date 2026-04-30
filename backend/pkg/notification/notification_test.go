package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenNSW/nsw/pkg/notification/internal/core"
)

// mockProvider is a test double that implements core.Provider.
type mockProvider struct {
	channelType core.ChannelType
	configureFn func(json.RawMessage) error
	sendFn      func(context.Context, core.Request) error
}

func (m *mockProvider) Type() core.ChannelType { return m.channelType }

func (m *mockProvider) Configure(cfg json.RawMessage) error {
	if m.configureFn != nil {
		return m.configureFn(cfg)
	}
	return nil
}

func (m *mockProvider) Send(ctx context.Context, req core.Request) error {
	if m.sendFn != nil {
		return m.sendFn(ctx, req)
	}
	return nil
}

func writeConfig(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	path := filepath.Join(t.TempDir(), "notification.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestNewManager_RoutesChannels(t *testing.T) {
	smsCalled := false
	emailCalled := false

	path := writeConfig(t, map[string]any{
		"sms":   map[string]string{"baseURL": "https://sms.example.com", "userName": "u"},
		"email": map[string]string{"baseURL": "https://email.example.com"},
	})

	providers := []core.Provider{
		&mockProvider{
			channelType: core.ChannelSMS,
			sendFn: func(_ context.Context, _ core.Request) error {
				smsCalled = true
				return nil
			},
		},
		&mockProvider{
			channelType: core.ChannelEmail,
			sendFn: func(_ context.Context, _ core.Request) error {
				emailCalled = true
				return nil
			},
		},
	}

	m, err := newManager(path, providers, http.DefaultClient)
	if err != nil {
		t.Fatalf("newManager: %v", err)
	}

	ctx := context.Background()
	if err := m.Send(ctx, core.Request{Channel: core.ChannelSMS, To: "+61400000000", Body: "hi"}); err != nil {
		t.Fatalf("SMS Send: %v", err)
	}
	if err := m.Send(ctx, core.Request{Channel: core.ChannelEmail, To: "u@example.com", Body: "hi"}); err != nil {
		t.Fatalf("Email Send: %v", err)
	}

	if !smsCalled {
		t.Error("expected SMS provider called")
	}
	if !emailCalled {
		t.Error("expected email provider called")
	}
}

func TestNewManager_UnsupportedChannel(t *testing.T) {
	path := writeConfig(t, map[string]any{})
	m, err := newManager(path, []core.Provider{}, http.DefaultClient)
	if err != nil {
		t.Fatalf("newManager: %v", err)
	}

	err = m.Send(context.Background(), core.Request{Channel: "push", To: "+61400000000", Body: "hi"})
	if err == nil {
		t.Fatal("expected error for unsupported channel")
	}
	if !strings.Contains(err.Error(), "unsupported channel") {
		t.Errorf("error %q does not mention unsupported channel", err.Error())
	}
}

func TestNewManager_ConfigureError(t *testing.T) {
	path := writeConfig(t, map[string]any{"sms": map[string]string{"x": "y"}})

	providers := []core.Provider{
		&mockProvider{
			channelType: core.ChannelSMS,
			configureFn: func(_ json.RawMessage) error {
				return errors.New("bad config")
			},
		},
	}

	_, err := newManager(path, providers, http.DefaultClient)
	if err == nil {
		t.Fatal("expected error from Configure, got nil")
	}
	if !strings.Contains(err.Error(), "configure sms provider") {
		t.Errorf("error %q missing expected prefix", err.Error())
	}
}

func TestNewManager_MissingConfigFile(t *testing.T) {
	_, err := newManager("/nonexistent/path.json", nil, http.DefaultClient)
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestNewManager_MalformedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := newManager(path, []core.Provider{}, http.DefaultClient)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestNewManager_EnvExpansion(t *testing.T) {
	t.Setenv("TEST_NOTIF_TOKEN", "secret")

	var captured json.RawMessage
	path := writeConfig(t, map[string]any{
		"email": map[string]string{"baseURL": "https://email.example.com", "token": "${TEST_NOTIF_TOKEN}"},
	})

	providers := []core.Provider{
		&mockProvider{
			channelType: core.ChannelEmail,
			configureFn: func(raw json.RawMessage) error {
				captured = raw
				return nil
			},
		},
	}

	if _, err := newManager(path, providers, http.DefaultClient); err != nil {
		t.Fatalf("newManager: %v", err)
	}

	if !strings.Contains(string(captured), "secret") {
		t.Errorf("config %s does not contain expanded token", string(captured))
	}
}

func TestNewManager_MissingEnvVar(t *testing.T) {
	os.Unsetenv("DEFINITELY_MISSING_VAR_XYZ")
	path := writeConfig(t, map[string]any{
		"email": map[string]string{"token": "${DEFINITELY_MISSING_VAR_XYZ}"},
	})

	_, err := newManager(path, []core.Provider{}, http.DefaultClient)
	if err == nil {
		t.Fatal("expected error for missing env var")
	}
	if !strings.Contains(err.Error(), "DEFINITELY_MISSING_VAR_XYZ") {
		t.Errorf("error %q should name the missing variable", err.Error())
	}
}

func TestSend_ValidationError(t *testing.T) {
	path := writeConfig(t, map[string]any{})
	m, err := newManager(path, []core.Provider{}, http.DefaultClient)
	if err != nil {
		t.Fatalf("newManager: %v", err)
	}

	tests := []struct {
		name string
		req  core.Request
	}{
		{"missing channel", core.Request{To: "u@example.com", Body: "hi"}},
		{"missing to", core.Request{Channel: core.ChannelEmail, Body: "hi"}},
		{"missing body", core.Request{Channel: core.ChannelEmail, To: "u@example.com"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := m.Send(context.Background(), tt.req); err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

func TestSend_PropagatesProviderError(t *testing.T) {
	sentinel := errors.New("gateway down")
	path := writeConfig(t, map[string]any{"sms": map[string]string{}})

	providers := []core.Provider{
		&mockProvider{
			channelType: core.ChannelSMS,
			sendFn: func(_ context.Context, _ core.Request) error {
				return sentinel
			},
		},
	}

	m, err := newManager(path, providers, http.DefaultClient)
	if err != nil {
		t.Fatalf("newManager: %v", err)
	}

	err = m.Send(context.Background(), core.Request{Channel: core.ChannelSMS, To: "+61400000000", Body: "hi"})
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}

func TestSend_ForwardsRequest(t *testing.T) {
	want := core.Request{
		Channel:  core.ChannelEmail,
		To:       "user@example.com",
		Subject:  "Hello",
		Body:     "World",
		HTMLBody: "<p>World</p>",
	}

	var got core.Request
	path := writeConfig(t, map[string]any{"email": map[string]string{}})

	providers := []core.Provider{
		&mockProvider{
			channelType: core.ChannelEmail,
			sendFn: func(_ context.Context, req core.Request) error {
				got = req
				return nil
			},
		},
	}

	m, err := newManager(path, providers, http.DefaultClient)
	if err != nil {
		t.Fatalf("newManager: %v", err)
	}

	if err := m.Send(context.Background(), want); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got != want {
		t.Errorf("forwarded request = %+v, want %+v", got, want)
	}
}

func TestNewManager_ProviderNotInConfig_NotRegistered(t *testing.T) {
	// Email provider present but no "email" key in config — should not be registered
	path := writeConfig(t, map[string]any{"sms": map[string]string{}})

	smsCalled := false
	providers := []core.Provider{
		&mockProvider{channelType: core.ChannelSMS, sendFn: func(_ context.Context, _ core.Request) error {
			smsCalled = true
			return nil
		}},
		&mockProvider{channelType: core.ChannelEmail, sendFn: func(_ context.Context, _ core.Request) error {
			return fmt.Errorf("should not be called")
		}},
	}

	m, err := newManager(path, providers, http.DefaultClient)
	if err != nil {
		t.Fatalf("newManager: %v", err)
	}

	_ = m.Send(context.Background(), core.Request{Channel: core.ChannelSMS, To: "+61400000000", Body: "hi"})
	if !smsCalled {
		t.Error("expected SMS provider called")
	}

	err = m.Send(context.Background(), core.Request{Channel: core.ChannelEmail, To: "u@example.com", Body: "hi"})
	if err == nil || !strings.Contains(err.Error(), "unsupported channel") {
		t.Errorf("expected unsupported channel error, got %v", err)
	}
}
