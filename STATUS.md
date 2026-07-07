# Current Project Status

This document is the current snapshot of project progress. For the long-term mission,
see [MISSION.md](MISSION.md).

## Current Mission

Make Arduino CLI feel native on Android and turn the Android Compatibility Layer (ACL)
into reusable infrastructure for Linux-based development tools on Android.

Native Termux on Android is the production target and final validation environment.
Ubuntu inside `proot-distro` is used for development tooling and repository work, but it
does not replace native Android validation.

Project Zero is now the first EOS adoption pilot for this repository. It is
used to validate the EOS project adoption model against a production-scale
engineering codebase.

## STATUS Versus ROADMAP

`STATUS.md` is authoritative for the current snapshot:

- current mission
- completed milestones
- active work
- known blockers
- the next engineering milestone

`docs/android/ROADMAP.md` is authoritative for future ordering:

- long-range milestone sequence
- planned follow-on work
- broad direction beyond the current snapshot

Update both when a validated change changes the current state and the future
milestone order. If the two documents conflict, `STATUS.md` wins for the current
snapshot and `ROADMAP.md` wins for future ordering.

## Living Institutional Memory

- Project North Star: [docs/android/PROJECT_NORTH_STAR.md](docs/android/PROJECT_NORTH_STAR.md)
- Living Institutional Memory: [docs/android/LIVING_INSTITUTIONAL_MEMORY.md](docs/android/LIVING_INSTITUTIONAL_MEMORY.md)
- Engineering Decisions: [docs/android/ENGINEERING_DECISIONS.md](docs/android/ENGINEERING_DECISIONS.md)
- Engineering Knowledge Framework: [docs/android/ENGINEERING_KNOWLEDGE_FRAMEWORK.md](docs/android/ENGINEERING_KNOWLEDGE_FRAMEWORK.md)
- Architecture Overview: [docs/android/ARCHITECTURE_OVERVIEW.md](docs/android/ARCHITECTURE_OVERVIEW.md)
- Development Workflow: [docs/android/DEVELOPMENT_WORKFLOW.md](docs/android/DEVELOPMENT_WORKFLOW.md)
- Validation Policy: [docs/android/VALIDATION_POLICY.md](docs/android/VALIDATION_POLICY.md)
- Roadmap: [docs/android/ROADMAP.md](docs/android/ROADMAP.md)
- Validation Environment Research: [docs/android/VALIDATION_ENVIRONMENT_RESEARCH.md](docs/android/VALIDATION_ENVIRONMENT_RESEARCH.md)
- Emulated ARM64 Smoke Test Workflow: [docs/android/EMULATED_ARM64_SMOKE_TEST_WORKFLOW.md](docs/android/EMULATED_ARM64_SMOKE_TEST_WORKFLOW.md)
- USB Transport Research: [docs/android/USB_TRANSPORT_RESEARCH.md](docs/android/USB_TRANSPORT_RESEARCH.md)
- USB Transport Architecture: [docs/android/USB_TRANSPORT_ARCHITECTURE.md](docs/android/USB_TRANSPORT_ARCHITECTURE.md)
- Transport Provider Model: [docs/android/TRANSPORT_PROVIDER_MODEL.md](docs/android/TRANSPORT_PROVIDER_MODEL.md)
- Transport Stream Foundation: [docs/android/TRANSPORT_STREAM_FOUNDATION.md](docs/android/TRANSPORT_STREAM_FOUNDATION.md)
- Transport API Stabilization: [docs/android/TRANSPORT_API_STABILIZATION.md](docs/android/TRANSPORT_API_STABILIZATION.md)
- Termux USB Provider Validation Checklist: [docs/android/MILESTONE_TERMUX_USB_PROVIDER.md](docs/android/MILESTONE_TERMUX_USB_PROVIDER.md)
- Native Transport Stream Validation: [docs/android/MILESTONE_NATIVE_TRANSPORT_STREAM_VALIDATION.md](docs/android/MILESTONE_NATIVE_TRANSPORT_STREAM_VALIDATION.md)
- V1 Release Criteria: [docs/android/V1_RELEASE_CRITERIA.md](docs/android/V1_RELEASE_CRITERIA.md)
- Upload Workflow Preview: [docs/android/UPLOAD_WORKFLOW_PREVIEW.md](docs/android/UPLOAD_WORKFLOW_PREVIEW.md)
- Serial Monitor Preview: [docs/android/SERIAL_MONITOR_PREVIEW.md](docs/android/SERIAL_MONITOR_PREVIEW.md)
- Firmware Package Spec: [docs/specifications/FIRMWARE_PACKAGE_SPEC.md](docs/specifications/FIRMWARE_PACKAGE_SPEC.md)
- EOS Project Zero: [docs/android/MILESTONE_EOS_PROJECT_ZERO.md](docs/android/MILESTONE_EOS_PROJECT_ZERO.md)
- Validated Findings: [docs/android/VALIDATED_FINDINGS.md](docs/android/VALIDATED_FINDINGS.md)

Current validation posture:
- Native Termux remains the source of truth for Android behavior.
- Unit tests and CLI/build verification are useful, but not sufficient for Android claims.
- Emulated ARM64 smoke tests are useful preflight checks only and do not prove Android success.
- Canonical working repository for active work is
  `/data/data/com.termux/files/home/Development/GitHub/arduino-cli-android`;
  repo-local binaries should be rebuilt from that tree before asking for new
  validation.
- The next product milestone now has a dedicated native-Termux package-validation doc at `docs/android/MILESTONE_NATIVE_FULL_FLASH_BOOTLOADER_PACKAGE_VALIDATION.md`.
- Native Termux package-validation evidence now confirms the ESP32-S3 sketch
  package is `full-flash`, all required package artifacts exist, and the
  prepare-only upload consumer accepts the package in dry-run mode.
- The package warning was resolved by rebuilding the repo-local `arduino-cli`
  binary with `go build -o arduino-cli .` and regenerating the package; the
  regenerated package now reports `target_chip = esp32s3` in the manifest,
  flash plan, and validation report, and the validation warnings are now
  `null`.
- The Termux USB provider now has a native-Termux validation checklist documenting expected outputs for discovery, permission, stale-path handling, and fd handoff evidence.
- Discovery and permission acquisition for the Termux USB provider are now native-Termux validated on Samsung A17 / Android 16; byte-stream and upload behavior remain unvalidated.
- The Termux USB provider now also exposes a bounded `acl transport probe-fd`
  / `probe-fd-helper` stream-diagnostics path that records `TERMUX_USB_FD`
  evidence without claiming a usable byte stream.
- The Termux USB provider now also exposes `acl transport stream-validate`, a
  bounded stream wrapper diagnostic that can optionally exercise a one-byte
  read/write probe without claiming byte-stream readiness.
- Native Termux validation now shows that `TERMUX_USB_FD` is inspectable but
  raw write on the current fd can return `invalid argument`; the helper keeps
  read and write probes separate so the diagnostics do not overclaim generic
  byte-stream behavior.
- The Termux USB provider now also exposes a diagnostic-only USB topology
  bridge foundation that wraps the Termux fd through the helper handoff path
  and reports descriptors, interfaces, and endpoints without sending payload
  data or claiming claim/release readiness.
- The transport stream foundation now exists as a reusable bounded stream
  wrapper and report model; the stream state is still experimental until native
  Termux byte-stream validation proves read/write behavior.
- The transport API is now considered stabilizing: the provider/manager/session/
  stream contracts are acceptable for upload-engine foundation work, while the
  byte-stream implementation itself remains experimental.
- The probe now uses `termux-usb -r -E -e <helper> <device>` and the helper
  accepts both environment and positional-argument fd sources, which resolves
  the earlier TERMUX_USB_FD mismatch.
- Native Termux has now confirmed the working probe shape:
  `termux-usb -r -E -e "./arduino-cli acl transport probe-fd-helper --json" /dev/bus/usb/001/002`.
- Native Termux has now validated `./arduino-cli acl transport probe-fd --device /dev/bus/usb/001/002 --details`
  and `./arduino-cli --json acl transport probe-fd --device /dev/bus/usb/001/002`.
- Native Termux `stream-validate` evidence now shows baseline handoff success,
  `EOF` on the read probe, and `write TERMUX_USB_FD: invalid argument` on the
  write probe; that is evidence against treating the fd as a normal generic
  byte-stream endpoint.
- Native Termux transport stream validation now classifies the current Termux
  USB fd path as diagnostic-only, not stream-capable: `fd_source=environment`,
  `handoff_mode=env`, `fd_valid=true`, `fd_inspectable=true`, raw byte-stream
  write is unsupported on this path, and read remains unproven because it was
  skipped after write failure.
- The current blocker is not permission or fd handoff; the next safe evidence
  step, if more proof is needed, is a read-only fresh-helper probe that does
  not attempt write first.
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
- The Termux USB transport provider now exists as a diagnostic-only implementation with discovery, permission, session, endpoint-export, and bounded fd-probe contracts, plus `arduino-cli acl transport list|diagnose|acquire|probe-fd|stream-status` CLI surfaces.
- The Termux USB transport provider now also exposes `stream-status` as a
  diagnostic alias for the same stream state report.
- The diagnostic-only USB topology bridge foundation is now native-Termux
  validated and reports descriptors, interfaces, endpoints, and topology source
  evidence without payload transfers.
- Native Termux claim/release evidence now shows interface 0 and interface 1
  returning `LIBUSB_ERROR_BUSY` while interface 2 can be claimed and released
  successfully on the detected Espressif device; interface 2 exposes vendor-
  specific bulk OUT `0x02` and bulk IN `0x83`, and no payload transfers were
  attempted.
- Native Termux has validated the USB topology bridge foundation on the target
  device: descriptors, interfaces, endpoints, vendor/product identity, serial
  identity, and the `libusb` topology source are visible without payload
  transfers.
- The Termux USB transport provider now also exposes diagnostic-only interface
  claim/release validation for the native helper handoff path; native Termux
  evidence now shows interface 2 succeeding while interfaces 0 and 1 are busy.
- The ACL firmware package foundation now exists, including the build manifest, flash plan, firmware package wrapper, and binary validator.
- The firmware package now also emits `analysis.json` and `README_FLASHING.txt`, and the ACL compile path uses metadata-first bootloader detection with an app-only fallback warning when the bootloader artifacts are incomplete.
- Native full-flash bootloader package validation is complete on native
  Termux.
- The ACL upload engine foundation now exists as a transport-neutral prepare-only planner/executor stack that consumes firmware packages and flash plans without opening real transport streams or sending bytes.
- The ACL workflow upload command is now documented and validated as a positional prepare-only command (`acl workflow upload <firmware-package>`); `--dry-run` and `--package` are not part of the contract.
- The upload execution report surface now keeps the canonical package, plan, diagnostics, result, and progress data in one place and de-duplicates repeated professional details.
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
- The transport provider skeleton now exists as compile-safe provider,
  permission, session, endpoint-export, and diagnostics contracts with
  fake-provider tests, but it still does not touch real Android USB APIs.
- Platform and tool installs now flow through the shared Android patch pipeline instead of requiring manual permission or chmod repair.
- The Android install patch pipeline now has a dedicated runtime-permission repair stage that only fixes the copied loader execute bit, while the broader Android patch stage remains responsible for ELF rewriting.
- Platform and library install paths now consult the compatibility resolver so compatible versions can be preferred over source patching.
- ACL-facing scanner, verifier, patch-preview, and bootstrap wrapper packages now exist on top of the current architecture.
- The `arduino-cli acl` command group now exposes scanner, verifier, patch-preview, and bootstrap entry points for CLI-facing diagnostics.
- The `arduino-cli acl workflow` experimental subcommands now expose the ACL engine for bootstrap and diagnostics workflows.
- Bootstrap reporting now reuses the Android install patch pipeline in read-only form and carries the known `.acl/runtime/ld-linux-aarch64.so.1` permission-denied evidence path in validation details.
- Governance framework batch 1 is now in place:
  - `docs/android/CODEX_OPERATING_MODEL.md`
  - `docs/android/ENGINEERING_PRINCIPLES.md`
  - `docs/android/DECISION_FRAMEWORK.md`
  - `docs/android/DOCUMENTATION_ARCHITECTURE.md`
  - updated cross-references in `AGENTS.md`, `ROADMAP.md`, and the validation docs
- Governance framework batch 1.5 is now in place:
  - `docs/android/ENGINEERING_METHODOLOGY.md`
  - `docs/android/REPOSITORY_GOVERNANCE.md`
  - `docs/android/ARCHITECTURE_REVIEW_PROCESS.md`
  - `docs/android/TECHNICAL_DEBT_POLICY.md`
  - `docs/android/INTERFACE_STABILITY_POLICY.md`
  - updated cross-references in `AGENTS.md`, `STATUS.md`, `ROADMAP.md`, and
    `LIVING_INSTITUTIONAL_MEMORY.md`
- Governance framework batch 2 has started with high-risk documentation
  ownership cleanup across validation, transport, and workflow preview docs.
- Governance framework batch 3 is now in place:
  - `docs/android/ENGINEERING_KNOWLEDGE_FRAMEWORK.md`
  - `docs/android/ENGINEERING_LIFECYCLE.md`
  - `docs/android/DECISION_LOG.md`
  - `docs/android/UNCERTAINTY_REGISTER.md`
  - `docs/android/CONFIDENCE_MODEL.md`
  - `docs/android/LESSONS_LEARNED.md`

## Work In Progress

- Project Zero EOS adoption pilot:
  - canonical project adoption manifest
  - overlay-only `AGENTS.md` boundary
  - ownership classification and compatibility assessment
  - adoption milestone documentation
- Design a reusable native Android USB transport framework inside the existing Termux ecosystem.
- Define a generic Android upload and monitor bridge that stays board-agnostic, descriptor-driven, and provider-based.
- Validate firmware upload and serial monitor behavior through the new transport implementation on real hardware.
- Keep the ACL upload prepare-only executor aligned with the GUI contract.
- Implement the first real transport provider runtime after the contract layer is proven stable.
- Validate the native Termux transport stream boundary with a read-only fresh-helper probe if more evidence is needed.
- Research whether any interface-2 transfer diagnostic is safe and meaningful before adding live bulk I/O.
- Start bounded byte-stream bridge foundation work only after read/write probing has a concrete, native-Termux validation plan.
- The ACL evidence collection command now exists with unit-test coverage; native Termux validation of its output remains pending.
- Finish the Android preflight and dry-run reporting surfaces so they can be consumed by future UI workspaces.
- Keep the development workflow aligned with the two-environment model: native Termux as the source of truth, Ubuntu/proot as a tooling environment.
- Polish the ACL CLI diagnostic surfaces and wire them into future workspace UI layers.
- Validate the experimental ACL compile workflow end-to-end on real sketches and tighten the report surface before promoting it further.
- Extend the ACL engine with flash hooks once the lower-level hooks are ready to use end-to-end.
- Apply the governance framework batch 1 guidance to later documentation and
  architecture batches without turning this milestone into a full doc cleanup.
- Apply the repository governance and evolution framework to later interface,
  schema, and architecture work without rewriting the whole docs set at once.
- Continue applying the governance framework to the remaining high-risk Android
  docs, keeping canonical definitions centralized and local summaries concise.

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
- Termux USB discovery and permission acquisition are now validated on Samsung A17 / Android 16, but `TERMUX_USB_FD` handoff remains unproven.
- The bounded TERMUX_USB_FD probe exists in code and tests, and native Termux
  validation of the official `probe-fd` command now confirms fd handoff,
  observation, and inspection.
- The transport stream foundation exists in code and unit tests, but native
  Termux byte-stream read/write validation is still pending.
- The upload engine foundation exists in code and unit tests as a prepare-only
  planner/executor stack; real upload execution is still pending native
  transport proof.
- Successful proot execution does not prove Android-native compatibility.
- Firmware upload on real hardware has not yet been demonstrated.
- The Android USB transport bridge is not yet implemented, so end-to-end upload remains unproven.
- Native full-flash bootloader package validation is complete on native
  Termux.

## Next Engineering Milestone

Validate the native Termux transport stream boundary with a fresh read-only
helper probe if more evidence is needed, while keeping upload, flashing, and
serial monitor out of scope.
