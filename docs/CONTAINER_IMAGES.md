# Container Images Guide

This guide documents the Dockerfiles in the repository, how to build each image, and how to run each image with required environment configuration.

Use this guide when you want to run services individually (without `docker compose`).

For full-stack orchestration and deployment architecture, see `docs/DEPLOYMENT.md`.

## 1) Available Dockerfiles and Images

| Component | Dockerfile | Image Tag (local) | Purpose |
|---|---|---|---|
| NSW Backend API | `backend/Dockerfile` | `nsw-backend:local` | Core backend service |
| OGA Backend | `oga/Dockerfile` | `nsw-oga-backend:local` | OGA backend service (NPQS/FCAU/IRD instances) |
| Trader Portal | `portals/apps/trader-app/Dockerfile` | `nsw-trader-portal:local` | Trader frontend |
| OGA Portal | `portals/apps/oga-app/Dockerfile` | `nsw-oga-portal:local` | OGA frontend (instance-config driven) |

## 2) Build Images

From repository root:

```bash
docker build -f backend/Dockerfile -t nsw-backend:local backend
docker build -f oga/Dockerfile -t nsw-oga-backend:local oga
docker build -f portals/apps/trader-app/Dockerfile -t nsw-trader-portal:local portals
docker build -f portals/apps/oga-app/Dockerfile -t nsw-oga-portal:local portals
```

## CI Docker Validation

Pull requests to `main` validate Docker builds using `.github/workflows/docker-validation.yml`.

- Validated Docker targets:
  - `backend/Dockerfile` (context: `backend/`)
  - `oga/Dockerfile` (context: `oga/`)
  - `portals/apps/trader-app/Dockerfile` (context: `portals/`)
  - `portals/apps/oga-app/Dockerfile` (context: `portals/`)
- Trigger behavior:
  - Runs when changes are in `backend/**`, `oga/**`, `portals/**`, or the workflow file itself.
  - Builds execute only for matrix entries whose context changed.
  - Any Docker build failure fails the workflow.
- Scope boundary:
  - This workflow validates image builds only.
  - It does not push/publish images.
  - Publishing belongs to release CD workflows.

## 3) Runtime Configuration Model

- Backend and OGA backends use container environment variables directly.
- Frontend images (`trader-app`, `oga-app`) use runtime env injection into `runtime-env.js` via entrypoint scripts.
- In production/non-local environments, set all runtime values explicitly rather than relying on defaults.

## 4) Run Each Image Individually

## 4.1 NSW Backend API

```bash
docker run --rm --name nsw-backend \
  -p 8080:8080 \
  -e SERVER_PORT=8080 \
  -e SERVER_DEBUG=true \
  -e SERVER_LOG_LEVEL=info \
  -e DB_HOST=<db-host> \
  -e DB_PORT=<db-port> \
  -e DB_USERNAME=<db-user> \
  -e DB_PASSWORD=<db-password> \
  -e DB_NAME=<db-name> \
  -e DB_SSLMODE=disable \
  -e AUTH_JWKS_URL=https://<idp-host>:8090/oauth2/jwks \
  -e AUTH_ISSUER=https://<idp-host>:8090 \
  -e AUTH_CLIENT_IDS=TRADER_PORTAL_APP,FCAU_TO_NSW,NPQS_TO_NSW,IRD_TO_NSW \
  -e AUTH_AUDIENCE=NSW_API \
  -e AUTH_JWKS_INSECURE_SKIP_VERIFY=true \
  -e CORS_ALLOWED_ORIGINS=http://localhost:5173,http://localhost:5174,http://localhost:5175,http://localhost:5176 \
  -e STORAGE_TYPE=local \
  -e STORAGE_LOCAL_BASE_DIR=/data/uploads \
  -v nsw-backend-uploads:/data/uploads \
  nsw-backend:local
```

> Note: Set `AUTH_JWKS_INSECURE_SKIP_VERIFY=false` (or omit it) when using a valid TLS certificate on the IDP in non-local environments.

## 4.2 OGA Backend

```bash
docker run --rm --name nsw-oga-backend \
  -p 8081:8081 \
  -v nsw-oga-data:/data \
  -e OGA_PORT=8081 \
  -e OGA_DEFAULT_FORM_ID=default \
  -e OGA_ALLOWED_ORIGINS=http://localhost:5174,http://localhost:5175,http://localhost:5176 \
  -e OGA_NSW_API_BASE_URL=http://backend:8080/api/v1 \
  -e OGA_NSW_CLIENT_ID=IRD_TO_NSW \
  -e OGA_NSW_CLIENT_SECRET=1234 \
  -e OGA_NSW_TOKEN_URL=https://localhost:8090/oauth2/token \
  -e OGA_NSW_SCOPES= \
  -e OGA_NSW_TOKEN_INSECURE_SKIP_VERIFY=true \
  nsw-oga-backend:local
```

To run multiple OGA instances, start multiple containers with different names/ports and separate volumes.

## 4.3 Trader Portal

```bash
docker run --rm --name nsw-trader-portal \
  -p 5173:80 \
  -e VITE_API_BASE_URL=http://localhost:8080/api/v1 \
  -e VITE_IDP_BASE_URL=https://localhost:8090 \
  -e VITE_IDP_CLIENT_ID=TRADER_PORTAL_APP \
  -e VITE_APP_URL=http://localhost:5173 \
  -e VITE_IDP_SCOPES=openid,profile,email \
  -e VITE_IDP_PLATFORM=AsgardeoV2 \
  -e VITE_SHOW_AUTOFILL_BUTTON=true \
  nsw-trader-portal:local
```

## 4.4 OGA Portal

```bash
docker run --rm --name nsw-oga-portal \
  -p 5174:80 \
  -e VITE_INSTANCE_CONFIG=npqs \
  -e VITE_API_BASE_URL=http://localhost:8081 \
  -e VITE_IDP_BASE_URL=https://localhost:8090 \
  -e VITE_IDP_CLIENT_ID=OGA_PORTAL_APP_NPQS \
  -e VITE_APP_URL=http://localhost:5174 \
  -e VITE_IDP_SCOPES=openid,profile,email \
  -e VITE_IDP_PLATFORM=AsgardeoV2 \
  nsw-oga-portal:local
```

To run multiple OGA portal instances, start multiple containers with different ports and `VITE_INSTANCE_CONFIG` / `VITE_IDP_CLIENT_ID` values.

## 5) Recommended for Full Stack

For local end-to-end startup of all services with correct startup order and migrations, use:

```bash
cp .env.example .env
./start-dev.sh
```

See `docs/DEPLOYMENT.md` for architecture and multi-mode deployment guidance.
