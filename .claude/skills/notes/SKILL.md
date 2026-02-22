---
name: notes
description: View or update development notes for the current feature/task
user-invocable: true
---

# Notes

Manage development notes in the `notes/` directory. Each note file is named `YYYY-MM-DD-<slug>.md` where the date is when the task started and `<slug>` is a short kebab-case identifier for the feature/task.

## Triggers

Besides explicit `/notes` invocation, update notes proactively when:
- The user says "note that", "make a note", "keep track of", "add a todo", or similar
- A significant decision is made or an approach is chosen
- Requirements are clarified or refined
- A deliverable is completed or scope changes
- A concrete next step or action item emerges from discussion

## File naming

- Path: `notes/YYYY-MM-DD-<slug>.md` (e.g. `notes/2026-02-19-popup-slider.md`)
- One file per feature/task. Reuse the existing file if work continues on the same task.
- Rename the file (change the slug) if the task evolves and the original name no longer fits.
- If unsure which file to update, list `notes/` and pick the most recent relevant one.

## Status

Each note has a status shown as a suffix in the H1 title: `# Feature Name (Status)`

| Status | Meaning |
|---|---|
| *(none)* | New or early exploration, no status needed yet |
| Research | Gathering info, comparing approaches |
| In Progress | Actively being implemented |
| Done | Completed |
| Abandoned | Decided not to pursue; keep the note for context |

Update the status as work progresses. When marking Done, follow the closing-out guidance below.

## Format

Keep it lightweight. Use headers, bullets, checklists as needed — not every note needs all sections. Typical content includes:
- What we're working on and why
- Goals / acceptance criteria
- Deliverables — not individual functions/changes, but major outputs like a new release, a report, etc. Usually skipped.
- Alternatives considered — if multiple approaches were analyzed, list them. Use a comparison table when dimensions vary across options; use bullet points when a brief summary suffices. **Preserve this research even when marking a task as done** — it's valuable context for revisiting decisions later.
- **TODOs** — `- [ ]` checklist of action items and next steps
- Decisions made and rationale
- Open questions

## Behavior

- **New task, no matching notes file**: create one with today's date and a descriptive slug. Fill in whatever context is known from the conversation.
- **Existing notes file for current task**: read it, then update with new information.
- **On `/notes` with no obvious update**: display the current notes to the user.
- **TODOs**: when the user asks to "add a todo" or a clear action item comes up in discussion, add it as `- [ ]` under a `## TODO` section in the relevant notes file. Proactively add TODOs when implementation steps are agreed upon. Keep TODOs in sync with actual progress — check off (`- [x]`) items as they are completed during the session.
- Keep it concise. Prefer terse bullet points over prose.
- Do not remove old decisions — they form a log. Prefix superseded ones with ~~strikethrough~~.
- **Closing out a note** (marking done): simplify implementation details and remove TODOs, but keep alternatives considered, key decisions with rationale, and anything that explains *why* something was done a particular way.
