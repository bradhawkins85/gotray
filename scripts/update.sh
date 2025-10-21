#!/usr/bin/env bash
set -euo pipefail

APP_NAME="gotray"
SERVICE_NAME="gotray"
INSTALL_DIR="/opt/${APP_NAME}"
BIN_PATH="${INSTALL_DIR}/${APP_NAME}"
REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"

log() {
  printf '[%s] %s\n' "${APP_NAME}" "$1"
}

log "Pulling latest source"
if [[ -d "${REPO_DIR}/.git" ]]; then
  (cd "${REPO_DIR}" && git pull --rebase)
fi

log "Building updated binary"
(cd "${REPO_DIR}" && go build -o "${BIN_PATH}" ./cmd/gotray)
chmod 0755 "${BIN_PATH}"

log "Restarting service"
sudo systemctl restart "${SERVICE_NAME}.service"

log "Update complete"
