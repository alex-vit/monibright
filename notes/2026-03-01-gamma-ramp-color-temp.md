# Color Temperature via Windows Gamma Ramp API

Date: 2026-03-01

## Background

DDC/CI RGB gain buttons (VCP 0x16/0x18/0x1A) looked bad — coarse 0–100 steps, monitor-dependent, slow writes. Exploring the f.lux/redshift approach instead: **Windows gamma ramp API** (`SetDeviceGammaRamp` / `GetDeviceGammaRamp` from gdi32.dll).

## How gamma ramps work

- `SetDeviceGammaRamp(HDC, *[3][256]uint16)` — three 256-entry arrays of WORD (R, G, B)
- Each entry maps an 8-bit input value (0–255) to a 16-bit output (0–65535)
- Linear/identity ramp: `ramp[i] = i * 257` (or `i << 8 | i`)
- To warm the display: scale green ramp down slightly, blue ramp down more
- Software-only — works on any display, no DDC/CI needed

## Kelvin → RGB conversion

Tanner Helland algorithm (used by f.lux, redshift):
- Input: temperature in Kelvin (1000–40000K range, useful range ~2700–6500K)
- Output: RGB multipliers (0–255 scale)
- At 6500K → (255, 255, 255) = no adjustment (neutral daylight)
- At 4000K → (255, ~200, ~140) = warm
- At 2700K → (255, ~150, ~50) = very warm (incandescent)

Normalize each channel to 0.0–1.0 and multiply against the identity ramp:
```
ramp[ch][i] = uint16(float64(i * 257) * multiplier[ch])
```

## Existing code to build on

- `slider.go` — already loads gdi32.dll, has the flyout window infrastructure
- `hotkey.go` — can add Win+PgUp/PgDn or similar for color temp adjustment
- `main.go` — app lifecycle; need to save/restore ramps on startup/exit

## Gotchas

- **Must restore original ramps on exit.** Otherwise display stays warm after MoniBright closes. Save baseline from `GetDeviceGammaRamp` at startup.
- **Silent failures.** Windows heuristics may reject ramps that would make the screen unreadable. Log success/failure.
- **Conflicts with Night Light / f.lux.** Both write to the same gamma ramp. Last writer wins. Acceptable — user shouldn't run both.
- **Multi-monitor.** Need `GetDC(NULL)` for primary or enumerate per-monitor DCs. Start with primary display; extend later.
- **Sleep/wake.** Gamma ramps may reset on display reconnect. Re-apply on `WM_DISPLAYCHANGE` or periodic timer.
- **HDR mode.** Gamma ramps are undefined in HDR. Don't apply if HDR is active (detect via `DXGI_OUTPUT_DESC1.ColorSpace`? — probably overkill for v1).

## Alternatives considered

| Approach | Pros | Cons | Verdict |
|---|---|---|---|
| **Gamma ramp (this)** | Smooth, works everywhere, no DDC/CI | Software overlay, conflicts with Night Light | **Go with this** |
| **DDC/CI RGB gain** | Hardware-level | Coarse, slow, monitor-dependent, looked bad | Tried, rejected |
| **DDC/CI color preset (0x14)** | Simple | Doesn't work (OSD profile overrides) | Rejected |

## Open questions

- Should color temp persist across restarts? (Save to registry or file?)
- Scheduling (auto-warm at sunset) — future feature or part of v1?
- What Kelvin range to expose? f.lux does 1200–6500K. Simpler: 2700–6500K.
- Hotkeys? Win+PgUp/PgDn to step color temp? Or just the slider?
