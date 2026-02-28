# Stale DDC/CI Handle After Sleep/Wake

## Bug (v1: monitor sleep)

After the monitor sleeps and wakes (~1-2 hours idle), `GetBrightness()` returns
`current=0` even though the real brightness is 70-80%. This causes:

- Menu shows 10% checked (nearest preset to 0)
- Tray icon goes invisible (eclipse arc at t=0 = fully eclipsed = all transparent pixels)

## Bug (v2: full computer sleep)

After full computer sleep/wake (S3), `SetBrightness()` silently fails — returns
nil but the monitor brightness doesn't change. Both hotkeys and menu clicks
update the icon/checkmark but have no effect on the screen. The v1 fix
(`refreshCheck` detecting `current=0`) didn't catch this because `refreshCheck`
only runs when opening the menu, and hotkey presses bypass it entirely.

## Log Evidence

```
15:57:38  setting brightness to 70%        ← last interaction
17:40:10  GetBrightness: current=0          ← STALE (actual ~70%), ~1h43m idle
17:40:12  setting brightness to 70%         ← user corrects via menu
17:40:14  GetBrightness: current=70         ← works fine now

20:08:55  GetBrightness: current=0          ← STALE again, ~2h28m idle
20:08:59  GetBrightness: current=80         ← second click 4s later → correct
```

## Root Cause

`PhysicalMonitor` wraps a Win32 handle obtained from `dxva2.dll`
(`GetPhysicalMonitorsFromHMONITOR`). When the monitor powers off (sleep/DPMS),
Windows invalidates these handles. The first DDC/CI call on a stale handle
returns zero/garbage rather than an error. Subsequent calls (or new handles)
work fine — the DDC/CI bus recovers on its own.

## Fix (v1: read path)

In `refreshCheck()`, treat `current=0` as suspicious. Re-enumerate monitors
(`NewSystemMonitors` + `NewPhysicalMonitor`) to get fresh handles, then retry
`GetBrightness`. This replaces `allMonitors` globally so subsequent
`SetBrightness` calls also use fresh handles.

## Fix (v2: write path)

In `setBrightness()`, read back brightness after setting. If the readback
doesn't match the expected level (diff > 5), the handles are stale —
re-enumerate monitors and retry the set. This catches the silent-fail case
where `SetBrightness` returns nil but doesn't actually change the monitor.

## Alternatives Considered

- **Periodic re-enumeration** (e.g. every N minutes): unnecessary overhead when
  idle, and doesn't help if you happen to check right after wake.
- **Retry without re-enumeration**: doesn't work — same stale handle returns
  the same stale result.
- **Ignore 0 and keep last-known value**: hides the problem; user would see
  stale checkmark and icon if they'd changed brightness via monitor buttons.

## Observations

- `SetBrightness` on a stale handle reports "ok" (nil error). After monitor
  sleep, it may still change the monitor. After full computer sleep (S3), it
  silently does nothing — the monitor stays at the previous brightness.
- The recovery is instant — no delay needed between re-enumerate and retry.
- 0% brightness is theoretically valid but extremely unlikely in practice.
  If a user genuinely sets 0%, the retry will confirm it.
