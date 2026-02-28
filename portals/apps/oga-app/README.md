# OGA App

## Authentication configuration

This app uses Asgardeo/Thunder OIDC for sign-in.

Required environment variables:

- `VITE_IDP_BASE_URL`: IdP base URL (for example `https://localhost:8090`)
- `VITE_IDP_CLIENT_ID`: OGA-specific IdP application client id
- `VITE_APP_URL`: public URL of this OGA deployment
- `VITE_IDP_SCOPES` (optional): comma-separated scopes (defaults to `openid,profile,email`)
- `VITE_IDP_PLATFORM` (optional): SDK platform (defaults to `AsgardeoV2`)

## Per-OGA deployment model

Each OGA deployment should use its own IdP application configuration.

Example:

- NPQS deployment
  - `VITE_INSTANCE_CONFIG=npqs`
  - `VITE_IDP_CLIENT_ID=OGA_PORTAL_APP_NPQS`
- FCAU deployment
  - `VITE_INSTANCE_CONFIG=fcau`
  - `VITE_IDP_CLIENT_ID=OGA_PORTAL_APP_FCAU`

This allows IdP-level user access restriction per OGA app registration.

## Local development

```bash
pnpm install
pnpm run dev
```
