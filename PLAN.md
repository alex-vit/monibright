# Auto-Update Plan

## Approach: DIY with stdlib (zero new dependencies)

The ideas file mentions `go-selfupdate`, but that pulls in 30+ transitive deps (Gitea/GitLab SDKs, OAuth2, YAML, etc.). Following the same philosophy that led to reimplementing the hotkey wrapper (~80 lines vs 941-line lib), we'll implement auto-update with pure stdlib in ~150-200 lines.

**Alternatives considered:**
| Option | New deps | Verdict |
|---|---|---|
| `creativeprojects/go-selfupdate` | 30+ transitive | Overkill — Gitea/GitLab/OAuth2 bloat |
| `minio/selfupdate` | 1 (minisign) | Decent, but the rename trick is ~20 lines |
| `rhysd/go-github-selfupdate` | ~10 transitive | Moderate bloat, less maintained |
| **DIY stdlib** | **0** | **Best fit — full control, ~150 lines** |

## How it works

### 1. On startup: clean up previous update
Delete `monibright.exe.old` left over from a previous update (Windows allows deleting once the old process has exited).

### 2. Background check + download
A goroutine hits `GET https://api.github.com/repos/alex-vit/monibright/releases/latest`, parses the JSON for `tag_name` and the `monibright.exe` asset URL. If the tag is newer than the compiled-in `version`, it downloads the exe to a temp file in the same directory.

GitHub API rate limit: 60 req/hour unauthenticated — checking once per app launch is fine.

### 3. Tray menu: "Update to vX.Y.Z"
A menu item is added at startup but hidden. When the download completes, it becomes visible. On click:
1. Rename running `monibright.exe` → `monibright.exe.old` (Windows allows renaming a running exe)
2. Rename downloaded temp file → `monibright.exe`
3. Launch the new exe
4. Exit the current process

No UAC needed — the app lives in a user-writable directory (`%LocalAppData%\MoniBright` or wherever the user placed it).

## Files

### New: `update.go` (~150 lines)
```
- cleanOldBinary()          // delete .old leftover on startup
- checkForUpdate() (ver, url, error)  // GET GitHub API, compare semver
- downloadUpdate(url) (tmpPath, error) // download to .tmp in same dir
- applyAndRestart(tmpPath)  // rename trick + exec new + exit
- compareVersions(current, latest) bool // simple vX.Y.Z comparison
```

### Modified: `main.go`
- `main()`: call `cleanOldBinary()` early
- `onReady()`: add hidden `mUpdate` menu item after title, spawn update goroutine
- Update goroutine: check → download → show menu item → wire click handler

### Menu layout (when update available)
```
MoniBright v1.1.0
Update to v1.2.0 — restart    ← NEW (hidden until ready)
[Open log]                      ← debug only
───────────────────
100% ... 10%
───────────────────
Start with Windows
───────────────────
Quit
```

## Scope exclusions
- No signature verification (HTTPS from GitHub is sufficient for a personal tool)
- No automatic restart — user clicks when ready
- No rollback mechanism (`.old` file remains for manual recovery)
- No periodic re-checking — only checks once at startup
- Installer users (`monibright-setup.exe`) are not affected — they can still use InnoSetup to update; the self-update targets the portable exe workflow

## Dev notes
Will create `notes/2026-02-22-auto-update.md` documenting the design decisions.
