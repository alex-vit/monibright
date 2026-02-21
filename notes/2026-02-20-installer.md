# Installer (Done)

InnoSetup installer for monibright. Ships `monibright-setup.exe` alongside the standalone exe.

## Key Decisions

- **InnoSetup standalone** — free, low CI complexity, pre-installed on `windows-2022` runners
- **Install dir:** `%LocalAppData%\MoniBright` — no admin elevation (`PrivilegesRequired=lowest`)
- **Auto-start:** optional task writes `HKCU\...\Run\MoniBright` — same registry key the app's tray toggle uses; no conflict
- **Process handling:** `AppMutex=MoniBrightMutex` + `CloseApplications=yes` for upgrades; `taskkill` for uninstall
- **Version:** `#define AppVersion` defaults to `"dev"`, overridden via `iscc /DAppVersion=v1.2.3` in CI
- **CI:** `windows-2022` runner, uploads both `monibright.exe` and `monibright-setup.exe` to GitHub Releases
