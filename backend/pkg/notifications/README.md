# pkg/notifications

`pkg/notifications` is the single notification package for the NSW backend.

It handles:
- routing requests to the correct provider by channel
- rendering templates (email and SMS) when a `TemplateID` is provided
- returning an error if delivery fails

It does not handle:
- batching
- queues
- retries

## Package layout

```text
pkg/notifications/
├── notifications.go   — types, Manager, Send/SendEmail/SendSMS
├── template.go        — template cache and rendering
└── providers/
    ├── email/
    │   └── service.go — external HTTP email service provider
    └── sms/
        └── govsms.go  — GovSMS provider
```

## Core types

### Request

Low-level type passed to providers:

```go
type Request struct {
    Channel  ChannelType
    To       string
    Subject  string // email only
    Body     string
    HTMLBody string // email only, optional
}
```

### EmailRequest / SMSRequest

Typed convenience types for `SendEmail` and `SendSMS`:

```go
type EmailRequest struct {
    To           string
    Subject      string
    Body         string
    HTMLBody     string
    TemplateID   string
    TemplateData map[string]any
}

type SMSRequest struct {
    To           string
    Body         string
    TemplateID   string
    TemplateData map[string]any
}
```

When `TemplateID` is set, the Manager renders the template and ignores the raw `Body`/`Subject`/`HTMLBody` fields.

### Provider

Every delivery backend implements:

```go
type Provider interface {
    Send(ctx context.Context, req Request) error
    Type() ChannelType
}
```

### Manager

`Manager` is the package entry point. It stores providers by channel and owns the template cache.

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

```text
caller
  -> Manager.SendEmail / SendSMS
  -> render template (if TemplateID set)
  -> Manager.Send
  -> provider selected by channel
  -> external service
```

Callers can also call `Manager.Send` directly with pre-rendered content.

## Templates

### Email templates

Files: `<EmailTemplateRoot>/<id>.tmpl`

Each file must define three named templates:

```
{{define "subject"}}...{{end}}
{{define "plainBody"}}...{{end}}
{{define "htmlBody"}}...{{end}}
```

`htmlBody` is optional — omitted from the request if not defined.

### SMS templates

Files: `<SMSTemplateRoot>/<id>.txt`

The entire file is the template body. Rendered output becomes `req.Body`.

Templates are parsed on first use and cached in memory.

## Included providers

### Email: external HTTP service

File: `providers/email/service.go`

POSTs to `{BaseURL}/emails`:

```json
{
  "to": "...",
  "subject": "...",
  "text_body": "...",
  "html_body": "..."   // omitted if empty
}
```

Config:

```go
type ServiceConfig struct {
    BaseURL    string
    Token      string       // optional bearer token
    HTTPClient *http.Client // defaults to 10s timeout
}
```

Any 2xx response is a success. Non-2xx returns an error containing the status code and response body.

### SMS: GovSMS

File: `providers/sms/govsms.go`

POSTs JSON to the GovSMS API.

Config:

```go
type GovSMSConfig struct {
    UserName   string
    Password   string
    SIDCode    string
    BaseURL    string       // must use https://
    HTTPClient *http.Client // defaults to 10s timeout
}
```

## Wiring example

```go
import (
    "github.com/OpenNSW/nsw/pkg/notifications"
    "github.com/OpenNSW/nsw/pkg/notifications/providers/email"
    "github.com/OpenNSW/nsw/pkg/notifications/providers/sms"
)

manager := notifications.New(
    notifications.Config{
        EmailTemplateRoot: cfg.Notification.EmailTemplateRoot,
        SMSTemplateRoot:   cfg.Notification.SMSTemplateRoot,
    },
    email.NewService(email.ServiceConfig{
        BaseURL: cfg.Notification.EmailServiceURL,
        Token:   cfg.Notification.EmailServiceToken,
    }),
    sms.NewGovSMS(sms.GovSMSConfig{
        BaseURL:  cfg.Notification.GovSMSBaseURL,
        UserName: cfg.Notification.GovSMSUsername,
        Password: cfg.Notification.GovSMSPassword,
        SIDCode:  cfg.Notification.GovSMSSIDCode,
    }),
)
```

## Adding a new provider

1. Add a `ChannelType` constant in `notifications.go` if needed
2. Implement `Provider`
3. Pass it to `notifications.New(...)`
