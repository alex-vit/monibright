# Installer (Done)

InnoSetup installer for monibright. Ships `monibright-setup.exe` alongside the standalone exe.

## Key Decisions

- **InnoSetup standalone** — free, low CI complexity, pre-installed on `windows-2022` runners
- **Install dir:** `%LocalAppData%\MoniBright` — no admin elevation (`PrivilegesRequired=lowest`)
- **Auto-start:** optional task writes `HKCU\...\Run\MoniBright` — same registry key the app's tray toggle uses; no conflict
- **Process handling:** `AppMutex=MoniBrightMutex` + `CloseApplications=yes` for upgrades; `taskkill` for uninstall
- **Version:** `#define AppVersion` defaults to `"dev"`, overridden via `iscc /DAppVersion=v1.2.3` in CI
- **CI:** `windows-2022` runner, uploads both `monibright.exe` and `monibright-setup.exe` to GitHub Releases

## Alternatives Considered

| Option | Price | CI Complexity | Updates | Verdict |
|---|---|---|---|---|
| **InnoSetup** (standalone) | Free (modified BSD) | Low — `.iss` script, `iscc` in CI | Re-run installer to upgrade | **Chosen** |
| GoReleaser + NSIS | Pro-only ($79/yr) | Medium | None built-in | Ruled out — paywall |
| GoReleaser + WiX (MSI) | Pro-only + WiX fee | High — verbose XML | MSI major upgrades | Overkill, paywall |
| go-selfupdate (no installer) | Free | Just `go build` + ~20 LOC | Self-updates from GitHub Releases | Solves updates, not installation |
| InnoSetup + go-selfupdate | Free | `.iss` + one Go dep | Install once, self-update after | Best combo — deferred self-update to future |
| Scoop / winget | Free | Low (Scoop) / Medium (winget) | Package manager commands | Complement, not primary |
