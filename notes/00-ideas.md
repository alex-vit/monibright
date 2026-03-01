# Ideas

- ~~Debug logging — write to a log file for diagnosing DDC/CI failures, monitor enumeration issues~~
- ~~Custom popup slider window anchored to tray icon (like Twinkle Tray) instead of menu presets~~
- Volume-style flyout redesign — dark panel, monitor name, expand to show multi-monitor list. See [`notes/2026-03-01-flyout-redesign-spec.md`](2026-03-01-flyout-redesign-spec.md)
- Investigate promoting tray icon to always-visible (not in overflow area) on app start
- Multi-monitor UX: per-monitor submenus? Separate hotkey sets? Currently sets all monitors together, reads from first
- `gen_icon.go`: accept a PNG/image file as input and convert to multi-size .ico
- ~~Self-update via GitHub releases — check on startup, silent download, tray "restart to update" option~~
- ~~Dynamic tray icon reflecting brightness — bright yellow sun at 100%, nearly eclipsed at 10%~~
- Customizable color temp lower bound — e.g. match desk lamp at 4000K instead of hardcoded 3500K. Slider min + night temp could be user-configurable.
- Auto brightness — schedule-based like auto color temp. E.g. desk lamp evening ~30%, sunny day 100%, overcast 80%. Could reuse the same sun schedule infrastructure. Needs configurable day/night brightness levels.
- Embed an app icon via Windows manifest so MoniBright has a proper icon in Start Menu / desktop shortcuts (currently shows generic exe icon)
- Input source switch — DDC/CI VCP code 0x60 can switch monitor inputs (HDMI1, DP1, etc.); add tray submenu or hotkey
- ~~Color temperature / "true tone" — f.lux-style warm shift on schedule; DDC/CI VCP 0x14 (color temp) or Windows gamma ramp API~~

## Bugs

- **Brightness commands silently fail after input switch** — hotkeys and menu selections update the UI checkmark but do not change actual monitor brightness. Started after switching monitor inputs. DDC/CI handle may go stale when the monitor's active input changes. Investigate: does the monitor need re-enumeration? Does `SetBrightness` return an error that we're missing?

## Done

- ~~Handle non-preset brightness values in UI (e.g. user sets 75% via monitor buttons — no checkmark matches)~~
- ~~Refactor `onReady` closures to top-level functions with package-level state~~
- ~~Self-update — DIY stdlib implementation, zero new deps~~
- Code signing — sign the exe/installer to eliminate the SmartScreen "Run anyway" scare dialog (requires a code signing certificate; EV certs get instant reputation, standard certs build it over time)
