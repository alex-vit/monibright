# Color Temperature / True Tone — Spike Results

Date: 2026-03-01

## Goal

Test whether DDC/CI can control monitor color temperature for a potential "true tone" feature.

## Test setup

Spike program: `cmd/colorspike/main.go` (throwaway, not committed).
Monitor had its OSD color profile set to "warm."

## Results

### DDC/CI Color Preset (VCP 0x14 — SelectColorPreset)

- **Read works partially:** `GetColorTemperature()` returned enum 3 (6500K). `GetVCPFeatureAndVCPFeatureReply` returned current=5 max=11.
- **Write is silently ignored:** `SetVCPFeature(Set4000K())` returned no error, but read-back stayed at the old value and no visible change occurred.
- **Likely cause:** Monitor's OSD color profile ("warm") overrides DDC/CI preset commands. Common behavior — many monitors lock out VCP 0x14 when a custom profile is active.
- `GetCapabilities()` fails entirely ("only works with MCCS 1.0/2.0 monitors") but individual VCP reads still work.

### RGB Gain (VCP 0x16/0x18/0x1A — VideoGainRed/Green/Blue)

- **Read works:** All three channels report min=0, cur=50, max=100.
- **Write works and is visible:** Setting Blue Gain from 50→30 produced an immediately visible greenish tint (less blue). Read-back confirmed the new value.
- **Restore works:** Setting all three back to 50/50/50 restored normal color.

### Summary table

| Method | Read | Write | Visible effect |
|---|---|---|---|
| VCP 0x14 (color preset) | Yes (partial) | No error, but ignored | None |
| VCP 0x16/0x18/0x1A (RGB gain) | Yes | Yes | Yes |

## Implications for true tone feature

**RGB gain is the viable DDC/CI path.** To simulate color temperature:
- Map Kelvin values to RGB ratios (e.g. 6500K = 50/50/50, 4000K = 50/45/35 — needs a proper lookup table)
- Adjust all three channels together to shift warm/cool
- Range is 0–100 per channel, giving reasonable granularity

**Alternatives considered:**

| Approach | Pros | Cons |
|---|---|---|
| DDC/CI RGB gain | Hardware-level, no overlay, works on this monitor | Coarse (0–100 steps), monitor-dependent, slow DDC/CI writes |
| DDC/CI color preset (0x14) | Simplest API | Doesn't work on this monitor (OSD profile overrides) |
| Windows gamma ramp (SetDeviceGammaRamp) | Works everywhere, smooth, continuous range | Software overlay, conflicts with Night Light/HDR/fullscreen games |

**Open questions:**
- What RGB gain values map to standard color temperatures? Need a Kelvin→RGB table calibrated for the 0–100 gain range.
- How slow are DDC/CI writes when changing all three channels? Noticeable flicker during transition?
- Should we save/restore the user's original RGB gain values on exit?
- Does changing RGB gain persist across monitor power cycles, or does the monitor reset?
