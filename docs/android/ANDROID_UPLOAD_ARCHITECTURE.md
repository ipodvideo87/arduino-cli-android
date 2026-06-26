# Android Upload Architecture

This document defines the upload and monitor architecture for native Android
Termux. It is a design record, not an implementation report.

The architecture is generic. ESP32-S3 is the first validation target, not the
design center.

## Scope

The goal is to make Android upload and serial monitor support work without
teaching Arduino CLI about Android internals.

Arduino CLI should continue to work against a serial-like endpoint. ACL should
own the Android USB transport, permission flow, descriptor parsing, and bridge
selection.

## Evidence Base

- Arduino CLI upload is recipe-driven and ultimately executes the selected upload
  pattern.
  - Source: `commands/service_upload.go`
- ESP32-family upload recipes use a serial port path and flash payload arguments.
  - Source: ESP32 platform test data in this repository
- Android USB permission is app-scoped and device access ends when the device is
  disconnected.
  - Source: [UsbManager API](https://developer.android.com/reference/android/hardware/usb/UsbManager)
- Termux already has a file-descriptor-based USB handoff path.
  - Source: [termux-api-package termux-api.c](https://github.com/termux/termux-api-package/blob/master/termux-api.c)
- esptool supports remote serial ports over RFC2217.
  - Source: [Remote Serial Ports](https://docs.espressif.com/projects/esptool/en/latest/esp32/remote-serial-ports.html)
- Android apps already demonstrate USB host serial, TCP bridge, and flashing
  behavior on the same class of hardware.
  - Source: [ArduinoDroid Play listing](https://play.google.com/store/apps/details?id=name.antonsmirnov.android.arduinodroid2)
  - Source: [ESP32_Flasher Play listing](https://play.google.com/store/apps/details?id=com.esp_flash.esp_flash_app)
  - Source: [TCPUART Play listing](https://play.google.com/store/apps/details?hl=en_US&id=com.hardcodedjoy.tcpuart)

## Architecture Decision

The Android upload path should be transport-based.

Decision:

- build a reusable ACL USB serial bridge
- expose the bridge as a PTY or PTY-compatible serial endpoint
- keep Arduino CLI unaware of Android USB internals
- keep the implementation descriptor-driven and board-agnostic
- treat ESP32-S3 as the first proof device, not a special case

This lets the same transport serve:

- upload
- serial monitor
- diagnostics
- chip identification

## Upload Flow

The intended end-to-end flow is:

`Download -> Extract -> Android Patch -> Permission / Runtime Fixes -> Executable Validation -> Toolchain Registration -> Self-Test -> Ready`

The pipeline should happen automatically after install. The user should not need
to manually patch installed packages.

### Where the pipeline belongs

The best home is a shared Android post-install stage that runs after extraction
and before package registration or first use.

Recommended behavior:

- core install should invoke it for installed platforms and required tools
- tool install should invoke it directly
- board installation should remain orchestration only
- library install should use the same stage only when the library package
  contains executable payloads or other runtime-facing artifacts

Pure source-only libraries should not be forced through ELF patching.

## Transport Layers

### 1. Discovery

Discovery identifies connected devices and collects enough metadata to choose a
transport path.

Discovery should not assume:

- a specific VID/PID
- a specific board family
- a specific serial chip
- a stable `/dev/ttyUSB*` or `/dev/ttyACM*` path

### 2. Permission

Android permission acquisition should be explicit and recorded.

The permission layer should know whether the device was opened by:

- a permission dialog
- a cached permission grant
- an attach intent
- a Termux-native file descriptor handoff

### 3. Bridge

The bridge converts Android USB access into a serial-like endpoint that existing
tools can consume.

Bridge responsibilities:

- claim the selected interface
- expose the data plane as a PTY or serial-compatible socket
- translate line coding changes
- translate DTR and RTS when supported
- keep diagnostics for disconnect and re-enumeration events

### 4. Tool Endpoint

The tool-facing endpoint should remain serial-shaped.

This is important because:

- Arduino CLI upload recipes expect a serial port abstraction
- esptool can use RFC2217 if a socket bridge becomes necessary
- serial monitor tooling already knows how to talk to a serial endpoint

## Firmware Package

The app should treat a build as a firmware package, not just a pair of loose
binary outputs.

Minimum package contents:

- application binary
- bootloader binary
- partition table binary
- `boot_app0`
- ELF
- MAP
- flash plan
- build manifest
- validation report
- human-readable build and flashing report

UX implication:

- beginner mode should present the package as a ready-to-flash result
- professional mode should expose the underlying artifacts and offsets
- the backend stays the same; only the presentation changes

## Current Implementation Notes

The compile path now emits a stable firmware package and the install paths now
run through the shared Android patch pipeline. That is infrastructure only; it
does not yet prove end-to-end Android USB flashing.

## Validation Plan

The architecture milestone is not complete until the following are demonstrated
on native Android hardware with the new implementation:

1. detect the target USB device
2. acquire Android permission or an equivalent Termux-native fd
3. open the USB connection
4. identify the serial-capable interface
5. expose a stable serial endpoint to the tool
6. run an `esptool` chip-identification probe through the bridge
7. flash a firmware package
8. confirm serial monitor output after upload

## Open Questions

- Is Termux-native USB acquisition sufficient on the target device?
- Does Android re-enumerate the device after permission is granted?
- Does the bridge need a PTY only, or PTY plus RFC2217 for some tools?
- Should self-test be a generic serial smoke test or a per-tool validation hook?
- Which package types should automatically invoke Android post-processing today?

## Native Android Experiments

Run these before implementation or during bridge bring-up:

1. `termux-usb -l`
2. `termux-usb -r <device>`
3. `termux-usb -e <probe> <device>`
4. dump raw USB descriptors from the opened connection
5. verify descriptor-based CDC detection on the target hardware
6. expose a PTY and connect a serial monitor to it
7. compare PTY behavior with an RFC2217 loopback when baud or modem-control
   semantics matter
8. flash a known-good firmware package and verify boot output

## References

- Arduino CLI upload implementation:
  `commands/service_upload.go`
- Android USB host overview:
  <https://developer.android.com/develop/connectivity/usb/host>
- Android `UsbManager` / `UsbDevice` / `UsbDeviceConnection` docs:
  <https://developer.android.com/reference/android/hardware/usb/UsbManager>
  <https://developer.android.com/reference/android/hardware/usb/UsbDevice>
  <https://developer.android.com/reference/android/hardware/usb/UsbDeviceConnection>
- Termux API USB fd handoff:
  <https://github.com/termux/termux-api-package/blob/master/termux-api.c>
- esptool remote serial ports:
  <https://docs.espressif.com/projects/esptool/en/latest/esp32/remote-serial-ports.html>
- ArduinoDroid:
  <https://play.google.com/store/apps/details?id=name.antonsmirnov.android.arduinodroid2>
- ESP32_Flasher:
  <https://play.google.com/store/apps/details?id=com.esp_flash.esp_flash_app>
- TCPUART:
  <https://play.google.com/store/apps/details?hl=en_US&id=com.hardcodedjoy.tcpuart>

## Decision Record

See [ADR 0002](adr/0002-android-post-install-pipeline.md) for the install
pipeline decision record that this architecture depends on.
