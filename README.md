# MoniBright

Windows system tray app for monitor brightness and color temperature control via DDC/CI.

[Download v1.4.0](https://github.com/alex-vit/monibright/releases/tag/v1.4.0) — **monibright-setup.exe** (installer) or **monibright.exe** (portable)

<img width="351" height="208" alt="Screenshot (20)" src="https://github.com/user-attachments/assets/e63cf17e-4828-4090-9de1-79cc3ad605db" />

## Features

- **Brightness slider** — left-click the tray icon for a popup slider, right-click for preset menu (10%–100%)
- **Color temperature** — adjustable warm shift from 3500K to 6500K via the slider
- **Auto color temperature** — f.lux-style automatic warm shift based on sunrise/sunset at your location
- **Global hotkeys** — <kbd>Win+Numpad1</kbd> (10%) through <kbd>Win+Numpad0</kbd> (100%)
- **Dynamic tray icon** — reflects current brightness level
- **Self-update** — checks for new releases on startup
- **Start with Windows** — optional autostart via installer or tray menu toggle

## Build

```bash
go build -ldflags "-H=windowsgui" -o monibright.exe .
```
