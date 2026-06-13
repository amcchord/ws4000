# Building from Source

## Prerequisites

- Go 1.22 or newer
- A C compiler (cgo is required for SDL2 bindings)
- SDL2, SDL2_ttf, and SDL2_mixer development libraries

### macOS

```bash
brew install sdl2 sdl2_ttf sdl2_mixer pkg-config
```

### Debian / Ubuntu / Raspberry Pi OS

```bash
sudo apt update
sudo apt install -y golang-go gcc pkg-config \
  libsdl2-dev libsdl2-ttf-dev libsdl2-mixer-dev
```

(If your distro's Go is older than 1.22, install from https://go.dev/dl/.)

### Fedora

```bash
sudo dnf install golang gcc pkgconf-pkg-config \
  SDL2-devel SDL2_ttf-devel SDL2_mixer-devel
```

### Windows (MSYS2)

Install [MSYS2](https://www.msys2.org/), then in a MINGW64 shell:

```bash
pacman -S mingw-w64-x86_64-go mingw-w64-x86_64-gcc mingw-w64-x86_64-pkg-config \
  mingw-w64-x86_64-SDL2 mingw-w64-x86_64-SDL2_ttf mingw-w64-x86_64-SDL2_mixer
```

## Build

```bash
git clone https://github.com/amcchord/ws4000.git
cd ws4000

# Download fonts, backgrounds, icons, maps, and music from upstream ws4kp.
# Idempotent; re-run any time. Requires python3 and curl.
./tools/vendor-assets.sh

go build -ldflags "-X main.version=$(git describe --tags --always)" -o ws4000 ./cmd/ws4000

./ws4000
```

The binary looks for assets in `./assets/upstream`, next to the executable
(`<exe dir>/assets/upstream`), or at `$WS4000_ASSETS`.

## Running tests

```bash
go test ./...
go vet ./...
```

## Headless screenshot mode

Useful for development and layout comparison without opening a window:

```bash
./ws4000 --screenshot out.png --display current-weather --location "Orlando, FL"
```

## Pixel comparison against the web version

`tools/compare/compare.sh` helps diff local renders against reference
captures from https://weatherstar.netbymatt.com (requires ImageMagick):

```bash
./tools/compare/compare.sh capture-local current-weather.png
./tools/compare/compare.sh compare reference.png local.png
```

## Cross-platform release builds

Releases are built by the GitHub Actions workflow in
`.github/workflows/release.yml`, triggered by pushing a `v*` tag. It produces:

| Artifact | Target hardware |
|----------|-----------------|
| `ws4000-<ver>-macos-arm64.tar.gz` | Apple Silicon Macs |
| `ws4000-<ver>-macos-x64.tar.gz` | Intel Macs |
| `ws4000-<ver>-windows-x64.zip` | Windows 10/11 (SDL DLLs included) |
| `ws4000-<ver>-linux-x64.tar.gz` | x86_64 Linux |
| `ws4000-<ver>-linux-arm64.tar.gz` | Pi 3/4/5/Zero 2 on 64-bit OS, other arm64 |
| `ws4000-<ver>-linux-armv7.tar.gz` | Pi 3/Zero 2 on 32-bit Raspberry Pi OS |

Each archive contains the binary, the full asset directory, config examples,
and docs. Linux binaries link the system SDL2 (`sudo apt install libsdl2-2.0-0
libsdl2-ttf-2.0-0 libsdl2-mixer-2.0-0`); macOS binaries link Homebrew SDL2
(`brew install sdl2 sdl2_ttf sdl2_mixer`); the Windows zip is self-contained.
