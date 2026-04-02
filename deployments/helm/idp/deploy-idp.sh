#!/usr/bin/env bash
# =============================================================================
# deploy-idp.sh — Render, patch, and deploy the idp-thunder Helm chart
#                 to the national-single-window-platform namespace on OpenShift
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
python3 -c "import yaml" 2>/dev/null    || error "PyYAML not installed (pip3 install pyyaml)"

oc whoami >/dev/null 2>&1 || error "Not logged in to OpenShift (run: oc login ...)"
oc project "${NAMESPACE}" >/dev/null 2>&1 || error "Namespace '${NAMESPACE}' not found or not accessible"
success "Preflight checks passed."

# ─── Step 1: Apply prerequisite secrets / configmaps ────────────────────────
info "Step 1: Applying prerequisite resources..."
oc apply -f "${SCRIPT_DIR}/idp-admin-secrets.yaml" -n "${NAMESPACE}"
oc apply -f "${SCRIPT_DIR}/idp-manual-bootstrap-cm.yaml" -n "${NAMESPACE}"
success "Prerequisite resources applied."

# ─── Step 2: Verify the signing keys secret exists ──────────────────────────
info "Step 2: Verifying 'thunder-signing-keys' secret..."
if ! oc get secret thunder-signing-keys -n "${NAMESPACE}" >/dev/null 2>&1; then
  error "'thunder-signing-keys' secret not found in namespace '${NAMESPACE}'."
fi
success "  'thunder-signing-keys' secret confirmed."

# ─── Step 3: Render the Helm chart ──────────────────────────────────────────
info "Step 3: Rendering Helm chart ${CHART} @ ${CHART_VERSION}..."
helm template "${HELM_RELEASE}" "${CHART}" \
  --version "${CHART_VERSION}" \
  -f "${VALUES_FILE}" \
  -n "${NAMESPACE}" \
  > "${RAW_MANIFEST}"
success "Manifest rendered."

# ─── Step 4: Patch the rendered manifests ───────────────────────────────────
info "Step 4: Patching manifests (removing hardcoded UIDs/GIDs)..."
python3 "${PATCH_SCRIPT}" "${RAW_MANIFEST}" > "${PATCHED_MANIFEST}"
success "Patched manifests written to ${PATCHED_MANIFEST}"

# ─── Step 5: Diff check (informational) ─────────────────────────────────────
info "Step 5: Verifying no hardcoded securityContext fields remain..."
if grep -E "^\s+(runAsUser|runAsGroup|fsGroup):" "${PATCHED_MANIFEST}"; then
  warn "Some securityContext fields still remain after patching! Review ${PATCHED_MANIFEST}."
else
  success "No forbidden fields detected."
fi

# ─── Step 6a: Delete immutable Helm setup Job ────────────────────────────────
info "Step 6a: Deleting immutable Helm setup Job (idp-thunder-setup)..."
oc delete job idp-thunder-setup -n "${NAMESPACE}" --ignore-not-found=true
success "Old Job removed."

# ─── Step 6b: Apply patched manifests ────────────────────────────────────────
info "Step 6b: Applying patched manifests to namespace '${NAMESPACE}'..."
oc apply -f "${PATCHED_MANIFEST}" -n "${NAMESPACE}"
success "Manifests applied."

# ─── Step 6c: Cleanup crashed pods immediately ──────────────────────────────
info "Step 6c: Cleaning up any existing crashed pods..."
CRASHING_PODS=$(oc get pods -n "${NAMESPACE}" -l "app.kubernetes.io/name=thunder" -o json | python3 -c "import sys,json; d=json.load(sys.stdin); print(' '.join([p['metadata']['name'] for p in d.get('items', []) if any(cs.get('state', {}).get('waiting', {}).get('reason') in ['CrashLoopBackOff', 'Error'] for cs in p.get('status', {}).get('containerStatuses', []))]))")

if [[ -n "${CRASHING_PODS}" ]]; then
  info "  Deleting crashing pods: ${CRASHING_PODS}"
  oc delete pods ${CRASHING_PODS} -n "${NAMESPACE}" --grace-period=1 --ignore-not-found
else
  info "  No crashed pods found."
fi

# ─── Step 7: Wait for rollout ───────────────────────────────────────────────
info "Step 7: Waiting for idp-thunder Deployment rollout..."
if oc rollout status deployment "${HELM_RELEASE}-deployment" -n "${NAMESPACE}" --timeout=180s; then
  success "Deployment rolled out successfully."
else
  warn "Rollout did not complete within 180s. Pod status:"
  oc get pods -n "${NAMESPACE}" -l "app.kubernetes.io/name=thunder" --sort-by='.status.startTime'
fi

# ─── Step 8: Run the manual setup Job ───────────────────────────────────────
info "Step 8: Running idp-manual-setup Job..."
oc delete job idp-manual-setup -n "${NAMESPACE}" --ignore-not-found=true
sleep 2
oc apply -f "${SCRIPT_DIR}/idp-manual-setup.yaml" -n "${NAMESPACE}"

info "  Waiting for Job completion..."
if oc wait --for=condition=complete job/idp-manual-setup -n "${NAMESPACE}" --timeout=120s; then
  success "idp-manual-setup Job completed successfully."
else
  warn "Job still running or timed out. Check logs: oc logs -l job-name=idp-manual-setup"
fi

# ─── Step 9: Final status ───────────────────────────────────────────────────
info "Step 9: Final status..."
oc get pods -n "${NAMESPACE}" -l "app.kubernetes.io/name=thunder"
success "=== Deployment process complete ==="
