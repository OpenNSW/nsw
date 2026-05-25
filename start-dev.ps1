param(
    [string]${env-file} = "",
    [switch]${skip-idp},
    [switch]${skip-temporal},
    [switch]${clean-run},
    [switch]${migrations}
)

# Mutual exclusivity check
if (${clean-run} -and ${migrations}) {
    Write-Error "Error: You cannot use -clean-run and -migrations at the same time."
    exit 1
}

$ROOT_DIR = $PSScriptRoot
if (-not ${env-file}) {
    ${env-file} = Join-Path $ROOT_DIR ".env"
}

$RUN_IDP = -not ${skip-idp}
$RUN_TEMPORAL = -not ${skip-temporal}
$CLEAN_RUN = if (${clean-run}) { "true" } else { "false" }

if (-not (Test-Path ${env-file})) {
    Write-Host "Env file not found: ${env-file}"
    Write-Host "Create one from: cp $ROOT_DIR\.env.example $ROOT_DIR\.env"
    exit 1
}

# Load environment variables
Write-Host "Loading environment variables from ${env-file}..."
Get-Content ${env-file} | ForEach-Object {
    if ($_ -match '^([^#\s][^=]+)=(.*)$') {
        $name = $matches[1].Trim()
        $value = $matches[2].Trim().Trim('"').Trim("'")
        Set-Item "env:$name" $value
    }
}

# Check dependencies
$Dependencies = @("go", "pnpm", "docker", "temporal")
foreach ($cmd in $Dependencies) {
    if (-not (Get-Command $cmd -ErrorAction SilentlyContinue)) {
        Write-Error "$cmd is required but was not found in PATH"
        exit 1
    }
}

# Port Definitions (with defaults)
$IDP_PORT = if ($env:IDP_PORT) { $env:IDP_PORT } else { "8090" }
$BACKEND_PORT = if ($env:BACKEND_PORT) { $env:BACKEND_PORT } else { "8080" }
$TRADER_APP_PORT = if ($env:TRADER_APP_PORT) { $env:TRADER_APP_PORT } else { "5173" }


# Temporal settings
$TEMPORAL_HOST = if ($env:TEMPORAL_HOST) { $env:TEMPORAL_HOST } else { "localhost" }
$TEMPORAL_PORT = if ($env:TEMPORAL_PORT) { $env:TEMPORAL_PORT } else { "7233" }
$TEMPORAL_NAMESPACE = if ($env:TEMPORAL_NAMESPACE) { $env:TEMPORAL_NAMESPACE } else { "default" }

# Service Environment Variables
$DB_HOST = if ($env:DB_HOST) { $env:DB_HOST } else { "localhost" }
$DB_PORT = if ($env:DB_PORT) { $env:DB_PORT } else { "5432" }
$DB_NAME = if ($env:DB_NAME) { $env:DB_NAME } else { "nsw_db" }
$DB_USERNAME = if ($env:DB_USERNAME) { $env:DB_USERNAME } else { "postgres" }
$DB_PASSWORD = if ($env:DB_PASSWORD) { $env:DB_PASSWORD } else { "changeme" }
$DB_SSLMODE = if ($env:DB_SSLMODE) { $env:DB_SSLMODE } else { "disable" }
$SERVER_DEBUG = if ($env:SERVER_DEBUG) { $env:SERVER_DEBUG } else { "true" }
$SERVER_LOG_LEVEL = if ($env:SERVER_LOG_LEVEL) { $env:SERVER_LOG_LEVEL } else { "info" }
$CORS_ALLOWED_ORIGINS = if ($env:CORS_ALLOWED_ORIGINS) { $env:CORS_ALLOWED_ORIGINS } else { "http://localhost:3000,http://localhost:5173" }

$AUTH_ISSUER = if ($env:AUTH_ISSUER) { $env:AUTH_ISSUER } else { "https://localhost:$IDP_PORT" }
$AUTH_JWKS_URL = if ($env:AUTH_JWKS_URL) { $env:AUTH_JWKS_URL } else { "https://localhost:$IDP_PORT/oauth2/jwks" }
$AUTH_CLIENT_IDS = if ($env:AUTH_CLIENT_IDS) { $env:AUTH_CLIENT_IDS } else { "TRADER_PORTAL_APP" }
$AUTH_AUDIENCE = if ($env:AUTH_AUDIENCE) { $env:AUTH_AUDIENCE } else { "NSW_API" }
$AUTH_JWKS_INSECURE_SKIP_VERIFY = if ($env:AUTH_JWKS_INSECURE_SKIP_VERIFY) { $env:AUTH_JWKS_INSECURE_SKIP_VERIFY } else { "true" }

$IDP_PUBLIC_URL = if ($env:IDP_PUBLIC_URL) { $env:IDP_PUBLIC_URL } else { "https://localhost:$IDP_PORT" }
$TRADER_IDP_CLIENT_ID = if ($env:TRADER_IDP_CLIENT_ID) { $env:TRADER_IDP_CLIENT_ID } else { "TRADER_PORTAL_APP" }

$IDP_SCOPES = if ($env:IDP_SCOPES) { $env:IDP_SCOPES } else { "openid,profile,email,group,role" }
$IDP_PLATFORM = if ($env:IDP_PLATFORM) { $env:IDP_PLATFORM } else { "AsgardeoV2" }
$SHOW_AUTOFILL_BUTTON = if ($env:SHOW_AUTOFILL_BUTTON) { $env:SHOW_AUTOFILL_BUTTON } else { "true" }
$TRADER_IDP_TRADER_GROUP_NAME = if ($env:TRADER_IDP_TRADER_GROUP_NAME) { $env:TRADER_IDP_TRADER_GROUP_NAME } else { "Traders" }
$TRADER_IDP_CHA_GROUP_NAME = if ($env:TRADER_IDP_CHA_GROUP_NAME) { $env:TRADER_IDP_CHA_GROUP_NAME } else { "CHA" }






function ensure_node_modules() {
    $PortalsDir = Join-Path $ROOT_DIR "portals"
    $TraderAppModules = Join-Path $PortalsDir "apps/trader-app/node_modules"

    if (-not (Test-Path $TraderAppModules)) {
        Write-Host "Missing node_modules in frontend apps. Running pnpm install in $PortalsDir..."
        Push-Location $PortalsDir
        pnpm install
        Pop-Location
    }
}

function wait_for_temporal() {
    $Retries = 30
    $WaitSeconds = 2

    Write-Host "Waiting for Temporal at ${TEMPORAL_HOST}:${TEMPORAL_PORT}..."
    for ($i = 1; $i -le $Retries; $i++) {
        temporal operator cluster health --address "${TEMPORAL_HOST}:${TEMPORAL_PORT}" --namespace $TEMPORAL_NAMESPACE 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "Temporal is ready."
            return
        }
        Write-Host "  Temporal not ready yet (attempt $i/$Retries), retrying in ${WaitSeconds}s..."
        Start-Sleep -Seconds $WaitSeconds
    }
    Write-Error "Temporal did not become ready in time. Aborting."
    exit 1
}



$script:Jobs = @()

function Start-ServiceJob {
    param($Name, $Dir, $EnvVars, $ScriptBlock)
    $Job = Start-Job -Name $Name -ScriptBlock $ScriptBlock -ArgumentList $Name, $Dir, $EnvVars
    $script:Jobs += $Job
    return $Job
}

# Clean Run Logic
    if ($CLEAN_RUN -eq "true") {
        Write-Host "Clean run: wiping Docker volumes and databases..."
        if ($RUN_IDP) {
            Write-Host "Removing IDP containers and volumes..."
            docker compose -f (Join-Path $ROOT_DIR "idp/docker-compose.yml") down --volumes
        }
        if ($RUN_TEMPORAL) {
            Write-Host "Removing Temporal containers and volumes..."
            docker compose -f (Join-Path $ROOT_DIR "temporal/docker-compose.yml") down --volumes
        }
    }

if (${clean-run} -or ${migrations}) {
    Write-Host "Running backend migrations..."
    Push-Location (Join-Path $ROOT_DIR "backend/internal/database/migrations")
    $env:ENV_FILE = ${env-file}
    $MigrationParams = @()
    if (${clean-run}) { $MigrationParams += "-clean-run" }
    if (${migrations}) { $MigrationParams += "-migrations" }
    powershell.exe -File ./run.ps1 @MigrationParams
    Pop-Location
}

# Start Docker Services
if ($RUN_IDP) {
    Write-Host "Starting IDP..."
    docker compose -f (Join-Path $ROOT_DIR "idp/docker-compose.yml") up -d
}
if ($RUN_TEMPORAL) {
    Write-Host "Starting Temporal..."
    docker compose -f (Join-Path $ROOT_DIR "temporal/docker-compose.yml") up -d
}

Write-Host "Starting local development services..."

ensure_node_modules

# Trader App
$TraderEnv = @{
    VITE_API_BASE_URL = "http://localhost:${BACKEND_PORT}/api/v1"
    VITE_IDP_BASE_URL = $IDP_PUBLIC_URL
    VITE_IDP_CLIENT_ID = $TRADER_IDP_CLIENT_ID
    VITE_APP_URL = "http://localhost:${TRADER_APP_PORT}"
    VITE_IDP_SCOPES = $IDP_SCOPES
    VITE_IDP_PLATFORM = $IDP_PLATFORM
    VITE_IDP_TRADER_GROUP_NAME = $TRADER_IDP_TRADER_GROUP_NAME
    VITE_IDP_CHA_GROUP_NAME = $TRADER_IDP_CHA_GROUP_NAME
    VITE_SHOW_AUTOFILL_BUTTON = $SHOW_AUTOFILL_BUTTON
    TRADER_APP_PORT = $TRADER_APP_PORT
}
Start-ServiceJob -Name "trader-app" -Dir (Join-Path $ROOT_DIR "portals/apps/trader-app") -EnvVars $TraderEnv -ScriptBlock {
    param($Name, $Dir, $EnvVars)
    foreach ($Key in $EnvVars.Keys) { Set-Item "env:$Key" $EnvVars[$Key] }
    Set-Location $Dir
    pnpm run dev -- --port $env:TRADER_APP_PORT 2>&1 | ForEach-Object { Write-Host "[$Name] $_" }
}

# Backend (wait for Temporal)
if ($RUN_TEMPORAL) {
    wait_for_temporal
}

$BackendEnv = @{
    DB_HOST = $DB_HOST
    DB_PORT = $DB_PORT
    DB_NAME = $DB_NAME
    DB_USERNAME = $DB_USERNAME
    DB_PASSWORD = $DB_PASSWORD
    DB_SSLMODE = $DB_SSLMODE
    TEMPORAL_HOST = $TEMPORAL_HOST
    TEMPORAL_PORT = $TEMPORAL_PORT
    TEMPORAL_NAMESPACE = $TEMPORAL_NAMESPACE
    SERVER_PORT = $BACKEND_PORT
    SERVER_DEBUG = $SERVER_DEBUG
    SERVER_LOG_LEVEL = $SERVER_LOG_LEVEL
    CORS_ALLOWED_ORIGINS = $CORS_ALLOWED_ORIGINS
    AUTH_JWKS_URL = $AUTH_JWKS_URL
    AUTH_ISSUER = $AUTH_ISSUER
    AUTH_CLIENT_IDS = $AUTH_CLIENT_IDS
    AUTH_AUDIENCE = $AUTH_AUDIENCE
    AUTH_JWKS_INSECURE_SKIP_VERIFY = $AUTH_JWKS_INSECURE_SKIP_VERIFY
}
Start-ServiceJob -Name "backend" -Dir (Join-Path $ROOT_DIR "backend") -EnvVars $BackendEnv -ScriptBlock {
    param($Name, $Dir, $EnvVars)
    foreach ($Key in $EnvVars.Keys) { Set-Item "env:$Key" $EnvVars[$Key] }
    Set-Location $Dir
    go run ./cmd/server/main.go 2>&1 | ForEach-Object { Write-Host "[$Name] $_" }
}

# Status Banner
Write-Host ""
Write-Host "Started local services:"
Write-Host "  - backend       -> http://localhost:$BACKEND_PORT"
Write-Host "  - trader-app    -> http://localhost:$TRADER_APP_PORT"

Write-Host ""
Write-Host "IDP running:      $RUN_IDP"
Write-Host "Temporal running: $RUN_TEMPORAL"
Write-Host "Clean run:        $CLEAN_RUN"
Write-Host ""
Write-Host "Press Ctrl+C to stop all services."

# Main loop
try {
    while ($true) {
        foreach ($Job in $script:Jobs) {
            if ($Job.State -eq "Running") {
                $Data = Receive-Job -Job $Job
                if ($Data) { $Data | ForEach-Object { Write-Host $_ } }
            }
        }
        Start-Sleep -Milliseconds 500
    }
} finally {
    Write-Host ""
    Write-Host "Stopping services..."
    foreach ($Job in $script:Jobs) {
        Stop-Job -Job $Job
        Remove-Job -Job $Job
    }
    if ($RUN_IDP) {
        Write-Host "Stopping IDP containers..."
        docker compose -f (Join-Path $ROOT_DIR "idp/docker-compose.yml") stop
    }
    if ($RUN_TEMPORAL) {
        Write-Host "Stopping Temporal containers..."
        docker compose -f (Join-Path $ROOT_DIR "temporal/docker-compose.yml") stop
    }
}
