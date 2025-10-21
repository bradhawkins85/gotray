#!/usr/bin/env bash
set -euo pipefail

APP_NAME="gotray"
SERVICE_NAME="gotray"
INSTALL_DIR="/opt/${APP_NAME}"
BIN_PATH="${INSTALL_DIR}/${APP_NAME}"
ENV_FILE="/etc/${APP_NAME}/.env"
CONFIG_PATH="/var/lib/${APP_NAME}/config.enc"
REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"

log() {
  printf '[%s] %s\n' "${APP_NAME}" "$1"
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    log "Installing missing dependency: $1"
    if command -v apt-get >/dev/null 2>&1; then
      sudo apt-get update
      sudo apt-get install -y "$2"
    else
      log "Please install $1 manually"
      exit 1
    fi
  fi
}

log "Verifying dependencies"
require_command go golang
require_command python3 python3
require_command systemctl systemd

if ! python3 -m venv --help >/dev/null 2>&1; then
  log "Ensuring python3-venv is available"
  if command -v apt-get >/dev/null 2>&1; then
    sudo apt-get install -y python3-venv
  fi
fi

log "Building GoTray binary"
mkdir -p "${INSTALL_DIR}"
(cd "${REPO_DIR}" && go build -o "${BIN_PATH}" ./cmd/gotray)
chmod 0755 "${BIN_PATH}"

log "Preparing environment configuration"
sudo mkdir -p "$(dirname "${ENV_FILE}")"
if [[ ! -f "${ENV_FILE}" ]]; then
  sudo cp "${REPO_DIR}/.env.example" "${ENV_FILE}"
  sudo chmod 0600 "${ENV_FILE}"
fi

log "Preparing data directory"
sudo mkdir -p "$(dirname "${CONFIG_PATH}")"
sudo chown "$(whoami)":"$(whoami)" "$(dirname "${CONFIG_PATH}")"

log "Creating systemd service"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
cat <<SERVICE | sudo tee "${SERVICE_FILE}" >/dev/null
[Unit]
Description=GoTray system tray service
After=network.target

[Service]
Type=simple
EnvironmentFile=${ENV_FILE}
Environment=GOTRAY_CONFIG_PATH=${CONFIG_PATH}
ExecStart=${BIN_PATH}
Restart=on-failure
User=$(whoami)
Group=$(whoami)
WorkingDirectory=${INSTALL_DIR}

[Install]
WantedBy=default.target
SERVICE

log "Reloading systemd daemon"
sudo systemctl daemon-reload

log "Enabling and starting service"
sudo systemctl enable --now "${SERVICE_NAME}.service"

log "Installation complete"
