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
config.go            # JSON config load/save (%LocalAppData%\MoniBright\config.json)
autocolor.go         # auto color temp: location, sun schedule, interpolation, ticker
gamma.go             # gamma ramp API: kelvinToRGB, applyColorTemp, save/restore
slider.go            # Win32 popup slider (brightness + color temp trackbars)
hotkey.go            # own RegisterHotKey wrapper (inspired by golang.design/x/hotkey)
update.go            # self-update via GitHub releases
debug.go / release.go # build-tag controlled const debug bool
icon/                # embedded tray icon (yellow circle); go generate ./icon regenerates .ico
notes/               # development notes (YYYY-MM-DD-<slug>.md per task)
```

## Development Notes

Keep notes in `notes/` during feature work. Use `/notes` or say "note that..." to update them.

## Build

```bash
go build -ldflags "-H=windowsgui" -o monibright.exe .               # dev (project root)
pwsh ./scripts/build-windows-release.ps1 -Version vX.Y.Z            # release → out/monibright.exe + out/monibright-setup.exe
```

Logs are written to `%LocalAppData%\MoniBright\log.txt` in all builds.

## Debugging

Fullscreen DirectX/Vulkan games often suppress or hook the Win key and other modifiers. When investigating hotkey or keyboard issues, rule out other running apps (especially games) before suspecting our code. See `notes/2026-02-23-win-key-stuck.md`.

## Ideas

See [`notes/00-ideas.md`](notes/00-ideas.md).
