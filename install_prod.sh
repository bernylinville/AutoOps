#!/usr/bin/env bash
set -euo pipefail

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

download_file() {
  local output_path="$1"
  shift
  local url=""
  for url in "$@"; do
    if [[ -z "$url" ]]; then
      continue
    fi
    echo "Trying installer source: $url"
    if curl -fsSL "$url" -o "$output_path"; then
      return 0
    fi
  done
  return 1
}

INSTALLER_PATH="$TMP_DIR/install_prod.sh"

if ! download_file "$INSTALLER_PATH" 'https://github.com/bfly123/architec-releases/releases/latest/download/install_prod.sh' 'https://www.architec.top/downloads/latest/install_prod.raw.sh'; then
  echo "Failed to fetch the Architec installer from both GitHub and the website fallback." >&2
  exit 1
fi

chmod +x "$INSTALLER_PATH"

export ARCHITEC_DOWNLOAD_BASE_URL='https://github.com/bfly123/architec-releases/releases/latest/download'
export ARCHITEC_FALLBACK_DOWNLOAD_BASE_URL='https://www.architec.top/downloads/latest'
export ARCHITEC_PRIMARY_INSTALL_SCRIPT_URL='https://github.com/bfly123/architec-releases/releases/latest/download/install_prod.sh'
export ARCHITEC_FALLBACK_INSTALL_SCRIPT_URL='https://www.architec.top/downloads/latest/install_prod.raw.sh'
export ARCHITEC_PRIMARY_CHECKSUMS_URL='https://github.com/bfly123/architec-releases/releases/latest/download/SHA256SUMS.txt'
export ARCHITEC_FALLBACK_CHECKSUMS_URL='https://www.architec.top/downloads/latest/SHA256SUMS.txt'

exec bash "$INSTALLER_PATH" "$@"
