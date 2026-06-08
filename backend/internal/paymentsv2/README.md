# Payments V2

The `paymentsv2` package provides a modular and extensible payment orchestration system for the National Single Window (NSW) platform. It follows a gateway-based architecture, separating protocol-level concerns from domain logic.

## Architecture Overview

The system consists of several key parts:

1.  **PaymentGateway**: An interface for gateway-specific integrations (e.g., LankaPay, GovPay). It handles session creation, webhook parsing, and real-time validation formatting.
2.  **GatewayRegistry**: A pure discovery and lookup service. It manages gateway registration, configuration injection, and provides sanitized metadata for the UI.
3.  **PaymentRepository**: Handles persistence for `PaymentTransaction` records using GORM.
4.  **PaymentService**: The high-level orchestrator. It uses the Registry to find the correct Gateway and coordinates between the gateway logic, database, and internal events.
5.  **HTTPHandler**: Exposes the payment service via RESTful endpoints for both public and internal use.

## Getting Started

### 1. Implement a PaymentGateway

Each payment gateway requires a dedicated implementation of the `PaymentGateway` interface in the `gateways` sub-package.

```go
type MyGateway struct {}

func (g *MyGateway) ApplyConfig(config json.RawMessage) error {
    // Inject gateway-specific settings from JSON
    return nil
}

func (g *MyGateway) GetFlowType() gateways.InteractionType {
    return gateways.FlowTypeRedirect
}

func (g *MyGateway) CreateSession(ctx context.Context, req gateways.SessionRequest) (*gateways.SessionResponse, error) {
    // Logic to initialize session with gateway
    return &gateways.SessionResponse{...}, nil
}

func (g *MyGateway) ExtractReferenceNumber(ctx context.Context, reqData json.RawMessage) (string, error) {
    // Parse gateway-specific validation request to find the NSW reference
    return "NSW-REF-123", nil
}

func (g *MyGateway) HandleValidateReference(ctx context.Context, tx *gateways.ValidationTransaction, reqData json.RawMessage) (*gateways.ValidationResponse, error) {
    // Format the final response for the bank app/gateway
    return &gateways.ValidationResponse{...}, nil
}

func (g *MyGateway) ParseWebhook(ctx context.Context, body []byte, headers map[string][]string) (*gateways.WebhookPayload, error) {
    // Logic to parse and validate gateway webhook
    return &gateways.WebhookPayload{...}, nil
}
```

### 2. Configure Payment Methods

The `payment_methods.json` file is the source of truth for available methods.

```json
{
  "version": "1.0",
  "methods": [
    {
      "id": "lankapay",
      "is_active": true,
      "render_info": {
        "display_name": "Credit/Debit Card (LankaPay)",
        "description": "Pay securely using your card.",
        "display_order": 1
      },
      "config": {
        "base_url": "https://sandbox.govpay.lk"
      }
    }
  ]
}
```

### 3. Instantiate the Registry

The `GatewayRegistry` loads the configuration and maps each method ID to its implementation.

```go
gateways := map[string]gateways.PaymentGateway{
    "lankapay": &lankapay.Gateway{},
    "govpay":   &govpay.Gateway{},
}

registry, err := paymentsv2.NewRegistry("configs/payment_methods.json", gateways)
```

### 4. Setup the Orchestrator

The `PaymentService` acts as the "Caller" using the Registry as a lookup.

```go
repo := paymentsv2.NewPaymentRepository(db)
service := paymentsv2.NewPaymentService(repo, registry)

handler := paymentsv2.NewHTTPHandler(service)
```

## Key Flows

### Checkout Initialization
The frontend calls `CreateCheckoutSession`. The Service generates an NSW reference, looks up the gateway implementation via the Registry, and delegates the session creation to that gateway.

### Real-Time Validation
When a user enters a reference in a bank app, the gateway calls NSW. 
1. The Service uses the Gateway to **Extract** the reference number.
2. The Service fetches the transaction from the **Database**.
3. The Service passes the record back to the Gateway to **Validate** and format the protocol-specific response.

### Webhook Processing
Gateways notify NSW of results. The Service looks up the gateway via the Registry, delegates the parsing, and then performs domain actions: updating status, persisting metadata, and firing internal events.
