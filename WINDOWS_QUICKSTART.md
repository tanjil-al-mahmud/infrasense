# InfraSense — Windows Quick Start Guide

Everything runs in Docker Desktop. No Go, Node, or other tooling required on your machine.

## Prerequisites

- [Docker Desktop for Windows](https://www.docker.com/products/docker-desktop/) installed and running
- WSL 2 backend enabled (recommended — Docker Desktop → Settings → General)
- PowerShell 5.1+ (built into Windows 10/11)

---

## Option A — Automated Setup (Recommended)

```powershell
git clone <your-repo-url>
cd infrasense
.\setup.ps1
```

For development mode (all ports exposed):

```powershell
.\setup.ps1 -Dev
```

The script handles everything: `.env` creation, image builds, service startup, and admin user seeding.

---

## Option B — Manual Setup

**Step 1 — Configure environment**

```powershell
Copy-Item .env.example .env
# Edit .env with your values (JWT_SECRET, ENCRYPTION_KEY, passwords)
```

**Step 2 — Start the stack**

```powershell
docker compose up -d --build
```

First run builds all images — expect 5–10 minutes. Subsequent starts are fast.

**Step 3 — Seed the admin user**

```powershell
Get-Content backend\scripts\create_admin.sql | `
  docker exec -i infrasense-postgres psql -U infrasense -d infrasense
```

---

## Accessing the Platform

| Service    | URL                       | Credentials              |
|------------|---------------------------|--------------------------|
| Dashboard  | http://localhost          | admin / Admin@123456     |
| Grafana    | http://localhost/grafana/ | admin / Admin@123456     |
| API Health | http://localhost/health   | —                        |

Change all passwords after first login.

---

## Useful Commands

```powershell
# Check service status
docker compose ps

# View logs
docker compose logs -f

# View logs for a specific service
docker compose logs -f api-server

# Stop the stack
docker compose down

# Full reset (removes all data)
docker compose down -v
docker compose up -d --build
```

---

## Troubleshooting

**Port 80 already in use**
```powershell
netstat -ano | findstr ":80"
# Stop the conflicting process, or set HTTP_PORT=8090 in .env
```

**Build fails**
```powershell
docker compose build --no-cache api-server
docker compose build --no-cache frontend
```

**Services stuck in "starting" state**
```powershell
docker compose logs api-server --tail 50
docker compose logs postgres --tail 50
```
