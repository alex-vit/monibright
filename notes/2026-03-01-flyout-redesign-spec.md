# Flyout Redesign — Volume-Style Panel

Date: 2026-03-01

## Goal

Replace the current basic slider popup with a modern flyout matching the Windows 10 volume panel UX: dark panel above the taskbar, device name, slider, and an expandable device list.

## Reference: Windows Volume Flyout

**Collapsed state:**
```
┌──────────────────────────────────────┐
│ Headphones (High Definition Audio) ^ │
│                                      │
│ 🔊  ════════●══════════════   18     │
└──────────────────────────────────────┘
```

**Expanded state:**
```
┌──────────────────────────────────────┐
│ Select playback device               │
│                                      │
│   Speakers (Steam Streaming Mic)     │
│ ● Headphones (High Def Audio)        │
│   Speakers (Steam Streaming)         │
│   5 - 27G2G4 (AMD HD Audio)         │
│                                      │
│ 🔊  ════════●══════════════   18     │
└──────────────────────────────────────┘
```

Key traits:
- Anchored above taskbar, right-aligned near tray area
- Dark background (#202020-ish), light text
- No window border or title bar — clean floating panel
- Chevron (^) on the device name row toggles expand/collapse
- Expanded list shows all devices, selected one is highlighted
- Slider + value always visible at bottom

## MoniBright Equivalent

**Collapsed (or single monitor):**
```
┌──────────────────────────────────────┐
│ DELL U2722D                        ^ │
│                                      │
│ ☀  ════════●══════════════   65%     │
└──────────────────────────────────────┘
```

**Expanded (multi-monitor):**
```
┌──────────────────────────────────────┐
│ Select monitor                       │
│                                      │
│ ● DELL U2722D                        │
│   LG 27GL850                         │
│                                      │
│ ☀  ════════●══════════════   65%     │
└──────────────────────────────────────┘
```

- Chevron only shown when >1 monitor
- Clicking a monitor name selects it; slider adjusts that monitor
- Single-monitor: no chevron, just shows the monitor name as a label

## Current State

`slider.go` — 385 lines of Win32:
- `WS_POPUP | WS_BORDER` window, 260×72 fixed size
- Stock `msctls_trackbar32` control + two `STATIC` labels
- `WM_CTLCOLORSTATIC` hack for dark background on labels
- Background brush and text color forced via GDI
- Positioned above taskbar via `SHAppBarMessage` (just added)
- Dismisses on `WM_ACTIVATE` → `WA_INACTIVE`

## What Needs to Change

### 1. Monitor names

**Problem:** The ddcci library doesn't expose monitor names — `PhysicalMonitor` has an unexported `handle` and `description` field. `SystemMonitor` has no exported description either.

**Options:**
- (a) Fork/patch ddcci to export the description string (it's already stored internally from `GetPhysicalMonitorsFromHMONITOR` which returns `PHYSICAL_MONITOR.szPhysicalMonitorDescription`)
- (b) Call `GetPhysicalMonitorsFromHMONITOR` ourselves to get the description string alongside monitor init
- (c) Fall back to "Monitor 1", "Monitor 2" labels

Recommendation: **(b)** — we already call the ddcci library for brightness; adding one more Win32 call to get the name string is minimal and avoids forking. If names come back generic ("Generic PnP Monitor"), supplement with EDID model name via `SetupAPI`.

### 2. Owner-drawn slider

The stock `msctls_trackbar32` doesn't match the volume flyout's flat aesthetic. Options:

- (a) **Keep stock trackbar**, style with `WM_CTLCOLORSTATIC` and dark background — close enough, already working
- (b) **Owner-draw the trackbar** — handle `NM_CUSTOMDRAW` on the trackbar to paint a flat track and circular thumb
- (c) **Fully custom slider** — paint the entire control ourselves in `WM_PAINT`, handle mouse hit-testing manually

Recommendation: **(a) for v1, (b) later** — the stock trackbar on a dark background already looks decent. Owner-draw is polish, not blocking.

### 3. Window chrome

Current: `WS_BORDER` gives a thin 1px frame. Volume flyout has no visible border, just a shadow/drop-shadow.

To match:
- Remove `WS_BORDER`
- Add `WS_EX_NOREDIRECTIONBITMAP` + `DwmExtendFrameIntoClientArea` for shadow without border (or just `CS_DROPSHADOW` on the window class for a simpler shadow)
- Optional: `DwmSetWindowAttribute(DWMWA_WINDOW_CORNER_PREFERENCE, DWMWCP_ROUND)` for rounded corners (Windows 11 only; on Win10 the volume flyout has square corners anyway)

Recommendation: Remove `WS_BORDER`, add `CS_DROPSHADOW` to the window class style. Simple, no DWM dependency.

### 4. Layout & sizing

Current: Fixed 260×72.

New layout (collapsed):
```
Width: ~320px (wider to fit monitor names)
Height: ~70px

[12px pad] Monitor Name Label          [chevron] [12px pad]
[12px pad] [sun icon] [slider 200px] [pct 40px]  [12px pad]
```

New layout (expanded, N monitors):
```
Height: ~70px + N × 28px + 8px separator

[12px pad] "Select monitor"                      [12px pad]
[12px pad]   ● Monitor Name 1                     [12px pad]
[12px pad]     Monitor Name 2                     [12px pad]
[8px gap]
[12px pad] [sun icon] [slider 200px] [pct 40px]  [12px pad]
```

Window resizes with `MoveWindow` on expand/collapse. Could animate with a timer for smooth expand, but snapping is fine for v1.

### 5. Expand/collapse state

- Chevron is a clickable STATIC or custom-drawn `^`/`v` character
- `WM_LBUTTONUP` hit test on chevron rect toggles expanded state
- Monitor name rows: each is a STATIC with `SS_NOTIFY` for click events, or owner-drawn hit rects
- Selected monitor index stored in a package-level var
- On monitor select: update slider position from that monitor's brightness, resize window to collapsed

### 6. Per-monitor brightness

Currently `setBrightness` writes to ALL monitors. With per-monitor selection:
- Slider adjusts only the selected monitor
- Hotkeys still adjust all monitors (current behavior, unchanged)
- `syncSlider` needs to know which monitor was changed

### 7. Font

Current: Default system font (ugly). Volume flyout uses Segoe UI 9pt.

Fix: `CreateFontW` with "Segoe UI", send `WM_SETFONT` to all child controls and use in owner-draw paint.

## Implementation Plan

| Phase | Scope | Effort |
|---|---|---|
| **1. Layout & font** | Wider window, Segoe UI font, monitor name label, remove WS_BORDER + add CS_DROPSHADOW | Small |
| **2. Monitor names** | Call GetPhysicalMonitorsFromHMONITOR ourselves, display in label | Small |
| **3. Expand/collapse** | Chevron toggle, monitor list, per-monitor slider, window resize | Medium |
| **4. Visual polish** | Owner-drawn slider, hover highlights on monitor rows, smooth expand animation | Medium |

Phases 1–2 are a single session. Phase 3 is the meaty one. Phase 4 is optional polish.

## Alternatives Considered

| Approach | Pros | Cons |
|---|---|---|
| **Win32 owner-draw (this spec)** | No new deps, full control, consistent with codebase | Verbose Win32 code, manual hit-testing |
| **WebView2 / HTML panel** | Easy styling, modern look trivially | Huge dependency (WebView2 runtime), startup latency, overkill |
| **lxn/walk** | Higher-level Win32 wrappers | Already decided against walk for monibright (savior uses it); would add a big dep |
| **Direct2D custom draw** | GPU-accelerated, smooth | Complex COM setup, overkill for a flyout |

## Open Questions

- Should the flyout remember which monitor was last selected, or always default to "all"?
- Should we add an "All monitors" row in the expanded list for the current "set all at once" behavior?
- Does the tray icon / hotkey behavior change, or only the left-click flyout?
