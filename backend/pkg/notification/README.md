# Notification Manager

The `notification` package provides a centralized, type-safe, and asynchronous system for dispatching messages across various communication channels like Email, SMS, and WhatsApp.

## Features

- **Type-Safe Payloads**: Specialized payloads for different channels (e.g., `EmailPayload`, `SMSPayload`) to ensure data integrity.
- **Asynchronous Dispatch**: All notifications are sent in the background, preventing blocking of the main application flow.
- **Multipart Email Support**: Automatic discovery and rendering of both HTML and Plain Text templates for maximum email deliverability.
- **Dynamic Templating**: Templates are loaded at runtime from the filesystem, allowing updates without application restarts.
- **Granular Error Reporting**: Comprehensive logging of failures per recipient using structured logging (`slog`).
- **Pluggable Architecture**: Supports interchangeable providers for each channel type (e.g., different SMS or Email gateways).

## Architecture

### Core Components

- **`Manager`**: The orchestrator that manages registered channels and dispatches notifications asynchronously.
- **`EmailChannel`**: Interface and implementation for sending emails with multipart template support.
- **`SMSChannel`**: Interface for phone-based notifications, implemented by providers like `GovSMS` and `WhatsApp`.
- **`Payloads`**: 
    - `BasePayload`: Shared template ID and data.
    - `EmailPayload`: Adds recipients and subject.
    - `SMSPayload`: Optimized for phone numbers.

### Implementation Details

- **Asynchronous Execution**: Dispatch methods return immediately, while background goroutines handle the actual network calls and rendering.
- **Template Discovery**: `EmailChannel` looks for `{TemplateID}.html` and `{TemplateID}.txt` in its configured `TemplateRoot` to build `multipart/alternative` messages.
- **Provider Injection**: Credentials and API settings are injected into channel instances during initialization via dedicated `Config` structs.

## Usage

### 1. Initialize and Register Channels

```go
import (
    "github.com/OpenNSW/nsw/pkg/notification"
    "github.com/OpenNSW/nsw/pkg/notification/channels"
)

manager := notification.NewManager()

// Register Email Channel
emailCfg := notification.EmailConfig{
    SMTPHost:     "smtp.example.com",
    SMTPPort:     587,
    SMTPUsername: "user@example.com",
    SMTPPassword: "password",
    SMTPSender:   "noreply@example.com",
    TemplateRoot: "/path/to/email/templates",
}
manager.RegisterEmailChannel(channels.NewEmailChannel(emailCfg))

// Register Gov SMS Channel
// NOTE: BaseURL MUST use https:// to protect credentials sent in the request body.
smsCfg := channels.GovSMSConfig{
    UserName:     "api_user",
    Password:     "secret",
    BaseURL:      "https://api.sms.gov.lk",
    TemplateRoot: "/path/to/sms/templates",
}
manager.RegisterSMSChannel(channels.NewGovSMSChannel(smsCfg))
```

### 2. Dispatch Notifications

```go
// Send an Email
manager.SendEmail(ctx, notification.EmailPayload{
    Recipients: []string{"user@example.com"},
    Subject:    "Welcome!",
    BasePayload: notification.BasePayload{
        TemplateID: "welcome",
        TemplateData: map[string]any{"Name": "Alice"},
    },
})

// Send an SMS
manager.SendSMS(ctx, notification.SMSPayload{
    Recipients: []string{"+1234567890"},
    BasePayload: notification.BasePayload{
        TemplateID: "otp",
        TemplateData: map[string]any{"OTP": "123456"},
    },
})
```

## Testing

The package includes three layers of testing:

1.  **Manager Unit Tests**: `pkg/notification/manager_test.go`
2.  **Channel Unit Tests**: `pkg/notification/channels/gov_sms_test.go`, `pkg/notification/channels/email.go` (template rendering)
3.  **Integration Tests**: `pkg/notification/test/integration/integration_test.go` (uses a mock HTTP server for SMS and template validation for Email).

To run all tests:
```bash
go test -v ./pkg/notification/...
```

## Template Structure

Templates should be organized in folders per channel:

```text
templates/
├── email/
│   └── otp.tmpl  # Contains {{define "subject"}}, {{define "plainBody"}}, {{define "htmlBody"}}
└── sms/
    └── otp.txt
```

Email templates use Go's `text/template` with three blocks:
- `subject`: The email subject line
- `plainBody`: Plain text version
- `htmlBody`: HTML version
