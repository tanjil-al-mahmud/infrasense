#!/usr/bin/env bash
# InfraSense Platform - Linux/macOS Setup Script
# Usage: bash setup.sh [--dev]
set -euo pipefail

DEV_MODE=false
[[ "${1:-}" == "--dev" ]] && DEV_MODE=true

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "================================================"
echo "  InfraSense Platform Setup"
echo "================================================"

# 1. Check prerequisites
info "Checking prerequisites..."
command -v docker  >/dev/null 2>&1 || error "Docker is not installed. Install from https://docs.docker.com/get-docker/"
command -v docker compose version >/dev/null 2>&1 || \
  docker-compose version >/dev/null 2>&1 || \
  error "Docker Compose is not installed."

DOCKER_COMPOSE="docker compose"
docker compose version >/dev/null 2>&1 || DOCKER_COMPOSE="docker-compose"

# 2. Environment setup
if [ ! -f .env ]; then
  info "Creating .env from .env.example..."
  cp .env.example .env
  warn "Review and update .env before going to production (JWT_SECRET, ENCRYPTION_KEY, passwords)."
else
  info ".env already exists — skipping copy."
fi

# 3. Build images
info "Building Docker images (this may take a few minutes on first run)..."
if $DEV_MODE; then
  $DOCKER_COMPOSE -f docker-compose.yml -f docker-compose.dev.yml build
else
  $DOCKER_COMPOSE build
fi

# 4. Start services
info "Starting InfraSense services..."
if $DEV_MODE; then
  $DOCKER_COMPOSE -f docker-compose.yml -f docker-compose.dev.yml up -d
else
  $DOCKER_COMPOSE up -d
fi

# 5. Wait for database
info "Waiting for PostgreSQL to be ready..."
RETRIES=30
until $DOCKER_COMPOSE exec -T postgres pg_isready -U infrasense >/dev/null 2>&1 || [ $RETRIES -eq 0 ]; do
  echo -n "."
  sleep 2
  ((RETRIES--))
done
echo ""
[ $RETRIES -eq 0 ] && error "PostgreSQL did not become ready in time."
info "PostgreSQL is ready."

# 6. Wait for API server (runs migrations on startup)
info "Waiting for API server to be healthy..."
RETRIES=30
until curl -sf http://localhost:8080/health >/dev/null 2>&1 || [ $RETRIES -eq 0 ]; do
  echo -n "."
  sleep 3
  ((RETRIES--))
done
echo ""
[ $RETRIES -eq 0 ] && warn "API server health check timed out — check logs: docker compose logs api-server"

# 7. Seed admin user if needed
info "Seeding default admin user..."
$DOCKER_COMPOSE exec -T postgres psql -U infrasense -d infrasense \
  -f /dev/stdin < backend/scripts/create_admin.sql 2>/dev/null || \
  warn "Admin seed skipped (user may already exist)."

echo ""
echo "================================================"
echo "  InfraSense is ready!"
echo "  Dashboard:  http://localhost"
if $DEV_MODE; then
  echo "  API:        http://localhost:8080"
  echo "  Grafana:    http://localhost:3000"
  echo "  Prometheus: http://localhost:9090"
fi
echo "  Login:      admin / Admin@123456"
echo "  (Change the password after first login)"
echo "================================================"
