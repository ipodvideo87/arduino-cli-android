# Current Project Status

This document is the current snapshot of project progress. For the long-term mission,
see [MISSION.md](MISSION.md).

## Current Mission

Make Arduino CLI feel native on Android and turn the Android Compatibility Layer (ACL)
into reusable infrastructure for Linux-based development tools on Android.

## Completed Milestones

- The repository has been freshly cloned and rebuilt from GitHub.
- The Android patcher now patches Arduino toolchains during installation.
- The shared-library `.interp` bug has been fixed.
- Shared libraries without `PT_INTERP` are no longer treated as executables.
- Arduino CLI successfully installs the ESP32 core on Android.
- The ACL compatibility scanner can scan installed Arduino packages.
- Executables are classified as `loader-and-rpath`.
- Shared libraries are classified as `runtime-dependency-only` or `rpath-only`.
- Shell scripts are classified as `script-no-elf-patch`.
- Firmware images are ignored instead of being patched as host tools.
- The ACL execution planner exists.
- The initial execution backend exists behind `--apply`.
- The ACL toolchain compatibility validation mode exists and is covered by unit tests.
- Documentation policies are in place and kept aligned with implementation.

## Work In Progress

- Validate the installed toolchain against real Arduino packages before the first
  controlled compilation attempt.
- Prove a complete ESP32 toolchain compile end to end.
- Validate firmware upload on real hardware.
- Verify the full Arduino CLI workflow on native Android.

## Known Blockers

- Native Android execution still needs proof outside proot.
- Successful proot execution does not prove Android-native compatibility.
- A complete compile-and-upload loop has not yet been demonstrated.

## Next Engineering Milestone

Use `acl-scan validate-compat` on a fresh installed package tree, then use the validated
compatibility data to gate the first narrow Arduino compilation attempt.
