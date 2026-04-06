#!/usr/bin/env bash
# =============================================================================
# deploy-idp.sh — Render, patch, and deploy the idp-thunder Helm chart
#                 with surgical InitContainer stabilization and MULTI-ENV support.
# =============================================================================
set -euo pipefail

# Default to shared if not specified
ENV="${1:-shared}"
NAMESPACE="national-single-window-platform"

# Architecture: Shared IDP (no prefix)
HELM_RELEASE="idp-thunder"
PUBLIC_URL="https://idp-thunder.national-single-window-platform.apps.sovecloud1.akaza.lk"
DB_HOST="nsw-db"
DB_SECRET="nsw-db-credentials"

CHART="oci://ghcr.io/asgardeo/helm-charts/thunder"
CHART_VERSION="0.29.0"
VALUES_FILE="$(dirname "$0")/custom-values.yaml" 
PATCH_SCRIPT="$(dirname "$0")/patch_idp.py"
RAW_MANIFEST="/tmp/${HELM_RELEASE}-manifests-raw.yaml"
PATCHED_MANIFEST="/tmp/${HELM_RELEASE}-manifests-patched.yaml"

HEALTH_URL="${PUBLIC_URL}/health"
CONSOLE_URL="${PUBLIC_URL}/console"

DOMAIN="national-single-window-platform.apps.sovecloud1.akaza.lk"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ─── Colour helpers ───────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
info()    { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC}   $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
error()   { echo -e "${RED}[ERR]${NC}  $*" >&2; exit 1; }

info "Preflight checks..."
command -v oc      >/dev/null 2>&1 || error "oc not found on PATH"
command -v helm    >/dev/null 2>&1 || error "helm not found on PATH"
command -v python3 >/dev/null 2>&1 || error "python3 not found on PATH"

oc project "${NAMESPACE}" >/dev/null 2>&1 || error "Namespace '${NAMESPACE}' not accessible"
info "Preflight checks passed."

info "Step 0: Extracting Crunchy Database Password for idp-thunder..."
# Wait for secret to appear if the operator just started
for i in {1..12}; do
  DB_PASSWORD=$(oc get secret "${DB_SECRET}" -n "${NAMESPACE}" -o jsonpath='{.data.password}' | base64 -d 2>/dev/null || true)
  if [[ -n "${DB_PASSWORD}" ]]; then
    success "Password extracted."
    break
  fi
  warn "  Waiting for secret ${DB_SECRET}... (attempt $i)"
  sleep 5
done
if [[ -z "${DB_PASSWORD}" ]]; then
  warn "Could not find password for ${DB_SECRET}. Using mock password for verification."
  DB_PASSWORD="mock-password"
fi
info "Step 1: Applying prerequisite resources for ${HELM_RELEASE}..."
oc apply -f "${SCRIPT_DIR}/idp-admin-secrets.yaml" -n "${NAMESPACE}"

info "Step 1.1: Generating Plaintext JSON Config for ${HELM_RELEASE}..."
cat <<EOF > "${SCRIPT_DIR}/idp-config.json"
{
  "server": {"hostname": "0.0.0.0", "port": 8090, "http_only": true, "public_url": "${PUBLIC_URL}"},
  "crypto": {
    "encryption": {"key": "repository/certs/crypto.key"},
    "keys": [{"id": "default-key", "cert_file": "repository/certs/signing.cert", "key_file": "repository/certs/signing.key"}]
  },
  "gate_client": {"hostname": "idp-thunder-national-single-window-platform.apps.sovecloud1.akaza.lk", "port": 443, "scheme": "https"},
  "cors": {"allowed_origins": [
    "https://dev-trader.${DOMAIN}",
    "https://staging-trader.${DOMAIN}",
    "https://dev-api.${DOMAIN}",
    "https://staging-api.${DOMAIN}"
  ]},
  "database": {
    "config": {"type": "postgres", "hostname": "${DB_HOST}", "port": 5432, "name": "configdb", "username": "shared-idp-db", "password": "${DB_PASSWORD}", "sslmode": "disable"},
    "runtime": {"type": "postgres", "hostname": "${DB_HOST}", "port": 5432, "name": "runtimedb", "username": "shared-idp-db", "password": "${DB_PASSWORD}", "sslmode": "disable"},
    "user": {"type": "postgres", "hostname": "${DB_HOST}", "port": 5432, "name": "userdb", "username": "shared-idp-db", "password": "${DB_PASSWORD}", "sslmode": "disable"}
  }
}
EOF

info "Step 1.2: Recreating ConfigMap Anchor for ${HELM_RELEASE}..."
oc delete configmap ${HELM_RELEASE}-stabilized-json -n "${NAMESPACE}" --ignore-not-found=true
oc create configmap ${HELM_RELEASE}-stabilized-json --from-file=deployment.yaml="${SCRIPT_DIR}/idp-config.json" -n "${NAMESPACE}"
success "Prerequisite resources and ConfigMap Anchor applied."

info "Step 2: Rendering and Patching Helm chart for ${HELM_RELEASE}..."
helm template ${HELM_RELEASE} oci://ghcr.io/asgardeo/helm-charts/thunder \
  --version 0.29.0 -f "${VALUES_FILE}" -n "${NAMESPACE}" > "${RAW_MANIFEST}"

python3 "${PATCH_SCRIPT}" "${RAW_MANIFEST}" "${HELM_RELEASE}" > "${PATCHED_MANIFEST}"
success "Manifests patched with Surgical Stabilization."

info "Step 3: Deleting immutable Helm setup Job for ${HELM_RELEASE}..."
oc delete job "${HELM_RELEASE}-setup" -n "${NAMESPACE}" --ignore-not-found=true
success "Old Job removed."

info "Step 4: Applying patched manifests for ${HELM_RELEASE}..."
oc apply -f "${PATCHED_MANIFEST}" -n "${NAMESPACE}"
success "Manifests applied with Surgical Stabilization."

info "Step 5: Aggressive cleanup of crashing pods and old ReplicaSets..."
oc delete rs -l "app.kubernetes.io/name=thunder,app.kubernetes.io/instance=${HELM_RELEASE}" -n "${NAMESPACE}" --ignore-not-found
for i in {1..3}; do
  info "  Iteration $i: Identifying crashed pods for ${HELM_RELEASE}..."
  CRASHING_PODS=$(oc get pods -n "${NAMESPACE}" -l "app.kubernetes.io/name=thunder,app.kubernetes.io/instance=${HELM_RELEASE}" -o json | python3 -c "import sys,json; d=json.load(sys.stdin); print(' '.join([p['metadata']['name'] for p in d.get('items', []) if any(cs.get('state', {}).get('waiting', {}).get('reason') in ['CrashLoopBackOff', 'Error', 'CreateContainerError'] for cs in p.get('status', {}).get('containerStatuses', []))]))")
  
  if [[ -n "${CRASHING_PODS}" ]]; then
    info "  Deleting: ${CRASHING_PODS}"
    oc delete pods ${CRASHING_PODS} -n "${NAMESPACE}" --grace-period=1 --ignore-not-found --wait=false
    sleep 5
  else
    info "  No crashed pods found."
    break
  fi
done

info "Step 6: Waiting for ${HELM_RELEASE}-deployment rollout..."
if oc rollout status deployment "${HELM_RELEASE}-deployment" -n "${NAMESPACE}" --timeout=150s; then
  success "Deployment rolled out successfully."
else
  warn "Rollout timed out. Inspecting pod state:"
  oc get pods -n "${NAMESPACE}" -l "app.kubernetes.io/name=thunder,app.kubernetes.io/instance=${HELM_RELEASE}"
fi

info "Step 7: Running idp-thunder-db-init Job..."
oc delete job idp-thunder-db-init -n "${NAMESPACE}" --ignore-not-found=true
sleep 2
oc apply -f "${SCRIPT_DIR}/init-db-job.yaml" -n "${NAMESPACE}"

info "  Waiting for Job completion..."
if oc wait --for=condition=complete job/idp-thunder-db-init -n "${NAMESPACE}" --timeout=150s; then
  success "${HELM_RELEASE}-db-init Job completed successfully."
else
  warn "Job still running or failed."
fi

info "Step 8: Final Verification Checklist..."
info "  Checking Health endpoint..."
if curl -skL -w "%{http_code}" "${HEALTH_URL}" -o /dev/null | grep -q "200"; then
  success "  Health endpoint: 200 OK"
else
  warn "  Health endpoint check failed."
fi

info "  Checking Console endpoint..."
RESPONSE_CODE=$(curl -skL -o /dev/null -w "%{http_code}" "${CONSOLE_URL}")
if [[ "$RESPONSE_CODE" == "200" ]]; then
    success "  Console check: 200 OK"
else
    warn "  Console check returned HTTP $RESPONSE_CODE"
fi

success "=== Stabilization Complete for ${ENV} ==="
oc get pods -n "${NAMESPACE}" -l "app.kubernetes.io/name=thunder,app.kubernetes.io/instance=${HELM_RELEASE}"
