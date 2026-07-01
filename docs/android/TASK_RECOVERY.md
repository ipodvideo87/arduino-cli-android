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

- Objective: finish full-flash bootloader package validation on native Termux
  and leave the repo with a reusable native-validation package for the next
  session.
- Scope:
  - firmware package correctness
  - full-flash vs app-only package mode behavior
  - bootloader, partition table, boot_app0, and application artifact detection
  - manifest and flash-plan validation
  - prepare-only upload consumption if already supported
- Constraints:
  - do not implement real upload
  - do not implement flashing
  - do not implement USB transfer work
  - do not implement serial monitor
  - do not widen scope beyond package validation
  - keep the change docs-only unless validation uncovers a doc gap that must be
    closed
- Non-goals:
  - USB transport implementation
  - byte-stream readiness
  - ESP32 protocol framing
  - upload execution
  - flashing execution
  - monitor execution

## Intended Plan

- Step 1: create or update the milestone doc for native full-flash package
  validation so the next session has one canonical place for the validation
  package, success criteria, and decision matrix.
- Step 2: keep the recovery snapshot current with the full live task/plan, not
  a short summary.
- Step 3: expose the exact native Termux command sequence needed to validate a
  full-flash package, confirm artifact presence, and optionally exercise the
  existing prepare-only upload consumer.

## Progress

- Completed:
  - read AGENTS.md, STATUS.md, ROADMAP.md, VALIDATED_FINDINGS.md, and the
    existing milestone docs
  - confirmed firmware-package and prepare-only upload support already exist
  - confirmed there is no existing dedicated full-flash bootloader validation
    milestone doc
  - created the dedicated native full-flash bootloader package validation doc
- In progress:
  - updating the recovery snapshot
- Remaining:
  - provide the exact native validation package commands
  - keep the milestone narrow and docs-only
  - ensure the new doc points at canonical evidence destinations for the
    eventual validation result

## Files

- Touched:
  - `docs/android/TASK_RECOVERY.md`
  - `docs/android/MILESTONE_NATIVE_FULL_FLASH_BOOTLOADER_PACKAGE_VALIDATION.md`
  - `STATUS.md`
- Intended:
  - `docs/android/NATIVE_TERMUX_SMOKE_TEST_WORKFLOW.md`

## Validation

- Status: planning and docs update only
- Evidence collected:
  - firmware package code already supports `full-flash` vs `app-only`
  - prepare-only upload already validates package, flash plan, artifacts, and
    readiness without opening a transport stream
- Evidence still needed:
  - native Termux confirmation that a chosen sketch/core combination produces a
    `full-flash` package
  - confirmation that `manifest.json`, `flash-plan.json`,
    `validation-report.json`, `analysis.json`, and `README_FLASHING.txt` are
    present
  - confirmation that bootloader, partition, boot_app0, and application
    artifacts are present when metadata supports full-flash packaging
  - confirmation that prepare-only upload consumption accepts the package
    without widening scope

## Safest Next Action

- Next action: finish the native Termux validation package in the dedicated
  milestone doc and then pause for user confirmation before running any device
  commands.

## Canonical Follow-Through

- `VALIDATED_FINDINGS.md`:
- `DECISION_LOG.md`:
- `LESSONS_LEARNED.md`:
- `UNCERTAINTY_REGISTER.md`:

## Reset State

When there is no active task, leave this file in place with only the template
above and blank fields.
