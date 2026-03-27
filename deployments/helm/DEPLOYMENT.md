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

The `oga-backend/` chart is a **single generic chart** (Deployment, Service, Route) reused for every OGA agency. It listens on port `8081` and exposes the `/api/oga/applications` endpoint.
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

## 3. Database Setup

The platform uses a shared PostgreSQL instance (`nsw-db`).

### 3.1 Automatic Migrations (nsw-api Init Container)

The `nsw-api` Helm chart includes an **init container** that automatically runs all idempotent migrations before the server starts. This handles:

- Schema creation (`001_initial_schema.sql`)
- All seed data (HS codes, forms, workflow templates, CHA entities, pre-consignment templates)
- V2 workflow table and mappings (`002_workflow_tem_v2.sql`)

**No manual steps needed** — migrations run automatically on every deploy.

To **disable** the init container (e.g. for debugging):
```bash
helm upgrade --install nsw-api ./deployments/helm/nsw-api \
  --set migrations.enabled=false --history-max 3
```

#### How It Works

1. An init container (`copy-migrations`) copies `.sql` files from the nsw-api image to a shared volume
2. A second init container (`run-migrations`) uses `postgres:16-alpine` to run each file via `psql`
3. All seed INSERTs use `ON CONFLICT DO NOTHING` — safe to re-run on every restart

#### OGA Submission URLs

The init container injects OGA backend service URLs into workflow node template configs via psql variables.
Configure in `values.yaml`:

```yaml
migrations:
  ogaSubmissionUrls:
    npqs: "http://oga-npqs-backend.<namespace>.svc.cluster.local/api/oga/inject"
    fcau: "http://oga-fcau-backend.<namespace>.svc.cluster.local/api/oga/inject"
    preconsignment: "http://oga-ird-backend.<namespace>.svc.cluster.local/api/oga/inject"
```

### 3.2 One-Time Destructive Migration

The `002_workflow_table.sql` migration is **excluded** from the init container because it contains `ALTER TABLE DROP COLUMN` statements. Run it **once** on a fresh or upgrading DB:

```bash
DB_POD=$(oc get pods -l deployment=nsw-db -o name | head -n 1)
oc exec -i $DB_POD -- psql -U postgres -d nsw_db < \
  backend/internal/database/migrations/002_workflow_table.sql
```

### 3.3 Manual Seeding (Emergency / Re-Seed)

If you need to manually re-seed:

```bash
DB_POD=$(oc get pods -l deployment=nsw-db -o name | head -n 1)

# Run individual seed files with psql variable substitution
cat backend/internal/database/migrations/001_insert_seed_workflow_node_templates.sql | \
  oc exec -i $DB_POD -- psql -U postgres -d nsw_db \
    -v ON_ERROR_STOP=1 \
    -v NPQS_OGA_SUBMISSION_URL="http://oga-npqs-backend.<namespace>.svc.cluster.local/api/oga/inject" \
    -v FCAU_OGA_SUBMISSION_URL="http://oga-fcau-backend.<namespace>.svc.cluster.local/api/oga/inject"

# Simple seed files (no psql variables)
oc exec -i $DB_POD -- psql -U postgres -d nsw_db < \
  backend/internal/database/migrations/001_insert_seed_hscodes.sql
```

> **Note:** The DB pod label is `deployment=nsw-db` (not `app=nsw-db`).

### 3.4 OGA Backend Databases

Each OGA backend needs an isolated database:

```bash
DB_POD=$(oc get pods -l deployment=nsw-db -o name | head -n 1)

# Create roles
oc exec $DB_POD -- psql -U postgres -c "CREATE ROLE oga_fcau_user WITH LOGIN PASSWORD 'oga_fcau_pw';"
oc exec $DB_POD -- psql -U postgres -c "CREATE ROLE oga_ird_user  WITH LOGIN PASSWORD 'oga_ird_pw';"
oc exec $DB_POD -- psql -U postgres -c "CREATE ROLE oga_npqs_user WITH LOGIN PASSWORD 'oga_npqs_pw';"

# Create databases (must be separate commands)
oc exec $DB_POD -- psql -U postgres -c "CREATE DATABASE \"oga-backend-fcau\" OWNER oga_fcau_user;"
oc exec $DB_POD -- psql -U postgres -c "CREATE DATABASE \"oga-backend-ird\"  OWNER oga_ird_user;"
oc exec $DB_POD -- psql -U postgres -c "CREATE DATABASE \"oga-backend-npqs\" OWNER oga_npqs_user;"
```

### 3.5 Thunder IDP Schema & App Registration

Since the official Thunder Helm setup hooks may be skipped or fail due to SCC quota constraints, you must manually initialize the Postgres schemas (`configdb`, `runtimedb`, `userdb`) and complete the bootstrap process to register `TRADER_PORTAL_APP` and `OGA_PORTAL_APP_NPQS`.

```bash
DB_POD=$(oc get pods -l deployment=nsw-db -o name | head -n 1)

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

## 4. Migration File Reference

| File | Type | Idempotent | Auto-Run |
|:---|:---|:---|:---|
| `001_initial_schema.sql` | Schema DDL | ✅ `IF NOT EXISTS` | ✅ |
| `001_insert_seed_hscodes.sql` | Seed data | ✅ `ON CONFLICT` | ✅ |
| `001_insert_seed_form_templates.sql` | Seed data | ✅ `ON CONFLICT` | ✅ |
| `001_insert_seed_workflow_node_templates.sql` | Seed data (needs psql vars) | ✅ `ON CONFLICT` | ✅ |
| `001_insert_seed_workflow_templates.sql` | Seed data | ✅ `ON CONFLICT` | ✅ |
| `001_insert_seed_workflow_hscode_map.sql` | Seed data | ✅ `ON CONFLICT` | ✅ |
| `001_insert_seed_pre_consignment_template.sql` | Seed data (needs psql vars) | ✅ `ON CONFLICT` | ✅ |
| `001_insert_cha_entity.sql` | Seed data | ✅ `ON CONFLICT` | ✅ |
| `002_workflow_table.sql` | **Destructive** (DROP COLUMN) | ❌ | ❌ Manual |
| `002_workflow_tem_v2.sql` | Schema + Seed | ✅ `IF NOT EXISTS` + `ON CONFLICT` | ✅ |

---

## 5. Verification & Troubleshooting

### Watch Pod Status
```bash
oc get pods -w
```

### Verify Init Container Logs
```bash
oc logs deployment/nsw-api -c copy-migrations
oc logs deployment/nsw-api -c run-migrations
```

### Common Failure Signatures

| Error | Cause | Fix |
|:---|:---|:---|
| `failed quota: must specify limits` | Init-containers lack resource limits | Add `initResources` in `custom-values.yaml` |
| `Permission denied (signing.key)` | SCC `anyuid` not granted | `oc adm policy add-scc-to-user anyuid …` |
| `Exec format error` | ARM image on AMD host | Rebuild with `--platform linux/amd64` |
| `ImagePullBackOff` | Missing pull secret | Verify `oc get secret ghcr-secret` |
| `record not found` (500 on consignment init) | Missing seed data in `workflow_template_maps` | Check init container logs; re-seed manually (§3.3) |
| CORS preflight failure | Missing OGA portal origin in `nsw-api` CORS config | Update `cors.allowedOrigins` in `nsw-api/values.yaml` |
| `OGA_DB_PASSWORD is required` | Missing env var in backend deployment | Verify `OGA_DB_PASSWORD` exists in backend values (§2) |

### Database Connectivity Check
```bash
DB_POD=$(oc get pods -l deployment=nsw-db -o name | head -n 1)
oc exec $DB_POD -- psql -U postgres -d nsw_db -c "\dt"
oc exec $DB_POD -- psql -U thunder_user -d configdb -c "\dt"
```

### Success Logs
```bash
oc logs deployment/trader-app  | grep "listening on 0.0.0.0:8080"
oc logs deployment/idp-thunder | grep "Server started"
oc logs deployment/nsw-api -c run-migrations | grep "Migrations completed"
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
│       ├── _helpers.tpl
│       ├── deployment.yaml      (includes migration init containers)
│       ├── db-migrations-cm.yaml (migration runner script)
│       ├── route.yaml
│       └── service.yaml
├── trader-app/                 ← Trader portal frontend
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/
├── oga-app/                    ← generic OGA portal frontend chart
│   ├── Chart.yaml
│   ├── values.yaml             ← generic base defaults
│   ├── fcau-app-values.yaml    ← FCAU portal override
│   ├── ird-app-values.yaml     ← IRD portal override
│   ├── npqs-app-values.yaml    ← NPQS portal override
│   └── templates/
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