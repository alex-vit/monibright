# Auto-Update Implementation

Implements self-update via GitHub Releases with zero new dependencies.

## Design Decisions

### DIY vs library
Chose pure stdlib (~150 lines in `update.go`) over `creativeprojects/go-selfupdate` (30+ transitive deps) and `minio/selfupdate` (3 deps). Same reasoning as the hotkey reimplementation — thin wrapper wins over heavy lib when usage is narrow.

See [auto-update-research.md](2026-02-22-auto-update-research.md) for full comparison.

### Silent apply, no restart
The update is applied silently in the background — no menu item, no restart prompt. The new exe takes effect on the next natural launch (reboot, autostart, or manual). This avoids interrupting the user for something that isn't urgent.

### Windows exe replacement
Windows locks running executables but allows renaming them. The update sequence:
1. Rename `monibright.exe` → `monibright.exe.old`
2. Rename `monibright.exe.tmp` → `monibright.exe`
3. On next startup, delete `.old`

If step 2 fails, we restore `.old` → `monibright.exe`.

### Dev builds skip update
`isNewer()` returns false when `version == ""` or `"dev"`, so debug/dev builds never trigger updates.

## Files Changed
- **`update.go`** (new) — `cleanOldBinary`, `checkForUpdate`, `downloadUpdate`, `applyUpdate`, `isNewer`, `parseSemver`
- **`main.go`** — calls `cleanOldBinary()` in `main()`, spawns background goroutine in `onReady()` for silent check+download+apply
