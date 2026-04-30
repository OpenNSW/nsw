package email

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

func newTLSProvider(t *testing.T, h http.HandlerFunc) (*emailProvider, *httptest.Server) {
	t.Helper()
	srv := httptest.NewTLSServer(h)
	t.Cleanup(srv.Close)
	p := NewProvider(srv.Client()).(*emailProvider)
	raw, _ := json.Marshal(emailConfig{BaseURL: srv.URL, Token: "tok"})
	if err := p.Configure(raw); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	return p, srv
}

func TestProvider_Type(t *testing.T) {
	p := NewProvider(http.DefaultClient)
	if p.Type() != core.ChannelEmail {
		t.Errorf("Type() = %q, want %q", p.Type(), core.ChannelEmail)
	}
}

func TestProvider_Configure_RequiresHTTPS(t *testing.T) {
	p := NewProvider(http.DefaultClient).(*emailProvider)
	raw, _ := json.Marshal(emailConfig{BaseURL: "http://insecure.example.com"})
	if err := p.Configure(raw); err == nil {
		t.Fatal("expected error for HTTP URL, got nil")
	}
}

func TestProvider_Configure_RequiresBaseURL(t *testing.T) {
	p := NewProvider(http.DefaultClient).(*emailProvider)
	raw, _ := json.Marshal(emailConfig{})
	if err := p.Configure(raw); err == nil {
		t.Fatal("expected error for empty baseURL, got nil")
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
			name: "success 201",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
			},
		},
		{
			name: "non-2xx error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = fmt.Fprint(w, "invalid recipient")
			},
			wantErr: "422",
		},
		{
			name: "500 error includes body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = fmt.Fprint(w, "upstream failure")
			},
			wantErr: "upstream failure",
		},
		{
			name: "posts to /emails path",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/emails" {
					http.Error(w, fmt.Sprintf("unexpected path: %s", r.URL.Path), http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusOK)
			},
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
				Channel: core.ChannelEmail,
				To:      "user@example.com",
				Subject: "Hello",
				Body:    "World",
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

func TestProvider_Send_PayloadShape(t *testing.T) {
	var captured emailPayload

	p, _ := newTLSProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	req := core.Request{
		Channel:  core.ChannelEmail,
		To:       "to@example.com",
		Subject:  "Subject line",
		Body:     "Plain text body",
		HTMLBody: "<p>HTML body</p>",
	}
	if err := p.Send(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captured.To != req.To {
		t.Errorf("To = %q, want %q", captured.To, req.To)
	}
	if captured.Subject != req.Subject {
		t.Errorf("Subject = %q, want %q", captured.Subject, req.Subject)
	}
	if captured.TextBody != req.Body {
		t.Errorf("TextBody = %q, want %q", captured.TextBody, req.Body)
	}
	if captured.HTMLBody != req.HTMLBody {
		t.Errorf("HTMLBody = %q, want %q", captured.HTMLBody, req.HTMLBody)
	}
}

func TestProvider_Send_HTMLBodyOmittedWhenEmpty(t *testing.T) {
	var raw map[string]any

	p, _ := newTLSProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	if err := p.Send(context.Background(), core.Request{
		Channel: core.ChannelEmail,
		To:      "to@example.com",
		Subject: "Hi",
		Body:    "Text only",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := raw["html_body"]; ok {
		t.Error("html_body should be omitted when empty")
	}
}

func TestProvider_Send_BearerTokenHeader(t *testing.T) {
	var capturedAuth string

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := NewProvider(srv.Client()).(*emailProvider)
	raw, _ := json.Marshal(emailConfig{BaseURL: srv.URL, Token: "secret-token"})
	if err := p.Configure(raw); err != nil {
		t.Fatalf("Configure: %v", err)
	}

	if err := p.Send(context.Background(), core.Request{
		Channel: core.ChannelEmail,
		To:      "to@example.com",
		Subject: "Hi",
		Body:    "Text",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedAuth != "Bearer secret-token" {
		t.Errorf("Authorization = %q, want %q", capturedAuth, "Bearer secret-token")
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
		Channel: core.ChannelEmail,
		To:      "to@example.com",
		Subject: "Hi",
		Body:    "Text",
	})
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if !strings.Contains(err.Error(), "email service request failed") {
		t.Errorf("error %q does not contain expected prefix", err.Error())
	}
}
