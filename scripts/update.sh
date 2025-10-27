#!/usr/bin/env bash
set -euo pipefail

APP_NAME="gotray"
INSTALL_DIR="/opt/${APP_NAME}"
BIN_PATH="${INSTALL_DIR}/${APP_NAME}"
REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TARGET_USER="${GOTRAY_INSTALL_USER:-${SUDO_USER:-${USER}}}"
GO_BUILD_FLAGS=(-trimpath)
GOTRAY_ENABLE_COMPRESSION="${GOTRAY_ENABLE_COMPRESSION:-0}"
GOTRAY_COMPRESSION_TOOL="${GOTRAY_COMPRESSION_TOOL:-upx}"
GOTRAY_COMPRESSION_ARGS="${GOTRAY_COMPRESSION_ARGS:---best --lzma}"
GOTRAY_SKIP_COMPRESSION_OS="${GOTRAY_SKIP_COMPRESSION_OS:-darwin}"
STRIP_SYMBOLS="${GOTRAY_STRIP_SYMBOLS:-1}"

configure_ldflags() {
  local -a ldflags
  if [[ "${STRIP_SYMBOLS}" == "1" ]]; then
    ldflags+=(-s -w)
  fi
  if [[ -n "${GOTRAY_EXTRA_LDFLAGS:-}" ]]; then
    # shellcheck disable=SC2206
    ldflags+=(${GOTRAY_EXTRA_LDFLAGS})
  fi
  if [[ ${#ldflags[@]} -eq 0 ]]; then
    return
  fi

  local joined
  joined="$(printf '%s ' "${ldflags[@]}")"
  joined="${joined% }"
  GO_BUILD_FLAGS+=("-ldflags=${joined}")
}

log() {
  printf '[%s] %s\n' "${APP_NAME}" "$1"
}

if [[ "${STRIP_SYMBOLS}" == "0" ]]; then
  log "Symbol stripping disabled (GOTRAY_STRIP_SYMBOLS=0)"
fi

configure_ldflags

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

log "Pulling latest source"
if [[ -d "${REPO_DIR}/.git" ]]; then
  (cd "${REPO_DIR}" && git pull --rebase)
fi

log "Building updated binary"
(
  cd "${REPO_DIR}"
  go build "${GO_BUILD_FLAGS[@]}" -o "${BIN_PATH}" ./cmd/gotray
)
sudo chmod 0755 "${BIN_PATH}"
maybe_compress_binary "${BIN_PATH}"

log "Restarting user service gotray@${TARGET_USER}.service"
sudo systemctl restart "${APP_NAME}@${TARGET_USER}.service"

log "Update complete"
