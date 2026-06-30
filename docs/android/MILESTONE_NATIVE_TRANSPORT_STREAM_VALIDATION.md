# Native Transport Stream Validation Milestone

## Milestone

Validate whether the current Termux USB fd/session/stream path can safely
support a bounded transport stream on native Termux.

This milestone stays transport-runtime oriented and generic. The implementation
target is the Termux USB provider, but the architecture should remain board-
agnostic.

## Scope

Validate only:

- fd handoff observation
- session stream exposure
- bounded stream diagnostics
- supported / unsupported / experimental stream state reporting

Do not claim or implement:

- upload
- flashing
- serial monitor
- ESP32 protocol behavior
- protocol framing
- reset behavior
- destructive USB communication

## Safe Diagnostic Commands

Use only the diagnostic surfaces already present in the repo:

```sh
./arduino-cli acl transport list
./arduino-cli acl transport diagnose --details
./arduino-cli acl transport acquire --device <device-path>
./arduino-cli acl transport probe-fd --device <device-path> --details
./arduino-cli acl transport stream-status --device <device-path> --details
./arduino-cli acl transport stream-validate --device <device-path> --details
```

These commands must remain diagnostic-only. They should never write firmware,
reset the board, or frame a protocol exchange.

## Validation Split

### Host-testable

Host tests can prove:

- CLI wiring for `probe-fd` and `stream-status`
- CLI wiring for `stream-validate`
- report shaping for supported / unsupported / experimental states
- helper fd-source parsing
- bounded stream wrapper behavior
- safe error handling when a stream is not available

### Native Termux required

Native Termux must prove:

- the actual `termux-usb` discovery and permission path
- the current fd handoff shape on-device
- whether the session can expose a bounded stream boundary
- whether a bounded read/write probe can be exercised safely
- whether the stream state should remain `experimental` or can move to
  `ready`

## Evidence Required Before Stream Readiness

Do not claim stream readiness until the native Termux evidence includes:

- the exact command used
- the device path
- whether `TERMUX_USB_FD` was present
- whether the fd was observed, valid, and inspectable
- the reported stream state and state reason
- the command trace for `termux-usb`
- explicit confirmation that no destructive USB communication occurred

If bounded read/write behavior is not proven on-device, keep the stream state
`experimental`.

If native Termux shows `write TERMUX_USB_FD: invalid argument`, treat that as
evidence that the fd should not be assumed to be a generic byte-stream endpoint.
Use the result to choose the next transport-oriented milestone rather than
promoting stream readiness.

## Current Milestone Status

This milestone is defined and ready for implementation and validation.
The repository already has the diagnostic-only transport provider, but native
Termux stream readiness remains unproven.
