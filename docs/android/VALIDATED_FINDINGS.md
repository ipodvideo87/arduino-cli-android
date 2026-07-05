# Validated Findings

This file is a logbook of tested findings from the project history.

## 2026-07-05

- Environment: native Termux on the target Android device
- Command/evidence:
  `./arduino-cli acl transport probe-fd --json --device /dev/bus/usb/001/002`,
  `./arduino-cli acl transport stream-validate --json --device /dev/bus/usb/001/002 --details --validate-read --validate-write`, and
  `termux-usb -r -E -e "./arduino-cli acl transport stream-validate-helper --json --validate-read --validate-write --timeout 2s" /dev/bus/usb/001/002`
- Result: native Termux fd handoff is proven and inspectable with `fd_source=environment`, `handoff_mode=env`, `fd_valid=true`, and `fd_inspectable=true`; permission and handoff are not the blocker; raw byte-stream write is unsupported on this path; read remains unproven because it was skipped after write failure; the current transport boundary is diagnostic-only, not stream-capable
- Confidence: high
- Notes: no upload, flashing, serial monitor, or source changes were attempted. The next safe step, if more evidence is needed, is a read-only fresh-helper probe that does not attempt write first

- Environment: native Termux on the target Android device
- Command/evidence: `./arduino-cli acl evidence collect --device /dev/bus/usb/001/002 --output-dir .acl/evidence` and inspection of `.acl/evidence/evidence-20260705T231207Z-867257893921.json`
- Result: the collector writes a nested evidence schema with fields under `repository.*`, `environment.*`, `binary.*`, and `device_path`; a correct jq summary example is `jq -r '{repo: .repository.root, branch: .repository.branch, commit: .repository.commit, native_termux: .environment.native_termux, binary: .binary.path, device: .device_path}'`; the earlier flat-field jq query was a query mismatch rather than a collector failure
- Confidence: high
- Notes: the collector validation remains successful on native Termux, and the warning-level transport details inside the evidence bundle are expected for the current diagnostic-only transport boundary

## 2026-07-04

- Environment: native Termux on the target Android device
- Command/evidence: package validation checks followed by
  `./arduino-cli acl workflow upload /data/data/com.termux/files/home/Development/Sketches/esp32ultimate/build/esp32.esp32.esp32s3/firmware-package --details`
- Result: the validated `full-flash` package is accepted by the prepare-only
  upload consumer with `dry-run=true`, `prepare-only=true`, `operations=4`,
  `ordered=true`, and `complete=true`; the consumer reports `stream required:
  false` and keeps the run read-only; after rebuilding the repo-local
  `arduino-cli` binary with `go build -o arduino-cli .` and regenerating the
  package, the regenerated package now reports `target_chip = esp32s3` in the
  manifest, flash plan, and validation report, and the validation warnings are
  now `null`
- Confidence: high
- Notes: the earlier jq loop failure was a diagnostic-script portability issue,
  not a package validation failure. The previous target-chip warning came from
  stale binary / stale package state. Future native-Termux command blocks
  should prefer direct file checks, consistent quoting, and simpler shell forms
  when verifying artifact existence. Future recurrence should check binary
  freshness before patching source

## 2026-07-03

- Environment: repository documentation review
- Command/evidence: `AGENTS.md`, `STATUS.md`, `ROADMAP.md`,
  `DECISION_LOG.md`, `ENGINEERING_DECISIONS.md`,
  `docs/android/MILESTONE_EOS_PROJECT_ZERO.md`, and `eos.project.json`
- Result: the repository now states the EOS adoption boundary as a
  manifest-first contract with a thin `AGENTS.md` overlay, the canonical
  ownership split is documented, and Project Zero is visible in the current
  status and roadmap as the next adoption milestone
- Confidence: medium-high
- Notes: this is static review evidence only. It confirms the adoption docs are
  aligned, but it does not answer the remaining EOS schema or constraints
  question

## 2026-07-03

- Environment: repository workspace in a PRoot-based Termux toolchain shell,
  not native Android Termux
- Command/evidence: compile of `esp32ultimate` with the resolved Arduino
  libraries, package inspection under
  `/data/data/com.termux/files/home/Development/Sketches/esp32ultimate/build/esp32.esp32.esp32s3/firmware-package`,
  and `./arduino-cli acl workflow upload "$PKG_DIR" --details`
- Result: compile now produces a `full-flash` firmware package with
  `manifest.json`, `flash-plan.json`, `validation-report.json`, `analysis.json`,
  and `README_FLASHING.txt`; the manifest records bootloader, partition table,
  boot_app0, and application artifacts; the prepare-only upload consumer
  accepts the package in dry-run mode; the validation report still warns that
  `target chip metadata is not set`
- Confidence: medium-high
- Notes: this is strong package-boundary evidence, but it is not native-Termux
  evidence and it does not resolve whether the target-chip warning reflects a
  metadata propagation gap or the current package state

## 2026-06-27

- Environment: repository unit-test environment
- Command/evidence: `go test ./internal/acl/upload/... ./internal/acl/... ./internal/cli/... ./commands/...` and `go build -o arduino-cli .`
- Result: the upload workflow now uses a prepare-only executor, the positional `acl workflow upload <firmware-package>` contract remains intact, and the executor report compiles and round-trips through unit tests
- Confidence: high
- Notes: this validates the code path and report shape only; it does not validate real upload, flashing, or transport execution

## 2026-06-30

- Environment: repository unit-test environment
- Command/evidence: `go test ./internal/acl/transport/termuxusb ./internal/cli/acl` and `go build -o arduino-cli .`
- Result: the diagnostic-only interface claim/release command surface compiles and the Termux USB provider helper boundary passes focused unit tests
- Confidence: high
- Notes: this validates the CLI and helper contract only; it does not validate native Termux claim/release behavior

## 2026-07-01

- Environment: native Termux on Samsung A17 / Android 16
- Command/evidence: `./arduino-cli acl transport claim-release --device /dev/bus/usb/001/002 --interface 0 --details`, `./arduino-cli acl transport claim-release --device /dev/bus/usb/001/002 --interface 1 --details`, and `./arduino-cli acl transport claim-release --device /dev/bus/usb/001/002 --interface 2 --details`
- Result: interface 0 claim failed with `LIBUSB_ERROR_BUSY`; interface 1 claim failed with `LIBUSB_ERROR_BUSY`; interface 2 claim succeeded and release succeeded; interface 2 is vendor-specific with bulk OUT `0x02` and bulk IN `0x83`; no payload transfers were attempted
- Confidence: high
- Notes: this is diagnostic-only interface lifecycle evidence. It does not validate byte-stream readiness, upload, flashing, serial monitor, or any transfer-diagnostic behavior. Zero-length or other bulk-transfer probes remain unproven and should not be treated as a safe generic next step.

- Environment: native Termux on Samsung A17 / Android 16
- Command/evidence: `./arduino-cli acl transport diagnose --device /dev/bus/usb/001/002 --details` and `./arduino-cli --json acl transport diagnose --device /dev/bus/usb/001/002`
- Result: the diagnostic-only USB topology bridge foundation reports `vid=0x303a pid=0x1001`, manufacturer `Espressif`, product `USB JTAG/serial debug unit`, serial identity, `interfaces=3`, `endpoints=5`, CDC bulk endpoints `0x01/0x81`, vendor-specific bulk endpoints `0x02/0x83`, and `topology_source=libusb` without payload transfers; `claim_release_state` remains `not_attempted`
- Confidence: high
- Notes: this proves the topology bridge works on-device, but it does not yet prove interface claim/release or any byte-stream behavior

- Environment: native Termux on Samsung A17 / Android 16
- Command/evidence: `./arduino-cli acl workflow upload --help`
- Result: the upload workflow is positional and dry-run only; the help text
  advertises `acl workflow upload <firmware-package>` and does not expose
  `--dry-run` or `--package`
- Confidence: high
- Notes: the command contract is now explicit for GUI and documentation use

- Environment: native Termux on Samsung A17 / Android 16
- Command/evidence: `./arduino-cli --json acl workflow upload ~/Development/Sketches/build/esp32.esp32.esp32s3/firmware-package`
- Result: upload dry-run planning works from the positional package path, with
  `dry_run=true`, `ready=true`, ordered steps, and a warning about missing
  target-chip metadata
- Confidence: high
- Notes: this validates dry-run planning only; it does not validate real upload,
  flashing, or transport execution. The observed plan currently has three
  entries: partitions, boot_app0, and application; the bootloader is still
  package-dependent

- Environment: repository unit-test environment
- Command/evidence: `go test ./commands`, `go test ./internal/acl/engine`, and
  `go test ./internal/acl/firmware`
- Result: target-chip metadata now propagates from `build.mcu` through the
  compile service and workflow snapshot into `FirmwarePackage` generation, so
  manifest and flash-plan metadata are populated before validation; the binary
  validator no longer emits `target chip metadata is not set` when the metadata
  is present
- Confidence: high
- Notes: this confirms the warning was caused by missing propagation in the
  compile/package construction path, not by the validator needing to be
  suppressed. Native Termux should still rerun the package milestone to confirm
  the on-device output matches the host evidence

- Environment: repository unit-test environment
- Command/evidence: upload report unit tests and workflow tests
- Result: the upload dry-run report now keeps a canonical package/plan/
  diagnostics/result/progress surface and de-duplicates repeated professional
  details
- Confidence: high
- Notes: this is a report-shaping validation, not hardware behavior

## 2026-06-26

- Environment: native Termux on Samsung A17 / Android 16
- Command/evidence: native Termux compile validation and workflow validation from
  the current project history
- Result: Blink compiles successfully in native Termux
- Confidence: high
- Notes: this is the current native Android compile baseline

- Environment: repository unit-test environment
- Command/evidence: `go test ./internal/acl/transport/...`
- Result: the transport stream foundation is implemented and unit-tested, with
  bounded stream wrappers, lifecycle states, timeout configuration, EOF and
  disconnect reporting, and the Termux USB provider/session stream boundary
  covered by tests
- Confidence: high
- Notes: this validates the contract and wrapper behavior only; native Termux
  byte-stream read/write behavior still needs device proof

## Unknown / from project history

- Environment: native Termux on the target Android device
- Command/evidence: project history references a larger multi-file ESP32-S3 sketch
  including WiFi, ESP Async WebServer, Async TCP, LittleFS, Adafruit NeoPixel, and
  Hash
- Result: native Termux compile works for the esp32ultimate multi-file sketch
- Confidence: medium
- Notes: keep this entry tied to native validation if a fresh reproduction is needed

- Environment: target ESP32-S3 hardware
- Command/evidence: firmware package from the current build was flashed using ESP32
  Flasher
- Result: generated firmware booted and served the expected web response
- Confidence: medium
- Notes: this proves runtime behavior on hardware, but the exact firmware package
  version should be revalidated whenever package semantics change

- Environment: Android USB host / Termux USB access
- Command/evidence: `termux-usb -l` and permission dialogs on the target device
- Result: devices can be enumerated and Android permission behavior is observable,
  but acquisition/open behavior still needs deeper investigation
- Confidence: medium
- Notes: this remains transport work, not a flashing success claim

- Environment: ACL compile workflow
- Command/evidence: workflow compile now generates firmware-package outputs in the
  repository history
- Result: ACL workflow compile produces package outputs, pending the latest native
  validation of full-flash bootloader inclusion
- Confidence: medium
- Notes: package content should be rechecked as firmware-package semantics evolve

- Environment: repository unit-test environment
- Command/evidence: `go test ./internal/acl/transport/... ./internal/cli/acl/...`
- Result: the transport provider contracts, Termux USB provider, and transport CLI
  surfaces compile and pass unit tests
- Confidence: high
- Notes: this validates the code boundary only; native Termux USB hardware behavior
  still needs on-device validation

## 2026-06-29

- Environment: native Termux on Samsung A17 / Android 16
- Command/evidence: `./arduino-cli acl transport stream-validate --device /dev/bus/usb/001/002 --details`,
  `./arduino-cli acl transport stream-validate --device /dev/bus/usb/001/002 --details --validate-read`,
  `./arduino-cli acl transport stream-validate --device /dev/bus/usb/001/002 --details --validate-write`, and
  `./arduino-cli acl transport stream-validate --device /dev/bus/usb/001/002 --details --validate-read --validate-write`
- Result: baseline `stream-validate` succeeds through the Termux fd handoff path; the fd is observed, valid, and inspectable; read-only validation returns `EOF`; write-only validation returns `write TERMUX_USB_FD: invalid argument`; combined validation does not prove a generic byte stream
- Confidence: high
- Notes: this is native evidence against treating `TERMUX_USB_FD` as a normal byte-stream fd. It does not prove upload, flashing, or serial-monitor behavior, and it points toward a USB-transfer-oriented follow-on milestone rather than byte-stream readiness

- Environment: repository unit-test environment
- Command/evidence: `go test ./internal/acl/transport/... ./internal/acl/transport/termuxusb/...` and `go test ./internal/cli/acl/...`
- Result: the bounded transport stream now has explicit host coverage for read/write bound exhaustion, short-write handling, EOF handling, and closed-state reporting; the Termux USB provider CLI diagnostics continue to report the stream boundary as experimental
- Confidence: high
- Notes: this is host validation only; native Termux byte-stream readiness remains unproven

- Environment: repository unit-test environment
- Command/evidence: `go test ./internal/cli/acl/...`
- Result: `acl transport stream-validate` now exists as a diagnostic-only CLI surface that wraps the live Termux USB stream in the bounded transport contract and can exercise one-byte read/write probes against a scripted stream fixture
- Confidence: high
- Notes: this validates the CLI wiring and report shape only; native Termux validation is still required before any byte-stream readiness claim

- Environment: native Termux validation planning
- Command/evidence: `docs/android/MILESTONE_TERMUX_USB_PROVIDER.md`
- Result: a native-Termux checklist now exists for `acl transport list`,
  `acl transport diagnose --details`, `acl transport acquire --device`, and the
  bounded `acl transport probe-fd` fd-handoff probe
- Confidence: high
- Notes: documentation only; no new device result is claimed by this entry

- Environment: repository unit-test environment
- Command/evidence: `go test ./internal/acl/transport/... ./internal/cli/acl/...`
- Result: the Termux USB provider now includes a bounded fd-handoff probe and
  the helper/CLI surface passes unit tests, including env and argv fd-source
  handling
- Confidence: high
- Notes: this validates the contract boundary only; native Termux validation of
  the probe is still pending

- Environment: native Termux on Samsung A17 / Android 16
- Command/evidence: `termux-usb -r -E -e "./arduino-cli acl transport probe-fd-helper --json" /dev/bus/usb/001/002`
- Result: helper JSON observes `TERMUX_USB_FD` via environment and reports
  `fd_source=environment`, `handoff_mode=env`, and `status=warning`
- Confidence: high
- Notes: this is the confirmed working fd-handoff shape; it does not prove
  upload, flashing, or serial monitor behavior

- Environment: native Termux on Samsung A17 / Android 16
- Command/evidence: `./arduino-cli acl transport probe-fd --device /dev/bus/usb/001/002 --details`
  and `./arduino-cli --json acl transport probe-fd --device /dev/bus/usb/001/002`
- Result: the official `probe-fd` command is native-Termux validated for
  fd handoff, fd observation, and fd inspection
- Confidence: high
- Notes: this does not validate byte-stream read/write, serial bridge, upload,
  flashing, or serial monitor behavior

- Environment: native Termux on Samsung A17 / Android 16
- Command/evidence: `./arduino-cli acl transport list`,
  `./arduino-cli acl transport diagnose --details`,
  `termux-usb -l`, and `./arduino-cli acl transport acquire --device
  /dev/bus/usb/001/002`
- Result: Termux USB discovery and permission acquisition are validated on the
  target device; `acquire` reports permission granted and diagnostics report the
  expected `TERMUX_USB_FD` limitation outside `termux-usb -e`
- Confidence: high
- Notes: this does not prove upload, flashing, serial monitor, or usable
  byte-stream endpoint behavior

- Environment: ESP32-S3 / Arduino-ESP32 flash recipe metadata
- Command/evidence: external flashing history and Arduino-ESP32 recipe conventions
- Result: external flashing previously used bootloader, partitions, boot_app0, and
  application layout
- Confidence: high
- Notes: this is the current canonical flash layout target for full-flash packages

- Environment: current Ubuntu/proot development environment for emulated
  validation research
- Command/evidence: `uname -m`, `/dev/kvm` probe, `command -v qemu-aarch64`, and
  `command -v apt-get`
- Result: ARM64 user-mode smoke testing is practical; a full Android emulator is
  not yet proven practical here because `/dev/kvm` is absent
- Confidence: medium
- Notes: this is a host-environment finding, not an Android-device claim

## 2026-07-02

- Environment: repository architecture review
- Command/evidence: `AGENTS.md`, `STATUS.md`, `ROADMAP.md`, `TASK_RECOVERY.md`,
  `docs/android/ENGINEERING_KNOWLEDGE_FRAMEWORK.md`, and the EOS adoption docs
- Result: `AGENTS.md` currently mixes universal methodology with project-local
  Android guidance, while the repository already has distinct homes for current
  state, future sequencing, recovery, evidence, and decision history; the EOS
  adoption boundary is cleaner when represented as `eos.project.json` plus a
  thin overlay
- Confidence: medium-high
- Notes: this is a static architecture finding, not a runtime or native-device
  validation result. It supports manifest-first EOS adoption and suggests that
  any future AGENTS reduction should preserve local Android guidance without
  duplicating EOS canon
