# Future Ideas

- Custom popup slider window anchored to tray icon (like Twinkle Tray) instead of menu presets
- Investigate promoting tray icon to always-visible (not in overflow area) on app start
- Multi-monitor UX: per-monitor submenus? Separate hotkey sets? Currently sets all monitors together, reads from first
- `gen_icon.go`: accept a PNG/image file as input and convert to multi-size .ico
- Self-update via [go-selfupdate](https://github.com/creativeprojects/go-selfupdate) — check on startup, silent download, tray "restart to update" option

## Done

- ~~Handle non-preset brightness values in UI (e.g. user sets 75% via monitor buttons — no checkmark matches)~~
- ~~Refactor `onReady` closures to top-level functions with package-level state~~
