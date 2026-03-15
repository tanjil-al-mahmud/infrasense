#!/usr/bin/env bash
# InfraSense Ubuntu 24.04 Native Installer
# This script installs InfraSense from source onto a fresh Ubuntu 24.04 system.

set -e

echo "InfraSense Ubuntu 24.04 Native Installer"
echo "========================================"

if [ "$EUID" -ne 0 ]; then
  echo "[ERROR] Please run this script as root (sudo bash scripts/install-ubuntu.sh)"
  exit 1
fi

echo "[1/7] Installing system dependencies..."
apt-get update
# Non-interactive installs
DEBIAN_FRONTEND=noninteractive apt-get install -y wget curl git build-essential postgresql postgresql-contrib nginx jq

echo "[2/7] Installing Go and Node.js..."
# Install Go 1.22+
if ! command -v go &> /dev/null; then
  echo "Downloading Go 1.22..."
  wget -q https://go.dev/dl/go1.22.1.linux-amd64.tar.gz
  tar -C /usr/local -xzf go1.22.1.linux-amd64.tar.gz
  rm go1.22.1.linux-amd64.tar.gz
fi
export PATH=$PATH:/usr/local/go/bin

# Install Node.js 20.x
if ! command -v node &> /dev/null; then
  echo "Downloading Node 20.x..."
  curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
  DEBIAN_FRONTEND=noninteractive apt-get install -y nodejs
fi

echo "[3/7] Setting up PostgreSQL..."
echo "Creating database and user 'infrasense'..."
# Create the user and database if they don't exist
sudo -u postgres psql -tc "SELECT 1 FROM pg_roles WHERE rolname='infrasense'" | grep -q 1 || sudo -u postgres psql -c "CREATE USER infrasense WITH PASSWORD 'infrasense';"
sudo -u postgres psql -tc "SELECT 1 FROM pg_database WHERE datname='infrasense'" | grep -q 1 || sudo -u postgres psql -c "CREATE DATABASE infrasense OWNER infrasense;"

echo "[4/7] Installing VictoriaMetrics..."
if [ ! -f /usr/local/bin/victoria-metrics-prod ]; then
    echo "Downloading VictoriaMetrics v1.99.0..."
    wget -q https://github.com/VictoriaMetrics/VictoriaMetrics/releases/download/v1.99.0/victoria-metrics-linux-amd64-v1.99.0.tar.gz
    tar -xzf victoria-metrics-linux-amd64-v1.99.0.tar.gz -C /usr/local/bin victoria-metrics-prod
    rm victoria-metrics-linux-amd64-v1.99.0.tar.gz
fi

cat << 'EOF' > /etc/systemd/system/victoriametrics.service
[Unit]
Description=VictoriaMetrics
After=network.target

[Service]
ExecStart=/usr/local/bin/victoria-metrics-prod -storageDataPath=/var/lib/victoria-metrics -retentionPeriod=90d -httpListenAddr=127.0.0.1:8428
Restart=always

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable --now victoriametrics

echo "[5/7] Building InfraSense Services from Source..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
APP_DIR="/opt/infrasense"

mkdir -p $APP_DIR/bin $APP_DIR/frontend $APP_DIR/config /var/log/infrasense

cd "$REPO_ROOT"

# Build Backend
echo "Building backend..."
cd backend
go build -o $APP_DIR/bin/api-server ./cmd/server/main.go
# Copy migrations for the backend to use
cp -r migrations $APP_DIR/migrations
# Copy example config as a base
cp config.example.yml $APP_DIR/config.yml
cd ..

# Build frontend
echo "Building React frontend..."
cd frontend
npm install
npm run build
cp -r dist/* $APP_DIR/frontend/
cd ..

# Build collectors
echo "Building Go collectors..."
for c in ipmi-collector proxmox-collector redfish-collector snmp-collector; do
  if [ -d "collectors/$c" ]; then
    echo "  - $c"
    cd collectors/$c
    go build -o $APP_DIR/bin/$c ./cmd/main.go || echo "Warning: Failed to build $c"
    cd ../..
  fi
done

echo "Building notification service..."
if [ -d "notification-service" ]; then
  cd notification-service
  go build -o $APP_DIR/bin/notification-service ./cmd/main.go || echo "Warning: Failed to build notification-service"
  cd ..
fi

# Generate Configuration
echo "Writing configuration..."
cat << 'EOF' > $APP_DIR/config/.env
DB_HOST=127.0.0.1
DB_PORT=5432
DB_NAME=infrasense
DB_USER=infrasense
DB_PASSWORD=infrasense
DB_SSLMODE=disable
JWT_SECRET=install_auto_jwt_key_32_chars_12
ENCRYPTION_KEY=install_auto_enc_key_32_chars_12
VICTORIAMETRICS_URL=http://127.0.0.1:8428/api/v1/write
API_HOST=127.0.0.1
API_PORT=8080
ENVIRONMENT=production
LOG_LEVEL=info
EOF

echo "[6/7] Setting up Systemd services..."

# Create systemd unit for API Server
cat << 'EOF' > /etc/systemd/system/infrasense-api.service
[Unit]
Description=InfraSense API Server
After=network.target postgresql.service victoriametrics.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/infrasense
EnvironmentFile=/opt/infrasense/config/.env
ExecStart=/opt/infrasense/bin/api-server
Restart=always

[Install]
WantedBy=multi-user.target
EOF

# Create systemd unit for Notification Service
cat << 'EOF' > /etc/systemd/system/infrasense-notification.service
[Unit]
Description=InfraSense Notification Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/infrasense
EnvironmentFile=/opt/infrasense/config/.env
ExecStart=/opt/infrasense/bin/notification-service
Restart=always

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now infrasense-api
systemctl enable --now infrasense-notification

echo "[7/7] Configuring Nginx reverse proxy..."
cat << 'EOF' > /etc/nginx/sites-available/infrasense
server {
    listen 80;
    server_name _;

    root /opt/infrasense/frontend;
    index index.html;

    # API Proxy
    location /api/v1/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # SPA Fallback
    location / {
        try_files $uri $uri/ /index.html;
    }
}
EOF

ln -sf /etc/nginx/sites-available/infrasense /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default
systemctl restart nginx

echo "====================================================="
echo "   Installation of InfraSense is Complete!"
echo "   Access your dashboard at: http://localhost"
echo "   (or your server's public IP address)"
echo "====================================================="
