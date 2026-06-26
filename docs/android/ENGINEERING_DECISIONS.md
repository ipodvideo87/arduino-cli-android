# Engineering Decisions

These decisions use an ADR-style format.

## FirmwarePackage is the canonical build output

- Status: accepted
- Context: The GUI needs one stable package boundary instead of scattered build
  artifacts.
- Decision: `FirmwarePackage` is the canonical compile output for UI consumption.
- Alternatives considered: exposing raw build folders directly; teaching the GUI
  to parse build internals.
- Consequences: package generation becomes a first-class step and raw ELF/MAP files
  remain authoritative inputs, not the primary UI contract.
- Validation/evidence if available: firmware-package tests and compile workflow
  integration.

## ACL Engine orchestrates workflows

- Status: accepted
- Context: The project needs a reusable orchestration layer for compile, package,
  diagnostics, install, and future transport workflows.
- Decision: the ACL Engine owns ordered workflow execution and structured reporting.
- Alternatives considered: wiring UI layers directly to command helpers.
- Consequences: workflows stay testable and reusable across CLI and future GUI
  layers.
- Validation/evidence if available: `acl workflow` command surfaces and engine tests.

## Beginner, Advanced, Professional are UI layers

- Status: accepted
- Context: Users should get different information density without losing capability.
- Decision: the same engine and package model feeds all modes.
- Alternatives considered: separate beginner and pro code paths.
- Consequences: the backend stays single-sourced; the UI chooses disclosure level.
- Validation/evidence if available: workflow report separation in code and tests.

## Metadata-first firmware packaging

- Status: accepted
- Context: ESP32-S3 packages need a reliable way to derive bootloader and flash
  plan entries from actual Arduino build metadata.
- Decision: prefer generated flash arguments first, then build-property patterns,
  then filesystem resolution; fall back to app-only with warning if metadata is
  incomplete.
- Alternatives considered: hardcoding offsets or failing normal compile on missing
  bootloader metadata.
- Consequences: full-flash packages are produced when the core exposes enough data;
  app-only remains available when metadata is incomplete.
- Validation/evidence if available: firmware package tests and native Termux
  workflow behavior.

## QEMU/emulated testing is preflight only

- Status: accepted
- Context: Emulated tests can catch regressions early but do not prove Android-native
  behavior.
- Decision: treat QEMU and emulated ARM64 smoke tests as preflight checks only.
- Alternatives considered: using emulation as a substitute for native validation.
- Consequences: stronger evidence is required before Android success claims.
- Validation/evidence if available: native-vs-proot divergences and project policy.

## Native Termux validation is required for Android compile claims

- Status: accepted
- Context: Android behavior differs from proot and desktop Linux.
- Decision: compile success on Android must be validated in native Termux.
- Alternatives considered: relying on Ubuntu/proot or build success alone.
- Consequences: native Termux remains the source of truth for Android compile claims.
- Validation/evidence if available: project policy and native validation history.

## Real hardware validation is required for upload/flash claims

- Status: accepted
- Context: Upload and flash behavior depends on transport and device state.
- Decision: do not claim upload/flash success until tested on physical hardware.
- Alternatives considered: desktop-only or emulator-only upload verification.
- Consequences: upload milestones remain gated by real hardware proof.
- Validation/evidence if available: current upload milestone policy.

## USB upload must be transport-based and device-agnostic

- Status: accepted
- Context: Android USB access is permission-gated and board-specific upload logic
  does not scale.
- Decision: build a generic transport layer and keep upload workflows device-agnostic.
- Alternatives considered: hardcoding board-specific serial paths or VID/PID logic.
- Consequences: Arduino CLI stays unaware of Android internals and UI layers can
  reuse the same transport abstraction.
- Validation/evidence if available: transport architecture docs and Termux USB findings.

## Android compatibility belongs in reusable ACL infrastructure

- Status: accepted
- Context: Android compatibility work should benefit compile, install, diagnostics,
  and future tools.
- Decision: put Android-specific logic in ACL layers that can be reused and tested.
- Alternatives considered: patching only individual commands.
- Consequences: lower maintenance cost and clearer ownership boundaries.
- Validation/evidence if available: install patch pipeline, compatibility layer,
  and engine work.

## analysis.json is the GUI-facing build analysis source

- Status: accepted
- Context: The GUI should not need to parse raw ELF or map files.
- Decision: emit a structured `analysis.json` that carries the machine-readable
  build analysis surface, while `firmware.elf` and `firmware.map` remain the
  authoritative raw artifacts.
- Alternatives considered: making the GUI parse ELF/MAP directly.
- Consequences: future analysis work can evolve behind a versioned schema without
  breaking the UI contract.
- Validation/evidence if available: firmware package schema and analysis placeholder
  implementation.

