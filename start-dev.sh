#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
RUN_IDP=true
RUN_TEMPORAL=true
RUN_MIGRATIONS=true

for arg in "$@"; do
  case "$arg" in
    --env-file=*)
      ENV_FILE="${arg#*=}"
      ;;
    --skip-idp)
      RUN_IDP=false
      ;;
    --skip-migrations)
      RUN_MIGRATIONS=false
      ;;
    --skip-temporal)
      RUN_TEMPORAL=false
      ;;
    *)
      echo "Unknown argument: $arg"
      echo "Usage: ./start-dev.sh [--env-file=/path/to/.env] [--skip-idp] [--skip-migrations] [--skip-temporal]"
      exit 1
      ;;
  esac
done

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Env file not found: $ENV_FILE"
  echo "Create one from: cp $ROOT_DIR/.env.example $ROOT_DIR/.env"
  exit 1
fi

set -a
source "$ENV_FILE"
set +a

for cmd in go pnpm docker; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "$cmd is required but was not found in PATH"
    exit 1
  fi
done

if [[ "$RUN_IDP" == "true" ]]; then
  echo "Starting IDP..."
  (
    cd "$ROOT_DIR/idp"
    docker compose up -d
  )
fi

IDP_PORT="${IDP_PORT:-8090}"
BACKEND_PORT="${BACKEND_PORT:-8080}"
TRADER_APP_PORT="${TRADER_APP_PORT:-5173}"
# Port Definitions
OGA_APP_NPQS_PORT="${OGA_APP_NPQS_PORT:-5174}"
OGA_APP_FCAU_PORT="${OGA_APP_FCAU_PORT:-5175}"
OGA_APP_IRD_PORT="${OGA_APP_IRD_PORT:-5176}"
OGA_APP_CDA_PORT="${OGA_APP_CDA_PORT:-5177}"
OGA_NPQS_PORT="${OGA_NPQS_PORT:-8081}"
OGA_FCAU_PORT="${OGA_FCAU_PORT:-8082}"
OGA_IRD_PORT="${OGA_IRD_PORT:-8083}"
OGA_CDA_PORT="${OGA_CDA_PORT:-8084}"


if [[ "$RUN_MIGRATIONS" == "true" ]]; then
  echo "Running backend migrations..."
  (
    cd "$ROOT_DIR/backend/internal/database/migrations"
    ENV_FILE="$ENV_FILE" \
      bash ./run.sh
  )
fi

if [[ "$RUN_TEMPORAL" == "true" ]]; then
  echo "Starting Temporal Workflow Manager..."
  (
    cd "$ROOT_DIR/temporal"
    docker compose up -d
  )
fi

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-nsw_db}"
DB_USERNAME="${DB_USERNAME:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-changeme}"
DB_SSLMODE="${DB_SSLMODE:-disable}"

TEMPORAL_HOST="${TEMPORAL_HOST:-localhost}"
TEMPORAL_PORT="${TEMPORAL_PORT:-7233}"
TEMPORAL_NAMESPACE="${TEMPORAL_NAMESPACE:-default}"

SERVER_DEBUG="${SERVER_DEBUG:-true}"
SERVER_LOG_LEVEL="${SERVER_LOG_LEVEL:-info}"
CORS_ALLOWED_ORIGINS="${CORS_ALLOWED_ORIGINS:-http://localhost:3000,http://localhost:5173,http://localhost:5174,http://localhost:5175,http://localhost:5176,http://localhost:5177}"

AUTH_ISSUER="${AUTH_ISSUER:-https://localhost:${IDP_PORT}}"
AUTH_JWKS_URL="${AUTH_JWKS_URL:-https://localhost:${IDP_PORT}/oauth2/jwks}"
AUTH_CLIENT_IDS="${AUTH_CLIENT_IDS:-TRADER_PORTAL_APP,FCAU_TO_NSW,NPQS_TO_NSW,IRD_TO_NSW,CDA_TO_NSW}"
AUTH_AUDIENCE="${AUTH_AUDIENCE:-NSW_API}"
AUTH_JWKS_INSECURE_SKIP_VERIFY="${AUTH_JWKS_INSECURE_SKIP_VERIFY:-true}"

IDP_PUBLIC_URL="${IDP_PUBLIC_URL:-https://localhost:${IDP_PORT}}"
TRADER_IDP_CLIENT_ID="${TRADER_IDP_CLIENT_ID:-TRADER_PORTAL_APP}"
NPQS_IDP_CLIENT_ID="${NPQS_IDP_CLIENT_ID:-OGA_PORTAL_APP_NPQS}"
FCAU_IDP_CLIENT_ID="${FCAU_IDP_CLIENT_ID:-OGA_PORTAL_APP_FCAU}"
IRD_IDP_CLIENT_ID="${IRD_IDP_CLIENT_ID:-OGA_PORTAL_APP_IRD}"
CDA_IDP_CLIENT_ID="${CDA_IDP_CLIENT_ID:-OGA_PORTAL_APP_CDA}"
IDP_SCOPES="${IDP_SCOPES:-openid,profile,email,group,role}"
IDP_PLATFORM="${IDP_PLATFORM:-AsgardeoV2}"
SHOW_AUTOFILL_BUTTON="${SHOW_AUTOFILL_BUTTON:-true}"
TRADER_IDP_TRADER_GROUP_NAME="${TRADER_IDP_TRADER_GROUP_NAME:-Traders}"
TRADER_IDP_CHA_GROUP_NAME="${TRADER_IDP_CHA_GROUP_NAME:-CHA}"

OGA_FORMS_PATH="${OGA_FORMS_PATH:-./data/forms}"
OGA_DEFAULT_FORM_ID="${OGA_DEFAULT_FORM_ID:-default}"
OGA_ALLOWED_ORIGINS="${OGA_ALLOWED_ORIGINS:-*}"

OGA_DB_DRIVER="${OGA_DB_DRIVER:-sqlite}"
OGA_DB_HOST="${OGA_DB_HOST:-localhost}"
OGA_DB_PORT="${OGA_DB_PORT:-5432}"
OGA_DB_USER="${OGA_DB_USER:-postgres}"
OGA_DB_PASSWORD="${OGA_DB_PASSWORD:-changeme}"
OGA_DB_NAME="${OGA_DB_NAME:-oga_db}"
OGA_DB_SSLMODE="${OGA_DB_SSLMODE:-disable}"

OGA_NPQS_DB_PATH="${OGA_NPQS_DB_PATH:-./npqs.db}"
OGA_FCAU_DB_PATH="${OGA_FCAU_DB_PATH:-./fcau.db}"
OGA_IRD_DB_PATH="${OGA_IRD_DB_PATH:-./ird.db}"
OGA_CDA_DB_PATH="${OGA_CDA_DB_PATH:-./cda.db}"
OGA_APP_NPQS_INSTANCE_CONFIG="${OGA_APP_NPQS_INSTANCE_CONFIG:-npqs}"
OGA_APP_FCAU_INSTANCE_CONFIG="${OGA_APP_FCAU_INSTANCE_CONFIG:-fcau}"
OGA_APP_IRD_INSTANCE_CONFIG="${OGA_APP_IRD_INSTANCE_CONFIG:-ird}"
OGA_APP_CDA_INSTANCE_CONFIG="${OGA_APP_CDA_INSTANCE_CONFIG:-cda}"

OGA_NSW_NPQS_CLIENT_ID="${OGA_NSW_NPQS_CLIENT_ID:-NPQS_TO_NSW}"
OGA_NSW_FCAU_CLIENT_ID="${OGA_NSW_FCAU_CLIENT_ID:-FCAU_TO_NSW}"
OGA_NSW_IRD_CLIENT_ID="${OGA_NSW_IRD_CLIENT_ID:-IRD_TO_NSW}"
OGA_NSW_CDA_CLIENT_ID="${OGA_NSW_CDA_CLIENT_ID:-CDA_TO_NSW}"
OGA_NSW_NPQS_CLIENT_SECRET="${OGA_NSW_NPQS_CLIENT_SECRET:-1234}"
OGA_NSW_FCAU_CLIENT_SECRET="${OGA_NSW_FCAU_CLIENT_SECRET:-1234}"
OGA_NSW_IRD_CLIENT_SECRET="${OGA_NSW_IRD_CLIENT_SECRET:-1234}"
OGA_NSW_CDA_CLIENT_SECRET="${OGA_NSW_CDA_CLIENT_SECRET:-1234}"
OGA_NSW_TOKEN_URL="${OGA_NSW_TOKEN_URL:-https://localhost:${IDP_PORT}/oauth2/token}"
OGA_NSW_SCOPES="${OGA_NSW_SCOPES:-}"
OGA_NSW_TOKEN_INSECURE_SKIP_VERIFY="${OGA_NSW_TOKEN_INSECURE_SKIP_VERIFY:-true}"

pids=()
names=()

cleanup() {
  local code=$?
  if [[ ${#pids[@]} -gt 0 ]]; then
    echo
    echo "Stopping services..."
    for pid in "${pids[@]}"; do
      kill "$pid" >/dev/null 2>&1 || true
    done
    wait >/dev/null 2>&1 || true
  fi
  exit "$code"
}

trap cleanup INT TERM

start_service() {
  local name="$1"
  local dir="$2"
  shift 2

  (
    cd "$dir"
    "$@" 2>&1 | sed -u "s/^/[${name}] /"
  ) &

  pids+=("$!")
  names+=("$name")
}

echo "Starting local development services (non-Docker)..."

start_service "backend" "$ROOT_DIR/backend" env \
  DB_HOST="$DB_HOST" \
  DB_PORT="$DB_PORT" \
  DB_NAME="$DB_NAME" \
  DB_USERNAME="$DB_USERNAME" \
  DB_PASSWORD="$DB_PASSWORD" \
  DB_SSLMODE="$DB_SSLMODE" \
  TEMPORAL_HOST="$TEMPORAL_HOST" \
  TEMPORAL_PORT="$TEMPORAL_PORT" \
  TEMPORAL_NAMESPACE="$TEMPORAL_NAMESPACE" \
  SERVER_PORT="$BACKEND_PORT" \
  SERVER_DEBUG="$SERVER_DEBUG" \
  SERVER_LOG_LEVEL="$SERVER_LOG_LEVEL" \
  CORS_ALLOWED_ORIGINS="$CORS_ALLOWED_ORIGINS" \
  AUTH_JWKS_URL="$AUTH_JWKS_URL" \
  AUTH_ISSUER="$AUTH_ISSUER" \
  AUTH_CLIENT_IDS="$AUTH_CLIENT_IDS" \
  AUTH_AUDIENCE="$AUTH_AUDIENCE" \
  AUTH_JWKS_INSECURE_SKIP_VERIFY="$AUTH_JWKS_INSECURE_SKIP_VERIFY" \
  go run ./cmd/server/main.go

# OGA instance registry
# Each row: name | backend_port | db_path | nsw_client_id | nsw_client_secret | app_port | instance_config | idp_client_id
OGA_INSTANCES=(
  "npqs|$OGA_NPQS_PORT|$OGA_NPQS_DB_PATH|$OGA_NSW_NPQS_CLIENT_ID|$OGA_NSW_NPQS_CLIENT_SECRET|$OGA_APP_NPQS_PORT|$OGA_APP_NPQS_INSTANCE_CONFIG|$NPQS_IDP_CLIENT_ID"
  "fcau|$OGA_FCAU_PORT|$OGA_FCAU_DB_PATH|$OGA_NSW_FCAU_CLIENT_ID|$OGA_NSW_FCAU_CLIENT_SECRET|$OGA_APP_FCAU_PORT|$OGA_APP_FCAU_INSTANCE_CONFIG|$FCAU_IDP_CLIENT_ID"
  "ird|$OGA_IRD_PORT|$OGA_IRD_DB_PATH|$OGA_NSW_IRD_CLIENT_ID|$OGA_NSW_IRD_CLIENT_SECRET|$OGA_APP_IRD_PORT|$OGA_APP_IRD_INSTANCE_CONFIG|$IRD_IDP_CLIENT_ID"
  "cda|$OGA_CDA_PORT|$OGA_CDA_DB_PATH|$OGA_NSW_CDA_CLIENT_ID|$OGA_NSW_CDA_CLIENT_SECRET|$OGA_APP_CDA_PORT|$OGA_APP_CDA_INSTANCE_CONFIG|$CDA_IDP_CLIENT_ID"
)

# Launch OGA backends
for entry in "${OGA_INSTANCES[@]}"; do
  IFS='|' read -r name port db_path nsw_client_id nsw_client_secret app_port instance_config idp_client_id <<< "$entry"

  start_service "oga-${name}" "$ROOT_DIR/oga" env \
    OGA_PORT="$port" \
    OGA_DB_DRIVER="$OGA_DB_DRIVER" \
    OGA_DB_PATH="$db_path" \
    OGA_DB_HOST="$OGA_DB_HOST" \
    OGA_DB_PORT="$OGA_DB_PORT" \
    OGA_DB_USER="$OGA_DB_USER" \
    OGA_DB_PASSWORD="$OGA_DB_PASSWORD" \
    OGA_DB_NAME="$OGA_DB_NAME" \
    OGA_DB_SSLMODE="$OGA_DB_SSLMODE" \
    OGA_FORMS_PATH="$OGA_FORMS_PATH" \
    OGA_DEFAULT_FORM_ID="$OGA_DEFAULT_FORM_ID" \
    OGA_ALLOWED_ORIGINS="$OGA_ALLOWED_ORIGINS" \
    OGA_NSW_API_BASE_URL="http://localhost:${BACKEND_PORT}/api/v1" \
    OGA_NSW_CLIENT_ID="$nsw_client_id" \
    OGA_NSW_CLIENT_SECRET="$nsw_client_secret" \
    OGA_NSW_TOKEN_URL="$OGA_NSW_TOKEN_URL" \
    OGA_NSW_SCOPES="$OGA_NSW_SCOPES" \
    OGA_NSW_TOKEN_INSECURE_SKIP_VERIFY="$OGA_NSW_TOKEN_INSECURE_SKIP_VERIFY" \
    go run ./cmd/server
done

# Trader portal
start_service "trader-app" "$ROOT_DIR/portals/apps/trader-app" env \
  VITE_API_BASE_URL="http://localhost:${BACKEND_PORT}/api/v1" \
  VITE_IDP_BASE_URL="$IDP_PUBLIC_URL" \
  VITE_IDP_CLIENT_ID="$TRADER_IDP_CLIENT_ID" \
  VITE_APP_URL="http://localhost:${TRADER_APP_PORT}" \
  VITE_IDP_SCOPES="$IDP_SCOPES" \
  VITE_IDP_PLATFORM="$IDP_PLATFORM" \
  VITE_IDP_TRADER_GROUP_NAME="$TRADER_IDP_TRADER_GROUP_NAME" \
  VITE_IDP_CHA_GROUP_NAME="$TRADER_IDP_CHA_GROUP_NAME" \
  VITE_SHOW_AUTOFILL_BUTTON="$SHOW_AUTOFILL_BUTTON" \
  pnpm run dev -- --port "$TRADER_APP_PORT"

# Launch OGA portals
for entry in "${OGA_INSTANCES[@]}"; do
  IFS='|' read -r name port db_path nsw_client_id nsw_client_secret app_port instance_config idp_client_id <<< "$entry"

  start_service "oga-app-${name}" "$ROOT_DIR/portals/apps/oga-app" env \
    VITE_PORT="$app_port" \
    VITE_INSTANCE_CONFIG="$instance_config" \
    VITE_API_BASE_URL="http://localhost:${port}" \
    VITE_IDP_BASE_URL="$IDP_PUBLIC_URL" \
    VITE_IDP_CLIENT_ID="$idp_client_id" \
    VITE_APP_URL="http://localhost:${app_port}" \
    VITE_IDP_SCOPES="$IDP_SCOPES" \
    VITE_IDP_PLATFORM="$IDP_PLATFORM" \
    pnpm run dev
done

# Status banner (generated from registry)
{
  echo
  echo "Started local services:"
  echo "  - backend       -> http://localhost:${BACKEND_PORT}"
  echo "  - trader-app    -> http://localhost:${TRADER_APP_PORT}"
  for entry in "${OGA_INSTANCES[@]}"; do
    IFS='|' read -r name port _ _ _ app_port _ _ <<< "$entry"
    printf "  - oga-%-9s -> http://localhost:%s\n" "$name" "$port"
    printf "  - oga-app-%-5s -> http://localhost:%s\n" "$name" "$app_port"
  done
  echo
  echo "IDP start: $RUN_IDP"
  echo "Migrations run: $RUN_MIGRATIONS"
  echo
  echo "Press Ctrl+C to stop all services started by this script."
}

wait
