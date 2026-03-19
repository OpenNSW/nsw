# Notification Manager Design Document

## Overview
The `notification` package provides a centralized, type-safe, and asynchronous system for dispatching messages across various phone-based communication channels (SMS, WhatsApp).

## Current Architecture

### Core Components
- **`Manager`**: The central hub that manages a registry for `SMSChannel` implementations. It provides a type-safe method for dispatching notifications.
- **`SMSChannel` (Interface)**: Defines the contract for phone-based providers (SMS, WhatsApp).
- **`SMSPayload`**: Contains phone-specific notification data (Recipients, Body, Templates).

### Design Patterns
- **Asynchronous Dispatch**: The `SendSMS` method is non-blocking and executes in the background.
- **Dependency Injection**: Provider credentials and settings are injected into channels via `Config` structs during initialization.

## Key Features & Implementation Details

### 1. Asynchronous Execution
The `Manager`'s `SendSMS` method returns immediately. Dispatching happens in background goroutines, ensuring that the main application flow is never blocked by slow notification APIs.

### 2. Granular Error Reporting & Logging
Errors are handled via **Structured Logging** (`log/slog`). Each channel returns a `map[string]error` for its recipients, and the `Manager` logs any failures with full context.

### 3. Channel-Specific Templating
Templates are loaded at runtime from the filesystem, allowing updates without rebuilding the application.
- `SMSChannel` and `WhatsAppChannel` use `text/template` with `.txt` files.

## Roadmap

### Phase 1: Foundation (Completed)
- [x] Type-safe `SMSPayload` structure.
- [x] `SMSChannel` interface.
- [x] Asynchronous `SendSMS` logic in Manager.
- [x] Runtime filesystem template loading.
- [x] Gov SL SMS provider integration.

### Phase 2: Expansion
- [ ] **Email Support**: Re-introduce `EmailChannel` with multipart template support.
- [ ] Internal worker pools for high-volume dispatching.

### Phase 3: Advanced Features
- [ ] **Reliability**: Add a retry decorator with exponential backoff.
- [ ] **Audit Log**: Persist notification history to a database.

## Usage Example

```go
// Initialize Manager
manager := notification.NewManager()

// Register SMS Channel with Credentials
smsCfg := channels.GovSMSConfig{
    UserName: "api_user",
    Password: "password",
    BaseURL:  "https://api.sms.com",
    TemplateRoot: "/templates/sms",
}
manager.RegisterSMSChannel(channels.NewGovSMSChannel(smsCfg))

// Dispatch Asynchronously
manager.SendSMS(ctx, notification.SMSPayload{
    Recipients: []string{"+123456789"},
    BasePayload: notification.BasePayload{
        TemplateID: "otp",
        TemplateData: map[string]any{"OTP": "123456"},
    },
})
```
