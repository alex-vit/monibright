# Debug Logging

## Status: Done

## Summary

Debug builds (`-tags debug`) log to `%APPDATA%\monibright\debug.log`. Release builds discard all log output via `io.Discard`.

## What changed

- Debug mode controlled by build tag (`//go:build debug`) setting `const debug = true/false`
- `main()` sets up file logging (append mode) for debug builds, `io.Discard` for release
- "Open log" tray menu item appears only in debug builds (between version title and separator)
- Log lines added at: startup, monitor enumeration, physical monitor init, `SetBrightness` (success and error), `GetBrightness` (success and error), hotkey registration

## Build behavior

| Build | debug | Logging | "Open log" menu |
|---|---|---|---|
| `go build -tags debug .` | `true` | `%APPDATA%\monibright\debug.log` | Yes |
| `go build .` | `false` | Discarded | No |
