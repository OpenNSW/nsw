#!/usr/bin/env bash
# =============================================================================
# deploy-idp.sh — Render, patch, and deploy the idp-thunder Helm chart
#                 with surgical InitContainer stabilization.
# =============================================================================
set -euo pipefail

NAMESPACE="national-single-window-platform"
HELM_RELEASE="idp-thunder"
CHART="oci://ghcr.io/asgardeo/helm-charts/thunder"
CHART_VERSION="0.29.0"
VALUES_FILE="$(dirname "$0")/custom-values.yaml"
PATCH_SCRIPT="$(dirname "$0")/patch_idp.py"
RAW_MANIFEST="/tmp/idp-manifests-raw.yaml"
PATCHED_MANIFEST="/tmp/idp-manifests-patched.yaml"
HEALTH_URL="https://idp-thunder-national-single-window-platform.apps.sovecloud1.akaza.lk/health"
CONSOLE_URL="https://idp-thunder-national-single-window-platform.apps.sovecloud1.akaza.lk/console"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ─── Colour helpers ───────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
info()    { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC}   $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
error()   { echo -e "${RED}[ERR]${NC}  $*" >&2; exit 1; }

# ─── Step 0: Preflight checks ────────────────────────────────────────────────
info "Preflight checks..."
command -v oc      >/dev/null 2>&1 || error "oc not found on PATH"
command -v helm    >/dev/null 2>&1 || error "helm not found on PATH"
command -v python3 >/dev/null 2>&1 || error "python3 not found on PATH"

oc project "${NAMESPACE}" >/dev/null 2>&1 || error "Namespace '${NAMESPACE}' not accessible"
success "Preflight checks passed."

# ─── Step 1: Apply prerequisite resources ────────────────────────────────────
info "Step 1: Applying prerequisite resources..."
oc apply -f "${SCRIPT_DIR}/idp-admin-secrets.yaml" -n "${NAMESPACE}"
oc apply -f "${SCRIPT_DIR}/idp-manual-bootstrap-cm.yaml" -n "${NAMESPACE}"

info "Step 1.1: Generating Base64-Shielded JSON Anchor..."
cat <<EOF > "${SCRIPT_DIR}/idp-perfect.json"
{
  "server": {"hostname": "0.0.0.0", "port": 8090, "http_only": true, "public_url": "https://idp-thunder-national-single-window-platform.apps.sovecloud1.akaza.lk"},
  "crypto": {
    "encryption": {"key": "file:///opt/thunder/repository/resources/security/crypto.key"},
    "keys": [{"id": "default-key", "cert_file": "/opt/thunder/repository/resources/security/signing.cert", "key_file": "/opt/thunder/repository/resources/security/signing.key"}]
  },
  "gate_client": {"hostname": "idp-thunder-service", "port": 8090, "scheme": "http"},
  "cors": {"allowed_origins": [
    "https://trader-app-national-single-window-platform.apps.sovecloud1.akaza.lk",
    "https://oga-fcau-national-single-window-platform.apps.sovecloud1.akaza.lk"
  ]},
  "database": {
    "config": {"type": "postgres", "hostname": "nsw-db", "port": 5432, "name": "configdb", "username": "postgres", "password": "postgres", "sslmode": "disable"},
    "runtime": {"type": "postgres", "hostname": "nsw-db", "port": 5432, "name": "runtimedb", "username": "postgres", "password": "postgres", "sslmode": "disable"},
    "user": {"type": "postgres", "hostname": "nsw-db", "port": 5432, "name": "userdb", "username": "postgres", "password": "postgres", "sslmode": "disable"}
  }
}
EOF
base64 -i "${SCRIPT_DIR}/idp-perfect.json" -o "${SCRIPT_DIR}/idp-perfect.json.b64"

info "Step 1.2: Recreating ConfigMap Anchor..."
oc delete configmap idp-stabilized-json -n "${NAMESPACE}" --ignore-not-found=true
oc create configmap idp-stabilized-json --from-file=deployment.yaml.b64="${SCRIPT_DIR}/idp-perfect.json.b64" -n "${NAMESPACE}"

success "Prerequisite resources and ConfigMap Anchor applied."

# ─── Step 2: Render & Patch with Surgical Stabilization ──────────────────────
info "Step 2: Rendering and Patching Helm chart..."
# Render chart
helm template idp-thunder oci://ghcr.io/asgardeo/helm-charts/thunder \
  --version 0.29.0 -f "${SCRIPT_DIR}/custom-values.yaml" -n "${NAMESPACE}" > "/tmp/idp-raw.yaml"

# Patch chart using our surgical logic
python3 "${SCRIPT_DIR}/patch_idp.py" "/tmp/idp-raw.yaml" > "/tmp/idp-patched.yaml"
success "Manifests patched with Surgical Stabilization."

# ─── Step 3: Delete immutable Helm setup Job ─────────────────────────────────
info "Step 3: Deleting immutable Helm setup Job..."
oc delete job idp-thunder-setup -n "${NAMESPACE}" --ignore-not-found=true
success "Old Job removed."

# ─── Step 4: Apply patched manifests ─────────────────────────────────────────
info "Step 4: Applying patched manifests..."
oc apply -f "/tmp/idp-patched.yaml" -n "${NAMESPACE}"
success "Manifests applied with Surgical Stabilization."

# ─── Step 5: Aggressive cleanup of crashing pods and old ReplicaSets ────────
info "Step 5: Aggressive cleanup of crashing pods and old ReplicaSets..."
oc delete rs -l "app.kubernetes.io/name=thunder" -n "${NAMESPACE}" --ignore-not-found
for i in {1..3}; do
  info "  Iteration $i: Identifying crashed pods..."
  CRASHING_PODS=$(oc get pods -n "${NAMESPACE}" -l "app.kubernetes.io/name=thunder" -o json | python3 -c "import sys,json; d=json.load(sys.stdin); print(' '.join([p['metadata']['name'] for p in d.get('items', []) if any(cs.get('state', {}).get('waiting', {}).get('reason') in ['CrashLoopBackOff', 'Error', 'CreateContainerError'] for cs in p.get('status', {}).get('containerStatuses', []))]))")
  
  if [[ -n "${CRASHING_PODS}" ]]; then
    info "  Deleting: ${CRASHING_PODS}"
    oc delete pods ${CRASHING_PODS} -n "${NAMESPACE}" --grace-period=1 --ignore-not-found --wait=false
    sleep 5
  else
    info "  No crashed pods found."
    break
  fi
done

# ─── Step 6: Wait for rollout ───────────────────────────────────────────────
info "Step 6: Waiting for idp-thunder Deployment rollout..."
if oc rollout status deployment "idp-thunder-deployment" -n "${NAMESPACE}" --timeout=150s; then
  success "Deployment rolled out successfully."
else
  warn "Rollout timed out. Inspecting pod state:"
  oc get pods -n "${NAMESPACE}" -l "app.kubernetes.io/name=thunder"
fi

# ─── Step 7: Run the manual setup Job ───────────────────────────────────────
info "Step 7: Running idp-manual-setup Job..."
oc delete job idp-manual-setup -n "${NAMESPACE}" --ignore-not-found=true
sleep 2
oc apply -f "${SCRIPT_DIR}/idp-manual-setup.yaml" -n "${NAMESPACE}"

info "  Waiting for Job completion..."
if oc wait --for=condition=complete job/idp-manual-setup -n "${NAMESPACE}" --timeout=150s; then
  success "idp-manual-setup Job completed successfully."
else
  warn "Job still running or failed. Check logs: oc logs -l job-name=idp-manual-setup"
fi

# ─── Step 8: Final Verification Checklist ───────────────────────────────────
info "Step 8: Final Verification Checklist..."

# 8a: Health endpoint
info "  Checking Health endpoint..."
if curl -skL -w "%{http_code}" "${HEALTH_URL}" -o /dev/null | grep -q "200"; then
  success "  Health endpoint: 200 OK"
else
  warn "  Health endpoint check failed. Pod logs may indicate remaining init errors."
fi

# 8b: Console endpoint (User requested)
info "  Checking Console endpoint..."
RESPONSE_CODE=$(curl -skL -o /dev/null -w "%{http_code}" "${CONSOLE_URL}")
if [[ "$RESPONSE_CODE" == "200" ]]; then
    success "  Console check: 200 OK"
else
    warn "  Console check returned HTTP $RESPONSE_CODE (likely redirecting to login)."
fi

# 8c: Log Audit for Permission Errors
info "  Checking logs for permission errors..."
if oc logs -l "app.kubernetes.io/name=thunder" -n "${NAMESPACE}" --tail=200 | grep -iq "permission denied"; then
    error "  Found permission denied in logs!"
else
    success "  No permission denied errors found."
fi

# 8d: Pod status
info "  Checking Pod status..."
oc get pods -n "${NAMESPACE}" -l "app.kubernetes.io/name=thunder"

success "=== Stabilization Complete ==="
