# Installer & Update Distribution

## Goal

Make monibright installable and upgradeable on Windows — proper Start Menu shortcut, Add/Remove Programs entry, seamless updates.

## Options Researched

| Option | Price | CI Complexity | Updates | Verdict |
|---|---|---|---|---|
| **InnoSetup** (standalone) | Free (modified BSD) | Low — `.iss` script, `iscc` in CI | Re-run installer to upgrade | Best traditional installer |
| GoReleaser + NSIS | **Pro-only** ($79/yr) | Medium | None built-in | Ruled out — paywall |
| GoReleaser + WiX (MSI) | **Pro-only** + WiX fee | High — verbose XML | MSI major upgrades | Overkill, paywall |
| **go-selfupdate** (no installer) | Free | Just `go build` + ~20 LOC | Self-updates from GitHub Releases | Solves updates, not installation |
| **InnoSetup + go-selfupdate** | Free | `.iss` + one Go dep | Install once, self-update after | **Best combo** |
| Scoop / winget | Free | Low (Scoop) / Medium (winget) | Package manager commands | Complement, not primary |

## Decision

**Recommended: InnoSetup + go-selfupdate** (can start with InnoSetup alone, add self-update later).

- **InnoSetup**: free, runs headless via `iscc.exe` (or Docker in CI). Handles install dir, shortcuts, uninstaller, auto-start.
- **go-selfupdate**: [creativeprojects/go-selfupdate](https://github.com/creativeprojects/go-selfupdate) — actively maintained (Jan 2025), supports GitHub Releases, Windows `.exe` naming, rollback on failure.
- Go itself used InnoSetup before switching to MSI. Very proven.

## Self-Update UX

go-selfupdate uses a rename trick on Windows: renames running `.exe` to `.old`, puts new exe in place. Old process keeps running from memory. Flow for monibright:

1. Check for update on startup (or periodically)
2. Download + apply silently in background (rename works while running)
3. Tray notification: "Update installed — restart to apply"
4. New exe runs on next natural restart (or user clicks "Restart now")
5. Clean up `.old` file on next launch

User is never interrupted. No popup, no forced restart.

## TODO

- [ ] Write InnoSetup `.iss` script (install dir, Start Menu shortcut, uninstaller, optional auto-start)
- [ ] Add InnoSetup build step to GitHub Actions release workflow
- [ ] Integrate `creativeprojects/go-selfupdate` — check on startup, silent download, tray "restart to update" option
- [ ] Clean up `.old` file on launch
- [ ] Decide: auto-start via registry Run key or Startup folder

## Open Questions

- Start with InnoSetup only, or go straight to InnoSetup + self-update?
- Auto-start on login — registry Run key or Startup folder?
- Should the installer kill a running monibright.exe before upgrading?
- Where to install by default? `%LocalAppData%\monibright` (no admin) vs `%ProgramFiles%` (needs admin)?
