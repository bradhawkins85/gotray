#!/usr/bin/env bash
set -euo pipefail

APP_NAME="gotray"
INSTALL_DIR="/opt/${APP_NAME}"
BIN_PATH="${INSTALL_DIR}/${APP_NAME}"
REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TARGET_USER="${GOTRAY_INSTALL_USER:-${SUDO_USER:-${USER}}}"

log() {
  printf '[%s] %s\n' "${APP_NAME}" "$1"
}

log "Pulling latest source"
if [[ -d "${REPO_DIR}/.git" ]]; then
  (cd "${REPO_DIR}" && git pull --rebase)
fi

log "Building updated binary"
(cd "${REPO_DIR}" && go build -o "${BIN_PATH}" ./cmd/gotray)
sudo chmod 0755 "${BIN_PATH}"

log "Restarting user service gotray@${TARGET_USER}.service"
sudo systemctl restart "${APP_NAME}@${TARGET_USER}.service"

log "Update complete"
