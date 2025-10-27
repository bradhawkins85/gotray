#!/usr/bin/env bash
set -euo pipefail

APP_NAME="gotray"
REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
OUTPUT_DIR="${GOTRAY_OUTPUT_DIR:-${REPO_DIR}/dist}"
TARGET_OS="${GOOS:-$(go env GOOS)}"
TARGET_ARCH="${GOARCH:-$(go env GOARCH)}"
GO_BUILD_FLAGS=(-trimpath "-ldflags=-s -w")
GOTRAY_ENABLE_COMPRESSION="${GOTRAY_ENABLE_COMPRESSION:-0}"
GOTRAY_COMPRESSION_TOOL="${GOTRAY_COMPRESSION_TOOL:-upx}"
GOTRAY_COMPRESSION_ARGS="${GOTRAY_COMPRESSION_ARGS:---best --lzma}"
GOTRAY_SKIP_COMPRESSION_OS="${GOTRAY_SKIP_COMPRESSION_OS:-darwin}"

log() {
  printf '[%s] %s\n' "${APP_NAME}" "$1"
}

should_compress() {
  if [[ "${GOTRAY_ENABLE_COMPRESSION}" != "1" ]]; then
    return 1
  fi

  for disabled in ${GOTRAY_SKIP_COMPRESSION_OS}; do
    if [[ "${TARGET_OS}" == "${disabled}" ]]; then
      log "Skipping compression because ${TARGET_OS} is in the disabled list"
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

if ! command -v go >/dev/null 2>&1; then
  log "Go toolchain is required"
  exit 1
fi

mkdir -p "${OUTPUT_DIR}"

BINARY_NAME="${APP_NAME}-${TARGET_OS}-${TARGET_ARCH}"
if [[ "${TARGET_OS}" == "windows" ]]; then
  BINARY_NAME+=".exe"
fi
BIN_PATH="${OUTPUT_DIR}/${BINARY_NAME}"

log "Building ${APP_NAME} for ${TARGET_OS}/${TARGET_ARCH}"
(
  cd "${REPO_DIR}"
  GOOS="${TARGET_OS}" GOARCH="${TARGET_ARCH}" go build "${GO_BUILD_FLAGS[@]}" -o "${BIN_PATH}" ./cmd/gotray
)
chmod 0755 "${BIN_PATH}"
maybe_compress_binary "${BIN_PATH}"

log "Release artifact stored at ${BIN_PATH}"
