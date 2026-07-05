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

- Objective: record the native Termux transport stream boundary finding and
  preserve the evidence boundary between diagnostic fd handoff and any future
  transport implementation work.
- Scope:
  - record the native Termux fd-handoff evidence
  - preserve the diagnostic-only classification for the current fd path
  - capture the read/write probe result boundary without expanding into upload
    or flashing
  - keep the next safe evidence step available if a fresh read-only helper probe
    is needed later
- Constraints:
  - do not patch sketch code
  - do not patch third-party libraries
  - do not change board/core/FQBN
  - do not broaden into upload execution, flashing execution, serial monitor,
    ESP32 protocol framing, EEL, EOS adoption work, or source changes
  - do not claim native-Termux success from host or proot evidence
- Non-goals:
  - upload execution
  - flashing execution
  - serial monitor
  - byte-stream readiness
  - EOS canon changes unless transport evidence requires them

## Intended Plan

- Step 1: keep the recovery snapshot aligned with the current transport
  boundary finding.
- Step 2: record the fd-handoff proof, the unsupported write result, and the
  diagnostic-only classification in canonical docs.
- Step 3: preserve the next safe evidence step as a read-only fresh-helper
  probe only if more detail is needed later.
- Step 4: leave the recovery snapshot ready for commit/push without broadening
  scope into upload, flashing, or serial-monitor work.

## Progress

- Completed:
  - reviewed the current repository state and the transport-related canonical
    docs before recording the new finding
  - confirmed the native Termux fd handoff is proven through `probe-fd` and
    `stream-validate`
  - confirmed `fd_source=environment`, `handoff_mode=env`, `fd_valid=true`,
    and `fd_inspectable=true`
  - confirmed raw byte-stream write is unsupported on this path and that read
    remains unproven because it was skipped after write failure
  - confirmed the evidence supports a diagnostic-only, not stream-capable,
    classification for the current Termux USB fd path
- In progress:
  - none
- Remaining:
  - commit and push the documentation update if requested
  - clear this file after the closeout is committed and pushed

## Files

- Touched:
  - `docs/android/TASK_RECOVERY.md`
- Intended:
  - none

## Validation

- Status: native Termux transport boundary evidence collected
- Evidence collected:
  - the transport docs define the diagnostic-only boundary and the next safe
    evidence step
  - native Termux `probe-fd` proves fd handoff and inspection
  - native Termux `stream-validate` proves the current path is not stream-capable
    because write returns `invalid argument`
  - the same result shape reproduces through `termux-usb -r -E -e`
  - no upload, flashing, serial monitor, or source changes were attempted
- Evidence still needed:
  - none for this milestone

## Safest Next Action

- Next action: if more detail is needed, run a fresh helper read probe without
  write first; otherwise commit and push this documentation update if requested
  and then clear this file after the repository state is published.

## Canonical Follow-Through

- `VALIDATED_FINDINGS.md`: updated
- `DECISION_LOG.md`:
- `LESSONS_LEARNED.md`:
- `UNCERTAINTY_REGISTER.md`:

## Reset State

When there is no active task, leave this file in place with only the template
above and blank fields.
