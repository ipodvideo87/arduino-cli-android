# Diagnostic Validation Standard

This document defines the repository standard for validation reporting.

This is the canonical home for diagnostic validation reporting requirements. Other documents may include brief reporting reminders for readability, but they should reference this standard rather than duplicating it.

For guidance on how Codex should apply this standard in practice, see
[CODEX_OPERATING_MODEL.md](CODEX_OPERATING_MODEL.md) and
[DECISION_FRAMEWORK.md](DECISION_FRAMEWORK.md).

## Principle

Evidence is more important than conclusions.

No engineering claim should stand alone as a slogan. Every claim must be tied to the concrete evidence that supports it, the environment in which that evidence was gathered, and the validation scope that evidence actually covers.

## Required Claim Structure

Every engineering claim should identify:

1. What was executed.
2. What environment it ran in.
3. What evidence was collected.
4. What engineering claims that evidence supports.
5. What it does not prove.
6. Remaining unknowns.
7. The recommended next validation step.

If any of these items are missing, the report is incomplete.

## Validation Levels

Validation levels describe increasing confidence, not interchangeable success labels.

### Level 0
Static analysis

Examples:

- code review
- architecture review
- linting
- format checks
- type or compile-time inspection that does not execute the target behavior

What it proves:

- the code or document is structurally consistent enough to inspect

What it does not prove:

- runtime behavior
- host compatibility
- Android compatibility
- hardware interaction

### Level 1
Unit / Integration

Examples:

- package unit tests
- package integration tests
- isolated test doubles

What it proves:

- local logic behaves as expected in controlled tests

What it does not prove:

- host execution
- Android execution
- real device interaction

### Level 2
Host Validation

Examples:

- `go build`
- `go test`
- CLI command execution on the host
- local package execution in a non-target environment

What it proves:

- the code builds and runs in the host environment used for validation

What it does not prove:

- Android behavior
- native Termux behavior
- real hardware behavior

### Level 3
ARM64 Smoke Validation

Examples:

- ARM64 emulator smoke tests
- QEMU-based preflight checks

What it proves:

- ARM64-oriented behavior is plausible under a controlled smoke environment

What it does not prove:

- native Android behavior
- native Termux behavior
- real hardware behavior

### Level 4
Native Termux Validation

Examples:

- execution on a real Android device in native Termux
- command-line behavior verified on-device
- runtime behavior verified in the Termux environment

What it proves:

- the behavior works in the production Android/Termux environment that matters for this repository

What it does not prove:

- all Android environments
- all device models
- real hardware upload success unless that specific hardware path was validated

### Level 5
Real Hardware Validation

Examples:

- upload to a physical board
- flash verification on attached hardware
- boot verification on the device under test

What it proves:

- the end-to-end hardware path works on the tested board and transport path

What it does not prove:

- broad board compatibility
- behavior on other boards
- behavior on other transports

## Highest Validation Level

Every completion report, milestone update, and substantive engineering summary must explicitly state the highest validation level achieved.

The highest level is the strongest level actually demonstrated for the specific claim being made.

Do not imply a higher validation level than the evidence supports.

## Evidence Reporting

Do not write:

- `tests passed`
- `validation passed`
- `feature complete`
- `works`

Instead, report:

- Environment
- Commands executed
- Evidence collected
- Capabilities proven
- Capabilities not proven
- Warnings
- Limitations
- Confidence level
- Next recommended validation

Prefer concrete artifacts over conclusions. Examples:

- command output
- logs
- file metadata
- version information
- test names
- observed device behavior
- generated reports

## Capability Reporting States

Every capability report should use one of the following states.

### Not Implemented

The capability does not exist in code.

### Implemented

The capability exists in code, but has not been validated in the target environment.

### Integrated

The capability is wired into the product or workflow, but not yet proven end to end in the target environment.

### Host Validated

The capability has been exercised successfully in a host validation environment.

### Smoke Validated

The capability has been exercised successfully in an ARM64 smoke or emulator environment.

### Native Termux Validated

The capability has been demonstrated on a real Android device in native Termux.

### Hardware Validated

The capability has been demonstrated on real attached hardware for the tested path.

### Production Ready

The capability has been validated sufficiently for the repository's current release or milestone expectations.

Production Ready is a stronger statement than Hardware Validated. It implies the implementation has the required evidence, coverage, and stability expectations for the intended milestone, not merely that one test succeeded once.

## Beginner / Advanced / Professional Output

These output modes represent increasing diagnostic depth.

### Beginner

- plain-language explanation
- what happened
- safe next step

### Advanced

- validation level
- evidence collected
- known limitations
- confidence

### Professional

- full diagnostic report
- environment
- commands executed
- versions
- validation level
- detailed evidence
- artifacts
- remaining risks
- next engineering milestone

The three modes should add depth and specificity, not just length.

## Reporting Rules

- State the validation level explicitly.
- Separate implemented, tested, validated, and production-ready claims.
- Distinguish environment success from target-environment success.
- Distinguish experimental behavior from validated behavior.
- Record what was observed, not only what was intended.
- Capture the next validation step when the evidence is incomplete.
- Use precise wording for limitations and unknowns.
- Keep local context brief and readable when it helps the current document, but do not create competing definitions for validation reporting.

## Recommended Completion Report Template

Environment:

- ...

Commands executed:

- ...

Evidence collected:

- ...

Validation level achieved:

- Level N

Capabilities proven:

- ...

Capabilities not yet proven:

- ...

Warnings:

- ...

Limitations:

- ...

Confidence:

- ...

Next recommended validation:

- ...

## Relationship to Existing Policy

This standard complements the existing validation policy in `VALIDATION_POLICY.md`.

Use `VALIDATION_POLICY.md` to determine the validation hierarchy and scope.
Use this document to determine how to report evidence and how to phrase claims.
