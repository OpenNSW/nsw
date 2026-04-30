#!/bin/sh
set -eu

# One-shot schema initialization for Temporal using PostgreSQL.
#
# Required env:
#   POSTGRES_SEEDS, DB_PORT, POSTGRES_USER, SQL_PASSWORD
# Optional:
#   TEMPORAL_DB_NAME (default: temporal)
#   TEMPORAL_VISIBILITY_DB_NAME (default: temporal_visibility)
#
# temporal-sql-tool is provided by the temporalio/admin-tools image.

: "${POSTGRES_SEEDS:?POSTGRES_SEEDS is required}"
: "${DB_PORT:?DB_PORT is required}"
: "${POSTGRES_USER:?POSTGRES_USER is required}"
: "${SQL_PASSWORD:?SQL_PASSWORD is required}"

DB_NAME="${TEMPORAL_DB_NAME:-temporal}"
VIS_DB_NAME="${TEMPORAL_VISIBILITY_DB_NAME:-temporal_visibility}"

echo "[setup-postgres] Initializing Temporal databases and schema..."

temporal-sql-tool \
  --plugin postgres12 \
  --endpoint "${POSTGRES_SEEDS}" \
  --port "${DB_PORT}" \
  --user "${POSTGRES_USER}" \
  --password "${SQL_PASSWORD}" \
  create-database \
  --database "${DB_NAME}" || true

temporal-sql-tool \
  --plugin postgres12 \
  --endpoint "${POSTGRES_SEEDS}" \
  --port "${DB_PORT}" \
  --user "${POSTGRES_USER}" \
  --password "${SQL_PASSWORD}" \
  create-database \
  --database "${VIS_DB_NAME}" || true

# Main Temporal schema

temporal-sql-tool \
  --plugin postgres12 \
  --endpoint "${POSTGRES_SEEDS}" \
  --port "${DB_PORT}" \
  --user "${POSTGRES_USER}" \
  --password "${SQL_PASSWORD}" \
  --database "${DB_NAME}" \
  setup-schema -v 0.0 || true

temporal-sql-tool \
  --plugin postgres12 \
  --endpoint "${POSTGRES_SEEDS}" \
  --port "${DB_PORT}" \
  --user "${POSTGRES_USER}" \
  --password "${SQL_PASSWORD}" \
  --database "${DB_NAME}" \
  update-schema -d /etc/temporal/schema/postgresql/v12/temporal/versioned

# Visibility schema (PostgreSQL-based visibility)

temporal-sql-tool \
  --plugin postgres12 \
  --endpoint "${POSTGRES_SEEDS}" \
  --port "${DB_PORT}" \
  --user "${POSTGRES_USER}" \
  --password "${SQL_PASSWORD}" \
  --database "${VIS_DB_NAME}" \
  setup-schema -v 0.0 || true

temporal-sql-tool \
  --plugin postgres12 \
  --endpoint "${POSTGRES_SEEDS}" \
  --port "${DB_PORT}" \
  --user "${POSTGRES_USER}" \
  --password "${SQL_PASSWORD}" \
  --database "${VIS_DB_NAME}" \
  update-schema -d /etc/temporal/schema/postgresql/v12/visibility/versioned

echo "[setup-postgres] Done."

