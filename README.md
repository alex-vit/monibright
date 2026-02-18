# MoniBright

Windows system tray app for monitor brightness control via DDC/CI.

[Download v1.0.0](https://github.com/alex-vit/monibright/releases/tag/v1.0.0)

<img width="264" height="429" alt="Screenshot (16)" src="https://github.com/user-attachments/assets/1d280d0f-e689-46db-9dda-da2eb86e13bc" />

## Features

- Brightness presets (10%–100%) from the system tray
- Global hotkeys: <kbd>Win+Numpad1</kbd> (10%) through <kbd>Win+Numpad0</kbd> (100%)
- Start with Windows option
- Single portable executable, no installer

## Build

```bash
go build -ldflags "-H=windowsgui" -o monibright.exe .
```
