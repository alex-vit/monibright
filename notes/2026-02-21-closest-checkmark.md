# Handle non-preset brightness in UI

## Task
When the current brightness doesn't match a preset exactly (e.g. 75% set via monitor buttons), show the closest preset as checked instead of showing no checkmark.

## Goals
- Closest preset gets the checkmark (standard rounding: 65→70, 75→80)
- No UI or menu structure changes — just smarter checkmark logic

## Deliverables
- [x] Update `checkItem` in `main.go` to round to nearest preset

## Decisions
- **Rounding**: standard math rounding (midpoints round up) — e.g. 25→30, 35→40
- **Scope**: only `checkItem` needs to change; `setBrightness` still sets exact values
