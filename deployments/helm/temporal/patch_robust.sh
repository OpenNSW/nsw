#!/bin/bash
set -e
CHART_DIR="/Users/tmp/nsw-health-cert/deployments/helm/temporal"
PATCH_DIR="${CHART_DIR}/patch_temporal"

cd "${CHART_DIR}"
rm -rf "${PATCH_DIR}"
mkdir -p "${PATCH_DIR}"

tar -xzf charts/temporal-1.0.0.tgz -C "${PATCH_DIR}"
find "${PATCH_DIR}" -name "._*" -delete

# Using python for safe string replacement in YAML
python3 -c "
import sys
path = '${PATCH_DIR}/temporal/templates/server-deployment.yaml'
with open(path, 'r') as f:
    lines = f.readlines()
with open(path, 'w') as f:
    for line in lines:
        if 'temporal-server' in line and 'args:' in line:
            indent = line[:line.find('args:')]
            f.write(f'{indent}args: [\"mkdir -p /tmp/config \\&\\& export DB=postgresql \\&\\& export POSTGRES_SEEDS=nsw-db \\&\\& export POSTGRES_USER=postgres \\&\\& export POSTGRES_PWD=\\${TEMPORAL_DEFAULT_STORE_PASSWORD} \\&\\& export DBPORT=5432 \\&\\& dockerize -template /etc/temporal/config/config_template.yaml:/tmp/config/docker.yaml \\&\\& /usr/local/bin/temporal-server --root / --config /tmp/config --env docker start --service \\${SERVICES}\"]\\n')
        else:
            f.write(line)
"

# Package it back
rm charts/temporal-1.0.0.tgz
cd "${PATCH_DIR}"
COPYFILE_DISABLE=1 tar -czf ../charts/temporal-1.0.0.tgz temporal
cd ..
rm -rf "${PATCH_DIR}"

echo "Robust python patch applied successfully"
