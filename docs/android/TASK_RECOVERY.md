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

- Objective: investigate and fix the recurring `target chip metadata is not
  set` warning at the smallest architecturally correct layer.
- Scope:
  - compile pipeline metadata flow
  - firmware package generation
  - manifest generation
  - flash-plan generation
  - validation-report generation
  - prepare-only upload consumption
  - workflow reporting and diagnostics that surface target-chip metadata
- Constraints:
  - do not implement upload, flashing, serial monitor, or USB transfer work
  - do not add broad refactors or parallel metadata systems
  - preserve existing package contracts and deterministic behavior
  - preserve existing validation behavior except where the root cause requires
    correction
- Non-goals:
  - live USB transfer work
  - stream readiness promotion
  - ESP32 protocol framing
  - upload execution
  - flashing execution
  - monitor execution

## Intended Plan

- Step 1: trace target-chip metadata from compile inputs through firmware
  package generation, validation, upload planning, and workflow reporting.
- Step 2: identify the smallest layer where the metadata is dropped or not
  propagated.
- Step 3: implement the narrowest production-quality fix that makes the
  package carry the correct target-chip metadata.
- Step 4: add tests that prove the propagation path and the warning behavior.
- Step 5: record the root cause and validation evidence in the canonical docs.

## Progress

- Completed:
  - read AGENTS.md, STATUS.md, ROADMAP.md, TASK_RECOVERY.md, and the canonical
    firmware-package / validation / workflow docs
  - located the warning source in `internal/acl/firmware/package.go`
  - traced the compile path through `commands/service_compile.go`
  - traced the workflow snapshot path through `internal/acl/engine/compile.go`
  - added a shared target-chip resolver in `internal/acl/firmware/metadata.go`
  - propagated target-chip metadata through the compile service and workflow
    snapshot package-generation paths
  - added regression tests for both propagation paths
  - recorded the root cause and host validation evidence in canonical docs
- In progress:
  - awaiting any additional native Termux confirmation the user wants to run
- Remaining:
  - native Termux revalidation of the firmware-package milestone if the user
    wants on-device confirmation
  - commit and push after the task is fully closed

## Files

- Touched:
  - `docs/android/TASK_RECOVERY.md`
  - `STATUS.md`
  - `docs/android/VALIDATED_FINDINGS.md`
  - `docs/android/DECISION_LOG.md`
  - `commands/service_compile.go`
  - `commands/service_compile_test.go`
  - `internal/acl/engine/compile.go`
  - `internal/acl/engine/compile_test.go`
  - `internal/acl/firmware/metadata.go`
- Intended:
  - none

## Validation

- Status: implementation complete, host validation complete, native Termux
  confirmation still pending
- Evidence collected:
  - `BuildManifest`, `FlashPlan`, `ValidationReport`, and `UploadPlan` all
    carry `target_chip`
  - `firmware.NewBinaryValidator()` warns only when both manifest and flash
    plan target chip fields are empty
  - `BuilderResultSnapshot.BuildInput` now populates `TargetChip` from
    `build.mcu`
  - `commands/service_compile.go` now also populates `TargetChip` when
    constructing the firmware package input
  - focused `go test` runs for `commands`, `internal/acl/engine`, and
    `internal/acl/firmware` passed
- Evidence still needed:
  - native Termux confirmation that the package output now matches the host
    result, if the user wants on-device verification

## Safest Next Action

- Next action: if native confirmation is desired, rerun the package milestone on
  native Termux and record the updated output; otherwise the current repo state
  is ready to be committed and pushed.

## Canonical Follow-Through

- `VALIDATED_FINDINGS.md`: updated
- `DECISION_LOG.md`: updated
- `LESSONS_LEARNED.md`:
- `UNCERTAINTY_REGISTER.md`:

## Reset State

When there is no active task, leave this file in place with only the template
above and blank fields.
