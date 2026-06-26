# Current Project Status

This document is the current snapshot of project progress. For the long-term mission,
see [MISSION.md](MISSION.md).

## Current Mission

Make Arduino CLI feel native on Android and turn the Android Compatibility Layer (ACL)
into reusable infrastructure for Linux-based development tools on Android.

Native Termux on Android is the production target and final validation environment.
Ubuntu inside `proot-distro` is used for development tooling and repository work, but it
does not replace native Android validation.

## Living Institutional Memory

- Project North Star: [docs/android/PROJECT_NORTH_STAR.md](docs/android/PROJECT_NORTH_STAR.md)
- Living Institutional Memory: [docs/android/LIVING_INSTITUTIONAL_MEMORY.md](docs/android/LIVING_INSTITUTIONAL_MEMORY.md)
- Engineering Decisions: [docs/android/ENGINEERING_DECISIONS.md](docs/android/ENGINEERING_DECISIONS.md)
- Architecture Overview: [docs/android/ARCHITECTURE_OVERVIEW.md](docs/android/ARCHITECTURE_OVERVIEW.md)
- Development Workflow: [docs/android/DEVELOPMENT_WORKFLOW.md](docs/android/DEVELOPMENT_WORKFLOW.md)
- Validation Policy: [docs/android/VALIDATION_POLICY.md](docs/android/VALIDATION_POLICY.md)
- Roadmap: [docs/android/ROADMAP.md](docs/android/ROADMAP.md)
- Validation Environment Research: [docs/android/VALIDATION_ENVIRONMENT_RESEARCH.md](docs/android/VALIDATION_ENVIRONMENT_RESEARCH.md)
- Emulated ARM64 Smoke Test Workflow: [docs/android/EMULATED_ARM64_SMOKE_TEST_WORKFLOW.md](docs/android/EMULATED_ARM64_SMOKE_TEST_WORKFLOW.md)
- USB Transport Research: [docs/android/USB_TRANSPORT_RESEARCH.md](docs/android/USB_TRANSPORT_RESEARCH.md)
- USB Transport Architecture: [docs/android/USB_TRANSPORT_ARCHITECTURE.md](docs/android/USB_TRANSPORT_ARCHITECTURE.md)
- Transport Provider Model: [docs/android/TRANSPORT_PROVIDER_MODEL.md](docs/android/TRANSPORT_PROVIDER_MODEL.md)
- Upload Workflow Preview: [docs/android/UPLOAD_WORKFLOW_PREVIEW.md](docs/android/UPLOAD_WORKFLOW_PREVIEW.md)
- Serial Monitor Preview: [docs/android/SERIAL_MONITOR_PREVIEW.md](docs/android/SERIAL_MONITOR_PREVIEW.md)
- Firmware Package Spec: [docs/specifications/FIRMWARE_PACKAGE_SPEC.md](docs/specifications/FIRMWARE_PACKAGE_SPEC.md)
- Validated Findings: [docs/android/VALIDATED_FINDINGS.md](docs/android/VALIDATED_FINDINGS.md)

Current validation posture:
- Native Termux remains the source of truth for Android behavior.
- Unit tests and CLI/build verification are useful, but not sufficient for Android claims.
- Emulated ARM64 smoke tests are useful preflight checks only and do not prove Android success.
- USB flashing is still unimplemented and must not be claimed as complete.

Repository cleanup:
- The queued-branch review is complete.
- `android-runtime-v2` is the canonical development branch for current Android work.
- Superseded `*-queued` branches are being retired only after their useful ideas
  have been preserved in docs, code, or validation history.

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
- Builtin tools installed from `packages/builtin/tools/...` now pass through the Android patcher, so builtin `ctags` follows the same loader and RPATH handling as other tools.
- ESP32-S3 Blink compilation now succeeds on native Termux.
- Native ESP32-S3 compile is validated on-device.
- `validate-compat` now treats Rust launcher wrappers as ELF executables again.
- ACL package building and runtime installation now preserve executable file modes instead of rewriting everything through a default `0644` create path.
- ACL compatibility validation now reports executable and script candidates that are missing execute bits.
- Runtime and package tests now cover executable-mode preservation through builder packaging, runtime installation, and build-runtime CLI packaging.
- Android tool/platform installation now repairs missing execute bits on executable and script payloads before ELF patching, so delegated launcher backends survive a clean `core install`.
- The ACL toolchain compatibility validation mode exists, is covered by unit tests, and now separates ignored resources from warnings.
- Documentation policies are in place and kept aligned with implementation.
- Native Termux compilation of a larger multi-file ESP32-S3 sketch is reported as successful in this workstream, including WiFi, ESP Async WebServer, Async TCP, LittleFS, Adafruit NeoPixel, Hash, and ESP32 core 3.3.10 usage.
- Firmware generated by the current build is reported as valid in this workstream because it was flashed from an Android application and booted the expected web server behavior.
- Android USB host access has been demonstrated on the same ESP32-S3 hardware by third-party apps, so the remaining work is transport integration rather than basic hardware discovery.
- Termux USB enumeration and permission flow are functional, but acquisition timing still needs root-cause analysis.
- The ACL transport manager now exists as a capability-based selector for native serial, Android USB fd, PTY, RFC2217, and future transports.
- The ACL firmware package foundation now exists, including the build manifest, flash plan, firmware package wrapper, and binary validator.
- The firmware package now also emits `analysis.json` and `README_FLASHING.txt`, and the ACL compile path uses metadata-first bootloader detection with an app-only fallback warning when the bootloader artifacts are incomplete.
- The ACL compatibility layer now exists as a rule-based decision layer for runtime, library, firmware, and transport compatibility.
- The ACL diagnostics workflow now exists as a shared pending/running/passed/warning/failed/skipped status model.
- The Android install patch pipeline now has a formal status-tracking foundation for download, extract, patch, runtime-fix, validation, register, self-test, and ready stages.
- The ACL engine now exists as the orchestration layer for ordered workflows, structured events, and machine-readable workflow reports.
- The experimental `acl workflow compile` command now exists and drives the ACL compile workflow on top of the existing compile service path.
- Successful compiles now produce a stable firmware package with manifest, flash plan, validation report, hashes, and copied artifacts.
- ESP32-S3 firmware packages now include the bootloader artifact and full-flash flash-plan entry when the Arduino core exposes the bootloader recipe metadata.
- The ACL compile workflow now reconstructs the firmware package from the compiler snapshot when the on-disk package is missing, so compile success and package-generation failure stay distinct.
- The validation-environment research docs and emulated ARM64 smoke-test workflow now exist as layered preflight guidance.
- The Android USB transport research, architecture, provider model, and upload
  / serial-monitor preview docs now exist as the design basis for future
  transport work.
- Platform and tool installs now flow through the shared Android patch pipeline instead of requiring manual permission or chmod repair.
- The Android install patch pipeline now has a dedicated runtime-permission repair stage that only fixes the copied loader execute bit, while the broader Android patch stage remains responsible for ELF rewriting.
- Platform and library install paths now consult the compatibility resolver so compatible versions can be preferred over source patching.
- ACL-facing scanner, verifier, patch-preview, and bootstrap wrapper packages now exist on top of the current architecture.
- The `arduino-cli acl` command group now exposes scanner, verifier, patch-preview, and bootstrap entry points for CLI-facing diagnostics.
- The `arduino-cli acl workflow` experimental subcommands now expose the ACL engine for bootstrap and diagnostics workflows.
- Bootstrap reporting now reuses the Android install patch pipeline in read-only form and carries the known `.acl/runtime/ld-linux-aarch64.so.1` permission-denied evidence path in validation details.

## Work In Progress

- Design a reusable native Android USB transport framework inside the existing Termux ecosystem.
- Define a generic Android upload and monitor bridge that stays board-agnostic, descriptor-driven, and provider-based.
- Validate firmware upload and serial monitor behavior through the new transport implementation on real hardware.
- Finish the Android preflight and dry-run reporting surfaces so they can be consumed by future UI workspaces.
- Keep the development workflow aligned with the two-environment model: native Termux as the source of truth, Ubuntu/proot as a tooling environment.
- Polish the ACL CLI diagnostic surfaces and wire them into future workspace UI layers.
- Validate the experimental ACL compile workflow end-to-end on real sketches and tighten the report surface before promoting it further.
- Extend the ACL engine with flash hooks once the lower-level hooks are ready to use end-to-end.

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
- GCC internal executables under `libexec/gcc/` now use wrapper launch instead of
  in-place interpreter rewriting; native Termux validation has advanced past the prior
  `cc1plus` startup crash.
- Native Android upload/flash architecture is not yet implemented end-to-end; it still needs a reusable USB host and serial bridge before upload is marked complete.
- The cause of the intermittent `termux-usb` acquisition/opening failures remains unknown.
- Successful proot execution does not prove Android-native compatibility.
- Firmware upload on real hardware has not yet been demonstrated.
- The Android USB transport bridge is not yet implemented, so end-to-end upload remains unproven.

## Next Engineering Milestone

Finish the generic native Android USB transport bridge, validate upload and monitor flow on real hardware, and move Android post-install repair into a shared automatic pipeline. In parallel, refine the ACL CLI diagnostic surfaces for future UI consumption.
