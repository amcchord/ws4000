# Asset Provenance

Graphics, fonts, icons, music, and static data files in `assets/upstream/` are vendored from the upstream WeatherStar 4000+ project:

- **Repository:** https://github.com/netbymatt/ws4kp
- **License:** MIT
- **Pinned commit:** see `UPSTREAM_COMMIT` in this directory

## Contents

| Path | Upstream Source | Notes |
|------|-----------------|-------|
| `backgrounds/` | `server/images/backgrounds/` | Display background PNGs |
| `icons/` | `server/images/icons/` | Animated GIF weather icons |
| `maps/` | `server/images/maps/` | Radar basemap and tile images |
| `fonts/` | `server/fonts/` | Star4000 WOFF fonts (converted to TTF at build time) |
| `music/` | `server/music/default/` | AI-generated background music tracks |
| `data/` | `datagenerators/output/` | Regional cities, travel cities, stations JSON |

## Font and Icon Credits

- Star4000 fonts originally by Nick Smith — [TWCClassics](http://twcclassics.com/downloads/fonts.html)
- Weather icons by Charles Abel, Nick Smith, and Malek Masoud — [TWCClassics](http://twcclassics.com/downloads/icons.html)
- Background graphics by Mike Battaglia and contributors to ws4kp

## Regenerating Assets

```bash
./tools/vendor-assets.sh
```

This script is idempotent and safe to re-run.
