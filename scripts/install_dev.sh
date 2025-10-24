#!/usr/bin/env bash
set -euo pipefail

APP_NAME="gotray"
SERVICE_NAME="${APP_NAME}-dev"
INSTALL_DIR="/opt/${SERVICE_NAME}"
BIN_PATH="${INSTALL_DIR}/${APP_NAME}"
ENV_ROOT="/etc/${APP_NAME}"
CONFIG_ROOT="/var/lib/${SERVICE_NAME}"
REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TARGET_USER="${GOTRAY_INSTALL_USER:-${SUDO_USER:-${USER}}}"

log() {
  printf '[%s] %s\n' "${SERVICE_NAME}" "$1"
}

if ! id -u "${TARGET_USER}" >/dev/null 2>&1; then
  log "Target user ${TARGET_USER} does not exist"
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  log "Go is required to build GoTray"
  exit 1
fi

if ! command -v systemctl >/dev/null 2>&1; then
  log "systemctl is required to manage services"
  exit 1
fi

log "Building development binary"
sudo mkdir -p "${INSTALL_DIR}"
(cd "${REPO_DIR}" && go build -o "${BIN_PATH}" ./cmd/gotray)
sudo chmod 0755 "${BIN_PATH}"

log "Preparing development environment file"
sudo mkdir -p "${ENV_ROOT}"
ENV_FILE="${ENV_ROOT}/dev-${TARGET_USER}.env"
if [[ ! -f "${ENV_FILE}" ]]; then
  sudo cp "${REPO_DIR}/.env.example" "${ENV_FILE}"
  sudo chmod 0600 "${ENV_FILE}"
fi
CONFIG_DIR="${CONFIG_ROOT}/${TARGET_USER}"
CONFIG_PATH="${CONFIG_DIR}/config.enc"
if ! sudo grep -q '^GOTRAY_CONFIG_PATH=' "${ENV_FILE}" 2>/dev/null; then
  printf '\nGOTRAY_CONFIG_PATH=%s\n' "${CONFIG_PATH}" | sudo tee -a "${ENV_FILE}" >/dev/null
fi

log "Preparing development data directory"
sudo mkdir -p "${CONFIG_DIR}"
sudo chown "${TARGET_USER}":"${TARGET_USER}" "${CONFIG_DIR}"
sudo chmod 0700 "${CONFIG_DIR}"

log "Creating development systemd template"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}@.service"
cat <<SERVICE | sudo tee "${SERVICE_FILE}" >/dev/null
[Unit]
Description=GoTray development tray (%i)
After=network.target

[Service]
Type=simple
EnvironmentFile=-${ENV_ROOT}/dev-%i.env
Environment=GOTRAY_CONFIG_PATH=${CONFIG_ROOT}/%i/config.enc
ExecStart=${BIN_PATH} run
Restart=on-failure
User=%i
WorkingDirectory=${INSTALL_DIR}
RuntimeDirectory=${SERVICE_NAME}-%i
RuntimeDirectoryMode=0700

[Install]
WantedBy=default.target
SERVICE

log "Reloading systemd daemon"
sudo systemctl daemon-reload

log "Enabling development service gotray-dev@${TARGET_USER}.service"
sudo systemctl enable --now "${SERVICE_NAME}@${TARGET_USER}.service"

log "Development installation complete"
