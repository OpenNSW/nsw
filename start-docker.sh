#!/usr/bin/env bash
# ==============================================================================
# NSW Docker Development — thin wrapper around docker compose
# ==============================================================================
# All service definitions live in docker-compose.yml.
# This script loads the env file, builds profiles, and delegates to
# `docker compose`.
# ==============================================================================

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env.docker}"
COMPOSE_FILE="$ROOT_DIR/docker-compose.yml"

RUN_IDP=true
RUN_POSTGRES=true
RUN_MIGRATIONS=true
RUN_BUILD=true
STOP_ONLY=false
REMOVE_VOLUMES=false

usage() {
  cat <<EOF
Usage: ./start-docker.sh [options]

Options:
  --env-file=/path/to/.env   Use a custom environment file (default: ./.env.docker)
  --skip-idp                 Do not start IDP services
  --skip-postgres            Do not start PostgreSQL container
  --skip-migrations          Do not apply backend SQL migrations
  --skip-build               Do not build images before running containers
  --stop                     Stop and remove NSW docker containers, then exit
  --remove-volumes           When used with --stop, also remove named volumes

Examples:
  ./start-docker.sh
  ./start-docker.sh --env-file=.env.docker
  ./start-docker.sh --skip-build
  ./start-docker.sh --stop
  ./start-docker.sh --stop --remove-volumes
EOF
}

for arg in "$@"; do
  case "$arg" in
    --env-file=*)
      ENV_FILE="${arg#*=}"
      ;;
    --skip-idp)
      RUN_IDP=false
      ;;
    --skip-postgres)
      RUN_POSTGRES=false
      ;;
    --skip-migrations)
      RUN_MIGRATIONS=false
      ;;
    --skip-build)
      RUN_BUILD=false
      ;;
    --stop)
      STOP_ONLY=true
      ;;
    --remove-volumes)
      REMOVE_VOLUMES=true
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $arg"
      usage
      exit 1
      ;;
  esac
done

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required but was not found in PATH"
  exit 1
fi

# Build the base compose command
compose_cmd=(docker compose -f "$COMPOSE_FILE")

# --- Resolve env file path ---------------------------------------------------
if [[ "$ENV_FILE" != /* ]]; then
  ENV_FILE="$ROOT_DIR/$ENV_FILE"
fi

if [[ -f "$ENV_FILE" ]]; then
  compose_cmd+=(--env-file "$ENV_FILE")
fi

# --- Build profile list -------------------------------------------------------
PROFILES=()
[[ "$RUN_IDP" == "true" ]]      && PROFILES+=(idp)
[[ "$RUN_POSTGRES" == "true" ]] && PROFILES+=(db)

for p in "${PROFILES[@]}"; do
  compose_cmd+=(--profile "$p")
done

# When --skip-migrations is used but postgres is still enabled,
# scale the migration init container to 0
SCALE_ARGS=()
if [[ "$RUN_POSTGRES" == "true" && "$RUN_MIGRATIONS" == "false" ]]; then
  SCALE_ARGS+=(--scale db-migration=0)
fi

# --- Stop mode ----------------------------------------------------------------
if [[ "$STOP_ONLY" == "true" ]]; then
  echo "Stopping NSW docker stack..."
  DOWN_ARGS=(down)
  if [[ "$REMOVE_VOLUMES" == "true" ]]; then
    DOWN_ARGS+=(-v)
  fi
  # Ensure all profiles are included when stopping so every service is torn down
  "${compose_cmd[@]}" --profile idp --profile db "${DOWN_ARGS[@]}"
  echo "Done."
  exit 0
fi

# --- Validate env file --------------------------------------------------------
if [[ ! -f "$ENV_FILE" ]]; then
  echo "Env file not found: $ENV_FILE"
  echo "Create one from: cp $ROOT_DIR/.env.docker.example $ROOT_DIR/.env.docker"
  exit 1
fi

# --- Load env for the summary output ------------------------------------------
set -a
source "$ENV_FILE"
set +a

IDP_PORT="${IDP_PORT:-8090}"
BACKEND_PORT="${BACKEND_PORT:-8080}"
TRADER_APP_PORT="${TRADER_APP_PORT:-5173}"
OGA_NPQS_PORT="${OGA_NPQS_PORT:-8081}"
OGA_FCAU_PORT="${OGA_FCAU_PORT:-8082}"
OGA_IRD_PORT="${OGA_IRD_PORT:-8083}"
OGA_APP_NPQS_PORT="${OGA_APP_NPQS_PORT:-5174}"
OGA_APP_FCAU_PORT="${OGA_APP_FCAU_PORT:-5175}"
OGA_APP_IRD_PORT="${OGA_APP_IRD_PORT:-5176}"
DB_PORT="${DB_PORT:-55432}"

# --- Build --------------------------------------------------------------------
if [[ "$RUN_BUILD" == "true" ]]; then
  echo "Building images..."
  "${compose_cmd[@]}" build
fi

# --- Start --------------------------------------------------------------------
echo "Starting NSW docker stack..."
if [[ ${#SCALE_ARGS[@]} -gt 0 ]]; then
  "${compose_cmd[@]}" up -d "${SCALE_ARGS[@]}"
else
  "${compose_cmd[@]}" up -d
fi

cat <<EOF

NSW Docker services:
  - backend        -> http://localhost:${BACKEND_PORT}
  - trader-app     -> http://localhost:${TRADER_APP_PORT}
  - oga-npqs       -> http://localhost:${OGA_NPQS_PORT}
  - oga-fcau       -> http://localhost:${OGA_FCAU_PORT}
  - oga-ird        -> http://localhost:${OGA_IRD_PORT}
  - oga-app-npqs   -> http://localhost:${OGA_APP_NPQS_PORT}
  - oga-app-fcau   -> http://localhost:${OGA_APP_FCAU_PORT}
  - oga-app-ird    -> http://localhost:${OGA_APP_IRD_PORT}

Supporting services:
  - postgres       -> localhost:${DB_PORT}   $(if [[ "$RUN_POSTGRES" == "true" ]]; then echo "(enabled)"; else echo "(skipped)"; fi)
  - idp (thunder)  -> https://localhost:${IDP_PORT}  $(if [[ "$RUN_IDP" == "true" ]]; then echo "(enabled)"; else echo "(skipped)"; fi)

Useful commands:
  - View logs:       docker compose -f $COMPOSE_FILE logs -f backend
  - Status:          docker compose -f $COMPOSE_FILE ps
  - Stop stack:      ./start-docker.sh --stop
  - Stop + volumes:  ./start-docker.sh --stop --remove-volumes
EOF
