#!/bin/bash
set -euo pipefail

# Load environment variables from backend/.env file when not running in Docker (env already set)
if [ -z "${DB_HOST:-}" ] && [ -f ../../../.env ]; then
    set -o allexport
    source ../../../.env
    set +o allexport
elif [ -z "${DB_HOST:-}" ]; then
    echo "Error: .env file not found and DB_HOST not set."
    exit 1
fi

# Ensure environment variables are set
for VAR in DB_HOST DB_PORT DB_USERNAME DB_PASSWORD DB_NAME; do
    if [ -z "${!VAR:-}" ]; then
        echo "Error: Required environment variable $VAR is not set."
        exit 1
    fi
done

# Force disconnect other users and drop the database
# Using the 'postgres' database as a maintenance DB to execute the drop
echo "Dropping database $DB_NAME..."
PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME WITH (FORCE);"

# Recreate the database
echo "Creating database $DB_NAME..."
PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d postgres -c "CREATE DATABASE $DB_NAME;"

# Define the file paths
MIGRATIONS=(
    "001_initial_schema.sql"
    "001_insert_seed_hscodes.sql"
    "001_insert_seed_form_templates.sql"
    "001_insert_seed_workflow_node_templates.sql"
    "001_insert_seed_workflow_templates.sql"
    "001_insert_seed_workflow_hscode_map.sql"
    "001_insert_seed_pre_consignment_template.sql"
)

echo "Starting database migrations..."

# Loop through and execute each file
for FILE in "${MIGRATIONS[@]}"; do
    if [ -f "$FILE" ]; then
        echo "Executing: $FILE"
        PGPASSWORD=$DB_PASSWORD psql -v ON_ERROR_STOP=1 -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d "$DB_NAME" -f "$FILE"
        
        if [ $? -ne 0 ]; then
            echo "Error executing $FILE. Aborting."
            exit 1
        fi
    else
        echo "Warning: File $FILE not found, skipping."
    fi
done

echo "Migrations completed successfully."