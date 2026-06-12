# ws4000

Native Go + SDL2 port of [WeatherStar 4000+](https://github.com/netbymatt/ws4kp) — the nostalgic Weather Channel local forecast experience, built for Raspberry Pi, desktop, and kiosk displays.

Live reference: [weatherstar.netbymatt.com](https://weatherstar.netbymatt.com)

## Attribution

This project is a native reimplementation inspired by and derived from the open source work of:

- **[netbymatt/ws4kp](https://github.com/netbymatt/ws4kp)** — WeatherStar 4000+ web application (MIT)
- **Mike Battaglia** — original WeatherStar drawing code and background graphics
- **[TWCClassics](http://twcclassics.com/)** — Star4000 fonts and weather icon sets

Graphics, fonts, icons, and default music are vendored from upstream ws4kp with permission under the MIT license. See [ASSETS.md](ASSETS.md) for provenance.

## Features

- All 13 forecast displays (current conditions, local/extended forecast, regional, radar, almanac, travel, hourly, hourly graph, SPC outlook, latest observations, hazards)
- SDL2 rendering at native 640×480 with windowed and fullscreen modes
- Cross-platform: Linux (including Raspberry Pi), macOS, Windows
- TOML configuration file + CLI flags
- Background music via SDL_mixer
- NOAA weather.gov API integration with disk caching
- Pixel-accurate layout matching the web version

## Quick Start

### Prerequisites

- Go 1.22+
- SDL2, SDL2_ttf, SDL2_mixer development libraries

**macOS (Homebrew):**
```bash
brew install sdl2 sdl2_ttf sdl2_mixer
```

**Debian/Ubuntu/Raspberry Pi OS:**
```bash
sudo apt install libsdl2-dev libsdl2-ttf-dev libsdl2-mixer-dev
```

### Build

```bash
# Download assets from upstream ws4kp (idempotent)
./tools/vendor-assets.sh

# Build
go build -o ws4000 ./cmd/ws4000

# Run
./ws4000 --location "Orlando International Airport, Orlando, FL, USA"
```

### Configuration

Copy the example config:

```bash
mkdir -p ~/.config/ws4000
cp configs/config.example.toml ~/.config/ws4000/config.toml
```

See [configs/config.example.toml](configs/config.example.toml) for all options.

### CLI

```
ws4000 [flags]

  --location string       Location query (geocoded via ArcGIS)
  --lat float             Latitude (use with --lon)
  --lon float             Longitude (use with --lat)
  --config string         Config file path (default ~/.config/ws4000/config.toml)
  --units string          us or metric (default us)
  --fullscreen            Start in fullscreen mode
  --scale int             Integer scale factor (default auto)
  --speed float           Playback speed multiplier (default 1.0)
  --enable string         Enable a display (repeatable)
  --disable string        Disable a display (repeatable)
  --music-dir string      Directory of MP3 files for background music
  --volume float          Music volume 0.0-1.0 (default 0.0)
  --scanlines             Enable CRT scanline overlay
  --fixture string        Use recorded API fixtures from directory
  --screenshot string     Save screenshot to path and exit
  --list-displays         List available displays and exit
```

### Keyboard Controls

| Key | Action |
|-----|--------|
| Space | Play / Pause |
| Right | Next display |
| Left | Previous display |
| F / Ctrl+F | Toggle fullscreen |
| Q / Esc | Quit |

## Raspberry Pi

See [docs/raspberry-pi.md](docs/raspberry-pi.md) for KMSDRM console kiosk setup and systemd service.

## Disclaimer

This application should NOT be used in life-threatening weather situations. Data comes from the public NOAA weather.gov API over the internet and is not suitable for mission-critical use. See the upstream [ws4kp disclaimer](https://github.com/netbymatt/ws4kp#disclaimer) for full details.

The WeatherSTAR 4000 unit and technology is owned by The Weather Channel. This is a free, non-profit fan project.

## License

MIT — see [LICENSE](LICENSE).
