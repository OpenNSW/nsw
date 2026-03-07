#!/bin/bash
set -euo pipefail

# Initialize all four PostgreSQL databases required by NSW
DATABASES=(nsw_db ird_db npqs_db fcau_db)

for DB_NAME in "${DATABASES[@]}"; do
    echo "=============================================="
    echo "Initializing database: $DB_NAME"
    echo "=============================================="
    export DB_NAME
    ./run.sh
done

echo "All databases initialized successfully."
