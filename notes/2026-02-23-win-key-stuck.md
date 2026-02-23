# Win Key Stuck/Broken — Investigation (Not Our Bug)

## Problem

Win key intermittently got "stuck" (every keypress acts as Win+key) or "broken"
(Win+anything registers as just the key). Suspected `RegisterHotKey` + `MOD_WIN`
interaction in `hotkey.go`.

## Investigation

Tested two fixes:

1. **`MOD_NOREPEAT` (0x4000)** on `RegisterHotKey` — prevent auto-repeat
   WM_HOTKEY flooding. Registration succeeded but Win key appeared completely
   broken during testing.
2. **`go fn(id)`** — async dispatch to avoid blocking the message loop during
   DDC/CI calls. Worked fine but unnecessary — DDC/CI calls complete sub-second
   per debug log, and async dispatch introduces race conditions on
   `allMonitors`/`brightItems`/systray calls.

## Actual Cause

Titan Quest (fullscreen DirectX game) was running and suppressing the Win key to
prevent accidental alt-tab. Closing the game restored normal Win key behavior.
The monibright hotkey code was not at fault.

Both fixes were reverted. The original `hotkey.go` code is correct as-is.

## Lesson

Fullscreen DirectX/Vulkan games commonly suppress or hook the Win key. Always
rule out other running software before investigating `RegisterHotKey` issues.
