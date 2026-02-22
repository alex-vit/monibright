# Build and run monibright

1. Kill any running monibright.exe: `powershell -Command "Stop-Process -Name monibright -Force -ErrorAction SilentlyContinue"` (ignore errors if none running)
2. If user asked to regenerate the icon (or you changed gen_icon.go): `go generate ./icon/`
3. Build with debug tag and no console: `go build -tags debug -ldflags "-H=windowsgui" -o monibright.exe .`
4. Launch: `start monibright.exe` (detached, so it doesn't block)
5. Confirm it's running.
