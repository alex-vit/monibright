# Build and run monibright

1. Kill any running monibright.exe: `powershell -Command "Stop-Process -Name monibright -Force -ErrorAction SilentlyContinue"` (ignore errors if none running)
2. If user asked to regenerate the icon (or you changed gen_icon.go): `go generate ./internal/icon/`
3. Determine variant:
   - Default: `./cmd/monibright-reference/`
   - If user asked for minimal: `./cmd/monibright-minimal/`
4. Build with no console: `go build -ldflags "-H=windowsgui" -o monibright.exe <variant>`
5. Launch: `start monibright.exe` (detached, so it doesn't block)
6. Confirm it's running.
