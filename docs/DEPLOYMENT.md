# Deployment Guide

This document explains how to deploy NSW in different environments and orchestration models.

## Quick Navigation

- [1) Deployment Scope](#1-deployment-scope)
- [2) Architecture (Platform-Agnostic)](#2-architecture-platform-agnostic)
- [3) Configuration Model](#3-configuration-model)
- [4) Choose a Deployment Mode](#4-choose-a-deployment-mode)
- [5) Option A — Local Development (start-dev.sh)](#5-option-a--local-development-start-devsh)
- [6) Option B — Individual Docker Image Deployment (Manual)](#6-option-b--individual-docker-image-deployment-manual)
- [7) Option C — Kubernetes Deployment](#7-option-c--kubernetes-deployment)
- [8) Access Endpoints (Default Local Ports)](#8-access-endpoints-default-local-ports)
- [9) Operations and Verification](#9-operations-and-verification)
- [10) Troubleshooting](#10-troubleshooting)
- [11) Security Notes](#11-security-notes)

Supported deployment approaches in this repository:

1. Local development via `start-dev.sh`
2. Individual Docker image deployment (supported with manual orchestration)
3. Kubernetes deployment (architecture guidance and required building blocks)


## 1) Deployment Scope

A full NSW system deployment contains:

- Identity Provider (Thunder)
- PostgreSQL
- NSW Backend API
- OGA backends (NPQS, FCAU, IRD)
- Trader portal frontend
- OGA portals (NPQS, FCAU, IRD)
- Database migration initializer

## 2) Architecture (Platform-Agnostic)

### 2.1 Service Domains

- Identity domain: Thunder bootstrap + Thunder runtime
- Core domain: PostgreSQL, DB migration, backend, trader portal
- OGA domains: separate NPQS/FCAU/IRD backend + portal pairs

### 2.2 Network/Isolation Model

Regardless of runtime platform:

- IDP traffic is isolated from OGA domain traffic.
- Core services (DB/backend/trader) are isolated from OGA service domains.
- Backend acts as the integration point that talks to DB, IDP, and OGA backends.

### 2.3 Deployment Architecture Diagram (Placeholder)

> TODO: Add deployment architecture diagram image/file here.

Suggested location:

- `docs/diagrams/deployment-architecture.png`

Markdown placeholder:

```md
![Deployment Architecture Diagram](docs/diagrams/deployment-architecture.png)
```

## 3) Configuration Model

### 3.1 Environment files in this repo

- `.env.example`: non-Docker local development (`start-dev.sh`)

### 3.2 Required configuration categories

- Database connectivity (`DB_*`)
- IDP and OIDC settings (`IDP_*`, `THUNDER_*`, `AUTH_*`, `*_IDP_CLIENT_ID`)
- Service ports and public URLs
- CORS and frontend runtime config (`VITE_*`)
- OGA behavior (`OGA_*`)
- Backend storage mode (`STORAGE_*`)
- Server behavior flags (`SERVER_DEBUG`, `SERVER_LOG_LEVEL`, `SHOW_AUTOFILL_BUTTON`)
- TLS skip for local IDP (`AUTH_JWKS_INSECURE_SKIP_VERIFY`) — set to `true` in local dev; remove or set to `false` in production

### 3.3 Remote Services Configuration (services.json)

The backend requires a `services.json` file to manage outbound connections to external agency portals.

- **Development:** Copy `backend/configs/services.example.json` to `backend/configs/services.json`.
- **Production (Docker/K8s):**
    - Mount the configuration file to `/app/configs/services.json` (or your preferred path).
    - Set the `SERVICES_CONFIG_PATH` environment variable to the file path.
    - **Security:** Do not bake secrets into Docker images. Use Kubernetes Secrets or environment variable expansion if your loading logic supports it.
- **Reference:** See [Services Migration Guide](SERVICES_MIGRATION.md) for schema details and security validation rules.

## 4) Choose a Deployment Mode

| Mode | Best For | Operational Complexity | Repository Support |
|------|----------|------------------------|-----------------|
| Local development (start-dev.sh) | Day-to-day dev workflows, local integration runs | Low | Full (reference dev script) |
| Individual Docker images | Custom runtime wiring, partial deployments | Medium/High | Supported (manual orchestration) |
| Kubernetes | Cluster deployment and production-style operations | High | Guidance in this document |

Recommendation:

- Use **start-dev.sh** for local development.
- For production-style deployments, use Kubernetes or manual Docker orchestration.

## 5) Option A — Local Development (start-dev.sh)

This is the supported local development path in this repository.

### 5.1 What is provided

- Root `start-dev.sh` dev runner
- `.env.example` for local configuration defaults
- Optional IDP and Temporal runtime via their own compose files

### 5.2 Quick start

```bash
cp .env.example .env
./start-dev.sh
```

Useful variants:

```bash
./start-dev.sh --skip-idp
./start-dev.sh --skip-temporal
./start-dev.sh --clean-run
```

## 6) Option B — Individual Docker Image Deployment (Manual)

If you want to run each image independently (without compose), you can do that, but you must manually handle:

- network creation
- startup ordering
- migration execution
- volume persistence
- environment injection per container

### 6.1 Minimum manual orchestration steps

1. Build/pull all required images.
2. Create required Docker networks.
3. Create named volumes for DB and uploads.
4. Start PostgreSQL and wait for readiness.
5. Run migration container/job.
6. Start IDP bootstrap and runtime.
7. Start backend.
8. Start OGA backends.
9. Start frontends.

### 6.2 Example network/volume bootstrap

```bash
docker network create idp-net
docker network create core-net
docker network create oga-npqs-net
docker network create oga-fcau-net
docker network create oga-ird-net

docker volume create postgres-data
docker volume create backend-uploads
docker volume create oga-npqs-data
docker volume create oga-fcau-data
docker volume create oga-ird-data
docker volume create thunder-db
```

### 6.3 Image run references

For per-image run examples (Dockerfiles, build commands, runtime env), see:

- `docs/CONTAINER_IMAGES.md`

Note: individual run commands are useful for testing single images, but for full-system reliability use `start-dev.sh` for local runs or Kubernetes in clusters.

## 7) Option C — Kubernetes Deployment

This repository currently does not ship complete production-ready Kubernetes manifests for the full NSW stack. However, the deployment architecture maps cleanly to Kubernetes.

### 7.1 Recommended Kubernetes building blocks

- Namespace(s) for environment isolation
- `Deployment` for stateless services (backend, portals, OGA backends where appropriate)
- `StatefulSet` or managed service for PostgreSQL
- PersistentVolumeClaims for DB and file storage
- `Job`/`InitContainer` for DB migrations
- `Service` objects for internal discovery
- `Ingress`/Gateway for external access
- `Secret` and `ConfigMap` for runtime configuration

### 7.2 Suggested k8s service grouping

- `idp` group: Thunder setup + Thunder runtime
- `core` group: postgres + migration job + backend + trader portal
- `oga-*` groups: per-agency backend + portal

### 7.3 Kubernetes deployment checklist

- Define all env vars from `.env.example` (and any environment-specific overrides) as `Secret`/`ConfigMap` keys.
- Ensure backend can resolve/connect to postgres, idp, and OGA services.
- Run migration job before backend becomes live.
- Configure ingress routes and TLS.
- Configure readiness/liveness probes.
- Configure persistent storage for DBs and backend uploads.

## 8) Access Endpoints (Default Local Ports)

- Backend: `http://localhost:8080`
- Trader portal: `http://localhost:5173`
- OGA NPQS backend: `http://localhost:8081`
- OGA FCAU backend: `http://localhost:8082`
- OGA IRD backend: `http://localhost:8083`
- OGA NPQS portal: `http://localhost:5174`
- OGA FCAU portal: `http://localhost:5175`
- OGA IRD portal: `http://localhost:5176`
- IDP: `https://localhost:8090`
- PostgreSQL: `localhost:55432`

## 9) Operations and Verification

Verification checklist (all modes):

- All required services are running and reachable
- Migrations completed successfully
- OIDC flows work (issuer/JWKS/client IDs aligned)
- Frontend → backend/API connectivity works
- Backend upload persistence is durable across restarts

## 10) Troubleshooting

- **Port conflict**: change host-mapped ports in environment config.
- **Migration failure**: verify DB credentials/host and migration execution order.
- **Auth validation errors**: verify `AUTH_ISSUER` and JWKS endpoint reachability.
- **Frontend API mismatch**: verify `VITE_API_BASE_URL` and ingress/host mappings.

## 11) Security Notes

- Treat deployment env files and secrets as sensitive.
- Avoid default credentials in shared/non-local environments.
- Restrict external exposure of internal-only services.
- Use proper TLS cert management in non-local deployments.
