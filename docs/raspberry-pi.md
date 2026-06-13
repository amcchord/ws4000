# Raspberry Pi Setup

ws4000 runs nicely on a Raspberry Pi as a dedicated weather display — windowed
on the desktop, or fullscreen straight on the console with no X11/Wayland via
SDL2's KMSDRM backend.

## Which download do I need?

| Board | OS | Artifact |
|-------|----|----------|
| Pi 4, Pi 5 | Raspberry Pi OS (64-bit) | `linux-arm64` |
| Pi 3, Pi Zero 2 W | Raspberry Pi OS (64-bit) | `linux-arm64` |
| Pi 3, Pi Zero 2 W | Raspberry Pi OS (32-bit) | `linux-armv7` |

Check with `uname -m`: `aarch64` → arm64, `armv7l` → armv7.

A Pi Zero 2 W handles the displays fine; radar processing takes a few extra
seconds at startup. The original Pi Zero / Pi 1 (ARMv6) are not supported.

## Install

```bash
# runtime libraries
sudo apt update
sudo apt install -y libsdl2-2.0-0 libsdl2-ttf-2.0-0 libsdl2-mixer-2.0-0

# download and unpack (pick your artifact from the releases page)
wget https://github.com/amcchord/ws4000/releases/latest/download/ws4000-linux-arm64.tar.gz
tar xzf ws4000-linux-arm64.tar.gz
cd ws4000-linux-arm64

# first run — geo-locates your IP automatically
./ws4000
```

## Configure

```bash
mkdir -p ~/.config/ws4000
cp configs/kiosk.toml ~/.config/ws4000/config.toml
nano ~/.config/ws4000/config.toml   # set your location
```

See [configuration.md](configuration.md) for all options.

## Console kiosk (no desktop required)

SDL2's KMSDRM backend draws directly to the display. From a text console
(not inside X/Wayland):

```bash
SDL_VIDEODRIVER=kmsdrm ./ws4000 --fullscreen
```

If you get a permissions error, add your user to the input/video groups:

```bash
sudo usermod -aG video,input,render $USER
# log out and back in
```

## Start on boot (systemd)

Install to a system path:

```bash
sudo mkdir -p /opt/ws4000
sudo cp -r ./* /opt/ws4000/
```

Create `/etc/systemd/system/ws4000.service`:

```ini
[Unit]
Description=WeatherStar 4000+
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=pi
WorkingDirectory=/opt/ws4000
Environment=SDL_VIDEODRIVER=kmsdrm
Environment=SDL_AUDIODRIVER=alsa
ExecStart=/opt/ws4000/ws4000 --fullscreen --config /home/pi/.config/ws4000/config.toml
Restart=on-failure
RestartSec=30

[Install]
WantedBy=multi-user.target
```

Enable it:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now ws4000
```

The display starts playing automatically once the forecast data loads.

## Desktop autostart (alternative)

If you run the Pi desktop, create `~/.config/autostart/ws4000.desktop`:

```ini
[Desktop Entry]
Type=Application
Name=WeatherStar 4000+
Exec=/opt/ws4000/ws4000 --fullscreen
X-GNOME-Autostart-enabled=true
```

## Tips

- **Performance:** all rendering happens on a 640x480 canvas scaled by the
  GPU, so even a Pi Zero 2 keeps up after the initial data load.
- **Music:** set `volume = 0.3` in the config; over HDMI make sure
  `SDL_AUDIODRIVER=alsa` and HDMI audio is the default sink.
- **Screen blanking:** disable console blanking for kiosks:
  `sudo raspi-config` → Display Options → Screen Blanking → No.
- **Offline testing:** record API responses once, then replay with
  `--fixture` (see configuration.md).
