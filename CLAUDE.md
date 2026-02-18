# monibright

Windows system tray app for monitor brightness control via DDC/CI. Single-monitor for now.

## Dependencies

- `github.com/niluan304/ddcci` — DDC/CI monitor control (no CGO)
- `github.com/energye/systray` — system tray (no CGO)

## Dependency Minimization

Goal: reduce dependency surface by reimplementing thin wrappers ourselves.
`cmd/monibright-reference` is the primary app — all regular work happens there.
`cmd/monibright-minimal` is an experiment that may fall behind; sync it on demand with `/sync-minimal`.

| Dependency | Lines | Used / Exported | Reimplement? |
|---|---|---|---|
| hotkey | 941 | 5 / 60+ (8%) | **Done** — `internal/hotkey/` (~85 lines) |
| ddcci | 3,019 | 4 / 35 (11%) | Maybe — EnumDisplayMonitors + dxva2.dll brightness calls |
| registry | 762 | 8 / 40+ (20%) | No — golang.org/x/sys, basically stdlib |
| systray | 4,100 | 10+ / 40+ (25%) | No — complex Win32 window mgmt |

## Project Layout

```
cmd/monibright-reference/   # primary app — uses external deps (golang.design/x/hotkey)
cmd/monibright-minimal/     # experiment — replaces hotkey dep with internal/hotkey
internal/hotkey/            # reimplementation of golang.design/x/hotkey (~85 lines)
internal/icon/              # embedded tray icon (yellow circle); go generate regenerates .ico
```

## Build

Both variants produce `monibright.exe`. See global `go` skill for ldflags reference.

```bash
# reference variant (default)
go build -o monibright.exe ./cmd/monibright-reference/                                                   # dev
go build -ldflags "-X main.version=0.1.0 -H=windowsgui" -o monibright.exe ./cmd/monibright-reference/   # release

# minimal variant
go build -o monibright.exe ./cmd/monibright-minimal/                                                     # dev
go build -ldflags "-X main.version=0.1.0 -H=windowsgui" -o monibright.exe ./cmd/monibright-minimal/     # release
```

## Future Ideas

- Custom popup slider window anchored to tray icon (like Twinkle Tray) instead of menu presets
- Investigate promoting tray icon to always-visible (not in overflow area) on app start
- Handle non-preset brightness values in UI (e.g. user sets 75% via monitor buttons — no checkmark matches)
- Multi-monitor UX: per-monitor submenus? Separate hotkey sets? Currently sets all monitors together, reads from first
- `gen_icon.go`: accept a PNG/image file as input and convert to multi-size .ico
