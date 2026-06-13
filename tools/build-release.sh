#!/usr/bin/env bash
# Build release artifacts for all supported targets. Idempotent.
#
# Local requirements:
#   - macOS with Homebrew SDL2 (for the native macos-arm64 build)
#   - Docker with binfmt/qemu (for linux x64/arm64/armv7 and windows cross)
#
# Usage: tools/build-release.sh v0.1.0 [target ...]
#   targets default to: macos-arm64 linux-x64 linux-arm64 linux-armv7 windows-x64

set -euo pipefail

VERSION="${1:?version (e.g. v0.1.0)}"
shift || true
TARGETS=("$@")
if [[ ${#TARGETS[@]} -eq 0 ]]; then
  TARGETS=(macos-arm64 linux-x64 linux-arm64 linux-armv7 windows-x64)
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
mkdir -p "${DIST_DIR}"

if [[ ! -d "${ROOT_DIR}/assets/upstream" ]]; then
  "${ROOT_DIR}/tools/vendor-assets.sh"
fi

GOFLAGS_LD="-X main.version=${VERSION}"

# SDL mingw development package versions for the Windows cross build
SDL2_VER="2.30.11"
SDL2_TTF_VER="2.24.0"
SDL2_MIXER_VER="2.8.1"

docker_linux_build() {
  local platform="$1"  # e.g. linux/amd64
  local target="$2"    # e.g. linux-x64
  echo "=== ${target} (docker ${platform}) ==="
  docker run --rm --platform "${platform}" \
    -v "${ROOT_DIR}:/src" -w /src \
    -v ws4000-gomod:/go/pkg/mod \
    -v "ws4000-aptcache-${target}:/var/cache/apt" \
    golang:1.26-bookworm bash -c "
      set -e
      apt-get update -qq
      apt-get install -y -qq pkg-config libsdl2-dev libsdl2-ttf-dev libsdl2-mixer-dev > /dev/null
      go build -ldflags '${GOFLAGS_LD}' -o /src/dist/ws4000-${target} ./cmd/ws4000
    "
  "${ROOT_DIR}/tools/package-release.sh" "${VERSION}" "${target}" "${DIST_DIR}/ws4000-${target}"
}

docker_windows_build() {
  echo "=== windows-x64 (docker mingw cross) ==="
  docker run --rm --platform linux/amd64 \
    -v "${ROOT_DIR}:/src" -w /src \
    -v ws4000-gomod:/go/pkg/mod \
    golang:1.26-bookworm bash -c "
      set -e
      apt-get update -qq
      apt-get install -y -qq gcc-mingw-w64-x86-64 curl > /dev/null

      mkdir -p /opt/sdl2-mingw && cd /opt/sdl2-mingw
      curl -fsSL -o sdl2.tar.gz https://github.com/libsdl-org/SDL/releases/download/release-${SDL2_VER}/SDL2-devel-${SDL2_VER}-mingw.tar.gz
      curl -fsSL -o ttf.tar.gz  https://github.com/libsdl-org/SDL_ttf/releases/download/release-${SDL2_TTF_VER}/SDL2_ttf-devel-${SDL2_TTF_VER}-mingw.tar.gz
      curl -fsSL -o mix.tar.gz  https://github.com/libsdl-org/SDL_mixer/releases/download/release-${SDL2_MIXER_VER}/SDL2_mixer-devel-${SDL2_MIXER_VER}-mingw.tar.gz
      tar xzf sdl2.tar.gz && tar xzf ttf.tar.gz && tar xzf mix.tar.gz
      PREFIX=/opt/sdl2-mingw/prefix
      mkdir -p \${PREFIX}
      cp -r SDL2-${SDL2_VER}/x86_64-w64-mingw32/* \${PREFIX}/
      cp -r SDL2_ttf-${SDL2_TTF_VER}/x86_64-w64-mingw32/* \${PREFIX}/
      cp -r SDL2_mixer-${SDL2_MIXER_VER}/x86_64-w64-mingw32/* \${PREFIX}/

      cd /src
      export GOOS=windows GOARCH=amd64 CGO_ENABLED=1
      export CC=x86_64-w64-mingw32-gcc
      export PKG_CONFIG_PATH=\${PREFIX}/lib/pkgconfig
      export CGO_CFLAGS=\"-I\${PREFIX}/include -I\${PREFIX}/include/SDL2\"
      export CGO_LDFLAGS=\"-L\${PREFIX}/lib\"
      go build -ldflags '${GOFLAGS_LD}' -tags static -o /src/dist/ws4000-windows-x64.exe ./cmd/ws4000 || \
      go build -ldflags '${GOFLAGS_LD}' -o /src/dist/ws4000-windows-x64.exe ./cmd/ws4000

      mkdir -p /src/dist/windows-dlls
      cp \${PREFIX}/bin/*.dll /src/dist/windows-dlls/
    "

  # package zip with DLLs
  local name="ws4000-${VERSION}-windows-x64"
  local stage="${DIST_DIR}/${name}"
  rm -rf "${stage}"
  mkdir -p "${stage}/assets" "${stage}/configs" "${stage}/docs"
  cp "${DIST_DIR}/ws4000-windows-x64.exe" "${stage}/ws4000.exe"
  cp "${DIST_DIR}/windows-dlls/"*.dll "${stage}/"
  cp -R "${ROOT_DIR}/assets/upstream" "${stage}/assets/upstream"
  cp "${ROOT_DIR}/configs/"*.toml "${stage}/configs/"
  cp "${ROOT_DIR}/docs/"*.md "${stage}/docs/"
  cp "${ROOT_DIR}/README.md" "${ROOT_DIR}/LICENSE" "${ROOT_DIR}/ASSETS.md" "${stage}/"
  (cd "${DIST_DIR}" && rm -f "${name}.zip" && zip -qr "${name}.zip" "${name}")
  rm -rf "${stage}"
  echo "created dist/${name}.zip"
}

for target in "${TARGETS[@]}"; do
  case "${target}" in
    macos-arm64)
      echo "=== macos-arm64 (native) ==="
      (cd "${ROOT_DIR}" && go build -ldflags "${GOFLAGS_LD}" -o "${DIST_DIR}/ws4000-macos-arm64" ./cmd/ws4000)
      "${ROOT_DIR}/tools/package-release.sh" "${VERSION}" macos-arm64 "${DIST_DIR}/ws4000-macos-arm64"
      ;;
    linux-x64)   docker_linux_build linux/amd64 linux-x64 ;;
    linux-arm64) docker_linux_build linux/arm64 linux-arm64 ;;
    linux-armv7) docker_linux_build linux/arm/v7 linux-armv7 ;;
    windows-x64) docker_windows_build ;;
    *) echo "unknown target ${target}" >&2; exit 1 ;;
  esac
done

echo
echo "Artifacts:"
ls -lh "${DIST_DIR}"/*.tar.gz "${DIST_DIR}"/*.zip 2>/dev/null || true
