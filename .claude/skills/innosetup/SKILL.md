---
name: innosetup
description: InnoSetup compiler location and bash quirks on Windows
disable-model-invocation: true
---

# InnoSetup

## ISCC Location

```
C:\Users\alex\AppData\Local\Programs\Inno Setup 6\ISCC.exe
```

Bash-compatible path: `/c/Users/alex/AppData/Local/Programs/Inno Setup 6/ISCC.exe`

## Bash Quirk: MSYS Path Conversion

MSYS/Git Bash converts arguments starting with `/` to Windows paths, which breaks ISCC flags like `/DAppVersion=1.0.0`. Disable this with `MSYS_NO_PATHCONV=1`:

```bash
MSYS_NO_PATHCONV=1 "/c/Users/alex/AppData/Local/Programs/Inno Setup 6/ISCC.exe" /DAppVersion=1.0.0 installer.iss
```
