---
name: notes
description: View or update development notes for the current feature/task
user-invocable: true
---

# Notes

Manage development notes in the `notes/` directory. Each note file is named `YYYY-MM-DD-<slug>.md` where the date is when the task started and `<slug>` is a short kebab-case identifier for the feature/task.

## Triggers

Besides explicit `/notes` invocation, update notes proactively when:
- The user says "note that", "make a note", "keep track of", or similar
- A significant decision is made or an approach is chosen
- Requirements are clarified or refined
- A deliverable is completed or scope changes

## File naming

- Path: `notes/YYYY-MM-DD-<slug>.md` (e.g. `notes/2026-02-19-popup-slider.md`)
- One file per feature/task. Reuse the existing file if work continues on the same task.
- Rename the file (change the slug) if the task evolves and the original name no longer fits.
- If unsure which file to update, list `notes/` and pick the most recent relevant one.

## Format

Keep it lightweight. Use headers, bullets, checklists as needed — not every note needs all sections. Typical content includes:
- What we're working on and why
- Goals / acceptance criteria
- Deliverables (checklist if useful)
- Decisions made and rationale
- Open questions

## Behavior

- **New task, no matching notes file**: create one with today's date and a descriptive slug. Fill in whatever context is known from the conversation.
- **Existing notes file for current task**: read it, then update with new information.
- **On `/notes` with no obvious update**: display the current notes to the user.
- Keep it concise. Prefer terse bullet points over prose.
- Do not remove old decisions — they form a log. Prefix superseded ones with ~~strikethrough~~.
