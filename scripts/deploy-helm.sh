#!/bin/bash
set -e

# Configuration: Default to 'dev' if not specified
ENV="${1:-dev}"
NAMESPACE="national-single-window-platform"
CLUSTER_DOMAIN="apps.sovecloud1.akaza.lk"

echo ">>> Deploying NSW Platform to environment: $ENV (Namespace: $NAMESPACE)"

# 0. Deploy NSW Database (StatefulSet)
echo ">>> 0. Deploying NSW Persistent Database"
helm upgrade --install nsw-db ./deployments/helm/nsw-db \
  --namespace "$NAMESPACE" \
  -f ./deployments/helm/nsw-db/values.yaml \
  --history-max 1 --wait

# 1. Build Dependencies
echo ">>> 1. Building Helm Dependencies"
helm dependency build ./deployments/helm/nsw-api
helm dependency build ./deployments/helm/oga-backend
helm dependency build ./deployments/helm/oga-app

# 2. Deploy IDP Thunder (EXCLUDED - Stable)
# echo ">>> 2. Deploying Declarative IDP Thunder"
# kustomize build --enable-helm ./deployments/helm/idp | kubectl apply -f -

# 3. Deploy Core NSW API
echo ">>> 3. Deploying Core NSW API"
# Clear old migration jobs to ensure the pre-upgrade hook can execute
kubectl delete job -l app.kubernetes.io/name=nsw-api -n "$NAMESPACE" --ignore-not-found
helm upgrade --install dev-nsw-api ./deployments/helm/nsw-api \
  --namespace "$NAMESPACE" \
  -f ./deployments/helm/nsw-api/values.yaml \
  -f ./deployments/helm/nsw-api/values-dev.yaml \
  --history-max 1 --wait

# 4. Deploy Temporal (EXCLUDED - Stable)
# echo ">>> 4. Deploying Temporal Workflow Engine"

# 5. Deploy OGA Agency Backends
echo ">>> 5. Deploying OGA Agency Backends"
for agency in fcau ird npqs; do
  VALUES_FILE="./deployments/helm/oga-backend/${agency}-backend-values.yaml"
  if [ ! -f "$VALUES_FILE" ]; then
    VALUES_FILE="./deployments/helm/oga-backend/values-dev.yaml"
  fi
  
  helm upgrade --install dev-oga-${agency}-backend ./deployments/helm/oga-backend \
    --namespace "$NAMESPACE" \
    -f ./deployments/helm/oga-backend/values.yaml \
    -f "$VALUES_FILE" \
    --history-max 1 --wait
done

# 6. Deploy Portal Applications
echo ">>> 6. Deploying OGA Portals & Trader App"
# Trader App
helm upgrade --install dev-trader-app ./deployments/helm/trader-app \
  --namespace "$NAMESPACE" \
  -f ./deployments/helm/trader-app/values.yaml \
  -f ./deployments/helm/trader-app/values-dev.yaml \
  --history-max 1 --wait

# OGA Portals
for agency in fcau ird npqs; do
  SPECIFIC_VALUES="./deployments/helm/oga-app/values/values-dev-${agency}.yaml"
  if [ ! -f "$SPECIFIC_VALUES" ]; then
    SPECIFIC_VALUES="./deployments/helm/oga-app/values/${agency}-values.yaml"
  fi

  helm upgrade --install dev-oga-${agency} ./deployments/helm/oga-app \
    --namespace "$NAMESPACE" \
    -f ./deployments/helm/oga-app/values.yaml \
    -f "$SPECIFIC_VALUES" \
    --set fullnameOverride=dev-oga-${agency} \
    --history-max 1 --wait
done

echo ">>> 7. Final Verification"
curl -k -I "https://dev-nsw-api.$NAMESPACE.$CLUSTER_DOMAIN/api/v1/health" || true

echo ""
echo ">>> Deployment Complete for environment: $ENV!"
