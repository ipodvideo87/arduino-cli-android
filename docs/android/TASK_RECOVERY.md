# Task Recovery

This file is the live recovery snapshot for the current Codex task.

Purpose:

- keep the active implementation and validation state visible after interruption
- let the next session resume without guessing
- preserve the current task plan without replacing canonical docs

Rules:

- keep this file tracked in the repo
- do not delete this file
- do not turn this into a governance layer
- after a task is fully completed, validated, committed, and pushed, clear the
  contents of this file but keep the file itself for the next task

How to use:

- at task start, replace the contents with the current live snapshot
- after any meaningful checkpoint, refresh it
- after interruption, read this file first before re-planning
- after completion, clear the contents and leave the file present

## Active Task

- none

## Intended Plan

- none

## Progress

- Completed:
  - none
- In progress:
  - none
- Remaining:
  - none

## Files

- Touched:
  - none
- Intended:
  - none

## Validation

- Status: idle
- Evidence collected:
  - none
- Evidence still needed:
  - none

## Safest Next Action

- Next action: start the next task from a fresh live snapshot.

## Canonical Follow-Through

- `STATUS.md`:
- `VALIDATED_FINDINGS.md`:
- `DECISION_LOG.md`:
- `LESSONS_LEARNED.md`:
- `UNCERTAINTY_REGISTER.md`:

## Reset State

When there is no active task, leave this file in place with only the template
above and blank fields.
