# Current Project Status

This document is the current snapshot of project progress. For the long-term mission,
see [MISSION.md](MISSION.md).

## Current Mission

Make Arduino CLI feel native on Android and turn the Android Compatibility Layer (ACL)
into reusable infrastructure for Linux-based development tools on Android.

Native Termux on Android is the production target and final validation environment.
Ubuntu inside `proot-distro` is used for development tooling and repository work, but it
does not replace native Android validation.

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
- The Android runtime installer preserves the loader execute bit and installs local
  linker aliases for the copied glibc runtime.
- ACL-owned `.acl/runtime` files are ignored by compatibility validation.
- Foreign Windows `.exe` tools are treated as unsupported warnings, not validation
  failures.
- The ACL execution planner exists.
- The initial execution backend exists behind `--apply`.
- ACL execution now distinguishes direct execution for Rust launcher wrappers and Android-native ELF from explicit-loader execution for patched Linux ELF binaries.
- ACL execution now sanitizes the launch environment to remove LD, QEMU, and PROOT leakage while preserving normal Termux and Android variables.
- ACL runtime discovery now prefers `ACL_RUNTIME_ROOT`, then `HOME/.arduino-cli-android/acl-runtime`, then `PREFIX/opt/arduino-cli-android/acl-runtime`, then `./acl-runtime`.
- ACL execution now emits structured diagnostics for dry-run and `--apply` so native Termux failures can be triaged with target, runtime, environment, and result evidence.
- `validate-compat` now treats Rust launcher wrappers as ELF executables again.
- ACL package building and runtime installation now preserve executable file modes instead of rewriting everything through a default `0644` create path.
- ACL compatibility validation now reports executable and script candidates that are missing execute bits.
- Runtime and package tests now cover executable-mode preservation through builder packaging, runtime installation, and build-runtime CLI packaging.
- Android tool/platform installation now repairs missing execute bits on executable and script payloads before ELF patching, so delegated launcher backends survive a clean `core install`.
- The ACL toolchain compatibility validation mode exists, is covered by unit tests, and now separates ignored resources from warnings.
- Documentation policies are in place and kept aligned with implementation.

## Work In Progress

- Validate the installed toolchain against real Arduino packages before the first
  controlled compilation attempt. The Termux ESP32 scan has already been used to tune the
  compatibility validation policy, but Arduino sketch compilation is still unproven.
- Prove a complete ESP32 toolchain compile end to end.
- Validate firmware upload on real hardware.
- Verify the full Arduino CLI workflow on native Android.
- Keep the development workflow aligned with the two-environment model: native Termux as
  the source of truth, Ubuntu/proot as a tooling environment.

## Known Blockers

- Native Android execution still needs proof outside proot.
- The ESP32 Rust launcher wrappers still break under explicit loader invocation because
  `/proc/self/exe` resolves to `ld`; ACL now prefers direct kernel exec for patched
  executables, but that path still needs native proof.
- The copied glibc loader still has to resolve its libraries without leaking Termux
  `LD_PRELOAD` or `LD_LIBRARY_PATH` state from the shell environment.
- The current Android runtime bundle vendors the Termux glibc runtime set in
  `internal/android/runtime`; any future non-vendored runtime library still needs a
  native Termux source, but the currently required set is already shipped with the
  repo.
- Successful proot execution does not prove Android-native compatibility.
- A complete compile-and-upload loop has not yet been demonstrated.
- The remaining native gap is broader Android compile-and-upload proof, not launcher execute-bit preservation.

## Next Engineering Milestone

Use `acl-scan validate-compat` on a fresh installed package tree, then use the validated
compatibility data to gate the first narrow Arduino compilation attempt.
