---
name: sync-minimal
description: Sync cmd/monibright-minimal with cmd/monibright-reference after reference has changed
disable-model-invocation: true
---

# Sync minimal variant with reference

The reference impl (`cmd/monibright-reference/main.go`) is the primary app.
The minimal impl (`cmd/monibright-minimal/main.go`) is an experiment that replaces
`golang.design/x/hotkey` with `internal/hotkey/`. It may fall behind.

Steps:

1. Diff the two main.go files to see what's changed in reference but not minimal.
2. Apply non-hotkey changes from reference to minimal. The only intentional differences are:
   - minimal imports `internal/hotkey` instead of `golang.design/x/hotkey`
   - minimal uses `hotkey.RegisterHotkeys` batch API instead of per-hotkey goroutines
3. Verify both variants build: `go vet ./...`
4. Summarize what was synced.
