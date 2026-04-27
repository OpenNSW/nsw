# Payments V2

The `paymentsv2` package provides a modular and extensible payment orchestration system for the National Single Window (NSW) platform. It follows a provider-based architecture, allowing for easy integration with multiple payment gateways (e.g., LankaPay, GovPay).

## Architecture Overview

The system consists of several key components:

1.  **PaymentProvider**: An interface for gateway-specific integrations. Focused strictly on logic (sessions, webhooks, validation).
2.  **PaymentRegistry**: Manages discovery, lookup, and **UI metadata** (loaded from configuration).
3.  **PaymentRepository**: Handles persistence for `PaymentTransaction` records using GORM.
4.  **PaymentService**: The high-level orchestrator that coordinates between the registry, repository, and external gateways.
5.  **HTTPHandler**: Exposes the payment service via RESTful endpoints for both public and internal use.

## Getting Started

### 1. Implement a PaymentProvider

Each payment gateway requires a dedicated implementation of the `PaymentProvider` interface. It does **not** handle UI metadata; that is managed by the Registry and loaded from `payment_methods.json`.

```go
type MyProvider struct {}

func (p *MyProvider) CreateSession(ctx context.Context, req paymentsv2.CreateCheckoutRequest) (*paymentsv2.CreateCheckoutResponse, error) {
    // Logic to initialize session with gateway
    return &paymentsv2.CreateCheckoutResponse{...}, nil
}

func (p *MyProvider) ParseWebhook(ctx context.Context, body []byte, headers map[string][]string) (*paymentsv2.WebhookPayload, error) {
    // Logic to parse and validate gateway webhook
    return &paymentsv2.WebhookPayload{...}, nil
}

func (p *MyProvider) HandleValidateReference(ctx context.Context, tx *paymentsv2.PaymentTransaction) (*paymentsv2.ValidateReferenceResponse, error) {
    // Logic to validate reference for real-time bank apps
    return &paymentsv2.ValidateReferenceResponse{...}, nil
}
```

### 2. Configure Payment Methods

The `payment_methods.json` file is the source of truth for available payment methods and their UI metadata.

```json
{
  "version": "1.0",
  "methods": [
    {
      "id": "lankapay",
      "is_active": true,
      "render_info": {
        "display_name": "Credit/Debit Card (LankaPay)",
        "description": "Pay securely using your card via LankaPay gateway.",
        "logo_url": "credit-card",
        "display_order": 1
      },
      "type": "REDIRECT",
      "gateway_url": "https://sandbox.govpay.lk/checkout"
    }
  ]
}
```

### 3. Instantiate the Registry

The `PaymentRegistry` loads the configuration and maps each method ID to its corresponding `PaymentProvider` implementation.

```go
// Example Registry instantiation
providers := map[string]paymentsv2.PaymentProvider{
    "lankapay": &lankapay.Provider{},
    "govpay":   &govpay.Provider{},
}

registry, err := paymentsv2.NewRegistry("configs/payment_methods.json", providers)
```

### 4. Instantiate the Service

Combine the repository and registry into the `PaymentService`.

```go
repo := paymentsv2.NewPaymentRepository(db)
service := paymentsv2.NewPaymentService(repo, registry)
```

### 5. Setup HTTP Handlers

The `HTTPHandler` can be integrated into your router.

```go
handler := paymentsv2.NewHTTPHandler(service)

// Example with standard library Mux (Go 1.22+)
mux := http.NewServeMux()
mux.HandleFunc("POST /api/v1/payments/{providerId}/validate", handler.HandleValidateReference)
mux.HandleFunc("POST /api/v1/payments/{providerId}/webhook", handler.HandleWebhook)
```

## Key Flows

### Checkout Initialization
The frontend or Task Engine calls `CreateCheckoutSession`. The service generates a NSW reference, selects the provider from the registry, and initializes a session with the gateway.

### Real-Time Validation
When a user enters a reference number in a bank app, the gateway calls `HandleValidateReference`. The service looks up the transaction in the database and delegates the validation logic to the specific provider.

### Webhook Processing
Gateways notify NSW of payment results via webhooks. The service uses the registry to find the correct provider, parses the payload, updates the transaction status, and triggers internal events for the Task Engine.
