param(
    [string]${env-file} = "",
    [switch]${skip-idp},
    [switch]${skip-temporal},
    [switch]${clean-run}
)

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
$OGA_APP_NPQS_PORT = if ($env:OGA_APP_NPQS_PORT) { $env:OGA_APP_NPQS_PORT } else { "5174" }
$OGA_APP_FCAU_PORT = if ($env:OGA_APP_FCAU_PORT) { $env:OGA_APP_FCAU_PORT } else { "5175" }
$OGA_APP_IRD_PORT = if ($env:OGA_APP_IRD_PORT) { $env:OGA_APP_IRD_PORT } else { "5176" }
$OGA_APP_CDA_PORT = if ($env:OGA_APP_CDA_PORT) { $env:OGA_APP_CDA_PORT } else { "5177" }
$OGA_NPQS_PORT = if ($env:OGA_NPQS_PORT) { $env:OGA_NPQS_PORT } else { "8081" }
$OGA_FCAU_PORT = if ($env:OGA_FCAU_PORT) { $env:OGA_FCAU_PORT } else { "8082" }
$OGA_IRD_PORT = if ($env:OGA_IRD_PORT) { $env:OGA_IRD_PORT } else { "8083" }
$OGA_CDA_PORT = if ($env:OGA_CDA_PORT) { $env:OGA_CDA_PORT } else { "8084" }

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
$CORS_ALLOWED_ORIGINS = if ($env:CORS_ALLOWED_ORIGINS) { $env:CORS_ALLOWED_ORIGINS } else { "http://localhost:3000,http://localhost:5173,http://localhost:5174,http://localhost:5175,http://localhost:5176,http://localhost:5177" }

$AUTH_ISSUER = if ($env:AUTH_ISSUER) { $env:AUTH_ISSUER } else { "https://localhost:$IDP_PORT" }
$AUTH_JWKS_URL = if ($env:AUTH_JWKS_URL) { $env:AUTH_JWKS_URL } else { "https://localhost:$IDP_PORT/oauth2/jwks" }
$AUTH_CLIENT_IDS = if ($env:AUTH_CLIENT_IDS) { $env:AUTH_CLIENT_IDS } else { "TRADER_PORTAL_APP,FCAU_TO_NSW,NPQS_TO_NSW,IRD_TO_NSW,CDA_TO_NSW" }
$AUTH_AUDIENCE = if ($env:AUTH_AUDIENCE) { $env:AUTH_AUDIENCE } else { "NSW_API" }
$AUTH_JWKS_INSECURE_SKIP_VERIFY = if ($env:AUTH_JWKS_INSECURE_SKIP_VERIFY) { $env:AUTH_JWKS_INSECURE_SKIP_VERIFY } else { "true" }

$IDP_PUBLIC_URL = if ($env:IDP_PUBLIC_URL) { $env:IDP_PUBLIC_URL } else { "https://localhost:$IDP_PORT" }
$TRADER_IDP_CLIENT_ID = if ($env:TRADER_IDP_CLIENT_ID) { $env:TRADER_IDP_CLIENT_ID } else { "TRADER_PORTAL_APP" }
$NPQS_IDP_CLIENT_ID = if ($env:NPQS_IDP_CLIENT_ID) { $env:NPQS_IDP_CLIENT_ID } else { "OGA_PORTAL_APP_NPQS" }
$FCAU_IDP_CLIENT_ID = if ($env:FCAU_IDP_CLIENT_ID) { $env:FCAU_IDP_CLIENT_ID } else { "OGA_PORTAL_APP_FCAU" }
$IRD_IDP_CLIENT_ID = if ($env:IRD_IDP_CLIENT_ID) { $env:IRD_IDP_CLIENT_ID } else { "OGA_PORTAL_APP_IRD" }
$CDA_IDP_CLIENT_ID = if ($env:CDA_IDP_CLIENT_ID) { $env:CDA_IDP_CLIENT_ID } else { "OGA_PORTAL_APP_CDA" }
$IDP_SCOPES = if ($env:IDP_SCOPES) { $env:IDP_SCOPES } else { "openid,profile,email,group,role" }
$IDP_PLATFORM = if ($env:IDP_PLATFORM) { $env:IDP_PLATFORM } else { "AsgardeoV2" }
$SHOW_AUTOFILL_BUTTON = if ($env:SHOW_AUTOFILL_BUTTON) { $env:SHOW_AUTOFILL_BUTTON } else { "true" }
$TRADER_IDP_TRADER_GROUP_NAME = if ($env:TRADER_IDP_TRADER_GROUP_NAME) { $env:TRADER_IDP_TRADER_GROUP_NAME } else { "Traders" }
$TRADER_IDP_CHA_GROUP_NAME = if ($env:TRADER_IDP_CHA_GROUP_NAME) { $env:TRADER_IDP_CHA_GROUP_NAME } else { "CHA" }

$OGA_FORMS_PATH = if ($env:OGA_FORMS_PATH) { $env:OGA_FORMS_PATH } else { "./data/forms" }
$OGA_DEFAULT_FORM_ID = if ($env:OGA_DEFAULT_FORM_ID) { $env:OGA_DEFAULT_FORM_ID } else { "default" }
$OGA_ALLOWED_ORIGINS = if ($env:OGA_ALLOWED_ORIGINS) { $env:OGA_ALLOWED_ORIGINS } else { "*" }

$OGA_DB_DRIVER = if ($env:OGA_DB_DRIVER) { $env:OGA_DB_DRIVER } else { "sqlite" }
$OGA_DB_HOST = if ($env:OGA_DB_HOST) { $env:OGA_DB_HOST } else { "localhost" }
$OGA_DB_PORT = if ($env:OGA_DB_PORT) { $env:OGA_DB_PORT } else { "5432" }
$OGA_DB_USER = if ($env:OGA_DB_USER) { $env:OGA_DB_USER } else { "postgres" }
$OGA_DB_PASSWORD = if ($env:OGA_DB_PASSWORD) { $env:OGA_DB_PASSWORD } else { "changeme" }
$OGA_DB_NAME = if ($env:OGA_DB_NAME) { $env:OGA_DB_NAME } else { "oga_db" }
$OGA_DB_SSLMODE = if ($env:OGA_DB_SSLMODE) { $env:OGA_DB_SSLMODE } else { "disable" }

$OGA_NPQS_DB_PATH = if ($env:OGA_NPQS_DB_PATH) { $env:OGA_NPQS_DB_PATH } else { "./npqs.db" }
$OGA_FCAU_DB_PATH = if ($env:OGA_FCAU_DB_PATH) { $env:OGA_FCAU_DB_PATH } else { "./fcau.db" }
$OGA_IRD_DB_PATH = if ($env:OGA_IRD_DB_PATH) { $env:OGA_IRD_DB_PATH } else { "./ird.db" }
$OGA_CDA_DB_PATH = if ($env:OGA_CDA_DB_PATH) { $env:OGA_CDA_DB_PATH } else { "./cda.db" }
$OGA_APP_NPQS_BRANDING_PATH = if ($env:OGA_APP_NPQS_BRANDING_PATH) { $env:OGA_APP_NPQS_BRANDING_PATH } else { "./src/configs/npqs.yaml" }
$OGA_APP_FCAU_BRANDING_PATH = if ($env:OGA_APP_FCAU_BRANDING_PATH) { $env:OGA_APP_FCAU_BRANDING_PATH } else { "./src/configs/fcau.yaml" }
$OGA_APP_IRD_BRANDING_PATH = if ($env:OGA_APP_IRD_BRANDING_PATH) { $env:OGA_APP_IRD_BRANDING_PATH } else { "./src/configs/ird.yaml" }
$OGA_APP_CDA_BRANDING_PATH = if ($env:OGA_APP_CDA_BRANDING_PATH) { $env:OGA_APP_CDA_BRANDING_PATH } else { "./src/configs/cda.yaml" }

$OGA_NSW_NPQS_CLIENT_ID = if ($env:OGA_NSW_NPQS_CLIENT_ID) { $env:OGA_NSW_NPQS_CLIENT_ID } else { "NPQS_TO_NSW" }
$OGA_NSW_FCAU_CLIENT_ID = if ($env:OGA_NSW_FCAU_CLIENT_ID) { $env:OGA_NSW_FCAU_CLIENT_ID } else { "FCAU_TO_NSW" }
$OGA_NSW_IRD_CLIENT_ID = if ($env:OGA_NSW_IRD_CLIENT_ID) { $env:OGA_NSW_IRD_CLIENT_ID } else { "IRD_TO_NSW" }
$OGA_NSW_CDA_CLIENT_ID = if ($env:OGA_NSW_CDA_CLIENT_ID) { $env:OGA_NSW_CDA_CLIENT_ID } else { "CDA_TO_NSW" }
$OGA_NSW_NPQS_CLIENT_SECRET = if ($env:OGA_NSW_NPQS_CLIENT_SECRET) { $env:OGA_NSW_NPQS_CLIENT_SECRET } else { "1234" }
$OGA_NSW_FCAU_CLIENT_SECRET = if ($env:OGA_NSW_FCAU_CLIENT_SECRET) { $env:OGA_NSW_FCAU_CLIENT_SECRET } else { "1234" }
$OGA_NSW_IRD_CLIENT_SECRET = if ($env:OGA_NSW_IRD_CLIENT_SECRET) { $env:OGA_NSW_IRD_CLIENT_SECRET } else { "1234" }
$OGA_NSW_CDA_CLIENT_SECRET = if ($env:OGA_NSW_CDA_CLIENT_SECRET) { $env:OGA_NSW_CDA_CLIENT_SECRET } else { "1234" }
$OGA_NSW_TOKEN_URL = if ($env:OGA_NSW_TOKEN_URL) { $env:OGA_NSW_TOKEN_URL } else { "https://localhost:$IDP_PORT/oauth2/token" }
$OGA_NSW_SCOPES = if ($env:OGA_NSW_SCOPES) { $env:OGA_NSW_SCOPES } else { "" }
$OGA_NSW_TOKEN_INSECURE_SKIP_VERIFY = if ($env:OGA_NSW_TOKEN_INSECURE_SKIP_VERIFY) { $env:OGA_NSW_TOKEN_INSECURE_SKIP_VERIFY } else { "true" }

# OGA Registry
$OGA_INSTANCES = @(
    @{ Name="npqs"; Port=$OGA_NPQS_PORT; DbPath=$OGA_NPQS_DB_PATH; NswClientId=$OGA_NSW_NPQS_CLIENT_ID; NswClientSecret=$OGA_NSW_NPQS_CLIENT_SECRET; AppPort=$OGA_APP_NPQS_PORT; BrandingPath=$OGA_APP_NPQS_BRANDING_PATH; IdpClientId=$NPQS_IDP_CLIENT_ID; AppName="National Plant Quarantine Service (NPQS)" },
    @{ Name="fcau"; Port=$OGA_FCAU_PORT; DbPath=$OGA_FCAU_DB_PATH; NswClientId=$OGA_NSW_FCAU_CLIENT_ID; NswClientSecret=$OGA_NSW_FCAU_CLIENT_SECRET; AppPort=$OGA_APP_FCAU_PORT; BrandingPath=$OGA_APP_FCAU_BRANDING_PATH; IdpClientId=$FCAU_IDP_CLIENT_ID; AppName="Food Control Administration Unit (FCAU)" },
    @{ Name="ird"; Port=$OGA_IRD_PORT; DbPath=$OGA_IRD_DB_PATH; NswClientId=$OGA_NSW_IRD_CLIENT_ID; NswClientSecret=$OGA_NSW_IRD_CLIENT_SECRET; AppPort=$OGA_APP_IRD_PORT; BrandingPath=$OGA_APP_IRD_BRANDING_PATH; IdpClientId=$IRD_IDP_CLIENT_ID; AppName="Inland Revenue Department (IRD)" },
    @{ Name="cda"; Port=$OGA_CDA_PORT; DbPath=$OGA_CDA_DB_PATH; NswClientId=$OGA_NSW_CDA_CLIENT_ID; NswClientSecret=$OGA_NSW_CDA_CLIENT_SECRET; AppPort=$OGA_APP_CDA_PORT; BrandingPath=$OGA_APP_CDA_BRANDING_PATH; IdpClientId=$CDA_IDP_CLIENT_ID; AppName="Coconut Development Authority (CDA)" }
)

function ensure_branding_file($FileName, $AppName) {
    $ConfigDir = Join-Path $ROOT_DIR "portals/apps/oga-app/src/configs"
    $FilePath = Join-Path $ConfigDir $FileName
    if (-not (Test-Path $FilePath)) {
        New-Item -ItemType Directory -Path $ConfigDir -Force | Out-Null
        $Content = @"
branding:
  appName: "$AppName"
  logoUrl: ""
  favicon: ""
"@
        Set-Content -Path $FilePath -Value $Content
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

function clean_oga_databases() {
    Write-Host "Cleaning OGA databases (driver: $OGA_DB_DRIVER)..."
    if ($OGA_DB_DRIVER -eq "sqlite") {
        foreach ($Instance in $OGA_INSTANCES) {
            $DbPath = $Instance.DbPath
            $ResolvedPath = ""
            if ([System.IO.Path]::IsPathRooted($DbPath)) {
                $ResolvedPath = $DbPath
            } else {
                $RelativePath = $DbPath -replace '^\./', ''
                $ResolvedPath = Join-Path $ROOT_DIR "oga/$RelativePath"
            }

            if (Test-Path $ResolvedPath) {
                Write-Host "  Deleting SQLite DB for $($Instance.Name): $ResolvedPath"
                Remove-Item -Path $ResolvedPath -Force
            } else {
                Write-Host "  SQLite DB for $($Instance.Name) not found (nothing to delete): $ResolvedPath"
            }
        }
    } elseif ($OGA_DB_DRIVER -eq "postgres") {
        if (-not (Get-Command psql -ErrorAction SilentlyContinue)) {
            Write-Error "psql is required for Postgres DB cleaning but was not found in PATH"
            exit 1
        }
        $env:PGPASSWORD = $OGA_DB_PASSWORD
        Write-Host "  Dropping and recreating Postgres database: $OGA_DB_NAME"
        $TermQuery = "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$OGA_DB_NAME' AND pid <> pg_backend_pid();"
        & psql -h $OGA_DB_HOST -p $OGA_DB_PORT -U $OGA_DB_USER -d postgres -c $TermQuery | Out-Null
        & psql -h $OGA_DB_HOST -p $OGA_DB_PORT -U $OGA_DB_USER -d postgres -c "DROP DATABASE IF EXISTS `"$OGA_DB_NAME`";"
        & psql -h $OGA_DB_HOST -p $OGA_DB_PORT -U $OGA_DB_USER -d postgres -c "CREATE DATABASE `"$OGA_DB_NAME`";"
    }
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
    clean_oga_databases
}

Write-Host "Running backend migrations..."
Push-Location (Join-Path $ROOT_DIR "backend/internal/database/migrations")
$env:ENV_FILE = ${env-file}
$env:CLEAN_RUN = $CLEAN_RUN
powershell.exe -File ./run.ps1
Pop-Location

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

# Start OGA Services
foreach ($Instance in $OGA_INSTANCES) {
    $OgaEnv = @{
        OGA_PORT = $Instance.Port
        OGA_DB_DRIVER = $OGA_DB_DRIVER
        OGA_DB_PATH = $Instance.DbPath
        OGA_DB_HOST = $OGA_DB_HOST
        OGA_DB_PORT = $OGA_DB_PORT
        OGA_DB_USER = $OGA_DB_USER
        OGA_DB_PASSWORD = $OGA_DB_PASSWORD
        OGA_DB_NAME = $OGA_DB_NAME
        OGA_DB_SSLMODE = $OGA_DB_SSLMODE
        OGA_FORMS_PATH = $OGA_FORMS_PATH
        OGA_DEFAULT_FORM_ID = $OGA_DEFAULT_FORM_ID
        OGA_ALLOWED_ORIGINS = $OGA_ALLOWED_ORIGINS
        OGA_NSW_API_BASE_URL = "http://localhost:${BACKEND_PORT}/api/v1"
        OGA_NSW_CLIENT_ID = $Instance.NswClientId
        OGA_NSW_CLIENT_SECRET = $Instance.NswClientSecret
        OGA_NSW_TOKEN_URL = $OGA_NSW_TOKEN_URL
        OGA_NSW_SCOPES = $OGA_NSW_SCOPES
        OGA_NSW_TOKEN_INSECURE_SKIP_VERIFY = $OGA_NSW_TOKEN_INSECURE_SKIP_VERIFY
    }
    Start-ServiceJob -Name "oga-$($Instance.Name)" -Dir (Join-Path $ROOT_DIR "oga") -EnvVars $OgaEnv -ScriptBlock {
        param($Name, $Dir, $EnvVars)
        foreach ($Key in $EnvVars.Keys) { Set-Item "env:$Key" $EnvVars[$Key] }
        Set-Location $Dir
        go run ./cmd/server 2>&1 | ForEach-Object { Write-Host "[$Name] $_" }
    }
    # Small delay to prevent resource contention during Go compilation
    Start-Sleep -Seconds 2

    ensure_branding_file "$($Instance.Name).yaml" "$($Instance.AppName)"

    $OgaAppEnv = @{
        VITE_PORT = $Instance.AppPort
        VITE_BRANDING_PATH = $Instance.BrandingPath
        VITE_API_BASE_URL = "http://localhost:$($Instance.Port)"
        VITE_IDP_BASE_URL = $IDP_PUBLIC_URL
        VITE_IDP_CLIENT_ID = $Instance.IdpClientId
        VITE_APP_URL = "http://localhost:$($Instance.AppPort)"
        VITE_IDP_SCOPES = $IDP_SCOPES
        VITE_IDP_PLATFORM = $IDP_PLATFORM
    }
    Start-ServiceJob -Name "oga-app-$($Instance.Name)" -Dir (Join-Path $ROOT_DIR "portals/apps/oga-app") -EnvVars $OgaAppEnv -ScriptBlock {
        param($Name, $Dir, $EnvVars)
        foreach ($Key in $EnvVars.Keys) { Set-Item "env:$Key" $EnvVars[$Key] }
        Set-Location $Dir
        pnpm run dev 2>&1 | ForEach-Object { Write-Host "[$Name] $_" }
    }
    # Small delay to prevent Vite filesystem conflicts in the shared oga-app directory
    Start-Sleep -Seconds 2
}

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
foreach ($Instance in $OGA_INSTANCES) {
    Write-Host ("  - oga-{0,-9} -> http://localhost:{1}" -f $Instance.Name, $Instance.Port)
    Write-Host ("  - oga-app-{0,-5} -> http://localhost:{1}" -f $Instance.Name, $Instance.AppPort)
}
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
