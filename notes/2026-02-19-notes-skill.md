# Notes skill (Done)

## Task
Create a Claude Code skill for keeping development notes during feature work.

## Goals
- Skill invocable via `/notes` and triggered by natural language ("note that", "make a note", etc.)
- Notes stored in `notes/` with date-prefixed filenames
- Structured format: Task, Goals, Deliverables, Decisions, Open Questions
- Notes kept up to date proactively during development

## Decisions
- **File location**: `notes/YYYY-MM-DD-<slug>.md` — one file per task, date is when work started
- **Decisions are append-only**: superseded ones get ~~strikethrough~~ rather than deleted
- **CLAUDE.md integration**: added `notes/` to project layout and a short "Development Notes" section pointing to the skill

## Decisions
- **Git tracking**: notes are committed to the repo — useful as project history
