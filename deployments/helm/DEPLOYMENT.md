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
| `idp-thunder` | **Official Thunder chart** | Identity Provider (WSO2) |
| `temporal` | `./deployments/helm/temporal` | Workflow Engine (Server + UI) |

---

## 1. Build & Push Images (GHCR)

Build with `linux/amd64` when on Apple Silicon:

```bash
# From the repository root
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/nsw-api:latest    -f backend/Dockerfile ./backend --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/trader-app:latest  -f portals/apps/trader-app/Dockerfile ./portals --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/oga-app:latest     -f portals/apps/oga-app/Dockerfile ./portals --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/oga-backend:latest -f oga/Dockerfile ./oga --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/idp:latest         -f idp/Dockerfile . --push
```

### 1.1 Temporal Image Mirroring

```bash
docker pull --platform linux/amd64 temporalio/auto-setup:1.28.3
docker tag temporalio/auto-setup:1.28.3 ghcr.io/opennsw/temporal-auto-setup:1.28.3
docker push ghcr.io/opennsw/temporal-auto-setup:1.28.3
```

---

## 2. Deploy / Upgrade Helm Charts

### Core NSW Services
```bash
helm upgrade --install nsw-api ./deployments/helm/nsw-api --history-max 1
helm upgrade --install trader-app ./deployments/helm/trader-app --history-max 1
```

### OGA Portal Frontends (Generic Chart)
Each agency deployment uses the same `oga-app` chart with instance-specific values.
```bash
helm upgrade --install oga-fcau-app ./deployments/helm/oga-app -f ./deployments/helm/oga-app/values/fcau-values.yaml --history-max 1
helm upgrade --install oga-ird-app  ./deployments/helm/oga-app -f ./deployments/helm/oga-app/values/ird-values.yaml  --history-max 1
helm upgrade --install oga-npqs-app ./deployments/helm/oga-app -f ./deployments/helm/oga-app/values/npqs-values.yaml --history-max 1
```

### OGA Backend Services (Generic Chart)
Standardized on **port 8081** for all internal communication.
```bash
helm upgrade --install oga-fcau-backend ./deployments/helm/oga-backend -f ./deployments/helm/oga-backend/fcau-backend-values.yaml --history-max 1
helm upgrade --install oga-ird-backend  ./deployments/helm/oga-backend -f ./deployments/helm/oga-backend/ird-backend-values.yaml  --history-max 1
helm upgrade --install oga-npqs-backend ./deployments/helm/oga-backend -f ./deployments/helm/oga-backend/npqs-backend-values.yaml --history-max 1
```

### External Services
```bash
helm upgrade --install idp-thunder oci://ghcr.io/asgardeo/helm-charts/thunder --version 0.29.0 -f ./deployments/helm/idp/custom-values.yaml --history-max 1
helm upgrade --install temporal ./deployments/helm/temporal --history-max 1
```

---

## 3. Database Setup (Standardization)

### 3.1 Logical Databases
Each service **must** have its own logical database within the `nsw-db` cluster.

```bash
# Create Databases
oc exec deployment/nsw-db -- psql -U postgres -c "CREATE DATABASE \"nsw_db\";"
oc exec deployment/nsw-db -- psql -U postgres -c "CREATE DATABASE \"oga-backend-fcau\";"
oc exec deployment/nsw-db -- psql -U postgres -c "CREATE DATABASE \"oga-backend-ird\";"
oc exec deployment/nsw-db -- psql -U postgres -c "CREATE DATABASE \"oga-backend-npqs\";"
```

### 3.2 Service URLs (Port 8081)
All OGA services are configured to listen on **port 8081**. This must be reflected in:
1.  **nsw-api Registry**: `services-cm.yaml` (ConfigMap)
2.  **Workflow Seeds**: `ogaSubmissionUrls` in `nsw-api/values.yaml`
3.  **OGA Frontend Proxy**: `ogaBackendUrl` in `oga-app` values.

```yaml
# nsw-api/values.yaml snippet
migrations:
  ogaSubmissionUrls:
    npqs: "http://oga-npqs-backend:8081/api/oga/inject"
    fcau: "http://oga-fcau-backend:8081/api/oga/inject"
    ird: "http://oga-ird-backend:8081/api/oga/inject"
```

---

## 4. Verification & Troubleshooting

| Issue | Cause | Fix |
|:---|:---|:---|
| `no registered service found` | URL/Port mismatch | Verify `services-cm.yaml` includes port `:8081`. |
| `Data not visible in OGA portal` | Proxy mismatch | Check `OGA_BACKEND_URL` in frontend pod matches backend service. |
| `Temporal connection failure` | Incorrect host | Set `TEMPORAL_HOST` to the cluster DNS of the temporal service. |

### Diagnostic Commands
```bash
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
```