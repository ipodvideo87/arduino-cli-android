# Repository Guidelines

## Mission
This repository aims to become a production-quality Arduino CLI that runs natively on Android without chroots, PRoot, Docker, virtual machines, or a traditional Linux distribution. It is also the flagship implementation of the Android Compatibility Layer (ACL), a reusable framework for running Linux developer tools reliably on Android.

`MISSION.md` is the high-level mission document. Use it to distinguish long-term direction from current implementation status.

## Development Architecture
There are intentionally two Linux environments in this project:

1. Native Termux on Android is the production target. It runs under the Android application sandbox, Android kernel, Bionic, SELinux, and Android filesystem and permission rules. Any shipped binary must work here. When native Termux and another environment disagree, native Termux is the source of truth.
2. Ubuntu inside `proot-distro` is a development environment only. It is useful for Codex, GitHub tooling, and other utilities that are easier to run under Ubuntu/glibc, but it is not the deployment target and passing tests there is not sufficient.

Native Termux validation always takes priority over container or desktop validation.

## Android Research Policy
- Before making Android-specific changes or any architecture-level change, read:
  - [docs/android/PROJECT_NORTH_STAR.md](docs/android/PROJECT_NORTH_STAR.md)
  - [docs/android/LIVING_INSTITUTIONAL_MEMORY.md](docs/android/LIVING_INSTITUTIONAL_MEMORY.md)
  - [docs/android/ENGINEERING_DECISIONS.md](docs/android/ENGINEERING_DECISIONS.md)
  - [docs/android/ARCHITECTURE_OVERVIEW.md](docs/android/ARCHITECTURE_OVERVIEW.md)
  - [docs/android/DEVELOPMENT_WORKFLOW.md](docs/android/DEVELOPMENT_WORKFLOW.md)
  - [docs/android/VALIDATION_POLICY.md](docs/android/VALIDATION_POLICY.md)
  - [docs/android/ROADMAP.md](docs/android/ROADMAP.md)
  - [docs/android/VALIDATION_ENVIRONMENT_RESEARCH.md](docs/android/VALIDATION_ENVIRONMENT_RESEARCH.md)
  - [docs/android/EMULATED_ARM64_SMOKE_TEST_WORKFLOW.md](docs/android/EMULATED_ARM64_SMOKE_TEST_WORKFLOW.md)
  - [docs/specifications/FIRMWARE_PACKAGE_SPEC.md](docs/specifications/FIRMWARE_PACKAGE_SPEC.md)
  - [docs/android/VALIDATED_FINDINGS.md](docs/android/VALIDATED_FINDINGS.md)
- Before making Android-specific changes, also read [docs/android/ANDROID_COMPATIBILITY_RESEARCH.md](docs/android/ANDROID_COMPATIBILITY_RESEARCH.md).
- Before designing Android upload or flashing behavior, also read
  [docs/android/ANDROID_UPLOAD_ARCHITECTURE.md](docs/android/ANDROID_UPLOAD_ARCHITECTURE.md)
  and [docs/android/ANDROID_USB_TRANSPORT_FRAMEWORK.md](docs/android/ANDROID_USB_TRANSPORT_FRAMEWORK.md)
  and the current milestone document under `docs/android/`.
- If the behavior is not already documented there, research it before guessing.
- Add new confirmed Android/Termux findings to that file as part of the same work.
- Record the source of each finding with device evidence, command output, file metadata, or a linked upstream reference.
- Distinguish confirmed device evidence from inference or assumption.
- Do not stack speculative patches for Android issues. Prefer one root-cause fix that matches native Termux behavior.
- For Android-specific failures, identify the exact failing layer before changing code: archive extraction, filesystem syscall, ELF patching, dynamic loader, runtime library closure, environment propagation, compiler runtime, or Arduino CLI logic.
- Collect evidence before changing code: exact command, stderr, `strace` excerpt, ELF metadata, file modes, runtime paths, and environment differences between native Termux and Ubuntu/proot.
- Do not assume an existing ELF error means the file is missing; on Android/Termux, `ENOENT` for a present executable often means the PT_INTERP path is wrong or unpatched.
- Ubuntu/proot validation is useful, but native Termux remains the final authority.
- For milestone-level Android work, add a dedicated milestone document under `docs/android/` and keep the validation evidence, root cause, and remaining work there.
- For Android upload and flash work, do not assume desktop serial-node access. Check whether the device requires Android USB host permission and an app-mediated bridge.
- Prefer a Termux-native or existing-Termux-tooling solution for Android USB
  access. Do not introduce a new companion APK unless the Termux-native path is
  proven impossible.
- Treat Android USB work as a generic transport problem. Do not hardcode VID/PID, interface numbers, baud rates, or board-specific control sequences unless the evidence shows there is no generic path.
- Keep Arduino CLI unaware of Android internals. ACL owns USB discovery, permission acquisition, bridge selection, diagnostics, and PTY export.
- The Android post-install pipeline should be shared and automatic: extract, patch or repair, validate, register, and self-test. Platform and tool installers should call it; library installs should opt in only when they contain executable payloads.
- Treat compatibility as a first-class layer. Prefer compatible library selection over source patching, record compatibility decisions in build and install reports, and surface beginner-friendly and professional detail separately.
- Successful compiles should emit a first-class firmware package and install flows should pass through the shared Android patch pipeline, but USB flashing remains a separate milestone until proven on-device.
- Keep docs and `STATUS.md` updated when architecture, assumptions, or validation evidence changes.
- Do not claim success beyond the validation level actually achieved.
- The `arduino-cli acl` command group is the CLI-facing diagnostic surface for scanner, verifier, patch preview, and bootstrap workflows. Keep its beginner/professional separation aligned with the underlying ACL reports.
- The ACL engine is the orchestration boundary for future workspace and GUI calls. Prefer adding workflow steps there instead of teaching UI layers to call scanner/verifier/patch-preview utilities directly.
- The known `.acl/runtime/ld-linux-aarch64.so.1` permission-denied case should remain visible in validation and bootstrap evidence instead of being hidden behind a manual repair step.

## Core Principles
- Android-first engineering.
- Automation over manual configuration.
- Reusable engineering over one-off fixes.
- Preserve upstream compatibility whenever practical.
- Keep Android-specific code isolated when possible.
- Prefer deterministic, reproducible behavior.
- Favor ELF-analysis-driven patch plans over blind Android fixes.
- Use wrapper launch for GCC libexec binaries when `patchelf --set-interpreter` is unsafe.
- Ensure builtin tools are included in the Android patching flow when they are installed under `packages/builtin/tools/...`.
- Use a transport abstraction for Android upload paths instead of hardcoding `/dev/ttyUSB*` or `/dev/ttyACM*`.
- Prefer descriptor-driven USB serial detection and PTY-based tool integration over board-specific serial assumptions.

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

## Validation Policy
- Unit tests come first, then integration tests, then build verification.
- Native Termux validation is the final authority for Android behavior.
- A successful Ubuntu/proot run is useful evidence, but it does not prove Android compatibility.
- When debugging filesystem, dynamic linking, executable patching, process execution, or archive extraction, always distinguish Android/Bionic behavior from Ubuntu/glibc behavior.
- Prefer generic fixes that apply across toolchains and archive formats instead of tool-specific workarounds.
- For Android-specific failures, collect evidence first: exact command, stderr, `strace` excerpt, ELF metadata, file modes, runtime paths, and environment differences between native Termux and Ubuntu/proot.
- Identify the failing layer explicitly before changing code: archive extraction, filesystem syscall, ELF patching, dynamic loader, runtime library closure, environment propagation, compiler runtime, or Arduino CLI logic.
- Preserve normal Linux behavior unless Android requires a scoped fallback.
- Add regression tests for the exact incompatibility you fix.
- Document newly discovered Android-vs-Linux differences in the project docs.

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
- Push after completed validated change sets unless instructed otherwise.
- When unsure, document assumptions and open questions explicitly.

## Repository Status
- `STATUS.md` is the canonical snapshot of current project progress.
- `MISSION.md` defines the long-term mission; `STATUS.md` should describe the current state against that mission.
- `STATUS.md` should always identify the current mission.
- `STATUS.md` should always identify completed milestones.
- `STATUS.md` should always identify work in progress.
- `STATUS.md` should always identify known blockers.
- `STATUS.md` should always identify the next engineering milestone.
- Update `STATUS.md` whenever a milestone meaningfully changes.
- Treat the docs listed above as living institutional memory and keep them current.

## Proof Before Claims
- Document functionality as complete only after it has been demonstrated through reproducible validation.
- `Implemented` means the code path exists, but may not have been exercised beyond basic development.
- `Tested` means automated or manual tests have exercised the behavior in a controlled way.
- `Validated` means the behavior has been demonstrated reproducibly in the target context that matters for the milestone.
- `Experimental` means the behavior exists but remains incomplete, unstable, or insufficiently proven.
- `Planned` means the behavior is intended work and should not be described as current capability.

## Documentation Maintenance
- When architecture, tooling, workflows, runtime behavior, supported platforms, commands, validation procedures, or engineering decisions change, update the corresponding documentation within the same change whenever practical.
- Review and update the relevant documents, including `README`, `ARCHITECTURE.md`, `RUNTIME.md`, `ROADMAP.md`, `STATUS.md`, ADRs, `acl/docs/LAYERING.md`, and any other affected documentation.

## ACL Code Placement
- Read `acl/docs/LAYERING.md` before deciding where new ACL-related code belongs.
- Keep layer boundaries explicit when adding or moving ACL behavior.

## Repository Notes
- ACL work lives under `acl/`.
- Keep generated binaries out of git.
- Shell wrappers, runtime checks, and scanner tooling should stay small and explicit.
