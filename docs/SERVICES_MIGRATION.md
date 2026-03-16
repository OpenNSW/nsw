# Services Migration & Integration Guide

This document describes how to migrate and integrate remote services into the NSW backend using the central `remote.Manager`.

## 1. Overview
The NSW backend uses a central registry (`services.json`) to manage outbound calls. This approach provides:
- **Centralized Auth:** OAuth2, Bearer, and API Key authentication managed in one place.
- **Resilience:** Consistent retry policies (`remote.DefaultRetryConfig`).
- **Security:** Protection against SSRF and credential leakage through strict URL validation.
- **Portability:** Easy environment-specific configuration via a JSON file.

---

## 2. The Configuration File (`services.json`)

The configuration file defines the "topology" of your external services.

### Example Entry:
```json
{
  "id": "npqs-portal",
  "url": "http://localhost:8081",
  "timeout": "30s",
  "auth": {
    "type": "bearer",
    "options": {
      "token": "YOUR_SECRET_TOKEN"
    }
  }
}
```

### Supported Auth Types:
- `bearer`: Static token authentication.
- `api_key`: Header-based API key.
- `oauth2`: Client credentials flow (auto-refreshes on demand).

---

## 3. Security: URL Validation (CRITICAL)

To prevent SSRF and accidental credential leakage, the `remote.Client` performs strict validation on absolute URLs:

- If you provide an absolute URL (e.g., `https://google.com`), the client will check it against the service's `baseURL`.
- **The Scheme (http/https) and Host (127.0.0.1:8080) must match exactly.**
- If they do not match, the request will be blocked with a `remote: absolute URL host ... does not match` error.

**Best Practice:** Use relative paths (e.g., `/api/v1/verify`) whenever possible.

---

## 4. Migration Paths

### Option A: Use Service ID (Recommended)
This is the cleanest approach. You refer to the service by its unique ID defined in the JSON.

```go
// Use the manager injected into your service/plugin
err := manager.Call(ctx, "npqs-portal", remote.Request{
    Method: "POST",
    Path:   "/api/v1/submissions",
    Body:   myData,
}, &response)
```

### Option B: Use Absolute URL (Backward Compatibility)
The `Manager` will automatically attempt to "identify" the service based on the URL provided.

```go
err := manager.Call(ctx, "", remote.Request{
    Method: "POST",
    Path:   "http://localhost:8081/api/v1/submissions", // Full URL
    Body:   myData,
}, &response)
```
*Note: If the URL's host matches a registered service, that service's authentication and timeouts will be applied automatically.*

---

## 5. Local Development vs. Production

### Local Development:
- Create your local config: `cp backend/configs/services.example.json backend/configs/services.json`.
- The `.gitignore` prevents `services.json` from being committed.

### Production:
- Provide a `services.json` through a Kubernetes ConfigMap or a safe file mount.
- Set the `SERVICES_CONFIG_PATH` environment variable to point to the file.
- **NEVER** put real secrets in the `services.example.json` file.

---

## 6. Troubleshooting

### "Remote Error: no registered service found"
The Manager cannot find a service ID that matches the URL host you provided. Check your `services.json` for typos in the `url` field.

### "Remote Error: host does not match"
You are trying to send a request to a host that differs from the one registered in the `Manager`. This is a security block.

### "Remote Error: oauth2 failed"
The OAuth2 client credentials flow failed. Verify your `client_id` and `client_secret`.
