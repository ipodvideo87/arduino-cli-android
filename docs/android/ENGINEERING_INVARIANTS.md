# Engineering Invariants

This document defines the non-negotiable process invariants that prevent
engineering drift in this repository.

It is intentionally small and operational. If an invariant belongs elsewhere
as a policy, standard, or lifecycle rule, this document should reference that
canonical owner rather than duplicate it.

## 1. Canonical Repository Invariant

- Rule: use `/data/data/com.termux/files/home/Development/GitHub/arduino-cli-android`
  as the only canonical working copy for `arduino-cli-android` unless a
  different copy is explicitly authorized.
- Verification: before implementation, validation, or closeout, confirm `pwd`,
  `git rev-parse --show-toplevel`, `git status -sb`, branch, and `HEAD`.
- Failure action: stop immediately, do not continue product work, and reconcile
  the working copy before proceeding.

## 2. Evidence-Before-Claims Invariant

- Rule: do not claim behavior, completion, or success beyond the evidence
  actually collected.
- Verification: every completion report must name the command, environment,
  evidence level, and the exact scope proven.
- Failure action: narrow the claim, add evidence, or mark the question
  unresolved.

## 3. Native Termux Source-Of-Truth Invariant

- Rule: when Android and non-Android behavior disagree, native Termux is the
  source of truth for Android behavior.
- Verification: check whether the evidence comes from native Termux, host,
  emulation, or real hardware, and label it explicitly.
- Failure action: treat non-Termux evidence as preflight or supporting evidence
  only until native Termux confirms the claim.

## 4. Binary Freshness Invariant

- Rule: before implementation or before asking the user to run commands, rebuild
  the local binary with `go build -o arduino-cli .` and verify the rebuilt
  binary exposes the expected functionality.
- Verification: confirm the rebuilt binary exists and the relevant command or
  help surface behaves as expected.
- Failure action: stop and rebuild before any further validation or claims.

## 5. Smallest-Safe-Milestone Invariant

- Rule: resolve the current engineering uncertainty with the smallest proven
  milestone instead of expanding scope.
- Verification: confirm the change set answers one bounded engineering question
  and does not introduce unrelated work.
- Failure action: split the work and defer unrelated follow-on ideas.

## 6. Structured Evidence Collector Default Invariant

- Rule: when the structured evidence collector is available, use it as the
  default path for native diagnostics instead of manual copy/paste logs.
- Verification: confirm the collector can emit a structured evidence bundle for
  the task and that the output captures the relevant commands and context.
- Failure action: use the collector or add the smallest adapter needed to make
  the evidence structured.

## 7. Validation-Before-Completion Invariant

- Rule: completion requires appropriate validation, not just a plausible code
  change or a passing host test.
- Verification: identify the highest validation level actually achieved and
  ensure it matches the claim being made.
- Failure action: downgrade the claim or continue validating.

## 8. Documentation Currency Invariant

- Rule: when validated engineering knowledge changes, the canonical document
  that owns that knowledge must be updated before task closeout unless an
  explicit documented deferral exists.
- Verification: confirm the changed knowledge was reflected in the owning
  document, or confirm a deferral note explains why the update was postponed.
- Failure action: update the owning document or stop the closeout until the
  deferral is documented.

## 9. One Canonical Owner Per Engineering Claim Invariant

- Rule: each engineering claim must have one canonical owner document.
- Verification: check document ownership before adding or repeating a rule.
- Failure action: move the claim to the correct canonical document and leave
  other docs as short references.

## 10. README Overview-Only Invariant

- Rule: `README.md` is a front-door overview and pointer document, not a
  competing source of current state or future ordering.
- Verification: confirm current state lives in `STATUS.md` and future ordering
  lives in `ROADMAP.md`.
- Failure action: trim README back to a high-level overview and pointers.

## 11. Closeout Chain Invariant

- Rule: the closeout sequence is `evidence -> validation -> documentation ->
  commit -> push -> recovery reset -> next milestone`.
- Verification: complete each step in order and record the result in the
  canonical docs.
- Failure action: do not close the task until the missing step is completed or
  the gap is explicitly documented.

## 12. TASK_RECOVERY Reset Invariant

- Rule: after a task is fully completed, validated, committed, and pushed,
  clear `docs/android/TASK_RECOVERY.md` so the next task starts from a fresh
  live snapshot.
- Verification: confirm the recovery file contains the current task only and is
  blank or reset after closeout.
- Failure action: reset the file immediately before starting the next task.
