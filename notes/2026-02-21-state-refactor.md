# Refactor onReady closures to package-level state

## Task
Extract closures from `onReady` into top-level functions backed by package-level variables, keeping `onReady` focused on initialization and wiring.

## Decisions
- Trivial one-liner lambdas (e.g. `item.Click(func() { setBrightness(level) })`) stay as closures — unavoidable with the callback API
- No state struct needed — package-level vars are sufficient for this codebase size
