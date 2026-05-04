#!/bin/sh
set -eu

: "${POSTGRES_SEEDS:?POSTGRES_SEEDS is required}"
: "${DB_PORT:?DB_PORT is required}"
: "${POSTGRES_USER:?POSTGRES_USER is required}"
: "${SQL_PASSWORD:?SQL_PASSWORD is required}"

DB_NAME="${TEMPORAL_DB_NAME:-temporal}"
VIS_DB_NAME="${TEMPORAL_VISIBILITY_DB_NAME:-temporal_visibility}"

# Global flags must come BEFORE the subcommand
sql_tool() {
  temporal-sql-tool \
    --plugin postgres12 \
    --endpoint "${POSTGRES_SEEDS}" \
    --port "${DB_PORT}" \
    --user "${POSTGRES_USER}" \
    --password "${SQL_PASSWORD}" \
    "$@"
}

echo "[setup-postgres] Initializing Temporal databases and schema..."

create_db_if_needed() {
  _db="$1"
  echo "[setup-postgres] Ensuring database '${_db}' exists..."
  # --database here sets the target DB to create; connect via default 'postgres' db
  if ! _out=$(sql_tool create-database --database "${_db}" 2>&1); then
    echo "${_out}" >&2
    echo "${_out}" | grep -qi "already exists" && return 0
    echo "[setup-postgres] Failed to create database '${_db}'." >&2
    exit 1
  fi
}

create_db_if_needed "${DB_NAME}"
create_db_if_needed "${VIS_DB_NAME}"

echo "[setup-postgres] Setting up main schema..."
sql_tool --database "${DB_NAME}" setup-schema -v 0.0 || true
sql_tool --database "${DB_NAME}" update-schema \
  -d /etc/temporal/schema/postgresql/v12/temporal/versioned

echo "[setup-postgres] Setting up visibility schema..."
sql_tool --database "${VIS_DB_NAME}" setup-schema -v 0.0 || true
sql_tool --database "${VIS_DB_NAME}" update-schema \
  -d /etc/temporal/schema/postgresql/v12/visibility/versioned

echo "[setup-postgres] Done."