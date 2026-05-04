#!/bin/bash
set -e
cd /Users/tmp/nsw-health-cert/deployments/helm/temporal
rm -rf patch_temporal
mkdir patch_temporal
tar -xzf charts/temporal-1.0.0.tgz -C patch_temporal
find patch_temporal -name "._*" -delete

sed -i '' 's|args: \["dockerize.*|args: \["mkdir -p /tmp/config \&\& dockerize -template /etc/temporal/config/config_template.yaml:/tmp/config/docker.yaml \&\& TEMPORAL_CONFIG_DIR=/tmp/config TEMPORAL_ENVIRONMENT=docker /usr/local/bin/temporal-server start"\]|' patch_temporal/temporal/templates/server-deployment.yaml

rm charts/temporal-1.0.0.tgz
cd patch_temporal
COPYFILE_DISABLE=1 tar -czf ../charts/temporal-1.0.0.tgz temporal
cd ..
rm -rf patch_temporal

echo "Patch applied successfully"
