# InfraSense Platform - Windows Setup Script
# Usage: .\setup.ps1 [-Dev]
# Requires: Docker Desktop for Windows
param([switch]$Dev)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ScriptDir

function Write-Info  { Write-Host "[INFO]  $args" -ForegroundColor Green }
function Write-Warn  { Write-Host "[WARN]  $args" -ForegroundColor Yellow }
function Write-Err   { Write-Host "[ERROR] $args" -ForegroundColor Red; exit 1 }

Write-Host "================================================" -ForegroundColor Cyan
Write-Host "  InfraSense Platform Setup (Windows)" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan

# 1. Check prerequisites
Write-Info "Checking prerequisites..."
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Err "Docker is not installed. Install Docker Desktop from https://docs.docker.com/desktop/windows/"
}
$ComposeCmd = if (docker compose version 2>$null) { "docker compose" } else { "docker-compose" }
Write-Info "Using compose command: $ComposeCmd"

# 2. Environment setup
if (-not (Test-Path ".env")) {
    Write-Info "Creating .env from .env.example..."
    Copy-Item ".env.example" ".env"
    Write-Warn "Review and update .env before going to production (JWT_SECRET, ENCRYPTION_KEY, passwords)."
} else {
    Write-Info ".env already exists — skipping copy."
}

# 3. Build images
Write-Info "Building Docker images (this may take a few minutes on first run)..."
if ($Dev) {
    Invoke-Expression "$ComposeCmd -f docker-compose.yml -f docker-compose.dev.yml build"
} else {
    Invoke-Expression "$ComposeCmd build"
}

# 4. Start services
Write-Info "Starting InfraSense services..."
if ($Dev) {
    Invoke-Expression "$ComposeCmd -f docker-compose.yml -f docker-compose.dev.yml up -d"
} else {
    Invoke-Expression "$ComposeCmd up -d"
}

# 5. Wait for database
Write-Info "Waiting for PostgreSQL to be ready..."
$retries = 30
do {
    Start-Sleep -Seconds 2
    $ready = docker exec infrasense-postgres pg_isready -U infrasense 2>$null
    $retries--
} while (-not $ready -and $retries -gt 0)
if ($retries -eq 0) { Write-Err "PostgreSQL did not become ready in time." }
Write-Info "PostgreSQL is ready."

# 6. Wait for API server
Write-Info "Waiting for API server to be healthy..."
$retries = 30
do {
    Start-Sleep -Seconds 3
    try {
        $resp = Invoke-WebRequest -Uri "http://localhost:8080/health" -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop
        $healthy = $resp.StatusCode -eq 200
    } catch { $healthy = $false }
    $retries--
} while (-not $healthy -and $retries -gt 0)
if (-not $healthy) { Write-Warn "API server health check timed out — check logs: docker compose logs api-server" }

# 7. Seed admin user
Write-Info "Seeding default admin user..."
try {
    Get-Content "backend\scripts\create_admin.sql" | docker exec -i infrasense-postgres psql -U infrasense -d infrasense 2>$null
} catch {
    Write-Warn "Admin seed skipped (user may already exist)."
}

Write-Host ""
Write-Host "================================================" -ForegroundColor Cyan
Write-Host "  InfraSense is ready!" -ForegroundColor Green
Write-Host "  Dashboard:  http://localhost" -ForegroundColor White
if ($Dev) {
    Write-Host "  API:        http://localhost:8080" -ForegroundColor White
    Write-Host "  Grafana:    http://localhost:3000" -ForegroundColor White
    Write-Host "  Prometheus: http://localhost:9090" -ForegroundColor White
}
Write-Host "  Login:      admin / Admin@123456" -ForegroundColor White
Write-Host "  (Change the password after first login)" -ForegroundColor Yellow
Write-Host "================================================" -ForegroundColor Cyan
