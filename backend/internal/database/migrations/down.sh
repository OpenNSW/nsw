#!/bin/bash
set -euo pipefail

# Get the directory where the script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Default: path to backend/.env
ENV_FILE_PATH="${ENV_FILE:-$SCRIPT_DIR/../../../.env}"
source "$ENV_FILE_PATH"

MIGRATION_DB_HOST="${MIGRATION_DB_HOST:-$DB_HOST}"
MIGRATION_DB_HOST="${MIGRATION_DB_HOST//host.docker.internal/localhost}"

DOWNS=(
  "014_fcau_workflow_seed.down.sql"
  "013_fcau_workflow_nodes_seed.down.sql"
  "012_fcau_forms_seed.down.sql"
  "011_workflow_tem_v2.down.sql"
  "010_workflow_table.down.sql"
  "009_insert_cha_entity.down.sql"
  "008_insert_payment_transactions.down.sql"
  "007_insert_seed_pre_consignment_template.down.sql"
  "006_insert_seed_workflow_hscode_map.down.sql"
  "005_insert_seed_workflow_templates.down.sql"
  "004_insert_seed_workflow_node_templates.down.sql"
  "003_insert_seed_form_templates.down.sql"
  "002_insert_seed_hscodes.down.sql"
  "001_initial_schema.down.sql"
)

# Move to the script's directory so file references work
cd "$SCRIPT_DIR"

for FILE in "${DOWNS[@]}"; do
  PGPASSWORD=$DB_PASSWORD psql -v ON_ERROR_STOP=1 \
    -h "$MIGRATION_DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d "$DB_NAME" -f "$FILE"
done