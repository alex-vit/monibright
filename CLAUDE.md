# monibright

Windows system tray app for monitor brightness control via DDC/CI.

## Dependencies

- `github.com/niluan304/ddcci` — DDC/CI monitor control (no CGO)
- `github.com/energye/systray` — system tray (no CGO)

## Dependency Minimization

Goal: reduce dependency surface by reimplementing thin wrappers ourselves.

| Dependency | Lines | Used / Exported | Reimplement? |
|---|---|---|---|
| hotkey | 941 | 5 / 60+ (8%) | **Done** — `hotkey.go` (~80 lines) |
| ddcci | 3,019 | 4 / 35 (11%) | Maybe — EnumDisplayMonitors + dxva2.dll brightness calls |
| registry | 762 | 8 / 40+ (20%) | No — golang.org/x/sys, basically stdlib |
| systray | 4,100 | 10+ / 40+ (25%) | No — complex Win32 window mgmt |

## Project Layout

```
main.go              # app entry point, tray menu, brightness control
hotkey.go            # own RegisterHotKey wrapper (inspired by golang.design/x/hotkey)
icon/                # embedded tray icon (yellow circle); go generate ./icon regenerates .ico
notes/               # development notes (YYYY-MM-DD-<slug>.md per task)
```

## Development Notes

Keep notes in `notes/` during feature work. Use `/notes` or say "note that..." to update them. See `.claude/skills/notes/SKILL.md` for full conventions.

## Build

See global `go` skill for ldflags reference.

```bash
go build -o monibright.exe .                                                   # dev
go build -ldflags "-X main.version=0.1.0 -H=windowsgui" -o monibright.exe .   # release
```

## Future Ideas

- Custom popup slider window anchored to tray icon (like Twinkle Tray) instead of menu presets
- Investigate promoting tray icon to always-visible (not in overflow area) on app start
- Handle non-preset brightness values in UI (e.g. user sets 75% via monitor buttons — no checkmark matches)
- Multi-monitor UX: per-monitor submenus? Separate hotkey sets? Currently sets all monitors together, reads from first
- `gen_icon.go`: accept a PNG/image file as input and convert to multi-size .ico
