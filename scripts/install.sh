#!/usr/bin/env bash
set -euo pipefail

APP_NAME="gotray"
SERVICE_NAME="gotray"
SERVICE_USER="gotray"
SERVICE_GROUP="gotray"
INSTALL_DIR="/opt/${APP_NAME}"
BIN_PATH="${INSTALL_DIR}/${APP_NAME}"
ENV_FILE="/etc/${APP_NAME}/.env"
CONFIG_DIR="/var/lib/${APP_NAME}"
CONFIG_PATH="${CONFIG_DIR}/config.enc"
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

if ! id -u "${SERVICE_USER}" >/dev/null 2>&1; then
  log "Creating service account ${SERVICE_USER}"
  sudo useradd --system --home "${CONFIG_DIR}" --shell /usr/sbin/nologin "${SERVICE_USER}"
fi

log "Preparing environment configuration"
sudo mkdir -p "$(dirname "${ENV_FILE}")"
if [[ ! -f "${ENV_FILE}" ]]; then
  sudo cp "${REPO_DIR}/.env.example" "${ENV_FILE}"
  sudo chmod 0600 "${ENV_FILE}"
fi

log "Preparing data directory"
sudo mkdir -p "${CONFIG_DIR}"
sudo chown "${SERVICE_USER}":"${SERVICE_GROUP}" "${CONFIG_DIR}"

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
ExecStart=${BIN_PATH} serve
Restart=on-failure
User=${SERVICE_USER}
Group=${SERVICE_GROUP}
WorkingDirectory=${INSTALL_DIR}

[Install]
WantedBy=multi-user.target
SERVICE

log "Reloading systemd daemon"
sudo systemctl daemon-reload

log "Enabling and starting service"
sudo systemctl enable --now "${SERVICE_NAME}.service"

log "Installation complete"
