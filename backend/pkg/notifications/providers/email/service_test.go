package email

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OpenNSW/nsw/pkg/notifications"
)

func newProvider(t *testing.T, h http.HandlerFunc) (*Provider, *httptest.Server) {
	t.Helper()
	srv := httptest.NewTLSServer(h)
	t.Cleanup(srv.Close)
	return New(Config{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	}), srv
}

func TestProvider_Type(t *testing.T) {
	p := New(Config{BaseURL: "http://example.com"})
	if p.Type() != notifications.ChannelEmail {
		t.Errorf("Type() = %q, want %q", p.Type(), notifications.ChannelEmail)
	}
}

func TestNew_DefaultsHTTPClient(t *testing.T) {
	p := New(Config{BaseURL: "http://example.com"})
	if p.config.HTTPClient == nil {
		t.Fatal("expected default HTTPClient, got nil")
	}
}

func TestProvider_Send(t *testing.T) {
	ctx := context.Background()

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
			name: "non-2xx includes status and body in error",
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
			name: "content-type is application/json",
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
			p, _ := newProvider(t, tt.handler)
			err := p.Send(ctx, notifications.Request{
				Channel: notifications.ChannelEmail,
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
	var captured serviceEmailPayload

	p, _ := newProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	req := notifications.Request{
		Channel:  notifications.ChannelEmail,
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
	var rawPayload map[string]any

	p, _ := newProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&rawPayload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	if err := p.Send(context.Background(), notifications.Request{
		Channel: notifications.ChannelEmail,
		To:      "to@example.com",
		Subject: "Hi",
		Body:    "Text only",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := rawPayload["html_body"]; ok {
		t.Error("html_body should be omitted when empty, but was present in payload")
	}
}

func TestProvider_Send_BearerTokenHeader(t *testing.T) {
	var capturedAuth string

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := New(Config{
		BaseURL:    srv.URL,
		Token:      "secret-token",
		HTTPClient: srv.Client(),
	})

	if err := p.Send(context.Background(), notifications.Request{
		Channel: notifications.ChannelEmail,
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

func TestProvider_Send_NoTokenHeader(t *testing.T) {
	var capturedAuth string

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := New(Config{BaseURL: srv.URL, HTTPClient: srv.Client()})

	if err := p.Send(context.Background(), notifications.Request{
		Channel: notifications.ChannelEmail,
		To:      "to@example.com",
		Subject: "Hi",
		Body:    "Text",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedAuth != "" {
		t.Errorf("expected no Authorization header, got %q", capturedAuth)
	}
}

func TestProvider_Send_InsecureURLRejected(t *testing.T) {
	p := New(Config{BaseURL: "http://email.svc.local", Token: "tok"})
	err := p.Send(context.Background(), notifications.Request{
		Channel: notifications.ChannelEmail,
		To:      "to@example.com",
		Subject: "Hi",
		Body:    "Text",
	})
	if err == nil {
		t.Fatal("expected error for HTTP URL, got nil")
	}
	if !strings.Contains(err.Error(), "HTTPS") {
		t.Errorf("error %q should mention HTTPS", err.Error())
	}
}

func TestProvider_Send_ContextCancellation(t *testing.T) {
	p, _ := newProvider(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		w.WriteHeader(http.StatusOK)
	})

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := p.Send(cancelCtx, notifications.Request{
		Channel: notifications.ChannelEmail,
		To:      "to@example.com",
		Subject: "Hi",
		Body:    "Text",
	})
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if !strings.Contains(err.Error(), "email service request failed") {
		t.Errorf("error %q does not contain %q", err.Error(), "email service request failed")
	}
}
