# NSW Platform — Helm Deployment Guide

> **Target:** OpenShift (Akaza GovCloud)  
> **Constraint:** Always pass `--history-max 3` to stay within the 20-secret quota.

---

## Chart Inventory

| Release Name | Chart Source | Values / Config |
|:---|:---|:---|
| `nsw-api` | `./deployments/helm/nsw-api` | `values.yaml` (inline) |
| `trader-app` | `./deployments/helm/trader-app` | `values.yaml` (inline) |
| `oga-app` | `./deployments/helm/oga-app` | `values.yaml` + per-agency overrides (e.g. `npqs-app-values.yaml`) |
| `oga-<agency>-backend` | `./deployments/helm/oga-backend` *(generic)* | Per-agency values file (e.g. `fcau-backend-values.yaml`) |
| `idp-thunder` | **Official Thunder chart** (`ghcr.io/asgardeo/helm-charts/thunder`) | `idp/custom-values.yaml` |

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

---

## 2. Deploy / Upgrade Helm Charts

All commands are run from the **repository root**.

### Core NSW Services

```bash
helm upgrade --install nsw-api    ./deployments/helm/nsw-api    --history-max 3
helm upgrade --install trader-app ./deployments/helm/trader-app --history-max 3
```

### OGA Portal Frontends (Generic Chart, 3 Distinct Releases)

The `oga-app/` chart is a **generic chart** reused for every OGA portal.
Each OGA gets its **own release name**, its own route, and a dedicated `-f` values file inside `oga-app/`:

```bash
helm upgrade --install oga-fcau-app ./deployments/helm/oga-app \
  -f ./deployments/helm/oga-app/fcau-app-values.yaml --history-max 3

helm upgrade --install oga-ird-app ./deployments/helm/oga-app \
  -f ./deployments/helm/oga-app/ird-app-values.yaml --history-max 3

helm upgrade --install oga-npqs-app ./deployments/helm/oga-app \
  -f ./deployments/helm/oga-app/npqs-app-values.yaml --history-max 3
```

This creates 3 distinct routes:
- `oga-fcau-app-national-single-window-platform.apps.sovecloud.akaza.lk`
- `oga-ird-app-national-single-window-platform.apps.sovecloud.akaza.lk`
- `oga-npqs-app-national-single-window-platform.apps.sovecloud.akaza.lk`

### OGA Backend Services (Generic Chart)

The `oga-backend/` chart is a **single generic chart** reused for every OGA agency.
Each agency gets its own **release name** and a dedicated `-f` values file inside `oga-backend/`:

```bash
helm upgrade --install oga-fcau-backend ./deployments/helm/oga-backend \
  -f ./deployments/helm/oga-backend/fcau-backend-values.yaml --history-max 3

helm upgrade --install oga-ird-backend  ./deployments/helm/oga-backend \
  -f ./deployments/helm/oga-backend/ird-backend-values.yaml  --history-max 3

helm upgrade --install oga-npqs-backend ./deployments/helm/oga-backend \
  -f ./deployments/helm/oga-backend/npqs-backend-values.yaml --history-max 3
```

To **add a new OGA**, create a new `<agency>-backend-values.yaml` in `oga-backend/` and run a new `helm upgrade --install` with a unique release name.

### Identity Provider (Thunder — Official Helm Chart)

Thunder uses the **official Asgardeo Helm chart** as a dependency. We only maintain a single override file at `idp/custom-values.yaml`.

```bash
# 1. Add / update the Thunder chart repo
helm repo add thunder https://asgardeo.github.io/helm-charts
helm repo update

# 2. Deploy with our custom overrides
helm upgrade --install idp-thunder thunder/thunder \
  -f ./idp/custom-values.yaml --history-max 3
```

> **Note:** The custom `idp/Dockerfile` wraps the official Thunder image to fix group permissions for OpenShift's random UID policy. The image is pushed as `ghcr.io/opennsw/idp` and referenced in `idp/custom-values.yaml`.

---

## 3. Database Initialization (PostgreSQL)

The platform uses a shared PostgreSQL instance (`nsw-db`). Each OGA needs an isolated database.

```bash
DB_POD=$(oc get pods -l app=nsw-db -o name)

# Create roles
oc exec $DB_POD -- psql -U postgres -c "CREATE ROLE oga_fcau_user WITH LOGIN PASSWORD 'oga_fcau_pw';"
oc exec $DB_POD -- psql -U postgres -c "CREATE ROLE oga_ird_user  WITH LOGIN PASSWORD 'oga_ird_pw';"
oc exec $DB_POD -- psql -U postgres -c "CREATE ROLE oga_npqs_user WITH LOGIN PASSWORD 'oga_npqs_pw';"

# Create databases (must be separate commands)
oc exec $DB_POD -- psql -U postgres -c "CREATE DATABASE \"oga-backend-fcau\" OWNER oga_fcau_user;"
oc exec $DB_POD -- psql -U postgres -c "CREATE DATABASE \"oga-backend-ird\"  OWNER oga_ird_user;"
oc exec $DB_POD -- psql -U postgres -c "CREATE DATABASE \"oga-backend-npqs\" OWNER oga_npqs_user;"
```

### 3.1 Thunder IDP Schema & App Registration

Since the official Thunder Helm setup hooks may be skipped or fail due to SCC quota constraints, you must manually initialize the Postgres schemas (`configdb`, `runtimedb`, `userdb`) and complete the bootstrap process to register `TRADER_PORTAL_APP` and `OGA_PORTAL_APP_NPQS`.

```bash
DB_POD=$(oc get pods -l app=nsw-db -o name | head -n 1)

# 1. Ensure the core IDP databases exist first
oc exec $DB_POD -- psql -U postgres -c "CREATE DATABASE configdb;" || true
oc exec $DB_POD -- psql -U postgres -c "CREATE DATABASE runtimedb;" || true
oc exec $DB_POD -- psql -U postgres -c "CREATE DATABASE userdb;" || true

# 2. Extract schema scripts from the Thunder image and inject them into Postgres
for db in configdb runtimedb userdb; do
  oc run tmp-idp --image=ghcr.io/opennsw/idp:latest --rm -i --restart=Never \
    --overrides='{"spec":{"containers":[{"name":"tmp-idp","image":"ghcr.io/opennsw/idp:latest","resources":{"limits":{"cpu":"500m","memory":"512Mi"},"requests":{"cpu":"200m","memory":"256Mi"}},"command":["cat","/opt/thunder/dbscripts/'$db'/postgres.sql"]}]}}' > /tmp/$db.sql
    
  # Filter out OpenShift's pod deletion noise and stream into DB
  grep -v "pod \"tmp-idp\" deleted" /tmp/$db.sql | oc exec -i $DB_POD -- psql -U postgres -d $db
done

# 3. Apply the custom bootstrap scripts and run the Setup Job
oc apply -f idp/idp-manual-bootstrap-cm.yaml
oc apply -f idp/idp-manual-setup.yaml

# 4. Wait for setup completion ("Complete" status)
oc get job idp-manual-setup -n national-single-window-platform -w
```

---

## 4. Verification & Troubleshooting

### Watch Pod Status
```bash
oc get pods -w
```

### Common Failure Signatures

| Error | Cause | Fix |
|:---|:---|:---|
| `failed quota: must specify limits` | Init-containers lack resource limits | Add `initResources` in `custom-values.yaml` |
| `Permission denied (signing.key)` | SCC `anyuid` not granted | `oc adm policy add-scc-to-user anyuid …` |
| `Exec format error` | ARM image on AMD host | Rebuild with `--platform linux/amd64` |
| `ImagePullBackOff` | Missing pull secret | Verify `oc get secret ghcr-secret` |

### Database Connectivity Check
```bash
oc exec $DB_POD -- psql -U thunder_user -d configdb -c "\dt"
```

### Success Logs
```bash
oc logs deployment/trader-app  | grep "listening on 0.0.0.0:8080"
oc logs deployment/idp-thunder | grep "Server started"
```

---

## Directory Structure

```
deployments/helm/
├── DEPLOYMENT.md               ← this file
├── nsw-api/                    ← NSW core backend API
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/
├── trader-app/                 ← Trader portal frontend
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/
└── oga-app/                    ← generic OGA portal frontend chart
    ├── Chart.yaml
    ├── values.yaml             ← generic base defaults
    ├── fcau-app-values.yaml    ← FCAU portal override
    ├── ird-app-values.yaml     ← IRD portal override
    ├── npqs-app-values.yaml    ← NPQS portal override
    └── templates/
└── oga-backend/                ← generic OGA backend chart
    ├── Chart.yaml
    ├── fcau-backend-values.yaml
    ├── ird-backend-values.yaml
    ├── npqs-backend-values.yaml
    └── templates/

idp/
├── custom-values.yaml          ← overrides for official Thunder Helm chart
└── Dockerfile                  ← wraps Thunder image for OpenShift UID fix
```