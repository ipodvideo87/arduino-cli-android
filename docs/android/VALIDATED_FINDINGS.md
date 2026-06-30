# Validated Findings

This file is a logbook of tested findings from the project history.

## 2026-06-27

- Environment: repository unit-test environment
- Command/evidence: `go test ./internal/acl/upload/... ./internal/acl/... ./internal/cli/... ./commands/...` and `go build -o arduino-cli .`
- Result: the upload workflow now uses a prepare-only executor, the positional `acl workflow upload <firmware-package>` contract remains intact, and the executor report compiles and round-trips through unit tests
- Confidence: high
- Notes: this validates the code path and report shape only; it does not validate real upload, flashing, or transport execution

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

- Environment: repository unit-test environment
- Command/evidence: `go test ./internal/acl/transport/... ./internal/acl/transport/termuxusb/...` and `go test ./internal/cli/acl/...`
- Result: the bounded transport stream now has explicit host coverage for read/write bound exhaustion, short-write handling, EOF handling, and closed-state reporting; the Termux USB provider CLI diagnostics continue to report the stream boundary as experimental
- Confidence: high
- Notes: this is host validation only; native Termux byte-stream readiness remains unproven

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
