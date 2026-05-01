#!/bin/bash
set -euo pipefail

# Load environment variables from env file
# Default: backend/.env
ENV_FILE_PATH="${ENV_FILE:-../../../.env}"

if [ -f "$ENV_FILE_PATH" ]; then
    set -o allexport
    source "$ENV_FILE_PATH"
    set +o allexport
else
    echo "Error: env file not found: $ENV_FILE_PATH"
    exit 1
fi

CLEAN_RUN="${CLEAN_RUN:-false}"

# Validate required DB environment variables
for VAR in DB_HOST DB_PORT DB_USERNAME DB_PASSWORD DB_NAME; do
    if [ -z "${!VAR:-}" ]; then
        echo "Error: Required environment variable $VAR is not set."
        exit 1
    fi
done

MIGRATION_DB_HOST="${MIGRATION_DB_HOST:-$DB_HOST}"
MIGRATION_DB_HOST="${MIGRATION_DB_HOST//host.docker.internal/localhost}"

NPQS_OGA_SUBMISSION_URL="${NPQS_OGA_SUBMISSION_URL:-http://localhost:8081/api/oga/inject}"
FCAU_OGA_SUBMISSION_URL="${FCAU_OGA_SUBMISSION_URL:-http://localhost:8082/api/oga/inject}"
PRECONSIGNMENT_OGA_SUBMISSION_URL="${PRECONSIGNMENT_OGA_SUBMISSION_URL:-http://localhost:8083/api/oga/inject}"
CDA_OGA_SUBMISSION_URL="${CDA_OGA_SUBMISSION_URL:-http://localhost:8084/api/oga/inject}"

if [[ "$CLEAN_RUN" == "true" ]]; then
    echo "Dropping database $DB_NAME..."
    PGPASSWORD=$DB_PASSWORD psql -h "$MIGRATION_DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d postgres \
        -c "DROP DATABASE IF EXISTS $DB_NAME WITH (FORCE);"

    echo "Creating database $DB_NAME..."
    PGPASSWORD=$DB_PASSWORD psql -h "$MIGRATION_DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d postgres \
        -c "CREATE DATABASE $DB_NAME;"
else
    echo "Skipping database drop/recreate. Run with --clean-run to wipe."
    exit 0
fi

MIGRATIONS=(
    "001_initial_schema.up.sql"
    "002_insert_seed_hscodes.up.sql"
    "003_insert_seed_form_templates.up.sql"
    "004_insert_seed_workflow_node_templates.up.sql"
    "005_insert_seed_workflow_templates.up.sql"
    "006_insert_seed_workflow_hscode_map.up.sql"
    "007_insert_seed_pre_consignment_template.up.sql"
    "008_insert_payment_transactions.up.sql"
    "009_insert_cha_entity.up.sql"
    "010_workflow_table.up.sql"
    "011_workflow_tem_v2.up.sql"
)

echo "Starting database migrations..."

# Loop through and execute each file
for FILE in "${MIGRATIONS[@]}"; do
    if [ -f "$FILE" ]; then
        echo "Executing: $FILE"
        PGPASSWORD=$DB_PASSWORD psql \
            -v ON_ERROR_STOP=1 \
            -v NPQS_OGA_SUBMISSION_URL="$NPQS_OGA_SUBMISSION_URL" \
            -v FCAU_OGA_SUBMISSION_URL="$FCAU_OGA_SUBMISSION_URL" \
            -v PRECONSIGNMENT_OGA_SUBMISSION_URL="$PRECONSIGNMENT_OGA_SUBMISSION_URL" \
            -v CDA_OGA_SUBMISSION_URL="$CDA_OGA_SUBMISSION_URL" \
            -h "$MIGRATION_DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d "$DB_NAME" -f "$FILE"
    else
        echo "Warning: File $FILE not found, skipping."
    fi
done

echo "Migrations completed successfully."