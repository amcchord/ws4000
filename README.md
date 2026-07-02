# ws4000 — WeatherStar 4000+ Native

A native Go + SDL2 port of [WeatherStar 4000+](https://github.com/netbymatt/ws4kp) —
the nostalgic 90s Weather Channel local forecast experience — built for
Raspberry Pi, desktop, and kiosk displays. No browser required.

Live web version for reference: [weatherstar.netbymatt.com](https://weatherstar.netbymatt.com)

## Features

- All the classic displays: current conditions, local/extended forecast,
  regional observations and forecasts, animated local radar, almanac with moon
  phases, hourly forecast and graph, travel cities, SPC severe weather outlook,
  and hazards
- **Zero-config startup** — geo-locates your public IP and shows your local weather
- Pixel-faithful rendering of the original Star4000 fonts, icons, and layouts
  at native 640x480, scaled to any window or fullscreen
- Single binary + assets folder; runs on Raspberry Pi (including straight on
  the console with no desktop via KMSDRM), macOS, Windows, and Linux
- Background music (bundled WeatherStar-inspired tracks or your own MP3s)
- TOML config file plus CLI flags; bottom ticker with custom messages
- Retro CRT scanline mode

## Quick Start

### Download a release

Grab the archive for your platform from
[Releases](https://github.com/amcchord/ws4000/releases), unpack, and run:

```bash
tar xzf ws4000-*-linux-arm64.tar.gz
cd ws4000-*-linux-arm64
./ws4000
```

| Platform | Archive | Runtime requirement |
|----------|---------|---------------------|
| Raspberry Pi 3/4/5, Zero 2 (64-bit OS) | `linux-arm64` | `sudo apt install libsdl2-2.0-0 libsdl2-ttf-2.0-0 libsdl2-mixer-2.0-0` |
| Raspberry Pi 3 / Zero 2 (32-bit OS) | `linux-armv7` | same as above |
| Linux x86_64 | `linux-x64` | same as above |
| macOS Apple Silicon | `macos-arm64` | `brew install sdl2 sdl2_ttf sdl2_mixer` |
| macOS Intel | build from source ([docs/building.md](docs/building.md)) | `brew install sdl2 sdl2_ttf sdl2_mixer` |
| Windows 10/11 | `windows-x64.zip` | none — SDL DLLs included |

### Or build from source

```bash
git clone https://github.com/amcchord/ws4000.git
cd ws4000
./tools/vendor-assets.sh     # fetch fonts/graphics/music from upstream ws4kp
go build -o ws4000 ./cmd/ws4000
./ws4000
```

Full instructions: [docs/building.md](docs/building.md)

## Usage

With no configuration, ws4000 geo-locates your IP address and starts playing
once the forecast loads. To pin a location:

```bash
./ws4000 --location "Orlando, FL"
./ws4000 --lat 28.431 --lon -81.3076
```

| Key | Action |
|-----|--------|
| Space | Play / pause |
| Right / Left | Next / previous display |
| F | Toggle fullscreen |
| Q or Esc | Quit |

## Configuration

Copy a starter config and edit:

```bash
mkdir -p ~/.config/ws4000
cp configs/config.example.toml ~/.config/ws4000/config.toml
```

- [configs/config.example.toml](configs/config.example.toml) — all options, documented
- [configs/kiosk.toml](configs/kiosk.toml) — fullscreen kiosk
- [configs/minimal.toml](configs/minimal.toml) — just a location
- [configs/metric.toml](configs/metric.toml) — metric units

Everything else: **[docs/configuration.md](docs/configuration.md)**

## Raspberry Pi

ws4000 was built with the Pi in mind — it runs fullscreen on the console
without X11/Wayland and starts at boot with a small systemd unit. The complete
guide, including which artifact fits which Pi model:
**[docs/raspberry-pi.md](docs/raspberry-pi.md)**

## Does it work outside the USA?

No — like the upstream project, this is tightly coupled to NOAA's
api.weather.gov, which covers the USA and its territories only.

## Documentation

- [docs/configuration.md](docs/configuration.md) — config file, CLI flags, env vars
- [docs/raspberry-pi.md](docs/raspberry-pi.md) — Pi install, kiosk mode, systemd
- [docs/building.md](docs/building.md) — building from source, cross-compilation
- [ASSETS.md](ASSETS.md) — asset provenance

## Attribution

This project is a native reimplementation derived from the open source work of:

- **[netbymatt/ws4kp](https://github.com/netbymatt/ws4kp)** — WeatherStar 4000+ web application (MIT)
- **Mike Battaglia** — original WeatherStar drawing code and background graphics
- **[TWCClassics](http://twcclassics.com/)** — Star4000 fonts and weather icon sets
  (fonts by Nick Smith; icons by Charles Abel, Nick Smith, and Malek Masoud)

Graphics, fonts, icons, and default music are vendored from upstream ws4kp
under the MIT license. See [ASSETS.md](ASSETS.md).

## Disclaimer

This application should NOT be used in life-threatening weather situations,
or be relied on to inform the public of such situations. It depends on
internet APIs that can and do go down. See the upstream
[ws4kp disclaimer](https://github.com/netbymatt/ws4kp#disclaimer).

The WeatherSTAR 4000 unit and technology is owned by The Weather Channel.
This is a free, non-profit fan project.

## License

MIT — see [LICENSE](LICENSE).
