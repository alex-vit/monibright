# Ideas

- ~~Debug logging — write to a log file for diagnosing DDC/CI failures, monitor enumeration issues~~
- Custom popup slider window anchored to tray icon (like Twinkle Tray) instead of menu presets
- Investigate promoting tray icon to always-visible (not in overflow area) on app start
- Multi-monitor UX: per-monitor submenus? Separate hotkey sets? Currently sets all monitors together, reads from first
- `gen_icon.go`: accept a PNG/image file as input and convert to multi-size .ico
- Self-update via [go-selfupdate](https://github.com/creativeprojects/go-selfupdate) — check on startup, silent download, tray "restart to update" option
- Dynamic tray icon reflecting brightness — bright yellow sun at 100%, nearly eclipsed at 10%
- Input source switch — DDC/CI VCP code 0x60 can switch monitor inputs (HDMI1, DP1, etc.); add tray submenu or hotkey
- Color temperature / "true tone" — f.lux-style warm shift on schedule; DDC/CI VCP 0x14 (color temp) or Windows gamma ramp API

## Bugs

- **Brightness commands silently fail after input switch** — hotkeys and menu selections update the UI checkmark but do not change actual monitor brightness. Started after switching monitor inputs. DDC/CI handle may go stale when the monitor's active input changes. Investigate: does the monitor need re-enumeration? Does `SetBrightness` return an error that we're missing?

## Done

- ~~Handle non-preset brightness values in UI (e.g. user sets 75% via monitor buttons — no checkmark matches)~~
- ~~Refactor `onReady` closures to top-level functions with package-level state~~
