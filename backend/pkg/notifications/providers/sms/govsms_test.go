package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/OpenNSW/nsw/pkg/notifications"
)

func newTLSProvider(t *testing.T, h http.HandlerFunc) (*GovSMSProvider, *httptest.Server) {
	t.Helper()
	srv := httptest.NewTLSServer(h)
	t.Cleanup(srv.Close)
	return NewGovSMS(GovSMSConfig{
		UserName:   "user",
		Password:   "pass",
		SIDCode:    "SID001",
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	}), srv
}

func TestGovSMSProvider_Type(t *testing.T) {
	p := NewGovSMS(GovSMSConfig{BaseURL: "https://example.com"})
	if p.Type() != notifications.ChannelSMS {
		t.Errorf("Type() = %q, want %q", p.Type(), notifications.ChannelSMS)
	}
}

func TestNewGovSMS_DefaultsHTTPClient(t *testing.T) {
	p := NewGovSMS(GovSMSConfig{BaseURL: "https://example.com"})
	if p.config.HTTPClient == nil {
		t.Fatal("expected default HTTPClient to be set, got nil")
	}
	if p.config.HTTPClient.Timeout != 10*time.Second {
		t.Errorf("HTTPClient.Timeout = %v, want %v", p.config.HTTPClient.Timeout, 10*time.Second)
	}
}

func TestGovSMSProvider_Send(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		handler     http.HandlerFunc
		cfgOverride *GovSMSConfig
		wantErr     string
	}{
		{
			name: "success 200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		},
		{
			name: "http scheme rejected",
			cfgOverride: &GovSMSConfig{
				BaseURL:    "http://example.com",
				HTTPClient: &http.Client{},
			},
			wantErr: "HTTPS is required",
		},
		{
			name: "server returns 400",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("bad request body"))
			},
			wantErr: "400",
		},
		{
			name: "server returns 500",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("internal error"))
			},
			wantErr: "500",
		},
		{
			name: "content-type header is application/json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if ct := r.Header.Get("Content-Type"); ct != "application/json" {
					http.Error(w, fmt.Sprintf("bad Content-Type: %s", ct), http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusOK)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p *GovSMSProvider
			if tt.cfgOverride != nil {
				p = NewGovSMS(*tt.cfgOverride)
			} else {
				p, _ = newTLSProvider(t, tt.handler)
			}

			err := p.Send(ctx, notifications.Request{
				Channel: notifications.ChannelSMS,
				To:      "+61400000000",
				Body:    "test message",
			})

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestGovSMSProvider_Send_PayloadFields(t *testing.T) {
	var captured govSMSRequestPayload

	p, _ := newTLSProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	req := notifications.Request{
		Channel: notifications.ChannelSMS,
		To:      "+61400000001",
		Body:    "hello world",
	}
	if err := p.Send(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captured.Data != "hello world" {
		t.Errorf("Data = %q, want %q", captured.Data, "hello world")
	}
	if captured.PhoneNumber != "+61400000001" {
		t.Errorf("PhoneNumber = %q, want %q", captured.PhoneNumber, "+61400000001")
	}
	if captured.SIDCode != "SID001" {
		t.Errorf("SIDCode = %q, want %q", captured.SIDCode, "SID001")
	}
	if captured.UserName != "user" {
		t.Errorf("UserName = %q, want %q", captured.UserName, "user")
	}
	if captured.Password != "pass" {
		t.Errorf("Password = %q, want %q", captured.Password, "pass")
	}
}

func TestGovSMSProvider_Send_ErrorBodyIncludedInMessage(t *testing.T) {
	p, _ := newTLSProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, "invalid credentials")
	})

	err := p.Send(context.Background(), notifications.Request{
		Channel: notifications.ChannelSMS,
		To:      "+61400000000",
		Body:    "hi",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error %q does not contain status code 401", err.Error())
	}
	if !strings.Contains(err.Error(), "invalid credentials") {
		t.Errorf("error %q does not contain response body", err.Error())
	}
}

func TestGovSMSProvider_Send_ContextCancellation(t *testing.T) {
	p, _ := newTLSProvider(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		w.WriteHeader(http.StatusOK)
	})

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := p.Send(cancelCtx, notifications.Request{
		Channel: notifications.ChannelSMS,
		To:      "+61400000000",
		Body:    "hi",
	})
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if !strings.Contains(err.Error(), "GovSMS request failed") {
		t.Errorf("error %q does not contain %q", err.Error(), "GovSMS request failed")
	}
}
