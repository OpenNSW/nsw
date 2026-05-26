#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
RUN_IDP=true
RUN_TEMPORAL=true
CLEAN_RUN=false

for arg in "$@"; do
  case "$arg" in
    --env-file=*)
      ENV_FILE="${arg#*=}"
      ;;
    --skip-idp)
      RUN_IDP=false
      ;;
    --skip-temporal)
      RUN_TEMPORAL=false
      ;;
    --clean-run)
      CLEAN_RUN=true
      ;;
    *)
      echo "Unknown argument: $arg"
      echo "Usage: ./start-dev.sh [--env-file=/path/to/.env] [--skip-idp] [--skip-temporal] [--clean-run]"
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

for cmd in go pnpm docker temporal; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "$cmd is required but was not found in PATH"
    exit 1
  fi
done

# Port defaults and env var fallbacks
IDP_PORT="${IDP_PORT:-8090}"
BACKEND_PORT="${BACKEND_PORT:-8080}"
TRADER_APP_PORT="${TRADER_APP_PORT:-5173}"

# Temporal settings with env var fallbacks
TEMPORAL_HOST="${TEMPORAL_HOST:-localhost}"
TEMPORAL_PORT="${TEMPORAL_PORT:-7233}"
TEMPORAL_NAMESPACE="${TEMPORAL_NAMESPACE:-default}"

# NSW Backednd env vars with defaults and fallbacks
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-nsw_db}"
DB_USERNAME="${DB_USERNAME:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-changeme}"
DB_SSLMODE="${DB_SSLMODE:-disable}"

SERVER_DEBUG="${SERVER_DEBUG:-true}"
SERVER_LOG_LEVEL="${SERVER_LOG_LEVEL:-info}"
CORS_ALLOWED_ORIGINS="${CORS_ALLOWED_ORIGINS:-http://localhost:3000,http://localhost:5173,http://localhost:5174,http://localhost:5175,http://localhost:5176,http://localhost:5177}"

AUTH_ISSUER="${AUTH_ISSUER:-https://localhost:${IDP_PORT}}"
AUTH_JWKS_URL="${AUTH_JWKS_URL:-https://localhost:${IDP_PORT}/oauth2/jwks}"
AUTH_CLIENT_IDS="${AUTH_CLIENT_IDS:-TRADER_PORTAL_APP,FCAU_TO_NSW,NPQS_TO_NSW,IRD_TO_NSW,CDA_TO_NSW}"
AUTH_AUDIENCE="${AUTH_AUDIENCE:-NSW_API}"
AUTH_JWKS_INSECURE_SKIP_VERIFY="${AUTH_JWKS_INSECURE_SKIP_VERIFY:-true}"

# Trader App settings with defaults and fallbacks
IDP_PUBLIC_URL="${IDP_PUBLIC_URL:-https://localhost:${IDP_PORT}}"
TRADER_IDP_CLIENT_ID="${TRADER_IDP_CLIENT_ID:-TRADER_PORTAL_APP}"
IDP_SCOPES="${IDP_SCOPES:-openid,profile,email,group,role}"
IDP_PLATFORM="${IDP_PLATFORM:-AsgardeoV2}"
SHOW_AUTOFILL_BUTTON="${SHOW_AUTOFILL_BUTTON:-true}"
TRADER_IDP_TRADER_GROUP_NAME="${TRADER_IDP_TRADER_GROUP_NAME:-Traders}"
TRADER_IDP_CHA_GROUP_NAME="${TRADER_IDP_CHA_GROUP_NAME:-CHA}"

# ---------------------------------------------------------------------------
# wait_for_temporal: wait until Temporal's gRPC endpoint is fully ready
# ---------------------------------------------------------------------------
wait_for_temporal() {
  local host="$TEMPORAL_HOST"
  local port="$TEMPORAL_PORT"
  local retries=30
  local wait=2

  echo "Waiting for Temporal at $host:$port..."

  for ((i=1; i<=retries; i++)); do
    if temporal operator cluster health \
        --address "$host:$port" \
        --namespace "$TEMPORAL_NAMESPACE" \
        >/dev/null 2>&1; then
      echo "Temporal is ready."
      return 0
    fi
    echo "  Temporal not ready yet (attempt $i/$retries), retrying in ${wait}s..."
    sleep "$wait"
  done

  echo "Temporal did not become ready in time. Aborting."
  exit 1
}


# ---------------------------------------------------------------------------
# MAIN SCRIPT
# ---------------------------------------------------------------------------

pids=()

cleanup() {
  # Ctrl+C: stop processes and containers, but destroy nothing
  echo
  echo "Stopping services..."

  if [[ ${#pids[@]} -gt 0 ]]; then
    for pid in "${pids[@]}"; do
      kill "$pid" >/dev/null 2>&1 || true
    done
    wait >/dev/null 2>&1 || true
  fi

  if [[ "$RUN_IDP" == "true" ]]; then
    echo "Stopping IDP containers..."
    docker compose -f "$ROOT_DIR/idp/docker-compose.yml" stop
  fi

  if [[ "$RUN_TEMPORAL" == "true" ]]; then
    echo "Stopping Temporal containers..."
    docker compose -f "$ROOT_DIR/temporal/docker-compose.yml" stop
  fi

  exit 0
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
}

# ---------------------------------------------------------------------------
# CLEAN RUN: wipe everything, recreate, migrate — then start services
# ---------------------------------------------------------------------------
if [[ "$CLEAN_RUN" == "true" ]]; then
  echo "Clean run: wiping Docker volumes and databases..."

  if [[ "$RUN_IDP" == "true" ]]; then
    echo "Removing IDP containers and volumes..."
    docker compose -f "$ROOT_DIR/idp/docker-compose.yml" down --volumes
  fi

  if [[ "$RUN_TEMPORAL" == "true" ]]; then
    echo "Removing Temporal containers and volumes..."
    docker compose -f "$ROOT_DIR/temporal/docker-compose.yml" down --volumes
  fi

  echo "Running backend migrations..."
  (
    cd "$ROOT_DIR/backend/internal/database/migrations"
    ENV_FILE="$ENV_FILE" \
    CLEAN_RUN="$CLEAN_RUN" \
      bash ./run.sh
  )
fi

# ---------------------------------------------------------------------------
# START DOCKER SERVICES
# ---------------------------------------------------------------------------
if [[ "$RUN_IDP" == "true" ]]; then
  echo "Starting IDP..."
  docker compose -f "$ROOT_DIR/idp/docker-compose.yml" up -d
fi

if [[ "$RUN_TEMPORAL" == "true" ]]; then
  echo "Starting Temporal..."
  docker compose -f "$ROOT_DIR/temporal/docker-compose.yml" up -d
fi

# ---------------------------------------------------------------------------
# START NON-DOCKER SERVICES
# ---------------------------------------------------------------------------
echo "Starting local development services..."

start_service "trader-app" "$ROOT_DIR/portals/apps/trader-app" env \
  VITE_API_BASE_URL="http://localhost:${BACKEND_PORT}" \
  VITE_IDP_BASE_URL="$IDP_PUBLIC_URL" \
  VITE_IDP_CLIENT_ID="$TRADER_IDP_CLIENT_ID" \
  VITE_APP_URL="http://localhost:${TRADER_APP_PORT}" \
  VITE_IDP_SCOPES="$IDP_SCOPES" \
  VITE_IDP_PLATFORM="$IDP_PLATFORM" \
  VITE_IDP_TRADER_GROUP_NAME="$TRADER_IDP_TRADER_GROUP_NAME" \
  VITE_IDP_CHA_GROUP_NAME="$TRADER_IDP_CHA_GROUP_NAME" \
  VITE_SHOW_AUTOFILL_BUTTON="$SHOW_AUTOFILL_BUTTON" \
  pnpm run dev -- --port "$TRADER_APP_PORT"


# Backend must wait for Temporal before starting
if [[ "$RUN_TEMPORAL" == "true" ]]; then
  wait_for_temporal
fi

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

# Status banner
{
  echo
  echo "Started local services:"
  echo "  - backend       -> http://localhost:${BACKEND_PORT}"
  echo "  - trader-app    -> http://localhost:${TRADER_APP_PORT}"
  echo
  echo "IDP running:      $RUN_IDP"
  echo "Temporal running: $RUN_TEMPORAL"
  echo "Clean run:        $CLEAN_RUN"
  echo
  echo "Press Ctrl+C to stop all services."
}

wait
