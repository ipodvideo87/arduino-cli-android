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

- Objective: close the package-validation milestone for the ESP32-S3 sketch
  package and preserve the evidence boundary between package generation and
  any future transport or flashing work.
- Scope:
  - validate that the generated package is `full-flash`
  - confirm the manifest, flash-plan, validation-report, analysis, and
    README_FLASHING artifacts exist
  - confirm bootloader, partitions, boot_app0, and application artifacts exist
  - exercise prepare-only upload consumption only if it is already supported
    and safe
- Constraints:
  - do not patch sketch code
  - do not patch third-party libraries
  - do not change board/core/FQBN
  - do not broaden into upload execution, flashing execution, USB transport,
    serial monitor, ESP32 protocol framing, EEL, or EOS adoption work
  - do not claim native-Termux success from host or proot evidence
- Non-goals:
  - upload execution
  - flashing execution
  - transport-stream validation
  - byte-stream readiness
  - EOS canon changes unless package evidence requires them

## Intended Plan

- Step 1: keep the recovery snapshot aligned with the completed closeout.
- Step 2: record the compile result, package validation result, and
  prepare-only dry-run result in canonical docs.
- Step 3: preserve the warning as a documented non-blocking follow-up.
- Step 4: leave the recovery snapshot ready for commit/push without
  broadening scope into transport or flashing work.

## Progress

- Completed:
  - reviewed the current repository state, `STATUS.md`, `ROADMAP.md`,
    `VALIDATED_FINDINGS.md`, `ENGINEERING_DECISIONS.md`, `DECISION_LOG.md`,
    and the package-validation milestone doc
  - confirmed the package milestone scope is package-only and explicitly
    excludes real upload, flashing, USB transport, serial monitor, ESP32
    protocol framing, EEL, and EOS adoption work
  - confirmed the current shell is native Termux for the latest validation
    evidence
  - confirmed package validation now passes and prepare-only upload dry-run
    succeeds without opening a real transport stream
  - confirmed the target-chip warning was resolved by rebuilding the repo-local
    `arduino-cli` binary with `go build -o arduino-cli .` and regenerating the
    package
  - noted the jq-loop artifact check failure as a diagnostic-script
    portability issue rather than a package evidence failure
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

- Status: package validation evidence collected in native Termux
- Evidence collected:
  - the milestone doc defines exact package-only checks and out-of-scope areas
  - the repository already has the package-validation milestone documented in
    `STATUS.md` and `ROADMAP.md`
  - package validation now produces a `full-flash` package with the required
    artifacts
  - prepare-only upload dry-run accepts the package and reports `dry-run: true`
    and `prepare-only: true`
  - rebuilding the repo-local `arduino-cli` binary with `go build -o arduino-cli .`
    and regenerating the package yielded `target_chip = esp32s3` in manifest,
    flash plan, and validation report, with validation warnings now `null`
  - the jq-loop check failure was a shell portability issue, not a package
    failure
- Evidence still needed:
  - none for this milestone

## Safest Next Action

- Next action: commit and push the closeout if requested, then clear this file
  after the repository state is published.

## Canonical Follow-Through

- `VALIDATED_FINDINGS.md`: updated
- `DECISION_LOG.md`:
- `LESSONS_LEARNED.md`:
- `UNCERTAINTY_REGISTER.md`:

## Reset State

When there is no active task, leave this file in place with only the template
above and blank fields.
