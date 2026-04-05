#!/bin/bash
set -e

echo ">>> 1. Synchronizing Database Migrations"
mkdir -p ./deployments/helm/nsw-api/migrations
cp ./backend/internal/database/migrations/*.sql ./deployments/helm/nsw-api/migrations/
echo "Successfully copied SQL migrations to Helm chart."

echo ""
echo ">>> 2. Deploying Core NSW Services"
helm upgrade --install nsw-api ./deployments/helm/nsw-api -f ./deployments/helm/nsw-api/values.yaml --history-max 1
helm upgrade --install trader-app ./deployments/helm/trader-app -f ./deployments/helm/trader-app/values.yaml --history-max 1

echo ""
echo ">>> 3. Deploying Legacy Temporal Server"
helm upgrade --install temporal ./deployments/helm/temporal -f ./deployments/helm/temporal/values.yaml --history-max 1

echo ""
echo ">>> 4. Deploying IDP Thunder (Direct OCI)"
# Apply administrative secrets and bootstrap ConfigMap first
oc apply -f deployments/helm/idp/idp-admin-secret.yaml
oc apply -f deployments/helm/idp/idp-manual-bootstrap-cm.yaml

# Pull and install directly from GHCR OCI Registry
helm upgrade --install idp-thunder oci://ghcr.io/asgardeo/helm-charts/thunder \
  --version 0.29.0 -f ./deployments/helm/idp/custom-values.yaml --history-max 1 -n national-single-window-platform

echo ""
echo ">>> 5. Initializing IDP Resources (Post-Install)"
# We apply the setup job manually as a post-install step
oc delete job idp-manual-setup 2>/dev/null || true
oc apply -f deployments/helm/idp/idp-manual-setup.yaml

echo ""
echo ">>> 6. Deploying OGA Agency Backends"
helm upgrade --install oga-fcau-backend ./deployments/helm/oga-backend -f ./deployments/helm/oga-backend/fcau-backend-values.yaml --history-max 1
helm upgrade --install oga-ird-backend ./deployments/helm/oga-backend -f ./deployments/helm/oga-backend/ird-backend-values.yaml --history-max 1
helm upgrade --install oga-npqs-backend ./deployments/helm/oga-backend -f ./deployments/helm/oga-backend/npqs-backend-values.yaml --history-max 1

echo ""
echo ">>> 7. Deploying OGA Frontends (Isolated)"
helm upgrade --install oga-fcau-app ./deployments/helm/oga-app -f ./deployments/helm/oga-app/values/fcau-values.yaml --history-max 1
helm upgrade --install oga-ird-app  ./deployments/helm/oga-app -f ./deployments/helm/oga-app/values/ird-values.yaml --history-max 1
helm upgrade --install oga-npqs-app ./deployments/helm/oga-app -f ./deployments/helm/oga-app/values/npqs-values.yaml --history-max 1

echo ""
echo ">>> Deployment Complete!"
