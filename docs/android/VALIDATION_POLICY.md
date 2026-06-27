# Validation Policy

This project uses evidence levels to avoid overclaiming.

This document is the canonical home for validation scope and provider-level policy. Other documents may summarize validation expectations for readability, but they should reference this document instead of duplicating the rules.

For task execution and report-writing behavior, see
[CODEX_OPERATING_MODEL.md](CODEX_OPERATING_MODEL.md) and
[DIAGNOSTIC_VALIDATION_STANDARD.md](DIAGNOSTIC_VALIDATION_STANDARD.md).

## Levels

The validation hierarchy is also defined in [DIAGNOSTIC_VALIDATION_STANDARD.md](DIAGNOSTIC_VALIDATION_STANDARD.md). That document defines how evidence should be reported; this document defines the validation scope for each level.

- Level 0: static analysis
- Level 1: unit / integration
- Level 2: host validation
- Level 3: ARM64 smoke validation
- Level 4: native Termux validation
- Level 5: real hardware validation

The repository's higher-level milestone claims should always refer to the highest validation level actually demonstrated.

## Validation Providers

Use a layered provider model instead of one hardcoded smoke test.

- Static analysis
- Unit / integration tests
- Host smoke tests
- ARM64 / QEMU smoke tests
- Native Termux
- Real hardware

Future providers may include:

- GitHub Actions
- desktop validation
- Android emulator validation if it can be proven practical
- Termux-like rootfs checks
- Android Bionic / sysroot compatibility checks

Every provider should emit:

- environment details
- architecture
- tool versions
- tests executed
- tests skipped
- results
- warnings
- limitations
- validation level or confidence boundary

## Rules

- Emulated ARM64 or QEMU smoke tests are useful preflight checks only.
- Emulated tests do not prove native Android success.
- Native Termux is required for Android compile claims.
- Real hardware is required for upload, flash, and runtime claims.
- Default bootstrap mode should not modify the host. Installation must be
  explicit.
- Passing Go tests alone does not validate Android behavior.
- Passing smoke tests does not validate native Termux behavior.
- Passing native Termux validation does not validate real hardware behavior.
- Every completion report should explicitly identify the validation level it achieved and the evidence that supports that claim.

## Claiming Success

- State the highest achieved validation level explicitly.
- Keep compile, packaging, validation, upload, flash, and runtime claims separate.
- Do not collapse a lower evidence level into a higher one.
- Label emulated validation as preflight only.
- Avoid vague success language such as `passed`, `works`, or `complete` unless it is immediately qualified with evidence, scope, and validation level.
