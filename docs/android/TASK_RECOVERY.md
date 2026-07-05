# Task Recovery

This file is the live recovery snapshot for the current Codex task.

Purpose:

- let the next session resume without guessing
- keep the active plan visible after interruption
- preserve the full working context without replacing canonical docs

Rules:

- keep this file tracked in the repo
- do not delete this file
- do not turn this into a governance layer
- do not archive completed task content here when it belongs in canonical docs
- after a task is fully completed, validated, committed, and pushed, clear the
  contents of this file but keep the file itself for the next task

How to use:

- at task start, replace the contents with the current live snapshot, including
  the full task/plan the work is currently following, not a short summary
- after any meaningful checkpoint, refresh it
- after interruption, read this file first before re-planning
- after completion, clear the contents and leave the file present

## Active Task

- Objective: record the standing repository workflow and canonical worktree
  path, so future implementation work starts from the same synced repository
  and the same Definition of Done.
- Scope:
  - document the canonical repository path and working-tree rule
  - document the binary rebuild / validation requirement before asking the user
    to run new commands
  - record the recurring-friction principle in reusable lessons
- Constraints:
  - do not change implementation code
  - do not broaden into upload execution, flashing execution, serial monitor,
    or EOS adoption work
  - keep the docs update limited to workflow, status, and reusable lesson
    records
- Non-goals:
  - new source behavior
  - transport changes
  - upload execution
  - flashing execution
  - serial monitor
  - EOS canon changes

## Intended Plan

- Step 1: keep the recovery snapshot aligned with the standing workflow update.
- Step 2: record the canonical worktree rule and Definition of Done in the
  repository guidance docs.
- Step 3: record the recurring-friction principle in the lessons doc.
- Step 4: leave the recovery snapshot ready for validation and commit without
  broadening scope into implementation work.

## Progress

- Completed:
  - reviewed the current repository state and the canonical workflow docs
  - identified the standing repository workflow updates that needed to be
    captured in `AGENTS.md`, `STATUS.md`, and `LESSONS_LEARNED.md`
- In progress:
  - patching the docs
- Remaining:
  - validate the docs-only change set
  - commit the documentation update if requested
  - clear this file after the closeout is committed and pushed

## Files

- Touched:
  - `AGENTS.md`
  - `STATUS.md`
  - `docs/android/LESSONS_LEARNED.md`
  - `docs/android/TASK_RECOVERY.md`
- Intended:
  - none

## Validation

- Status: standing workflow documentation update in progress
- Evidence collected:
  - the current repository state was checked before editing
  - the standing workflow now has a canonical repo path and rebuild/validation
    requirement in `AGENTS.md`
  - the current status now points future work at the canonical Termux clone
  - the reusable lesson about recurring friction now lives in
    `docs/android/LESSONS_LEARNED.md`
- Evidence still needed:
  - `git diff --check`
  - a final status review before commit

## Safest Next Action

- Next action: validate the docs-only diff, then commit the workflow update if
  requested and clear this file after the repository state is published.

## Canonical Follow-Through

- `VALIDATED_FINDINGS.md`:
- `DECISION_LOG.md`:
- `LESSONS_LEARNED.md`: updated
- `UNCERTAINTY_REGISTER.md`:

## Reset State

When there is no active task, leave this file in place with only the template
above and blank fields.
