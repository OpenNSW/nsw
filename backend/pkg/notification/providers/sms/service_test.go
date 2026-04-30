package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OpenNSW/nsw/pkg/notification/internal/core"
)

func newTLSProvider(t *testing.T, h http.HandlerFunc) (*smsProvider, *httptest.Server) {
	t.Helper()
	srv := httptest.NewTLSServer(h)
	t.Cleanup(srv.Close)
	p := NewProvider(srv.Client()).(*smsProvider)
	raw, _ := json.Marshal(smsConfig{
		BaseURL:  srv.URL,
		UserName: "user",
		Password: "pass",
		SIDCode:  "SID001",
	})
	if err := p.Configure(raw); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	return p, srv
}

func TestProvider_Type(t *testing.T) {
	p := NewProvider(http.DefaultClient)
	if p.Type() != core.ChannelSMS {
		t.Errorf("Type() = %q, want %q", p.Type(), core.ChannelSMS)
	}
}

func TestProvider_Configure_RequiresHTTPS(t *testing.T) {
	p := NewProvider(http.DefaultClient).(*smsProvider)
	raw, _ := json.Marshal(smsConfig{BaseURL: "http://insecure.example.com", UserName: "u"})
	if err := p.Configure(raw); err == nil {
		t.Fatal("expected error for HTTP URL, got nil")
	}
}

func TestProvider_Configure_RequiresUserName(t *testing.T) {
	p := NewProvider(http.DefaultClient).(*smsProvider)
	raw, _ := json.Marshal(smsConfig{BaseURL: "https://example.com"})
	if err := p.Configure(raw); err == nil {
		t.Fatal("expected error for missing userName, got nil")
	}
}

func TestProvider_Send(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr string
	}{
		{
			name: "success 200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		},
		{
			name: "server returns 400",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = fmt.Fprint(w, "bad request body")
			},
			wantErr: "400",
		},
		{
			name: "server returns 500 includes body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = fmt.Fprint(w, "internal error")
			},
			wantErr: "internal error",
		},
		{
			name: "content-type application/json",
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
			p, _ := newTLSProvider(t, tt.handler)
			err := p.Send(context.Background(), core.Request{
				Channel: core.ChannelSMS,
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

func TestProvider_Send_PayloadFields(t *testing.T) {
	var captured smsPayload

	p, _ := newTLSProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	req := core.Request{
		Channel: core.ChannelSMS,
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

func TestProvider_Send_ContextCancellation(t *testing.T) {
	p, _ := newTLSProvider(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		w.WriteHeader(http.StatusOK)
	})

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := p.Send(cancelCtx, core.Request{
		Channel: core.ChannelSMS,
		To:      "+61400000000",
		Body:    "hi",
	})
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if !strings.Contains(err.Error(), "sms request failed") {
		t.Errorf("error %q does not contain expected prefix", err.Error())
	}
}
