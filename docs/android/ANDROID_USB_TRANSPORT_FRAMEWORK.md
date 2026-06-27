# Android USB Transport Framework

This document is the short-form overview of the Android USB transport work.
The detailed research lives in:

- [USB Transport Research](USB_TRANSPORT_RESEARCH.md)
- [USB Transport Architecture](USB_TRANSPORT_ARCHITECTURE.md)
- [Transport Provider Model](TRANSPORT_PROVIDER_MODEL.md)
- [Upload Workflow Preview](UPLOAD_WORKFLOW_PREVIEW.md)
- [Serial Monitor Preview](SERIAL_MONITOR_PREVIEW.md)

It remains board-agnostic and does not assume ESP32-specific VID/PID values,
interface numbers, reset sequences, or baud rates.

The compile-safe transport provider skeleton now exists in
`internal/acl/transport`, along with fake-provider tests and manager helpers.
The Termux USB provider now exists in `internal/acl/transport/termuxusb`,
along with safe diagnostic CLI surfaces:

- `arduino-cli acl transport list`
- `arduino-cli acl transport diagnose`
- `arduino-cli acl transport acquire --device <path>`
- `arduino-cli acl transport probe-fd --device <path>`

It is still a contract and diagnostics layer only: no real Android USB API
calls, no USB flashing, and no upload or serial-monitor implementation yet.
The fd probe is intentionally bounded and only records `TERMUX_USB_FD`
handoff evidence.
The current probe uses the working `termux-usb -r -E -e <helper> <device>`
shape and the helper also accepts the positional fd handoff used by
`termux-usb -e`.

The framework exists so Arduino CLI and future tool integrations can keep
talking to a serial-like endpoint while ACL owns Android-specific USB
discovery, permission, and bridge logic.

## Current Direction

- discovery is separate from transport
- permission acquisition is separate from discovery
- descriptor parsing is separate from permission
- the tool-facing endpoint is separate from USB control details
- transport selection is capability-based rather than board-based
- provider contracts are compile-safe and fake-provider tested before any real
  Android USB integration is attempted
- Termux USB discovery, permission acquisition, and file-descriptor reporting
  are isolated behind the transport provider contract and remain diagnostic-only
  until native Termux validation proves the on-device behavior
- the fd-handoff probe exists as a transport-neutral diagnostic boundary, but a
  real byte-stream bridge remains unproven

## Related Docs

- [USB Transport Research](USB_TRANSPORT_RESEARCH.md)
- [USB Transport Architecture](USB_TRANSPORT_ARCHITECTURE.md)
- [Transport Provider Model](TRANSPORT_PROVIDER_MODEL.md)
- [Upload Workflow Preview](UPLOAD_WORKFLOW_PREVIEW.md)
- [Serial Monitor Preview](SERIAL_MONITOR_PREVIEW.md)
- [ADR 0001](adr/0001-generic-android-usb-serial-bridge.md)
