# Auto Color Temperature

Implemented f.lux-style automatic color temperature adjustment.

## Architecture

- `config.go` — JSON config at `%LocalAppData%\MoniBright\config.json` with auto_color_enabled, day/night temps, lat/lon
- `autocolor.go` — location detection, sun schedule API, interpolation math, 60s ticker goroutine

## Location Detection

1. Primary: IP geolocation via `https://iplocation.info` (HTTPS, no API key, ~100 req/day)
2. Fallback: embedded timezone→coordinates table (~45 major cities)
3. Result cached in config so subsequent launches skip the lookup

### Alternatives considered
- **ip-api.com** — HTTP only on free tier, HTTPS requires paid plan
- **ipinfo.io** — good free tier but requires signup for reliable use
- **Browser geolocation API** — not available in a tray app

## Sun Schedule

Fetched from `https://api.sunrisesunset.io/json` (free, no key). Parses sunrise, sunset, civil twilight begin/end. Re-fetches on date change. Falls back to hardcoded 06:00/18:00.

## Transition Math

Sunset/sunrise are the midpoints of their transitions (not the boundaries):
- Evening: ramp starts ~30 min before sunset, ends ~30 min after (at civil twilight end)
- Morning: symmetric around sunrise using civil twilight begin
- Linear interpolation, rounded to nearest 100K to avoid gamma ramp jitter

## Manual Override

Dragging the color temp slider while auto mode is active disengages auto mode, unchecks the menu item, and saves the config. Re-engaging requires clicking "Auto color temp" again.

## Vendor Removal

Removed `vendor/` directory (2.2MB). No build scripts referenced `-mod=vendor`. Module cache used instead.
