#!/usr/bin/env bash
set -euo pipefail

APP_NAME="gotray"
INSTALL_DIR="/opt/${APP_NAME}"
BIN_PATH="${INSTALL_DIR}/${APP_NAME}"
ENV_ROOT="/etc/${APP_NAME}"
CONFIG_ROOT="/var/lib/${APP_NAME}"
REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TARGET_USER="${GOTRAY_INSTALL_USER:-${SUDO_USER:-${USER}}}"

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

if ! id -u "${TARGET_USER}" >/dev/null 2>&1; then
  log "Target user ${TARGET_USER} does not exist"
  exit 1
fi

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
sudo mkdir -p "${INSTALL_DIR}"
(cd "${REPO_DIR}" && go build -o "${BIN_PATH}" ./cmd/gotray)
sudo chmod 0755 "${BIN_PATH}"

log "Preparing environment configuration for ${TARGET_USER}"
sudo mkdir -p "${ENV_ROOT}"
ENV_FILE="${ENV_ROOT}/${TARGET_USER}.env"
if [[ ! -f "${ENV_FILE}" ]]; then
  sudo cp "${REPO_DIR}/.env.example" "${ENV_FILE}"
  sudo chmod 0600 "${ENV_FILE}"
fi
CONFIG_DIR="${CONFIG_ROOT}/${TARGET_USER}"
CONFIG_PATH="${CONFIG_DIR}/config.enc"
if ! sudo grep -q '^GOTRAY_CONFIG_PATH=' "${ENV_FILE}" 2>/dev/null; then
  printf '\nGOTRAY_CONFIG_PATH=%s\n' "${CONFIG_PATH}" | sudo tee -a "${ENV_FILE}" >/dev/null
fi

log "Preparing data directory"
sudo mkdir -p "${CONFIG_DIR}"
sudo chown "${TARGET_USER}":"${TARGET_USER}" "${CONFIG_DIR}"
sudo chmod 0700 "${CONFIG_DIR}"

log "Creating systemd template service"
SERVICE_FILE="/etc/systemd/system/${APP_NAME}@.service"
cat <<SERVICE | sudo tee "${SERVICE_FILE}" >/dev/null
[Unit]
Description=GoTray per-user tray (%i)
After=network.target

[Service]
Type=simple
EnvironmentFile=-${ENV_ROOT}/%i.env
Environment=GOTRAY_CONFIG_PATH=${CONFIG_ROOT}/%i/config.enc
ExecStart=${BIN_PATH} run
Restart=on-failure
User=%i
WorkingDirectory=${INSTALL_DIR}
RuntimeDirectory=${APP_NAME}-%i
RuntimeDirectoryMode=0700

[Install]
WantedBy=default.target
SERVICE

log "Reloading systemd daemon"
sudo systemctl daemon-reload

log "Enabling and starting gotray@${TARGET_USER}.service"
sudo systemctl enable --now "${APP_NAME}@${TARGET_USER}.service"

log "Installation complete"
