# Android USB Transport Framework

This document defines the reusable Android USB transport framework for native
Termux. It is board-agnostic and does not assume ESP32-specific VID/PID values,
interface numbers, reset sequences, or baud rates.

The framework exists so Arduino CLI and esptool can keep talking to a serial-like
endpoint while ACL owns Android-specific USB discovery, permission, and bridge
logic.

## Evidence Base

- Android USB host mode enumerates connected devices and requires explicit
  permission before access.
  - Source: [Android USB host overview](https://developer.android.com/develop/connectivity/usb/host)
  - Source: [UsbManager API](https://developer.android.com/reference/android/hardware/usb/UsbManager)
  - Source: [UsbDeviceConnection API](https://developer.android.com/reference/android/hardware/usb/UsbDeviceConnection)
- Android grants USB permission until the device is disconnected.
  - Source: [UsbManager.requestPermission](https://developer.android.com/reference/android/hardware/usb/UsbManager)
- Termux USB support already passes file descriptors through `termux-callback`.
  - Source: [termux-api-package termux-api.c](https://github.com/termux/termux-api-package/blob/master/termux-api.c)
  - Source: [termux-api releases](https://github.com/termux/termux-api/releases)
- A generic Android USB serial library can discover CDC/ACM devices by interface
  type instead of only by VID/PID.
  - Source: [usb-serial-for-android](https://github.com/mik3y/usb-serial-for-android)
- PTYs are a standard serial-like host abstraction.
  - Source: [pty(7)](https://man7.org/linux/man-pages/man7/pty.7.html)
  - Source: [posix_openpt(3)](https://man7.org/linux/man-pages/man3/posix_openpt.3.html)

## Design Goal

The bridge must expose a host-side serial endpoint that existing tools can open
without knowing that Android USB was involved.

That means:

- discovery is separate from transport
- permission acquisition is separate from discovery
- descriptor parsing is separate from permission
- the tool-facing endpoint is separate from USB control details

## Layer Stack

The intended flow is:

`Android USB Host -> Acquisition -> Descriptor Parsing -> Serial Transport -> PTY -> arduino-cli / esptool / monitor`

The layers are intentionally explicit.

### Device Discovery

Discovery identifies connected devices and captures enough metadata to choose a
transport path.

Required inputs:

- raw USB descriptors
- Android USB device name
- bus and device path when available
- interface count and endpoint layout

Discovery must not require a predeclared ESP32 profile.

### Permission Acquisition

Android permission is app-scoped and must be requested before opening the device.

The acquisition layer should support both:

- enumerating already-connected devices
- responding to attach events when an app is allowed to handle them

The layer must record whether the device was opened after:

- a granted permission dialog
- a cached permission check
- a direct attach intent

### Descriptor Parsing

Descriptor parsing decides whether a device can be exposed as a generic serial
transport.

The parser should prefer interface class and protocol data over vendor-specific
assumptions.

Primary cases:

- CDC ACM interfaces
- composite devices with a dedicated serial interface
- vendor-specific serial adapters handled by a pluggable probe table

The parser should surface:

- control interface
- data interface
- bulk IN endpoint
- bulk OUT endpoint
- optional interrupt endpoint
- line coding support
- DTR and RTS support

### Serial Transport

The transport layer owns the live USB connection.

Responsibilities:

- open and close the Android USB connection
- claim and release interfaces
- perform control transfers
- perform bulk transfers
- translate line coding updates
- translate DTR and RTS requests when the device supports them

The transport must not interpret board-specific flashing protocols.

### PTY Export

The tool-facing contract should be a PTY or PTY-compatible serial endpoint.

Why PTY:

- it is a standard UNIX serial abstraction
- most serial-aware tools already understand it
- it keeps Android-specific USB details out of Arduino CLI

Important caveat:

- a PTY is a byte-stream contract, not a full USB control-plane emulator
- if a tool needs baud changes or modem-control semantics beyond what a PTY can
  express, the bridge needs a side channel or an RFC2217-compatible transport

### Diagnostics

The framework should emit structured diagnostics for each session:

- discovery source
- permission source
- selected interface
- raw descriptor summary
- endpoint summary
- bridge endpoint path
- line coding state
- DTR and RTS state
- disconnect or re-enumeration events

This is required so intermittent acquisition failures can be explained rather
than guessed.

## Candidate Implementations

### Termux Direct Device Node

Use a normal `/dev/tty*` path when Android has already exposed one and it is
usable.

This is opportunistic, not the default assumption.

### Termux USB File Descriptor

Use `termux-usb` or another Termux-native acquisition path to obtain a USB file
descriptor and hand it to native code.

This keeps acquisition inside the existing Termux ecosystem and is the preferred
first implementation path.

### Android UsbManager Bridge

Use Android host APIs directly from a Termux-side helper or a small Android-side
bridge when permission or descriptor handling requires it.

This option remains inside the existing Termux ecosystem unless native evidence
shows it cannot work.

### RFC2217-Compatible Socket Bridge

If a PTY is not sufficient for a given tool, expose a localhost transport that
preserves serial semantics closely enough for pySerial and esptool.

This is the right fallback shape when baud changes or modem-control semantics
need to survive across process boundaries.

## Proposed ACL Placement

The reusable pieces belong under ACL rather than in Arduino CLI command code.

Suggested structure:

- `acl/serial/detect`
- `acl/serial/bridge`
- `acl/serial/runtime`
- `acl/serial/diagnostics`

Responsibilities:

- parse descriptors
- decide whether a device is serial-capable
- acquire and hold the USB transport
- expose a PTY or serial-compatible endpoint
- report bridge diagnostics

Arduino CLI should consume the exposed endpoint and stay unaware of Android
internals.

## Open Questions

- Why do some `termux-usb` acquisition attempts succeed while others fail?
- Is Android re-enumerating the device after permission is granted?
- Does the device name or `/dev/bus/usb/...` path change between permission and open?
- Is the observed delay a Termux API bug, an Android USB lifecycle issue, or a
  timing bug in the bridge code?
- Is PTY alone sufficient for all supported upload and monitor flows, or will
  RFC2217 be required for some tools?

## Native Android Experiments

Before implementation, run these experiments on the target device:

1. `termux-usb -l`
2. `termux-usb -r <device>`
3. `termux-usb -e <probe> <device>`
4. `ls -l /dev/bus/usb/*/*` before and after disconnect/reconnect
5. dump raw USB descriptors from the opened fd
6. identify the serial interface class and endpoints
7. open the same fd from native code and prove the fd reaches the bridge
8. run an `esptool` chip-identification probe through the bridge

## Decision Record

See [ADR 0001](adr/0001-generic-android-usb-serial-bridge.md) for the transport
decision record that this framework implements.
