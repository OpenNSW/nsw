#!/usr/bin/env bash

set -e

# ==============================================================================
# deploy-helm.sh
# Purpose: Automatically copies backend SQL migrations to the Helm chart before 
# deploying or upgrading the National Single Window services.
# 
# Requirement: Must be run from the repository root.
# ==============================================================================

# Ensure we are in the repository root
if [ ! -d "deployments/helm" ]; then
  echo "Error: Must be run from the repository root."
  exit 1
fi

echo ">>> 1. Synchronizing Database Migrations"
mkdir -p ./deployments/helm/nsw-api/sql
cp ./backend/internal/database/migrations/*.sql ./deployments/helm/nsw-api/sql/
echo "Successfully copied SQL migrations to Helm chart."

echo ""
echo ">>> 2. Deploying Core NSW Services"
helm upgrade --install nsw-api    ./deployments/helm/nsw-api    --history-max 1
helm upgrade --install trader-app ./deployments/helm/trader-app --history-max 1

echo ""
echo ">>> 3. Deploying Legacy Temporal Server"
helm upgrade --install temporal ./deployments/helm/temporal --history-max 1

echo ""
echo ">>> 4. Deploying IDP Thunder"
helm upgrade --install idp-thunder oci://ghcr.io/asgardeo/helm-charts/thunder --version 0.29.0 -f ./idp/custom-values.yaml --history-max 1

echo ""
echo ">>> 5. Deploying OGA Frontends"
helm upgrade --install oga-multi-frontend ./deployments/helm/oga-multi-frontend --history-max 1

echo ""
echo ">>> 6. Deploying OGA Backends"
helm upgrade --install oga-fcau-backend ./deployments/helm/oga-backend -f ./deployments/helm/oga-backend/fcau-backend-values.yaml --history-max 1
helm upgrade --install oga-ird-backend  ./deployments/helm/oga-backend -f ./deployments/helm/oga-backend/ird-backend-values.yaml  --history-max 1
helm upgrade --install oga-npqs-backend ./deployments/helm/oga-backend -f ./deployments/helm/oga-backend/npqs-backend-values.yaml --history-max 1

echo ""
echo "========================================================="
echo "✅ Deployment completed successfully!"
echo "========================================================="
