# Dynamic Tray Icon Reflecting Brightness Level

## Summary

Replaced static yellow circle tray icon with a runtime-generated eclipse icon that reflects the current brightness level. A moon circle slides across the sun — full bright circle at 100%, half crescent at 50%, thin sliver at 10%.

## Visual Design

**Eclipse mechanic:** Two equal-radius circles — sun (bright, colored) and moon (transparent). The moon slides horizontally from fully overlapping (full eclipse at 0%) to fully off-screen (full sun at 100%). Visible crescent width directly maps to brightness.

**Hard pixel edges:** No anti-aliasing. Every pixel is either fully opaque or fully transparent. This is critical — at 16/32px, AA and gradients produce muddy artifacts that look worse than clean pixel edges.

**Color interpolation across three anchors:**
- 10%: deep amber `#B87333`
- 50%: warm yellow `#DCA51A`
- 100%: bright gold `#FFD700`

## Approaches Tried and Rejected

1. **Corona glow (radial gradient falloff from sun body):** Too small and fuzzy at 16px — unreadable. Glow disappears into noise at tray icon scale.
2. **Eclipse with anti-aliased edges:** Partial-alpha pixels at circle edges rendered as dim colored specks on the taskbar — looked like artifacts, not smooth edges.
3. **Color-only circle (no shape change):** Worked but flat — hard to distinguish brightness levels at a glance without the shape cue.
4. **Narrower AA band (0.3px instead of 0.5px):** Made edge artifacts worse, not better — fewer pixels but more visible.
5. **Hand-drawn MS Paint template with recoloring:** Windows tray icon rendering made it look bad too — the issue was partial alpha and scaling, not the drawing method.

**Key insight:** At system tray scale, binary opaque/transparent with hard geometric shapes beats any attempt at smooth rendering.

## Files Changed

- **New: `icon/gen.go`** — `Generate(level)` returns ICO bytes (16+32px). `eclipseImage()` draws sun minus moon. `sunColor()`/`lerpColor()` for color interpolation. `buildICO()` assembles ICO format.
- **Modified: `main.go`** — Added `updateIcon(level)` helper. Called at end of `refreshCheck()` and `setBrightness()`. Static icon still used at startup before brightness is known.
- **Modified: `.claude/commands/run.md`** — Added `-tags debug` to build command.

## Unchanged

- `icon/icon.go` — still provides embedded `Data` for instant startup icon
- `icon/gen_icon.go` — build-time generator stays as-is
