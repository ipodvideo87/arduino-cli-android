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

- Objective: record the native Termux stream-boundary closeout evidence as a
  docs-only update and preserve the boundary between diagnostic helper evidence
  and any future transport implementation work.
- Scope:
  - record the canonical repo preflight and stream-boundary evidence
  - preserve the diagnostic-only classification for the current fd path
  - keep upload, flashing, serial monitor, and payload transfer out of scope
  - keep the next milestone available for read-only claim/release diagnostics
- Constraints:
  - do not change implementation code
  - do not broaden into upload execution, flashing execution, serial monitor,
    or EOS adoption work
  - keep the docs update limited to workflow, status, and reusable evidence
    records
- Non-goals:
  - new source behavior
  - transport changes
  - upload execution
  - flashing execution
  - serial monitor
  - EOS canon changes

## Intended Plan

- Step 1: keep the recovery snapshot aligned with the stream-boundary closeout.
- Step 2: record the preflight and fd boundary in the canonical docs.
- Step 3: preserve the next milestone as read-only claim/release diagnostics.
- Step 4: leave the recovery snapshot ready for validation and commit without
  broadening scope into implementation work.

## Progress

- Completed:
  - reviewed the current repository state and the canonical workflow docs
  - confirmed canonical repo preflight passed
  - confirmed read probing returned `EOF`
  - confirmed write probing returned `invalid argument`
  - confirmed fd handoff remains valid and inspectable
  - confirmed `TERMUX_USB_FD` should not be treated as a generic byte stream
  - confirmed the stream boundary is not validated and the next milestone is
    read-only claim/release diagnostics
- In progress:
  - patching the docs
- Remaining:
  - validate the docs-only change set
  - commit the documentation update if requested
  - clear this file after the closeout is committed and pushed

## Files

- Touched:
  - `STATUS.md`
  - `docs/android/VALIDATED_FINDINGS.md`
  - `docs/android/TASK_RECOVERY.md`
- Intended:
  - none

## Validation

- Status: stream-boundary closeout documentation update in progress
- Evidence collected:
  - canonical repo preflight passed
  - the stream-boundary evidence is captured in canonical docs
  - the next milestone is read-only claim/release diagnostics
- Evidence still needed:
  - `git diff --check`
  - a final status review before commit

## Safest Next Action

- Next action: validate the docs-only diff, then commit the stream-boundary
  closeout if requested and clear this file after the repository state is
  published.

## Canonical Follow-Through

- `STATUS.md`: updated
- `VALIDATED_FINDINGS.md`: updated
- `DECISION_LOG.md`:
- `LESSONS_LEARNED.md`:
- `UNCERTAINTY_REGISTER.md`:

## Reset State

When there is no active task, leave this file in place with only the template
above and blank fields.
