# Build installer package

Build monibright.exe and the InnoSetup installer (`monibright-setup.exe`).

## Steps

1. Kill any running monibright.exe: `powershell -Command "Stop-Process -Name monibright -Force -ErrorAction SilentlyContinue"` (ignore errors if none running)
2. Determine version from the latest git tag: `git tag --sort=-version:refname | head -1`. If no tags, default to `dev`.
3. Build the exe:
   ```
   go build -ldflags "-X main.version=<version> -H=windowsgui" -o monibright.exe .
   ```
4. Build the installer with ISCC (see `innosetup` skill for path and flags):
   ```
   MSYS_NO_PATHCONV=1 "<iscc-path>" /DAppVersion=<version> installer.iss
   ```
5. Report the output path: `Output/monibright-setup.exe`
