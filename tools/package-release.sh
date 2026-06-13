#!/usr/bin/env bash
# Package a built binary + assets into a release archive. Idempotent.
# Usage: tools/package-release.sh <version> <target> <binary>
#   e.g. tools/package-release.sh v0.1.0 linux-arm64 ws4000

set -euo pipefail

VERSION="${1:?version (e.g. v0.1.0)}"
TARGET="${2:?target (e.g. linux-arm64)}"
BINARY="${3:?path to built binary}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
NAME="ws4000-${VERSION}-${TARGET}"
STAGE="${DIST_DIR}/${NAME}"

if [[ ! -f "${ROOT_DIR}/${BINARY}" && ! -f "${BINARY}" ]]; then
  echo "binary not found: ${BINARY}" >&2
  exit 1
fi
if [[ ! -d "${ROOT_DIR}/assets/upstream" ]]; then
  echo "assets missing; run tools/vendor-assets.sh first" >&2
  exit 1
fi

rm -rf "${STAGE}"
mkdir -p "${STAGE}/assets" "${STAGE}/configs" "${STAGE}/docs"

if [[ -f "${ROOT_DIR}/${BINARY}" ]]; then
  cp "${ROOT_DIR}/${BINARY}" "${STAGE}/"
else
  cp "${BINARY}" "${STAGE}/"
fi

cp -R "${ROOT_DIR}/assets/upstream" "${STAGE}/assets/upstream"
# exclude the python venv if it leaked into the tree
rm -rf "${STAGE}/assets/upstream/.venv"

cp "${ROOT_DIR}/configs/"*.toml "${STAGE}/configs/"
cp "${ROOT_DIR}/docs/"*.md "${STAGE}/docs/"
cp "${ROOT_DIR}/README.md" "${ROOT_DIR}/LICENSE" "${ROOT_DIR}/ASSETS.md" "${STAGE}/"

# normalize the binary name inside the archive
if [[ "${TARGET}" == windows-* ]]; then
  if [[ ! -f "${STAGE}/ws4000.exe" ]]; then
    mv "${STAGE}/$(basename "${BINARY}")" "${STAGE}/ws4000.exe"
  fi
  # bundle the SDL runtime DLLs next to the exe so the zip is portable (MSYS2 CI)
  if command -v ldd >/dev/null 2>&1; then
    ldd "${STAGE}/ws4000.exe" | awk '/mingw64/ {print $3}' | while read -r dll; do
      cp -f "${dll}" "${STAGE}/" || true
    done
  fi
else
  if [[ ! -f "${STAGE}/ws4000" ]]; then
    mv "${STAGE}/$(basename "${BINARY}")" "${STAGE}/ws4000"
  fi
  chmod +x "${STAGE}/ws4000"
fi

mkdir -p "${DIST_DIR}"
pushd "${DIST_DIR}" >/dev/null
if [[ "${TARGET}" == windows-* ]]; then
  rm -f "${NAME}.zip"
  zip -qr "${NAME}.zip" "${NAME}"
  echo "created dist/${NAME}.zip"
else
  rm -f "${NAME}.tar.gz"
  tar czf "${NAME}.tar.gz" "${NAME}"
  echo "created dist/${NAME}.tar.gz"
fi
popd >/dev/null

rm -rf "${STAGE}"
