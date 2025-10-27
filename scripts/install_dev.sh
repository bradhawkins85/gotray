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
GO_BUILD_FLAGS=(-trimpath "-ldflags=-s -w")
GOTRAY_ENABLE_COMPRESSION="${GOTRAY_ENABLE_COMPRESSION:-0}"
GOTRAY_COMPRESSION_TOOL="${GOTRAY_COMPRESSION_TOOL:-upx}"
GOTRAY_COMPRESSION_ARGS="${GOTRAY_COMPRESSION_ARGS:---best --lzma}"
GOTRAY_SKIP_COMPRESSION_OS="${GOTRAY_SKIP_COMPRESSION_OS:-darwin}"

log() {
  printf '[%s] %s\n' "${SERVICE_NAME}" "$1"
}

should_compress() {
  if [[ "${GOTRAY_ENABLE_COMPRESSION}" != "1" ]]; then
    return 1
  fi

  local target_os
  target_os="${GOOS:-$(go env GOOS)}"
  for disabled in ${GOTRAY_SKIP_COMPRESSION_OS}; do
    if [[ "${target_os}" == "${disabled}" ]]; then
      log "Skipping compression because ${target_os} is in the disabled list"
      return 1
    fi
  done

  if ! command -v "${GOTRAY_COMPRESSION_TOOL}" >/dev/null 2>&1; then
    log "Compression tool ${GOTRAY_COMPRESSION_TOOL} not found; skipping"
    return 1
  fi

  return 0
}

maybe_compress_binary() {
  if ! should_compress; then
    return
  fi

  # shellcheck disable=SC2206
  local -a args=(${GOTRAY_COMPRESSION_ARGS})
  if "${GOTRAY_COMPRESSION_TOOL}" "${args[@]}" "$1"; then
    log "Compressed binary with ${GOTRAY_COMPRESSION_TOOL} ${GOTRAY_COMPRESSION_ARGS}"
  else
    log "Compression attempt failed; leaving binary uncompressed"
  fi
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
(
  cd "${REPO_DIR}"
  go build "${GO_BUILD_FLAGS[@]}" -o "${BIN_PATH}" ./cmd/gotray
)
sudo chmod 0755 "${BIN_PATH}"
maybe_compress_binary "${BIN_PATH}"

log "Preparing development environment file"
sudo mkdir -p "${ENV_ROOT}"
ENV_FILE="${ENV_ROOT}/dev-${TARGET_USER}.env"
if [[ ! -f "${ENV_FILE}" ]]; then
  sudo cp "${REPO_DIR}/.env.example" "${ENV_FILE}"
  sudo chmod 0600 "${ENV_FILE}"
fi
CONFIG_DIR="${CONFIG_ROOT}/${TARGET_USER}"
CONFIG_PATH="${CONFIG_DIR}/config.b64"
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
Environment=GOTRAY_CONFIG_PATH=${CONFIG_ROOT}/%i/config.b64
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
