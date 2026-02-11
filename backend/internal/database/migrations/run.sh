#!/bin/bash

# Load environment variables from backend/.env file
# This filters out comments and exports each line as a variable
if [ -f ../../../.env ]; then
    export $(echo $(grep -v '^#' ../../../.env | xargs))
else
    echo "Error: .env file not found."
    exit 1
fi

# Force disconnect other users and drop the database
# Using the 'postgres' database as a maintenance DB to execute the drop
echo "Dropping database $DB_NAME..."
PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME WITH (FORCE);"

# Recreate the database
echo "Creating database $DB_NAME..."
PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d postgres -c "CREATE DATABASE $DB_NAME;"

# Ensure environment variables are set
if [ -z "$DB_PASSWORD" ]; then
    echo "Error: DB_PASSWORD is not set."
    exit 1
fi

# Define the file paths
MIGRATIONS=(
    "002_initial_schema.sql"
    "002_insert_seed_data.sql"
    "003_initial_schema.sql"
    "003_insert_seed_data.sql"
    "004_unify_task_parent_id.sql"
)

echo "Starting database migrations..."

# Loop through and execute each file
for FILE in "${MIGRATIONS[@]}"; do
    if [ -f "$FILE" ]; then
        echo "Executing: $FILE"
        PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d "$DB_NAME" -f "$FILE"
        
        if [ $? -ne 0 ]; then
            echo "Error executing $FILE. Aborting."
            exit 1
        fi
    else
        echo "Warning: File $FILE not found, skipping."
    fi
done

echo "Migrations completed successfully."