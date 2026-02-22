# Build Tags for Debug vs Release

## Status: Done

## Summary

Replaced runtime `version == "dev"` debug detection with compile-time build tags. Debug mode is now controlled by `//go:build debug` / `//go:build !debug` across two files (`debug.go`, `release.go`) that define `const debug = true/false`.

## Motivation

Decouple "debug mode" from "version string". Previously, `var version = "dev"` served double duty: display version and debug flag. This meant you couldn't have a versioned debug build (e.g. testing a release candidate with logging enabled).

## Alternatives considered

### 1. ldflags `version == "dev"` (previous approach)
- **Pros:** Simple, no extra files, already working
- **Cons:** Conflates version with debug intent; `var` not `const` so compiler can't dead-code-eliminate debug paths; can't build a versioned debug binary

### 2. ldflags with separate debug var (`-X main.debug=1`)
- **Pros:** Orthogonal to version, simple single-mechanism
- **Cons:** Must be a string (ldflags limitation), no dead-code elimination, easy to forget the flag

### 3. Build tags (chosen)
- **Pros:** Explicit intent (`-tags debug`); `const` bool enables dead-code elimination — release binaries contain zero debug code; can swap entire files per variant; orthogonal to version
- **Cons:** Two extra small files; one more flag to remember in build command

### 4. Hybrid (build tags + ldflags)
- Considered but unnecessary for this project — build tags alone cover the need

## What changed

- Added `debug.go` (`//go:build debug`, `const debug = true`)
- Added `release.go` (`//go:build !debug`, `const debug = false`)
- Removed `isDebugBuild()` function from `main.go`
- Changed `var version` default from `"dev"` to `""` — version is now purely for display
- Added `displayVersion()` helper that returns `"dev"` when version is unset
- All `if isDebugBuild()` checks replaced with `if debug`
- Updated CLAUDE.md build commands, debug logging note

## Build commands

```bash
go build -tags debug -ldflags "-H=windowsgui" -o monibright.exe .                          # dev
go build -ldflags "-X main.version=1.1.0 -H=windowsgui" -o monibright.exe .                # release
go build -tags debug -ldflags "-X main.version=1.1.0 -H=windowsgui" -o monibright.exe .    # versioned debug (now possible)
```
