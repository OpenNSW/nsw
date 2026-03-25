# NSW Platform: Deployment & Verification Guide

This guide covers the setup for the National Single Window (NSW) platform on OpenShift (Akaza GovCloud).

## 1. Database Initialization (PostgreSQL)
The NSW platform uses a shared PostgreSQL instance (`nsw-db`). Databases and users must be created individually to avoid transaction block errors.

### Setup Isolated OGA Databases
Run these commands to create roles and databases for the Other Government Agencies (OGAs). 
*Note: We use hyphenated database names to match the Helm values.*

```bash
# Define the DB pod name
DB_POD=$(oc get pods -l app=nsw-db -o name)

# 1. Create Roles (Users)
oc exec $DB_POD -- psql -U postgres -c "CREATE ROLE oga_fcau_user WITH LOGIN PASSWORD 'oga_fcau_pw';"
oc exec $DB_POD -- psql -U postgres -c "CREATE ROLE oga_ird_user WITH LOGIN PASSWORD 'oga_ird_pw';"
oc exec $DB_POD -- psql -U postgres -c "CREATE ROLE oga_npqs_user WITH LOGIN PASSWORD 'oga_npqs_pw';"

# 2. Create Databases (Must be separate commands)
oc exec $DB_POD -- psql -U postgres -c "CREATE DATABASE \"oga-backend-fcau\" OWNER oga_fcau_user;"
oc exec $DB_POD -- psql -U postgres -c "CREATE DATABASE \"oga-backend-ird\" OWNER oga_ird_user;"
oc exec $DB_POD -- psql -U postgres -c "CREATE DATABASE \"oga-backend-npqs\" OWNER oga_npqs_user;"
```

---

## 2. Build and Push (GHCR)
If building on **Apple Silicon (M1/M2/M3)**, you must use `buildx` to target `linux/amd64`.

```bash
# From the repository root
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/trader-app:latest -f portals/apps/trader-app/Dockerfile ./portals --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/oga-app:latest -f portals/apps/oga-app/Dockerfile ./portals --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/nsw-api:latest -f backend/Dockerfile ./backend --push
```

---

## 3. Deploy Helm Charts
Always use `--history-max 3` to stay within the 20-secret quota limit of the project.

### Core NSW Services
```bash
helm upgrade --install nsw-api ./deployments/helm/nsw-api --history-max 3
helm upgrade --install trader-app ./deployments/helm/trader-app --history-max 3
helm upgrade --install oga-app ./deployments/helm/oga-app --history-max 3
```

### OGA Backend Services (Generic Chart)
Use the generic `oga-backend` chart with agency-specific value files.
```bash
helm upgrade --install oga-fcau-backend ./deployments/helm/oga-backend -f ./deployments/helm/oga-backend/fcau-backend-values.yaml --history-max 3
helm upgrade --install oga-ird-backend ./deployments/helm/oga-backend -f ./deployments/helm/oga-backend/ird-backend-values.yaml --history-max 3
helm upgrade --install oga-npqs-backend ./deployments/helm/oga-backend -f ./deployments/helm/oga-backend/npqs-backend-values.yaml --history-max 3
```

### Identity Provider (Thunder)
**Critical:** The Thunder chart requires `anyuid` SCC privileges which must be granted by a Cluster Admin.

```bash
# 1. Ensure you are in install/helm/
cd install/helm/

# 2. Deploy using the rectified custom-values.yaml (handles Init-Container Quotas)
helm upgrade --install idp-thunder . -f values.yaml -f custom-values.yaml --history-max 3
```

---

## 4. Verification & Troubleshooting

### A. Watch Pod Status
```bash
oc get pods -w
```

### B. Common Failure Signatures
| Error | Cause | Fix |
| :--- | :--- | :--- |
| **`failed quota: must specify limits`** | Init-containers lack resources. | Update `initResources` in `custom-values.yaml`. |
| **`Permission denied (signing.key)`** | SCC `anyuid` not granted. | Contact Admin to run `oc adm policy add-scc-to-user anyuid`. |
| **`Exec format error`** | Wrong architecture (ARM vs AMD). | Rebuild image using `docker buildx --platform linux/amd64`. |
| **`ImagePullBackOff`** | Missing `ghcr-secret`. | Run `oc get secret ghcr-secret` to verify presence. |

### C. Database Connectivity Check
Verify that the `idp-thunder-setup` job successfully populated the tables:
```bash
oc exec $DB_POD -- psql -U thunder_user -d configdb -c "\dt"
```

### D. Success Signature (Logs)
```bash
oc logs deployment/trader-app | grep "listening on 0.0.0.0:8080"
oc logs deployment/idp-thunder | grep "Server started"
```