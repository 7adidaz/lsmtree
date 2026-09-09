#!/usr/bin/env bash
set -euo pipefail

GO_VERSION_TARBALL="go1.26.6.linux-amd64.tar.gz"
GO_DOWNLOAD_URL="https://go.dev/dl/${GO_VERSION_TARBALL}"
INSTALL_DIR="/usr/local"
REPO_URL="https://github.com/7adidaz/lsmtree"
APP_DIR="/opt/lsmtree"
DATA_DIR="${APP_DIR}/data"
BIN_NAME="lsm-tree"
LOG_FILE="/var/log/lsmtree.log"
SERVICE_NAME="lsmtree"
APP_PORT="8080"

echo "==> Updating packages"
sudo dnf update -y

echo "==> Installing git and nginx"
sudo dnf install -y git tar gzip nginx

echo "==> Downloading Go ${GO_VERSION_TARBALL}"
cd /tmp
curl -fsSL -o "${GO_VERSION_TARBALL}" "${GO_DOWNLOAD_URL}"

echo "==> Installing Go to ${INSTALL_DIR}"
sudo rm -rf "${INSTALL_DIR}/go"
sudo tar -C "${INSTALL_DIR}" -xzf "${GO_VERSION_TARBALL}"

export HOME="${HOME:-/root}"
export PATH="${INSTALL_DIR}/go/bin:${PATH}"
export GOPATH="${HOME}/go"
export GOCACHE="${HOME}/.cache/go-build"
export GOMODCACHE="${GOPATH}/pkg/mod"
mkdir -p "${GOPATH}" "${GOCACHE}" "${GOMODCACHE}"

sudo tee /etc/profile.d/go.sh > /dev/null <<'PROFILE_EOF'
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export GOCACHE=$HOME/.cache/go-build
export GOMODCACHE=$GOPATH/pkg/mod
PROFILE_EOF
sudo chmod +x /etc/profile.d/go.sh

echo "==> Verifying Go install"
go version

echo "==> Cloning repo into ${APP_DIR}"
sudo rm -rf "${APP_DIR}"
sudo git clone "${REPO_URL}" "${APP_DIR}"
sudo chown -R "$(whoami)":"$(whoami)" "${APP_DIR}"

echo "==> Creating data folder"
mkdir -p "${DATA_DIR}"

echo "==> Setting up swap (t3.micro has only 1GB RAM; Go build of gin's dependency tree can OOM without it)"
if [ ! -f /swapfile ]; then
  sudo dd if=/dev/zero of=/swapfile bs=1M count=1024
  sudo chmod 600 /swapfile
  sudo mkswap /swapfile
  sudo swapon /swapfile
  if ! grep -q '^/swapfile' /etc/fstab; then
    echo '/swapfile swap swap defaults 0 0' | sudo tee -a /etc/fstab > /dev/null
  fi
fi

echo "==> Building project (-p=1 limits parallel compiler jobs so small instances do not OOM)"
cd "${APP_DIR}"
GOFLAGS="-p=1" go build -o "${BIN_NAME}" main.go

echo "==> Creating systemd service for ${BIN_NAME}"
sudo tee "/etc/systemd/system/${SERVICE_NAME}.service" > /dev/null <<EOF
[Unit]
Description=LSM Tree KV Store
After=network.target

[Service]
Type=simple
WorkingDirectory=${APP_DIR}
ExecStart=${APP_DIR}/${BIN_NAME}
Restart=on-failure
RestartSec=2
User=$(whoami)
StandardOutput=append:${LOG_FILE}
StandardError=append:${LOG_FILE}

[Install]
WantedBy=multi-user.target
EOF

sudo touch "${LOG_FILE}"
sudo chown "$(whoami)":"$(whoami)" "${LOG_FILE}"

sudo systemctl daemon-reload
sudo systemctl enable "${SERVICE_NAME}"
sudo systemctl restart "${SERVICE_NAME}"

echo "==> Configuring nginx reverse proxy (port 80 -> ${APP_PORT})"
sudo tee "/etc/nginx/conf.d/${SERVICE_NAME}.conf" > /dev/null <<EOF
server {
    listen 80;
    server_name _;

    location / {
        proxy_pass http://127.0.0.1:${APP_PORT};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

if [ -f /etc/nginx/conf.d/default.conf ]; then
  sudo rm -f /etc/nginx/conf.d/default.conf
fi

sudo nginx -t
sudo systemctl enable nginx
sudo systemctl restart nginx

sleep 1
echo "==> Done."
echo "    App service: systemctl status ${SERVICE_NAME}"
echo "    App logs:    ${LOG_FILE}"
echo "    Nginx proxy: http://<instance-ip>/ping  (forwards to 127.0.0.1:${APP_PORT})"
echo "    Direct test: curl http://localhost:${APP_PORT}/ping"
