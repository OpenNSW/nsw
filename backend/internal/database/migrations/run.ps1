param(
    [switch]${clean-run},
    [switch]${migrations}
)

# Load environment variables from env file
# Default: backend/.env
$EnvFilePath = if ($env:ENV_FILE) { $env:ENV_FILE } else { Join-Path $PSScriptRoot "../../../.env" }

if (Test-Path $EnvFilePath) {
    Write-Host "Loading environment variables from $EnvFilePath"
    Get-Content $EnvFilePath | ForEach-Object {
        if ($_ -match '^([^#\s][^=]+)=(.*)$') {
            $name = $matches[1].Trim()
            $value = $matches[2].Trim().Trim('"').Trim("'")
            Set-Item "env:$name" $value
        }
    }
} else {
    Write-Error "Error: env file not found: $EnvFilePath"
    exit 1
}

# Ensure environment variables are set
$RequiredVars = @("DB_HOST", "DB_PORT", "DB_USERNAME", "DB_PASSWORD", "DB_NAME")
foreach ($Var in $RequiredVars) {
    if (-not (Get-ChildItem Env:$Var -ErrorAction SilentlyContinue)) {
        Write-Error "Error: Required environment variable $Var is not set."
        exit 1
    }
}

$MigrationDbHost = if ($env:MIGRATION_DB_HOST) { $env:MIGRATION_DB_HOST } else { $env:DB_HOST }
$MigrationDbHost = $MigrationDbHost -replace "host.docker.internal", "localhost"

$NpqsOgaSubmissionUrl = if ($env:NPQS_OGA_SUBMISSION_URL) { $env:NPQS_OGA_SUBMISSION_URL } else { "http://localhost:8081/api/oga/inject" }
$FcauOgaSubmissionUrl = if ($env:FCAU_OGA_SUBMISSION_URL) { $env:FCAU_OGA_SUBMISSION_URL } else { "http://localhost:8082/api/oga/inject" }
$PreconsignmentOgaSubmissionUrl = if ($env:PRECONSIGNMENT_OGA_SUBMISSION_URL) { $env:PRECONSIGNMENT_OGA_SUBMISSION_URL } else { "http://localhost:8083/api/oga/inject" }
$CdaOgaSubmissionUrl = if ($env:CDA_OGA_SUBMISSION_URL) { $env:CDA_OGA_SUBMISSION_URL } else { "http://localhost:8084/api/oga/inject" }

# Handle Clean Run
if (${clean-run}) {
    # Set PGPASSWORD for psql
    $env:PGPASSWORD = $env:DB_PASSWORD

    # Force disconnect other users and drop the database
    Write-Host "Dropping database `"$($env:DB_NAME)`"..."
    & psql -h "$MigrationDbHost" -p "$($env:DB_PORT)" -U "$($env:DB_USERNAME)" -d postgres -c "DROP DATABASE IF EXISTS `"$($env:DB_NAME)`" WITH (FORCE);"

    # Recreate the database
    Write-Host "Creating database `"$($env:DB_NAME)`"..."
    & psql -h "$MigrationDbHost" -p "$($env:DB_PORT)" -U "$($env:DB_USERNAME)" -d postgres -c "CREATE DATABASE `"$($env:DB_NAME)`";"
}

# Dynamically discover migration files
$AllMigrations = Get-ChildItem -Path $PSScriptRoot -Filter "*.up.sql" | Sort-Object Name

$MigrationsToRun = @()
if (${clean-run}) {
    $MigrationsToRun = $AllMigrations
} elseif (${migrations}) {
    $Prefix = Read-Host "Enter the migration prefix to start from (e.g., 013)"
    if ([string]::IsNullOrWhiteSpace($Prefix)) {
        Write-Error "Migration prefix is required when using -migrations."
        exit 1
    }
    
    # Filter files starting from the prefix
    $MigrationsToRun = $AllMigrations | Where-Object { $_.Name -ge $Prefix }
    
    if ($null -eq $MigrationsToRun -or $MigrationsToRun.Count -eq 0) {
        Write-Warning "No migration files found matching or following prefix: $Prefix"
        exit 0
    }
}

Write-Host "Selected $($MigrationsToRun.Count) migration(s) to execute."

Write-Host "Starting database migrations..."

# Loop through and execute each file
foreach ($File in $MigrationsToRun) {
    $FilePath = $File.FullName
    Write-Host "Executing: $($File.Name)"
    & psql `
        -v ON_ERROR_STOP=1 `
        -v NPQS_OGA_SUBMISSION_URL="$NpqsOgaSubmissionUrl" `
        -v FCAU_OGA_SUBMISSION_URL="$FcauOgaSubmissionUrl" `
        -v PRECONSIGNMENT_OGA_SUBMISSION_URL="$PreconsignmentOgaSubmissionUrl" `
        -v CDA_OGA_SUBMISSION_URL="$CdaOgaSubmissionUrl" `
        -h "$MigrationDbHost" -p "$($env:DB_PORT)" -U "$($env:DB_USERNAME)" -d "$($env:DB_NAME)" -f "$FilePath"
    
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Error executing $($File.Name). Aborting."
        exit 1
    }
}

Write-Host "Migrations completed successfully."
