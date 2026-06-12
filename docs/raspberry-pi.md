# Raspberry Pi Setup

WeatherStar 4000+ native (`ws4000`) runs on Raspberry Pi OS with SDL2. For a kiosk display attached via HDMI, you can run directly on the console without a desktop environment.

## Install Dependencies

```bash
sudo apt update
sudo apt install -y libsdl2-dev libsdl2-ttf-dev libsdl2-mixer-dev pkg-config git
```

## Build

```bash
git clone https://github.com/amcchord/ws4000.git
cd ws4000
./tools/vendor-assets.sh
go build -o ws4000 ./cmd/ws4000
```

Or download a prebuilt `ws4000-linux-armv7` binary from GitHub Releases.

## Configure

```bash
mkdir -p ~/.config/ws4000
cp configs/config.example.toml ~/.config/ws4000/config.toml
# Edit location and display settings
nano ~/.config/ws4000/config.toml
```

## Windowed Mode (Desktop)

```bash
./ws4000 --location "Your City, ST, USA"
```

Press **Space** to start playback, **F** for fullscreen.

## Console Kiosk (KMSDRM)

For a dedicated weather display on the console without X11/Wayland:

```bash
# Install to system path with assets
sudo mkdir -p /usr/share/ws4000
sudo cp ws4000 /usr/local/bin/
sudo cp -r assets/upstream /usr/share/ws4000/assets/

export WS4000_ASSETS=/usr/share/ws4000/assets/upstream
export SDL_VIDEODRIVER=kmsdrm
export SDL_AUDIODRIVER=alsa

/usr/local/bin/ws4000 --fullscreen --location "Your City, ST, USA"
```

## systemd Service

Create `/etc/systemd/system/ws4000.service`:

```ini
[Unit]
Description=WeatherStar 4000+ Native
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=pi
Environment=WS4000_ASSETS=/usr/share/ws4000/assets/upstream
Environment=SDL_VIDEODRIVER=kmsdrm
Environment=SDL_AUDIODRIVER=alsa
ExecStart=/usr/local/bin/ws4000 --fullscreen --config /home/pi/.config/ws4000/config.toml
Restart=on-failure
RestartSec=30

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable ws4000
sudo systemctl start ws4000
```

## Autostart on Boot (Desktop)

Add to `~/.config/autostart/ws4000.desktop`:

```ini
[Desktop Entry]
Type=Application
Name=WeatherStar 4000+
Exec=/home/pi/ws4000/ws4000 --fullscreen
X-GNOME-Autostart-enabled=true
```

## Notes

- Requires network access for NOAA `api.weather.gov` (US locations only)
- Assets must be present at `assets/upstream/` or via `WS4000_ASSETS`
- Use `--fixture ./testdata/fixtures` for offline layout testing
