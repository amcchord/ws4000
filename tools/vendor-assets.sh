#!/usr/bin/env bash
# Idempotent asset vendor script for ws4000.
# Downloads graphics, fonts, music, and data from upstream ws4kp.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
ASSETS_DIR="${ROOT_DIR}/assets/upstream"
UPSTREAM_REPO="https://github.com/netbymatt/ws4kp"
UPSTREAM_COMMIT="${UPSTREAM_COMMIT:-$(curl -fsSL "https://api.github.com/repos/netbymatt/ws4kp/commits/main" | python3 -c "import sys,json; print(json.load(sys.stdin)['sha'])")}"

echo "Vendoring assets from ${UPSTREAM_REPO}@${UPSTREAM_COMMIT:0:8}"

mkdir -p "${ASSETS_DIR}"

echo "${UPSTREAM_COMMIT}" > "${ASSETS_DIR}/UPSTREAM_COMMIT"

download() {
  local src_path="$1"
  local dest_path="$2"
  local url="${UPSTREAM_REPO}/raw/${UPSTREAM_COMMIT}/${src_path}"

  mkdir -p "$(dirname "${dest_path}")"

  if [[ -f "${dest_path}" ]]; then
    return 0
  fi

  echo "  downloading ${src_path}"
  curl -fsSL "${url}" -o "${dest_path}"
}

download_tree() {
  local src_dir="$1"
  local dest_dir="$2"
  local api_url="https://api.github.com/repos/netbymatt/ws4kp/contents/${src_dir}?ref=${UPSTREAM_COMMIT}"

  mkdir -p "${dest_dir}"

  python3 - "${api_url}" "${UPSTREAM_REPO}" "${UPSTREAM_COMMIT}" "${src_dir}" "${dest_dir}" <<'PY'
import json, sys, urllib.request, os
from urllib.parse import quote

api_url, repo, commit, src_dir, dest_dir = sys.argv[1:6]

def fetch_json(url):
    req = urllib.request.Request(url, headers={"User-Agent": "ws4000-vendor"})
    with urllib.request.urlopen(req) as resp:
        return json.load(resp)

def raw_url(path):
    encoded = "/".join(quote(part, safe="") for part in path.split("/"))
    return f"{repo}/raw/{commit}/{encoded}"

def walk(path, local_base):
    items = fetch_json(f"https://api.github.com/repos/netbymatt/ws4kp/contents/{path}?ref={commit}")
    if not isinstance(items, list):
        items = [items]
    for item in items:
        local_path = os.path.join(local_base, item["name"])
        if item["type"] == "dir":
            os.makedirs(local_path, exist_ok=True)
            walk(item["path"], local_path)
        else:
            if os.path.exists(local_path):
                continue
            url = raw_url(item["path"])
            print(f"  downloading {item['path']}")
            req = urllib.request.Request(url, headers={"User-Agent": "ws4000-vendor"})
            with urllib.request.urlopen(req) as resp:
                data = resp.read()
            os.makedirs(os.path.dirname(local_path), exist_ok=True)
            with open(local_path, "wb") as f:
                f.write(data)

walk(src_dir, dest_dir)
PY
}

echo "Backgrounds..."
download_tree "server/images/backgrounds" "${ASSETS_DIR}/backgrounds"

echo "Icons..."
download_tree "server/images/icons" "${ASSETS_DIR}/icons"

echo "Maps..."
download_tree "server/images/maps" "${ASSETS_DIR}/maps"

echo "Logos..."
download_tree "server/images/logos" "${ASSETS_DIR}/logos"

echo "Fonts..."
download_tree "server/fonts" "${ASSETS_DIR}/fonts"

echo "Music..."
download_tree "server/music/default" "${ASSETS_DIR}/music"

echo "Static data..."
download "datagenerators/output/regionalcities.json" "${ASSETS_DIR}/data/regionalcities.json"
download "datagenerators/output/travelcities.json" "${ASSETS_DIR}/data/travelcities.json"
download "datagenerators/output/stations.json" "${ASSETS_DIR}/data/stations.json"

echo "Converting WOFF fonts to TTF..."
FONTS_DIR="${ASSETS_DIR}/fonts"
TTF_DIR="${FONTS_DIR}/ttf"
VENV_DIR="${SCRIPT_DIR}/.venv"
mkdir -p "${TTF_DIR}"

if [[ ! -d "${VENV_DIR}" ]]; then
  python3 -m venv "${VENV_DIR}"
  "${VENV_DIR}/bin/pip" install --quiet fonttools
fi

"${VENV_DIR}/bin/python" - "${FONTS_DIR}" "${TTF_DIR}" <<'PY'
import glob, os, sys
from fontTools.ttLib import TTFont

fonts_dir, ttf_dir = sys.argv[1:3]

for woff in glob.glob(os.path.join(fonts_dir, "*.woff")):
    base = os.path.splitext(os.path.basename(woff))[0]
    ttf = os.path.join(ttf_dir, base + ".ttf")
    if os.path.exists(ttf):
        continue
    font = TTFont(woff)
    font.save(ttf)
    print(f"  converted {woff} -> {ttf}")
PY

echo "Done. Assets in ${ASSETS_DIR}"
