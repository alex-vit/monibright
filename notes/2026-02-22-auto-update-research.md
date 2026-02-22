# Auto-Update Research for Go Windows Desktop App

Research into self-update approaches for a single-binary Windows tray app using GitHub Releases.

## The Problem

MoniBright is distributed as a single `.exe`. We want to:
1. Check GitHub Releases for a newer version on startup (or periodically)
2. Download the new binary
3. Replace the running executable
4. Restart (or prompt user to restart)

## Windows-Specific Challenges

### File Locking
Windows locks a running `.exe` — you cannot write to or delete it. However, Windows **does** allow **renaming** a running executable. This is the key trick every library exploits:

1. Rename `monibright.exe` -> `monibright.exe.old` (works even while running)
2. Write/rename new binary to `monibright.exe`
3. Old file cannot be deleted while running — hide it or clean up on next launch

### UAC / Elevation
- If the app lives in `Program Files`, replacing requires admin elevation
- If the app lives in `%APPDATA%` or user-writable directory, no elevation needed
- MoniBright is a per-user app (autostart via `HKCU\...\Run`), so it should live in a user-writable location — **no UAC issue**

### Cleanup
Old `.exe.old` files linger until next launch. Libraries handle this by:
- Hiding the old file (Windows `FILE_ATTRIBUTE_HIDDEN`)
- Deleting `.old` files on startup

## Library Comparison

### Option A: `creativeprojects/go-selfupdate`

The most feature-complete library. Already referenced in `00-ideas.md`.

**Pros:**
- Directly supports GitHub Releases as a source provider (also Gitea, GitLab)
- Handles semver comparison, asset name matching (OS/arch), decompression
- Supports `.zip`, `.gzip`, `.tar.gz`, `.tar.xz`, `.bz2`
- Hash and signature validation (sha256, ECDSA)
- Rollback on failure
- Context-aware API
- Actively maintained (last updated Dec 2025)

**Cons:**
- **Heavy dependency tree**: pulls in `google/go-github/v74`, `Masterminds/semver/v3`, Gitea SDK, GitLab SDK, `golang.org/x/oauth2`, `ulikunitz/xz`, `gopkg.in/yaml.v3`, plus many transitive deps (HTTP retry, pageant, etc.)
- Overkill for a single-repo GitHub-only use case
- Contradicts project's dependency minimization goal

**Direct deps**: ~10 | **Transitive deps**: ~30+

### Option B: `minio/selfupdate`

Low-level building block — handles binary replacement only (not release detection).

**Pros:**
- Extremely minimal: 1 direct dep (`aead.dev/minisign`), 2 indirect (`x/crypto`, `x/sys`)
- Handles the hard part: atomic rename, Windows `.old` hiding, rollback
- `selfupdate.Apply(reader, opts)` — just feed it an `io.Reader` with the new binary
- Checksum and minisign signature verification built in
- Used in production by MinIO itself

**Cons:**
- Does NOT check GitHub releases — you must write that yourself
- Does NOT do semver comparison — you must do that yourself
- Does NOT decompress archives — you must handle `.zip` extraction yourself
- `minisign` dep may be unnecessary if you don't need signature verification

**Direct deps**: 1 | **Transitive deps**: 2 (both `x/` stdlib-adjacent)

### Option C: `rhysd/go-github-selfupdate`

The original GitHub-focused self-update library.

**Pros:**
- GitHub Releases focused (no Gitea/GitLab bloat)
- Semver comparison built in
- Asset name matching for OS/arch

**Cons:**
- Pulls in `google/go-github`, `golang.org/x/oauth2`, `inconshreveable/go-update`
- Less actively maintained than `creativeprojects/go-selfupdate`
- Still a heavy dependency tree

### Option D: DIY (stdlib + minimal helpers)

Write the update logic ourselves using `net/http` and `os`.

**Steps:**
1. `GET https://api.github.com/repos/OWNER/REPO/releases/latest` — returns JSON with `tag_name` and `assets[].browser_download_url`
2. Compare `tag_name` (e.g. `v1.2.0`) against compiled-in `version` string — simple string comparison or trivial semver
3. Download the asset (`.exe` directly, or `.zip` containing `.exe`)
4. Rename running exe -> `.old`, write new exe, clean up

**Pros:**
- Zero new dependencies (net/http, encoding/json, os, archive/zip all stdlib)
- Total implementation: ~150-200 lines
- Full control, no abstraction leaks
- Perfectly aligned with dependency minimization goal
- Can reuse existing `version` variable already in `main.go`

**Cons:**
- Must handle Windows rename trick ourselves (~20 lines, well-documented pattern)
- No rollback on failure (can add ~15 lines)
- No signature verification (acceptable for a personal tool; GitHub HTTPS is sufficient)
- No automatic OS/arch asset matching (unnecessary — only building for `windows/amd64`)
- Must handle GitHub API rate limiting (60 req/hr unauthenticated; checking once at startup is fine)

## GitHub API Rate Limits

- **Unauthenticated**: 60 requests/hour per IP
- **Authenticated** (personal access token): 5,000 requests/hour
- For a desktop app checking once per launch, unauthenticated is more than sufficient
- The endpoint `GET /repos/{owner}/{repo}/releases/latest` returns a single JSON object

## Recommended Approach

**Option D (DIY)** is the best fit for MoniBright, with Option B (`minio/selfupdate`) as a fallback if the Windows exe-replacement logic proves tricky.

**Rationale:**
- MoniBright already values dependency minimization (reimplemented hotkey wrapper to avoid a dep)
- Only targets `windows/amd64` — no need for cross-platform asset matching
- Only uses GitHub — no need for Gitea/GitLab providers
- The GitHub Releases API is trivial to call with `net/http`
- Semver comparison for `vX.Y.Z` tags is ~20 lines
- The Windows rename-running-exe trick is well-documented and ~20 lines

**If DIY proves insufficient**, `minio/selfupdate` adds only 3 deps (all stdlib-adjacent) and handles the tricky atomic-replace + rollback + Windows-hiding cleanly. You'd still write the GitHub release checking yourself, but the binary replacement would be battle-tested.

## Sketch of DIY Implementation

```go
// update.go — ~150 lines total

// checkForUpdate checks GitHub for a newer release.
// Returns (downloadURL, newVersion, needsUpdate).
func checkForUpdate() (string, string, bool, error) {
    resp, err := http.Get("https://api.github.com/repos/alex-vit/monibright/releases/latest")
    // parse JSON: tag_name, assets[0].browser_download_url
    // compare tag_name vs version
    // return asset URL if newer
}

// applyUpdate downloads the new exe and replaces the running binary.
func applyUpdate(url string) error {
    // 1. Download to temp file
    // 2. Rename current exe -> exe.old
    // 3. Rename temp -> current exe path
    // 4. (Optional) hide .old file
}

// cleanOldBinary removes leftover .old file from previous update.
func cleanOldBinary() {
    exe, _ := os.Executable()
    os.Remove(exe + ".old")
}

// Simple semver: split on ".", compare major/minor/patch as ints.
func isNewer(current, latest string) bool { ... }
```

## UX Options

1. **Silent check + tray notification**: Check on startup, show "Update available (v1.2.0)" menu item
2. **Silent check + auto-download**: Download in background, show "Restart to update" menu item
3. **Manual only**: "Check for updates" menu item that user clicks

Option 2 is the best UX — no interruption, update is ready when user is ready to restart.
