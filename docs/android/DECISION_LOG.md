# Decision Log

This document is the task-level decision provenance log for the repository.

It records the path from question to evidence to decision so future contributors
can understand why a change was made without reconstructing the full chat or
history.

## Purpose

Use this log for decisions that are important enough to keep, but narrower than
an ADR.

Examples:

- choosing between two validation strategies
- selecting a next milestone
- narrowing a transport boundary
- resolving an uncertainty that shaped implementation

## 2026-07-01 - propagate target-chip metadata from compile properties

- Question: why did firmware-package validation report `target chip metadata is
  not set` after the full-flash package milestone had already completed?
- Evidence: `internal/acl/firmware/package.go` warns only when both manifest and
  flash-plan target-chip fields are empty; `commands/service_compile.go` and
  `internal/acl/engine/compile.go` previously built `FirmwarePackage` input
  without populating `TargetChip`; the compile properties already contain
  `build.mcu`; host tests now show `build.mcu` propagates into
  `FirmwarePackage` generation and the validator no longer emits the warning
  when metadata is present.
- Decision: resolve target-chip metadata at compile-input construction time by
  reading `build.mcu` first and falling back to the known board identifier, and
  keep the package validator unchanged.
- Alternatives: suppress the warning; document it as expected behavior; teach
  the validator to infer missing metadata; add a separate metadata schema.
- Confidence: medium -> high
- Uncertainty removed: the warning came from missing propagation, not from a
  validator bug.
- Uncertainty remaining: native Termux package output should be rerun to
  confirm the on-device package now reports target-chip metadata without the
  warning.
- Docs updated: `VALIDATED_FINDINGS.md`, `STATUS.md`, `TASK_RECOVERY.md`
- Roadmap impact: none
- Follow-up: rerun the native Termux package milestone output and record the
  updated on-device finding if it still matches the host validation

## What A Log Entry Must Answer

Every entry should make the following obvious:

- the question being answered
- the evidence used
- the decision made
- the alternatives considered
- the confidence change
- the uncertainty that remains
- the docs that were updated
- the roadmap effect

## Recommended Entry Shape

```md
## YYYY-MM-DD - short title

- Question: ...
- Evidence: ...
- Decision: ...
- Alternatives: ...
- Confidence: before -> after
- Uncertainty removed: ...
- Uncertainty remaining: ...
- Docs updated: ...
- Roadmap impact: ...
- Follow-up: ...
```

## Relationship To Other Records

- Use `ENGINEERING_DECISIONS.md` for durable architecture decisions.
- Use `VALIDATED_FINDINGS.md` for evidence-backed findings.
- Use `UNCERTAINTY_REGISTER.md` for open questions.
- Use `LESSONS_LEARNED.md` for distilled reusable lessons.

The decision log connects those records but does not replace them.

## Entry Rules

- Keep entries chronological.
- Keep entries short enough to scan.
- Do not duplicate a full ADR.
- Do not omit uncertainty just because the decision was made.
- Link out to the evidence source when the full detail belongs elsewhere.

## Closure Rule

When a task is closed, the decision log entry should explain why the remaining
uncertainty is acceptable, or where it was moved if it is still active.

## 2026-07-02 - adopt EOS through a manifest-first boundary

- Question: what is the cleanest long-term contract between EOS and
  `arduino-cli-android` for the first real adoption pilot?
- Evidence: repository review showed `AGENTS.md` currently mixes universal
  methodology with Android-specific repository guidance; EOS already has a
  canonical project adoption model; the repo had no pre-existing adoption
  manifest; `STATUS.md` and `ROADMAP.md` already separate current state from
  future sequencing.
- Decision: implement the adoption boundary as `eos.project.json` plus a thin
  `AGENTS.md` overlay, and treat the manifest as the canonical adoption
  contract.
- Alternatives: keep AGENTS canonical; add a separate adoption lock file; leave
  adoption implicit in prose.
- Confidence: medium -> medium-high
- Uncertainty removed: the project needs an explicit manifest boundary more than
  another prose policy layer.
- Uncertainty remaining: whether the manifest should gain a dedicated
  safety/constraints section before EOS Foundation v0.1.0.
- Docs updated: `MILESTONE_EOS_PROJECT_ZERO.md`, `ENGINEERING_DECISIONS.md`,
  `VALIDATED_FINDINGS.md`, `STATUS.md`, `ROADMAP.md`, `TASK_RECOVERY.md`
- Roadmap impact: Project Zero becomes the next repo-level EOS adoption step
- Follow-up: validate the manifest shape against EOS schema and decide whether
  any additional AGENTS reduction is justified
