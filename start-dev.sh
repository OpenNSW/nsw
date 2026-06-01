#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"

for arg in "$@"; do
  case "$arg" in
    --env-file=*)
      ENV_FILE="${arg#*=}"
      ;;
    *)
      echo "Unknown argument: $arg"
      echo "Usage: ./start-dev.sh [--env-file=/path/to/.env]"
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

if ! command -v pnpm >/dev/null 2>&1; then
  echo "pnpm is required but was not found in PATH"
  exit 1
fi

# Port defaults and env var fallbacks
BACKEND_PORT="${BACKEND_PORT:-8080}"
TRADER_APP_PORT="${TRADER_APP_PORT:-5173}"
IDP_PORT="${IDP_PORT:-8090}"

# Trader App settings with defaults and fallbacks
IDP_PUBLIC_URL="${IDP_PUBLIC_URL:-https://localhost:${IDP_PORT}}"
TRADER_IDP_CLIENT_ID="${TRADER_IDP_CLIENT_ID:-TRADER_PORTAL_APP}"
IDP_SCOPES="${IDP_SCOPES:-openid,profile,email,group,role}"
IDP_PLATFORM="${IDP_PLATFORM:-AsgardeoV2}"
SHOW_AUTOFILL_BUTTON="${SHOW_AUTOFILL_BUTTON:-true}"
TRADER_IDP_TRADER_GROUP_NAME="${TRADER_IDP_TRADER_GROUP_NAME:-Traders}"
TRADER_IDP_CHA_GROUP_NAME="${TRADER_IDP_CHA_GROUP_NAME:-CHA}"

echo "Starting Trader Portal..."
echo "  - URL: http://localhost:${TRADER_APP_PORT}"
echo "  - Backend API: http://localhost:${BACKEND_PORT}"
echo "  - IDP Public URL: $IDP_PUBLIC_URL"
echo

cd "$ROOT_DIR/portals/apps/trader-app"
exec env \
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
