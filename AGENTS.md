# Repository Guidelines

## Mission
This repository aims to become a production-quality Arduino CLI that runs natively on Android without chroots, PRoot, Docker, virtual machines, or a traditional Linux distribution. It is also the flagship implementation of the Android Compatibility Layer (ACL), a reusable framework for running Linux developer tools reliably on Android.

`MISSION.md` is the high-level mission document. Use it to distinguish long-term direction from current implementation status.

## Core Principles
- Android-first engineering.
- Automation over manual configuration.
- Reusable engineering over one-off fixes.
- Preserve upstream compatibility whenever practical.
- Keep Android-specific code isolated when possible.
- Prefer deterministic, reproducible behavior.

## Project Goals
- Run Arduino CLI natively on Android.
- Build and maintain ACL as a general-purpose compatibility platform.
- Detect Linux ELF executables and analyze their dependencies and runtime requirements.
- Patch compatible executables safely.
- Build portable runtimes when required.
- Support ESP32, ESP8266, RP2040, AVR, STM32, and future platforms.

## Current Priorities
Work in this order unless a higher-priority issue is clearly required:
1. Complete ACL runtime architecture.
2. Implement reliable ELF analysis and compatibility verification.
3. Implement safe ELF patching.
4. Execute `esptool` successfully.
5. Execute the ESP32 compiler toolchain.
6. Compile Arduino sketches on Android.
7. Upload firmware to a physical ESP32-S3.

Do not claim a milestone is complete until it has been verified through real-world testing.

## Engineering Expectations
- Understand the existing architecture before changing it.
- Reuse existing components whenever possible.
- Avoid duplicating logic or creating application-specific hacks.
- Keep changes modular, maintainable, and reviewable.
- Update documentation when behavior changes.
- Never fabricate APIs, functionality, or test results.
- Clearly mark experimental behavior.

## Documentation Policy
- Documentation is part of the codebase, not a separate cleanup task.
- Whenever functionality changes, review and update the relevant documentation as part of the same change.
- Documentation should describe what the repository currently proves, not what is merely planned.
- Future work must be clearly identified as planned, experimental, or unverified.
- Avoid absolute statements unless they have been validated.
- Every change should leave the documentation in a more accurate state than before.

## Repository Status
- `STATUS.md` is the canonical snapshot of current project progress.
- `MISSION.md` defines the long-term mission; `STATUS.md` should describe the current state against that mission.
- `STATUS.md` should always identify the current mission.
- `STATUS.md` should always identify completed milestones.
- `STATUS.md` should always identify work in progress.
- `STATUS.md` should always identify known blockers.
- `STATUS.md` should always identify the next engineering milestone.
- Update `STATUS.md` whenever a milestone meaningfully changes.

## Proof Before Claims
- Document functionality as complete only after it has been demonstrated through reproducible validation.
- `Implemented` means the code path exists, but may not have been exercised beyond basic development.
- `Tested` means automated or manual tests have exercised the behavior in a controlled way.
- `Validated` means the behavior has been demonstrated reproducibly in the target context that matters for the milestone.
- `Experimental` means the behavior exists but remains incomplete, unstable, or insufficiently proven.
- `Planned` means the behavior is intended work and should not be described as current capability.

## Documentation Maintenance
- When architecture, tooling, workflows, runtime behavior, supported platforms, commands, validation procedures, or engineering decisions change, update the corresponding documentation within the same change whenever practical.
- Review and update the relevant documents, including `README`, `ARCHITECTURE.md`, `RUNTIME.md`, `ROADMAP.md`, `STATUS.md`, ADRs, and any other affected documentation.

## Repository Notes
- ACL work lives under `acl/`.
- Keep generated binaries out of git.
- Shell wrappers, runtime checks, and scanner tooling should stay small and explicit.
