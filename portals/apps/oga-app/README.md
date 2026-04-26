# OGA App

## Authentication configuration

This app uses Asgardeo/Thunder OIDC for sign-in.

Required environment variables:

- `VITE_INSTANCE_CONFIG`: OGA config id to load (`npqs`, `fcau`, ...)
- `VITE_API_BASE_URL`: OGA backend API base URL (for example `http://localhost:8081`)
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
- CDA deployment
  - `VITE_INSTANCE_CONFIG=cda`
  - `VITE_IDP_CLIENT_ID=OGA_PORTAL_APP_CDA`

This allows IdP-level user access restriction per OGA app registration.

## Configuration

OGA instance branding and feature configuration is defined via YAML files in `src/configs/`.

### How it works

1. `src/configs/default.yaml` provides base fallback values for all instances.
2. `src/configs/<instance>.yaml` overrides specific values for each OGA.
3. At build time, the default config and instance config are loaded and merged.
4. The merged config is validated before the app renders.

### Adding a new OGA instance

1. Copy `src/configs/example.yaml` to `src/configs/<instance-id>.yaml`
2. Edit the `branding.appName` field (required)
3. Set `VITE_INSTANCE_CONFIG=<instance-id>` in your environment

### Config schema

```yaml
branding:
  appName: "My OGA Name"       # Required
  logoUrl: ""                   # Optional
  favicon: ""                   # Optional
```

## Local development

```bash
pnpm install
pnpm run dev
```
