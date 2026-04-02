# NSW Platform — Helm Deployment Guide

> **Target:** OpenShift (Akaza GovCloud)  
> **Constraint:** Always pass `--history-max 1` to stay within the 20-secret quota limit for multiple OGA pods.

---

## Chart Inventory

| Release Name | Chart Source | Values / Config |
|:---|:---|:---|
| `nsw-api` | `./deployments/helm/nsw-api` | `values.yaml` (inline) |
| `trader-app` | `./deployments/helm/trader-app` | `values.yaml` (inline) |
| `oga-multi-frontend` | `./deployments/helm/oga-multi-frontend` | Unified Generic Frontend for all agencies. |
| `oga-<agency>-backend` | `./deployments/helm/oga-backend` | Per-agency values file (e.g. `fcau-backend-values.yaml`) |
| `idp-thunder` | **Official Thunder chart** (`ghcr.io/asgardeo/helm-charts/thunder`) | `idp/custom-values.yaml` |
| `temporal` | `./deployments/helm/temporal` | `values.yaml` (unified server/ui) |

---

## 1. Build & Push Images (GHCR)

Build with `linux/amd64` when on Apple Silicon:

```bash
# From the repository root
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/nsw-api:latest    -f backend/Dockerfile ./backend --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/trader-app:latest  -f portals/apps/trader-app/Dockerfile ./portals --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/oga-multi-frontend:latest -f deployments/helm/oga-multi-frontend/Dockerfile . --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/oga-backend:latest -f oga/Dockerfile ./oga --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/idp:latest         -f idp/Dockerfile . --push
```

### 1.1 Temporal Image Mirroring (Manual)

Due to Docker Hub rate limits on OpenShift, the Temporal `auto-setup` image must be mirrored to GHCR:

```bash
# Pull the amd64 variant (OpenShift is x86_64)
docker pull --platform linux/amd64 temporalio/auto-setup:1.28.3

# Tag and Push to your GHCR
docker tag temporalio/auto-setup:1.28.3 ghcr.io/opennsw/temporal-auto-setup:1.28.3
docker push ghcr.io/opennsw/temporal-auto-setup:1.28.3
```

---

## 2. Deploy / Upgrade Helm Charts

All deployments are executed automatically using the wrapper script from the **repository root**. This script securely synchronizes database migrations before configuring the OpenShift clusters.

```bash
./scripts/deploy-helm.sh
```

### Core NSW Services

The deploy script automatically manages memory quotas and service routes for the core backend and trader portal.

### OGA Portal Frontends (Consolidated Chart)

The `oga-multi-frontend/` chart is a single consolidated deployment that dynamically routes all OGA portal requests based on the host header.

```bash
helm upgrade --install oga-multi-frontend ./deployments/helm/oga-multi-frontend --history-max 1
```

This single deployment serves the 3 distinct routes seamlessly:
- `oga-fcau-app-national-single-window-platform.apps.sovecloud1.akaza.lk`
- `oga-ird-app-national-single-window-platform.apps.sovecloud1.akaza.lk`
- `oga-npqs-app-national-single-window-platform.apps.sovecloud1.akaza.lk`

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

Thunder officially hosts its Helm Charts on the GitHub Container Registry. We deploy it remotely via OCI and inject our `custom-values.yaml`.

```bash
helm upgrade --install idp-thunder oci://ghcr.io/asgardeo/helm-charts/thunder \
  --version 0.29.0 -f ./deployments/helm/idp/custom-values.yaml --history-max 1
```

> **Note:** We are using the official Thunder image (`ghcr.io/asgardeo/thunder:0.29.0`) to avoid permissions and ImagePullBackOff issues. The `custom-values.yaml` handles database connections and environment settings.

### 2.4 Temporal (Unified Server & UI)

Temporal is deployed as an "All-in-One" service. It uses an `emptyDir` mount at `/etc/temporal/config` to allow the `auto-setup` script to generate configuration on OpenShift's read-only filesystem.

```bash
helm upgrade --install temporal ./deployments/helm/temporal --history-max 3
```

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
3. All seed INSERTs now use `ON CONFLICT (id) DO UPDATE SET config = EXCLUDED.config` — this automatically synchronizes configuration changes between `values.yaml` and the database on every restart (eliminating manual SQL steps).

#### OGA Submission URLs

The init container injects OGA backend service URLs into workflow node template configs via psql variables.
Configure in `values.yaml`:

```yaml
migrations:
  ogaSubmissionUrls:
    npqs: "http://oga-npqs-backend.<namespace>.svc.cluster.local/api/oga/inject"
    fcau: "http://oga-fcau-backend.<namespace>.svc.cluster.local/api/oga/inject"
    ird: "http://oga-ird-backend.<namespace>.svc.cluster.local/api/oga/inject"
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
| `001_insert_seed_form_templates.sql` | Seed data | ✅ `DO UPDATE` | ✅ |
| `001_insert_seed_workflow_node_templates.sql` | Seed data | ✅ `DO UPDATE` | ✅ |
| `001_insert_seed_workflow_templates.sql` | Seed data | ✅ `DO UPDATE` | ✅ |
| `001_insert_seed_workflow_hscode_map.sql` | Seed data | ✅ `ON CONFLICT` | ✅ |
| `001_insert_seed_pre_consignment_template.sql` | Seed data | ✅ `DO UPDATE` | ✅ |
| `001_insert_cha_entity.sql` | Seed data | ✅ `ON CONFLICT` | ✅ |
| `002_workflow_table.sql` | **Destructive** (DROP COLUMN) | ❌ | ❌ Manual |
| `002_workflow_tem_v2.sql` | Schema + Seed | ✅ `IF NOT EXISTS` | ✅ |

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
| `no registered service found` (500 on task submit) | Port mismatch between DB and Registry lookup | Ensure OGA URLs in `values.yaml` include ports (81, 82) to match `services-cm.yaml`. |
| `CORS preflight failure` | Missing OGA portal origin in `nsw-api` CORS config | Update `cors.allowedOrigins` in `nsw-api/values.yaml`. Note: CORS is origin-sensitive (host + port). |
| `OGA_DB_PASSWORD is required` | Missing env var in backend deployment | Verify Kubernetes Secret `nsw-db-credentials` exists and is referenced (§2) |
| `unable to create open ... permission denied` | Temporal auto-setup cannot write config | Ensure `emptyDir` is mounted to `/etc/temporal/config` |
| `manifest unknown` (Temporal) | Incorrect mirrored tag in GHCR | Verify `ghcr.io/opennsw/temporal-auto-setup:1.28.3` exists |

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

## 6. Production Security & Robustness Best Practices

To comply with OpenShift `restricted-v2` Security Context Constraints (SCC) and ensure zero-manual-intervention migrations:

1. **Numeric Ports**: Always use numeric `targetPort` (e.g., `8081`) in Services and Routes to avoid named port resolution issues.
2. **Unprivileged UID**: All images must support running as any random high UID (UID 101/1000). Avoid `chown` in Dockerfiles; use `chmod -R g+w` for writable paths.
3. **Secret-Driven Configuration**: NEVER put plaintext passwords in `values.yaml`. Use Kubernetes Secrets (`nsw-db-credentials`, `temporal-certs`, etc.) and mount them via `secretKeyRef`.
4. **Automated Schema Sync**: All seed INSERTs must use `DO UPDATE SET config = EXCLUDED.config` to ensure that `values.yaml` updates are automatically reflected in the database.

---

## 7. Directory Structure

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
├── temporal/                   ← Temporal unified server/ui chart
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/

idp/
├── custom-values.yaml          ← overrides for official Thunder Helm chart
└── Dockerfile                  ← wraps Thunder image for OpenShift UID fix
```