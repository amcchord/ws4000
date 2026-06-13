# Configuration

ws4000 reads a TOML config file, then applies any command-line flags on top
(flags always win). Without any configuration at all, it geo-locates your
public IP and shows weather for your area.

## Config file location

| Platform | Default path |
|----------|--------------|
| Linux / Raspberry Pi | `~/.config/ws4000/config.toml` |
| macOS | `~/.config/ws4000/config.toml` |
| Windows | `%USERPROFILE%\.config\ws4000\config.toml` |

Use a different path with `--config /path/to/config.toml`.

Starter examples live in [`configs/`](../configs):

- [`config.example.toml`](../configs/config.example.toml) — every option, documented
- [`kiosk.toml`](../configs/kiosk.toml) — fullscreen kiosk for a dedicated display
- [`minimal.toml`](../configs/minimal.toml) — smallest useful config
- [`metric.toml`](../configs/metric.toml) — metric units

## Options

### Location

```toml
# One of three ways to pick a location:

# 1. Automatic (default): geo-locate your public IP address
location = "auto"

# 2. Free-text query, geocoded via ArcGIS (same service the web version uses)
# location = "Orlando International Airport, Orlando, FL, USA"
# location = "30301"            # ZIP codes work
# location = "Boise, ID"

# 3. Exact coordinates (overrides location)
# latitude = 28.431
# longitude = -81.3076
```

Weather data comes from NOAA's api.weather.gov, which only covers the USA
and its territories.

### Display

```toml
units = "us"          # "us" or "metric"
fullscreen = false    # start fullscreen (F key toggles at runtime)
scale = 2             # window size multiplier: 2 = 1280x960
speed = 1.0           # playback speed: 0.5 (very fast) .. 1.5 (very slow)
scanlines = false     # retro CRT scanline overlay
```

### Selected displays

Matches the checkboxes on the web version. Defaults shown:

```toml
[displays]
hazards = true              # severe weather warnings (auto-hidden when none)
current-weather = true      # current conditions
latest-observations = true  # nearby station observations
hourly = false              # hourly forecast table
hourly-graph = true         # 36-hour temperature/precip graph
travel = false              # national travel cities forecast
regional-forecast = true    # regional map with observations + forecasts
local-forecast = true       # text forecast
extended-forecast = true    # 3-day outlook cards
almanac = false             # sunrise/sunset and moon phases
spc-outlook = true          # storm prediction center outlook (auto-hidden when no risk)
radar = true                # local radar loop
```

The equivalent CLI flags are repeatable/comma-separated:

```bash
ws4000 --enable almanac,travel --disable radar
```

### Music

```toml
volume = 0.0                   # 0.0 (muted, default) to 1.0
# music_dir = "/path/to/mp3s"  # your own tracks; defaults to the built-in ones
```

Background music starts with playback. The bundled tracks are
WeatherStar-inspired pieces from the upstream project (see ASSETS.md).

### Misc

```toml
# custom_scroll = "Welcome to WeatherStar|Thanks for watching"  # | splits messages
refresh_ms = 600000   # weather refresh interval (10 minutes)
# fixture_dir = "./fixtures"  # replay recorded API responses (development)
```

## CLI reference

```text
ws4000 [flags]

--location string    Location query, or "auto" for geo-IP (default auto)
--lat float          Latitude  (use with --lon)
--lon float          Longitude (use with --lat)
--config string      Config file path
--units string       us or metric
--fullscreen         Start fullscreen
--scale int          Window scale factor (default 2)
--speed float        Playback speed multiplier (default 1.0)
--enable string      Comma-separated display ids to enable
--disable string     Comma-separated display ids to disable
--music-dir string   Directory of MP3 files
--volume float       Music volume 0.0-1.0
--scanlines          CRT scanline overlay
--screenshot string  Render one frame to a PNG and exit (headless)
--display string     Which display id to capture with --screenshot
--fixture string     Use recorded API fixtures from a directory
--list-displays      List display ids and exit
--version            Print version and exit
```

## Keyboard controls

| Key | Action |
|-----|--------|
| Space | Play / pause |
| Right / Left | Next / previous display |
| F | Toggle fullscreen |
| Q or Esc | Quit |

## Environment variables

| Variable | Purpose |
|----------|---------|
| `WS4000_ASSETS` | Path to the asset directory (default: `assets/upstream` next to the binary or working directory) |
| `SDL_VIDEODRIVER` | Force an SDL video backend (e.g. `kmsdrm` for console kiosks on Raspberry Pi) |
| `SDL_AUDIODRIVER` | Force an SDL audio backend (e.g. `alsa`) |
