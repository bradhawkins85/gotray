#!/usr/bin/env bash
set -euo pipefail

APP_NAME="gotray"
SERVICE_NAME="gotray-dev"
INSTALL_DIR="/opt/${APP_NAME}-dev"
BIN_PATH="${INSTALL_DIR}/${APP_NAME}"
ENV_FILE="/etc/${APP_NAME}/dev.env"
CONFIG_PATH="/var/lib/${APP_NAME}-dev/config.enc"
REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"

log() {
  printf '[%s] %s\n' "${SERVICE_NAME}" "$1"
}

log "Ensuring dependencies are installed"
if ! command -v go >/dev/null 2>&1; then
  echo "Go is required" >&2
  exit 1
fi
if ! python3 -m venv --help >/dev/null 2>&1; then
  echo "python3-venv is required" >&2
  exit 1
fi

log "Building development binary"
mkdir -p "${INSTALL_DIR}"
(cd "${REPO_DIR}" && go build -o "${BIN_PATH}" ./cmd/gotray)
chmod 0755 "${BIN_PATH}"

log "Preparing development environment file"
sudo mkdir -p "$(dirname "${ENV_FILE}")"
if [[ ! -f "${ENV_FILE}" ]]; then
  sudo cp "${REPO_DIR}/.env.example" "${ENV_FILE}"
  sudo chmod 0600 "${ENV_FILE}"
fi

log "Creating development data directory"
sudo mkdir -p "$(dirname "${CONFIG_PATH}")"
sudo chown "$(whoami)":"$(whoami)" "$(dirname "${CONFIG_PATH}")"

SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
cat <<SERVICE | sudo tee "${SERVICE_FILE}" >/dev/null
[Unit]
Description=GoTray development service
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

log "Enabling development service"
sudo systemctl enable --now "${SERVICE_NAME}.service"

log "Development installation complete"
