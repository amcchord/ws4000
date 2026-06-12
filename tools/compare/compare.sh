#!/usr/bin/env bash
# Idempotent pixel comparison helper against reference screenshots.
# Capture references from https://weatherstar.netbymatt.com and local ws4000 --screenshot output.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
OUTPUT_DIR="${ROOT_DIR}/tools/compare/output"
REFERENCE_DIR="${ROOT_DIR}/testdata/screenshots/reference"
LOCAL_DIR="${ROOT_DIR}/testdata/screenshots/local"

mkdir -p "${OUTPUT_DIR}" "${REFERENCE_DIR}" "${LOCAL_DIR}"

usage() {
  echo "Usage:"
  echo "  $0 capture-local [output-name]   Run ws4000 --screenshot to local dir"
  echo "  $0 compare [reference] [local]   Compare two PNG files with ImageMagick"
  echo "  $0 compare-all                   Compare all matching reference/local pairs"
}

capture_local() {
  local name="${1:-frame.png}"
  local bin="${ROOT_DIR}/ws4000"
  if [[ ! -x "${bin}" ]]; then
    echo "Build ws4000 first: go build -o ws4000 ./cmd/ws4000"
    exit 1
  fi
  if [[ ! -d "${ROOT_DIR}/assets/upstream" ]]; then
    ./tools/vendor-assets.sh
  fi
  "${bin}" --screenshot "${LOCAL_DIR}/${name}" --location "Orlando International Airport, Orlando, FL, USA"
  echo "Saved ${LOCAL_DIR}/${name}"
}

compare_files() {
  local ref="${1:?reference png}"
  local local_file="${2:?local png}"
  if ! command -v magick >/dev/null 2>&1 && ! command -v compare >/dev/null 2>&1; then
    echo "Install ImageMagick: brew install imagemagick"
    exit 1
  fi
  local out="${OUTPUT_DIR}/diff-$(basename "${ref}")"
  if command -v magick >/dev/null 2>&1; then
    magick compare -metric AE "${ref}" "${local_file}" "${out}" 2>"${OUTPUT_DIR}/metric.txt" || true
  else
    compare "${ref}" "${local_file}" "${out}" 2>"${OUTPUT_DIR}/metric.txt" || true
  fi
  echo "Diff image: ${out}"
  echo "Absolute error metric:"
  cat "${OUTPUT_DIR}/metric.txt"
}

compare_all() {
  local found=0
  for ref in "${REFERENCE_DIR}"/*.png; do
    if [[ ! -f "${ref}" ]]; then
      echo "No reference screenshots in ${REFERENCE_DIR}"
      echo "Add PNG captures from weatherstar.netbymatt.com named by display id."
      exit 0
    fi
    base="$(basename "${ref}")"
    local_file="${LOCAL_DIR}/${base}"
    if [[ ! -f "${local_file}" ]]; then
      echo "Missing local capture: ${local_file}"
      continue
    fi
    found=1
    echo "Comparing ${base}..."
    compare_files "${ref}" "${local_file}"
  done
  if [[ "${found}" -eq 0 ]]; then
    echo "No matching pairs found."
  fi
}

cmd="${1:-}"
shift || true

case "${cmd}" in
  capture-local) capture_local "${1:-frame.png}" ;;
  compare) compare_files "${1:?}" "${2:?}" ;;
  compare-all) compare_all ;;
  *) usage ;;
esac
