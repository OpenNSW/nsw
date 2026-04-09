#!/bin/bash
set -e

NAMESPACE="national-single-window-platform"
PULL_SECRET="ghcr-io-registry"

echo "=== 1. Starting Core API Rollout ==="
helm upgrade --install staging-api ./deployments/helm/nsw-api \
  -f ./deployments/helm/nsw-api/values.yaml \
  -f ./deployments/helm/nsw-api/values-staging.yaml \
  -n $NAMESPACE --history-max 1

helm upgrade --install dev-nsw-api ./deployments/helm/nsw-api \
  -f ./deployments/helm/nsw-api/values.yaml \
  -f ./deployments/helm/nsw-api/values-dev.yaml \
  -n $NAMESPACE --history-max 1

echo "=== 2. Starting Temporal Rollout ==="
helm upgrade --install temporal ./deployments/helm/temporal \
  -f ./deployments/helm/temporal/values-dev.yaml \
  -n $NAMESPACE --history-max 1

echo "=== 3. Starting Trader Portal Rollout ==="
helm upgrade --install dev-trader-app ./deployments/helm/trader-app \
  -f ./deployments/helm/trader-app/values.yaml \
  -f ./deployments/helm/trader-app/values-dev.yaml \
  -n $NAMESPACE --set "image.pullSecrets[0].name=$PULL_SECRET" --history-max 1

echo "=== 4. Starting IDP Stabilization Rollout ==="
# We use the imperative pipeline for stability as requested, but backed by definitive manifests
helm template idp-thunder ./deployments/helm/idp/charts/idp-umbrella \
  --namespace $NAMESPACE \
  -f ./deployments/helm/idp/values-dev.yaml > /tmp/idp-raw.yaml
python3 /tmp/patch_idp.py
kubectl apply -f /tmp/idp-patched.yaml
kubectl rollout status deployment idp-thunder-deployment -n $NAMESPACE --timeout=60s

echo "=== 5. Starting OGA Stack Rollout ==="
# FCAU
helm upgrade --install dev-oga-fcau-backend ./deployments/helm/oga-backend \
  -f ./deployments/helm/oga-backend/values.yaml \
  -f ./deployments/helm/oga-backend/fcau-backend-values.yaml \
  -n $NAMESPACE --set "image.pullSecrets[0].name=$PULL_SECRET" --history-max 1
helm upgrade --install dev-oga-fcau ./deployments/helm/oga-app \
  -f ./deployments/helm/oga-app/values.yaml \
  -f ./deployments/helm/oga-app/values-dev.yaml \
  -f ./deployments/helm/oga-app/values-dev-fcau.yaml \
  -n $NAMESPACE --set "image.pullSecrets[0].name=$PULL_SECRET" --history-max 1

# IRD
helm upgrade --install dev-oga-ird-backend ./deployments/helm/oga-backend \
  -f ./deployments/helm/oga-backend/values.yaml \
  -f ./deployments/helm/oga-backend/ird-backend-values.yaml \
  -n $NAMESPACE --set "image.pullSecrets[0].name=$PULL_SECRET" --history-max 1
helm upgrade --install dev-oga-ird ./deployments/helm/oga-app \
  -f ./deployments/helm/oga-app/values.yaml \
  -f ./deployments/helm/oga-app/values-dev.yaml \
  -f ./deployments/helm/oga-app/values-dev-ird.yaml \
  -n $NAMESPACE --set "image.pullSecrets[0].name=$PULL_SECRET" --history-max 1

# NPQS
helm upgrade --install dev-oga-npqs-backend ./deployments/helm/oga-backend \
  -f ./deployments/helm/oga-backend/values.yaml \
  -f ./deployments/helm/oga-backend/npqs-backend-values.yaml \
  -n $NAMESPACE --set "image.pullSecrets[0].name=$PULL_SECRET" --history-max 1
helm upgrade --install dev-oga-npqs ./deployments/helm/oga-app \
  -f ./deployments/helm/oga-app/values.yaml \
  -f ./deployments/helm/oga-app/values-dev.yaml \
  -f ./deployments/helm/oga-app/values-dev-npqs.yaml \
  -n $NAMESPACE --set "image.pullSecrets[0].name=$PULL_SECRET" --history-max 1

echo "=== 6. Platform-Wide Health Check ==="
curl -I https://idp-thunder-national-single-window-platform.apps.sovecloud1.akaza.lk/gate/signin
curl -I https://staging-trader-app-national-single-window-platform.apps.sovecloud1.akaza.lk
curl -I https://staging-oga-fcau-national-single-window-platform.apps.sovecloud1.akaza.lk
