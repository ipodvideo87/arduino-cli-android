# Architecture Overview

## High-Level Flow

```text
Future GUI / Workspaces
  ↓
ACL Engine
  ↓
Workflows
  ↓
Firmware Package / Compatibility / Diagnostics / Transport
  ↓
Arduino CLI backend
  ↓
Toolchains and hardware
```

## How It Fits Together

- The GUI should call workflows, not Arduino CLI internals.
- Compile, package, validate, flash, monitor, diagnostics, and install are workflow
  concerns.
- `FirmwarePackage` is the boundary between the build system and the UI.
- `TransportManager` and the future USB bridge sit below upload workflows.
- `TransportStream` is the reusable byte-oriented boundary between transport
  sessions and future upload / monitor consumers.
- `UploadEngine` sits between `FirmwarePackage` / `FlashPlan` and future
  transport execution. The current foundation is dry-run only.
- The compatibility layer, scanner, verifier, bootstrap, and patch preview are
  shared infrastructure used by multiple workflows.

## Layer Responsibilities

- GUI / workspaces:
  - present beginner, advanced, and professional views
  - consume package and workflow reports
- ACL Engine:
  - orchestrate ordered steps
  - emit structured events and reports
- Workflows:
  - compile, package, validate, flash, monitor, diagnostics, install
- Firmware / compatibility / diagnostics:
  - package artifacts
  - validate outputs
  - classify compatibility and runtime readiness
- Transport stream foundation:
  - represent bounded read/write/cancel/close semantics
  - preserve diagnostics even when a live stream is unavailable
  - keep upload and monitor consumers transport-neutral
- Arduino CLI backend:
  - perform actual Arduino compile and toolchain operations
- Toolchains and hardware:
  - provide the real execution environment and physical device behavior

## Boundary Rules

- The GUI should not parse raw Arduino build internals when a package field exists.
- `analysis.json` should carry machine-readable analysis data for the GUI.
- `firmware.elf` and `firmware.map` remain the authoritative raw build artifacts.
- USB flashing must remain transport-based, not board-specific command glue.
