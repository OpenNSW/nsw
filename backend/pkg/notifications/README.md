# pkg/notifications

Single notification package for the NSW backend.

Handles:
- routing requests to the correct provider by channel
- rendering templates (email and SMS) when a `TemplateID` is provided
- dispatching delivery in a background goroutine so callers are not blocked
- logging provider errors via `slog` (template errors are returned to the caller immediately)

Does not handle: batching, queues, retries.

## Package layout

```text
pkg/notifications/
├── notifications.go      — types, Manager, Send/SendEmail/SendSMS
├── template.go           — template cache and rendering
├── loader/
│   └── loader.go         — LoadFromFile: reads notifications.json, expands ${VAR}, builds Manager
└── providers/
    ├── email/
    │   └── service.go    — external HTTP email service provider
    └── sms/
        └── govsms.go     — GovSMS provider
```

## Wiring

Bootstrap loads configuration from a JSON file:

```go
import "github.com/OpenNSW/nsw/pkg/notifications/loader"

nm, err := loader.LoadFromFile(cfg.Notification.ConfigPath) // NOTIFICATIONS_CONFIG_PATH
```

See `configs/notifications.example.json` for the file format.
`${VAR}` placeholders in the JSON are expanded from environment variables before parsing.
Any unset variable is a hard error at startup.

`notifications.New(cfg, providers...)` remains available for tests.

## Core types

### Provider interface

```go
type Provider interface {
    Send(ctx context.Context, req Request) error
    Type() ChannelType
}
```

### Manager

Entry point. Stores providers by channel, owns the template cache.

```go
manager := notifications.New(
    notifications.Config{
        EmailTemplateRoot: "./configs/email-templates",
        SMSTemplateRoot:   "./configs/sms-templates",
    },
    emailProvider,
    smsProvider,
)
```

## How it works

```
caller
  -> Manager.SendEmail / SendSMS
  -> render template (if TemplateID set)  ← errors returned here
  -> goroutine fires
       -> Manager.Send
       -> provider selected by channel
       -> external service               ← errors logged via slog
```

`SendEmail` and `SendSMS` return immediately after template rendering.
Callers can call `Manager.Send` directly — synchronous, returns provider errors.

## Templates

### Email

Files: `<EmailTemplateRoot>/<id>.tmpl`

```
{{define "subject"}}...{{end}}
{{define "plainBody"}}...{{end}}
{{define "htmlBody"}}...{{end}}  // optional
```

### SMS

Files: `<SMSTemplateRoot>/<id>.txt` — entire file is the template body.

Templates are parsed on first use and cached.

## Providers

### email.Provider

POSTs to `{BaseURL}/emails`. BaseURL must use `https://`.

```go
type Config struct {
    BaseURL    string       `json:"baseURL"`  // https:// required
    Token      string       `json:"token"`    // optional bearer token
    HTTPClient *http.Client `json:"-"`        // defaults to 10s timeout
}
```

Request payload:

```json
{ "to": "...", "subject": "...", "text_body": "...", "html_body": "..." }
```

Any 2xx is success. Non-2xx returns an error with status code and body.

### sms.GovSMSProvider

POSTs JSON to the GovSMS API. BaseURL must use `https://`.

```go
type GovSMSConfig struct {
    BaseURL    string       `json:"baseURL"`  // https:// required
    UserName   string       `json:"userName"`
    Password   string       `json:"password"`
    SIDCode    string       `json:"sidCode"`
    HTTPClient *http.Client `json:"-"`        // defaults to 10s timeout
}
```

## Adding a provider

1. Add a `ChannelType` constant in `notifications.go` if needed
2. Implement the `Provider` interface
3. Add a `case` in `loader/loader.go` to wire it from JSON config
