#!/bin/bash
set -e

# Configuration: Default to 'dev' if not specified
ENV="${1:-dev}"
NAMESPACE="national-single-window-platform"
CLUSTER_DOMAIN="apps.sovecloud1.akaza.lk"

echo ">>> Deploying NSW Platform to environment: $ENV (Namespace: $NAMESPACE)"

# 0. Initialize Databases (Optional/First run)
echo ">>> 0. Initializing Logical Databases"
# Ignore errors if DB already exists
kubectl exec deployment/nsw-db -n "$NAMESPACE" -- psql -U postgres -c "CREATE DATABASE \"nsw_dev\";" || true
kubectl exec deployment/nsw-db -n "$NAMESPACE" -- psql -U postgres -c "CREATE DATABASE \"oga-backend-fcau\";" || true
kubectl exec deployment/nsw-db -n "$NAMESPACE" -- psql -U postgres -c "CREATE DATABASE \"oga-backend-ird\";" || true
kubectl exec deployment/nsw-db -n "$NAMESPACE" -- psql -U postgres -c "CREATE DATABASE \"oga-backend-npqs\";" || true
kubectl exec deployment/nsw-db -n "$NAMESPACE" -- psql -U postgres -c "CREATE DATABASE \"temporal_db_dev\";" || true

# 1. Build Dependencies
echo ">>> 1. Building Helm Dependencies"
helm dependency build ./deployments/helm/nsw-api
helm dependency build ./deployments/helm/idp
helm dependency build ./deployments/helm/oga-backend
helm dependency build ./deployments/helm/oga-app

# 2. Deploy IDP Thunder (Patched Deployment)
echo ">>> 2. Deploying Declarative IDP Thunder (with Patching)"
helm template idp-thunder ./deployments/helm/idp -n "$NAMESPACE" --values ./deployments/helm/idp/custom-values.yaml > /tmp/idp-raw.yaml
python3 /tmp/patch_idp.py
# Cleanup old seed job to trigger rerun
kubectl delete job idp-thunder-seed-job -n "$NAMESPACE" --ignore-not-found
kubectl apply -n "$NAMESPACE" -f /tmp/idp-patched.yaml

# 3. Deploy Core NSW API
echo ">>> 3. Deploying Core NSW API"
# Delete old migrations to prevent Helm timeout on pre-upgrade-hooks
kubectl delete job -l app.kubernetes.io/name=nsw-api -n "$NAMESPACE" --ignore-not-found
helm upgrade --install dev-nsw-api ./deployments/helm/nsw-api \
  --namespace "$NAMESPACE" \
  -f ./deployments/helm/nsw-api/values.yaml \
  -f ./deployments/helm/nsw-api/values-dev.yaml \
  --history-max 1 --wait

# 4. Deploy Temporal
echo ">>> 4. Deploying Temporal Workflow Engine"
helm upgrade --install dev-temporal ./deployments/helm/temporal \
  --namespace "$NAMESPACE" \
  -f ./deployments/helm/temporal/values-dev.yaml \
  --history-max 1

# 5. Deploy OGA Agency Backends
echo ">>> 5. Deploying OGA Agency Backends"
for agency in fcau ird npqs; do
  helm upgrade --install dev-oga-${agency}-backend ./deployments/helm/oga-backend \
    --namespace "$NAMESPACE" \
    -f ./deployments/helm/oga-backend/values-dev.yaml \
    -f ./deployments/helm/oga-backend/${agency}-backend-values.yaml \
    --set migrations.enabled=false \
    --history-max 1
done

# 6. Deploy OGA Portals
echo ">>> 6. Deploying OGA Portals"
for agency in fcau ird npqs; do
  helm upgrade --install dev-oga-${agency} ./deployments/helm/oga-app \
    --namespace "$NAMESPACE" \
    -f ./deployments/helm/oga-app/values-dev.yaml \
    -f ./deployments/helm/oga-app/values/${agency}-values.yaml \
    --set fullnameOverride=dev-oga-${agency} --history-max 1
done

echo ">>> 7. Final Verification"
curl -k -I "https://dev-nsw-api-$NAMESPACE.$CLUSTER_DOMAIN/health"
curl -k -I "https://idp-thunder-$NAMESPACE.$CLUSTER_DOMAIN/oauth2/jwks"

echo ""
echo ">>> Deployment Complete for environment: $ENV!"
