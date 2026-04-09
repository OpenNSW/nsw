# NSW Platform — Helm Deployment Guide

> **Target:** OpenShift (Akaza GovCloud)  
> **Constraint:** Always pass `--history-max 1` to stay within the 20-secret quota limit for multiple OGA pods.

---

## Chart Inventory

| Release Name | Chart Source | Description |
|:---|:---|:---|
| `nsw-api` | `./deployments/helm/nsw-api` | Core backend API (Go) |
| `trader-app` | `./deployments/helm/trader-app` | Trader portal frontend (React) |
| `oga-<agency>-app` | `./deployments/helm/oga-app` | Generic OGA portal frontend (React) |
| `oga-<agency>-backend` | `./deployments/helm/oga-backend` | Generic OGA backend API (Go) |
| `idp-thunder` | `./deployments/helm/idp` | Declarative Identity Provider (WSO2) Umbrella Chart |
| `temporal` | `./deployments/helm/temporal` | Workflow Engine (Server + UI) |

---

## 1. Build & Push Images (GHCR)

Build with `linux/amd64` when on Apple Silicon:

```bash
# From the repository root
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/nsw-api:latest -f backend/Dockerfile ./backend --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/nsw-api-migrations:latest -f backend/migrations.Dockerfile . --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/trader-app:latest -f portals/apps/trader-app/Dockerfile ./portals --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/oga-app:latest -f portals/apps/oga-app/Dockerfile ./portals --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/oga-backend:latest -f oga/Dockerfile ./oga --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/idp:latest -f deployments/helm/idp/Dockerfile ./deployments/helm/idp --push
```

### 1.1 Temporal Image Mirroring

```bash
docker pull --platform linux/amd64 temporalio/auto-setup:1.28.3
docker tag temporalio/auto-setup:1.28.3 ghcr.io/opennsw/temporal-auto-setup:1.28.3
docker push ghcr.io/opennsw/temporal-auto-setup:1.28.3
```

---

## 2. Deploy / Upgrade Helm Charts (Multi-Environment)

We utilize a hierarchical values approach (`values.yaml` + `values-dev.yaml`) and separate namespaces per environment to guarantee isolation without collisions. **Always explicitly execute Helm layering the environment values.**

### Developer Initialization
```bash
helm dependency build ./deployments/helm/idp
helm dependency build ./deployments/helm/nsw-api
```

### Option A: DEV Environment Deployments
```bash
# Core NSW Core API
helm upgrade --install dev-nsw-api ./deployments/helm/nsw-api -f ./deployments/helm/nsw-api/values.yaml -f ./deployments/helm/nsw-api/values-dev.yaml -n national-single-window-platform --history-max 1

# Temporal Server
helm upgrade --install dev-trader-app ./deployments/helm/trader-app -f ./deployments/helm/trader-app/values.yaml -f ./deployments/helm/trader-app/values-dev.yaml -n nsw-dev --set fullnameOverride=trader-app,image.tag=latest --history-max 1
helm upgrade --install dev-temporal ./deployments/helm/temporal -f ./deployments/helm/temporal/values-dev.yaml -n nsw-dev --history-max 1

# IDP Umbrella Chart (with Kustomize Patching)
# NOTE: Only redeploy idp-thunder if there is a version update. 
# Seeding and configuration are usually handled via hooks or manual scripts.
kustomize build --enable-helm ./deployments/helm/idp | oc apply -n national-single-window-platform -f -

# OGA Apps & Backends
helm upgrade --install dev-oga-fcau-backend ./deployments/helm/oga-backend -f ./deployments/helm/oga-backend/values-dev.yaml -f ./deployments/helm/oga-backend/fcau-backend-values.yaml -n national-single-window-platform --history-max 1
helm upgrade --install dev-oga-ird-backend ./deployments/helm/oga-backend -f ./deployments/helm/oga-backend/values-dev.yaml -f ./deployments/helm/oga-backend/ird-backend-values.yaml -n nsw-dev --history-max 1
helm upgrade --install dev-oga-npqs-backend ./deployments/helm/oga-backend -f ./deployments/helm/oga-backend/values-dev.yaml -f ./deployments/helm/oga-backend/npqs-backend-values.yaml -n nsw-dev --history-max 1

helm upgrade --install dev-oga-fcau-app ./deployments/helm/oga-app -f ./deployments/helm/oga-app/values-dev.yaml -f ./deployments/helm/oga-app/values/fcau-values.yaml -n nsw-dev --set fullnameOverride=dev-oga-fcau-app --history-max 1
helm upgrade --install dev-oga-ird-app ./deployments/helm/oga-app -f ./deployments/helm/oga-app/values-dev.yaml -f ./deployments/helm/oga-app/values/ird-values.yaml -n nsw-dev --set fullnameOverride=dev-oga-ird-app --history-max 1
helm upgrade --install dev-oga-npqs-app ./deployments/helm/oga-app -f ./deployments/helm/oga-app/values-dev.yaml -f ./deployments/helm/oga-app/values/npqs-values.yaml -n nsw-dev --set fullnameOverride=dev-oga-npqs-app --history-max 1
```

### Option B: STAGING Environment Deployments
Use the exact same pattern with `-f values-staging.yaml` overrides and `-n nsw-staging` namespaces.
```bash
# Core NSW Core API
helm upgrade --install staging-nsw-api ./deployments/helm/nsw-api -f ./deployments/helm/nsw-api/values.yaml -f ./deployments/helm/nsw-api/values-staging.yaml -n nsw-staging --history-max 1
```

---

## 3. Database Setup (Standardization)

### 3.1 Logical Databases
Each service **must** have its own logical database within the `nsw-db` cluster.

```bash
# Create Databases
oc exec statefulset/nsw-db -n national-single-window-platform -- psql -U postgres -c "CREATE DATABASE \"nsw_db\";"
oc exec statefulset/nsw-db -n national-single-window-platform -- psql -U postgres -c "CREATE DATABASE \"oga-backend-fcau\";"
oc exec statefulset/nsw-db -n national-single-window-platform -- psql -U postgres -c "CREATE DATABASE \"oga-backend-ird\";"
oc exec statefulset/nsw-db -n national-single-window-platform -- psql -U postgres -c "CREATE DATABASE \"oga-backend-npqs\";"
```

### 3.2 Dynamic Service Registration
Service discovery for external backends (OGA, ASYCUDA mock) is handled via the `services` block in `values.yaml`. This renders a ConfigMap (`services-cm.yaml`) that the API loads at runtime via `SERVICES_CONFIG_PATH`.

```yaml
# nsw-api/values-dev.yaml
service:
  services:
    - id: "npqs"
      url: "http://dev-oga-npqs-backend.{{ .Values.global.namespace }}.svc.cluster.local:8081"
      timeout: "30s"
    - id: "customs-asycuda"
      url: "https://7b0eb5f0-1ee3-4a0c-8946-82a893cb60c2.mock.pstmn.io"
      timeout: "10s"
```


---

## 4. Verification & Troubleshooting

| Issue | Cause | Fix |
|:---|:---|:---|
| `no registered service found` | URL/Port mismatch | Verify `services-cm.yaml` includes port `:8081`. |
| `Data not visible in OGA portal` | Proxy mismatch | Check `OGA_BACKEND_URL` in frontend pod matches backend service. |
| `Temporal connection failure` | Incorrect host or stale connection pool | Set `TEMPORAL_HOST` correctly. If it fails with "no usable database connection", restart temporal pods. |
| `ErrImagePull` on migrations | Missing `latest` tag | Use explicit version tags (e.g., `1.0.0`) in `values.yaml` instead of `latest`. |
| `Invalid client_id` in IDP | Apps not registered or stale | Run `seed.sh` manually inside the thunder pod or rerun the seed job. (Redeploying idp-thunder is rarely needed). |
| `Dirty database version X` | Migration failed mid-execution | Force version manually or perform a **Logical Database Reset** (see below). |
| `database accessed by users` | Active pod connections | Scale down deployments and terminate sessions before dropping DB. |
| `syntax error at ":"` | `psql` variables in `migrate` | `golang-migrate` does not support `psql` variables (`:'VAR'`). Use standard SQL strings instead. |

### 4.2 Logical Database Reset (Emergency Only)
If migrations are stuck in a `dirty` state and you need to start from a clean slate:

```bash
# 1. Scale down the API to stop new connections
oc scale deployment/dev-nsw-api --replicas=0 -n national-single-window-platform

# 2. Terminate zombie sessions (Run separately)
oc exec statefulset/nsw-db -n national-single-window-platform -- psql -U postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'nsw_dev' AND pid <> pg_backend_pid();"

# 3. Drop and Recreate (Run separately)
oc exec statefulset/nsw-db -n national-single-window-platform -- psql -U postgres -c "DROP DATABASE nsw_dev;"
oc exec statefulset/nsw-db -n national-single-window-platform -- psql -U postgres -c "CREATE DATABASE nsw_dev;"

# 4. Scale back up (Triggers fresh migration)
oc scale deployment/dev-nsw-api --replicas=1 -n national-single-window-platform
```

### Diagnostic Commands
```bash
# Check API logs
kubectl logs -l app.kubernetes.io/instance=dev-nsw-api -n national-single-window-platform

# Check OGA Connectivity
oc exec deployment/nsw-api -- curl -v http://oga-npqs-backend:8081/health

# Check Nginx Proxy Configuration
oc exec deployment/oga-npqs-app -- cat /etc/nginx/conf.d/default.conf | grep proxy_pass
```

---

## 5. Directory Structure
```
deployments/helm/
├── nsw-api/           (Core API)
├── trader-app/        (Trader Frontend)
├── oga-app/           (Generic OGA Frontend)
│   └── values/        (Instance overrides)
└── oga-backend/       (Generic OGA Backend)
    └── *-values.yaml  (Instance overrides)
└── idp/               (Declarative IDP Umbrella Chart)
```