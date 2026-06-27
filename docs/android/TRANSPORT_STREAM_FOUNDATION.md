# Transport Stream Foundation

This document defines the reusable byte-oriented transport layer that sits
between transport sessions and the future upload / serial monitor workflows.

It is deliberately conservative. It records the stream abstraction, the
bounded-lifecycle model, and the validation boundary without claiming upload,
flashing, or monitor success.

## Architecture

```text
Transport Provider
  ↓
Transport Session
  ↓
Transport Stream
  ↓
Future upload / monitor / diagnostics consumers
```

The stream layer is transport-neutral and board-neutral. It should work for:

- Termux USB
- desktop serial
- PTY
- RFC2217
- TCP
- Bluetooth
- HID
- DFU
- CMSIS-DAP
- JTAG / SWD
- future transports

## Contract Shape

The transport package now exposes small additive interfaces for:

- read
- write
- close
- stream capabilities
- diagnostics
- timeout configuration
- cancellation
- EOF detection
- disconnect detection
- last activity
- close reason
- stream session access

The main `TransportStream` contract stays intentionally small. More specialized
behaviors are optional interfaces so future transports can expose only what they
actually support.

## Lifecycle Model

The stream report now has explicit lifecycle states:

- unavailable
- experimental
- ready
- active
- closing
- closed
- disconnected
- failed

These states are diagnostic signals, not marketing claims.

Current guidance:

- `unavailable` means the transport cannot expose a stream yet
- `experimental` means the transport can expose a bounded stream boundary, but
  the byte path is not yet production-proven
- `ready` and `active` require real stream behavior, not just fd inspection
- `disconnected` and `failed` must preserve the reason in diagnostics

## Bounded Behavior

The first reusable stream implementation is intentionally cautious:

- bounded reads
- bounded writes
- configurable timeouts
- explicit cancellation
- graceful close
- EOF reporting
- disconnect reporting
- no auto-reconnect
- no background polling loops unless a later transport truly needs one
- no destructive device I/O
- no protocol framing
- no packet parsing

The current bounded stream wrapper is reusable infrastructure for future
Termux, serial, and socket transports. It is not an upload engine.

## Termux USB Integration

The Termux USB provider now carries stream state through its diagnostics and can
return an experimental stream only when the process already has a valid handed
off file descriptor.

What is validated today:

- discovery
- permission acquisition
- `TERMUX_USB_FD` handoff evidence
- file descriptor inspection
- diagnostic reporting

What is not yet validated:

- byte-stream read/write
- upload
- flashing
- serial monitor

Diagnostic commands:

- `arduino-cli acl transport probe-fd --device <path>`
- `arduino-cli acl transport stream-status --device <path>`
- `arduino-cli acl transport probe-fd-helper --json`

## Diagnostics

The stream report is first-class and machine-readable. It records:

- stream state
- state reason
- timeouts
- bytes read
- bytes written
- last activity
- close reason
- disconnect reason
- observation states for read, write, close, EOF, and disconnect
- warnings and limitations

The goal is for the GUI and future workflows to inspect the stream contract
without parsing Android USB internals.

## Validation Boundary

Validation levels matter:

- unit tests prove the contract and wrapper behavior
- native Termux proves Android behavior
- real hardware is still required for upload, flash, and runtime claims

This milestone only proves the stream foundation and bounded diagnostics. It
does not prove that upload, serial monitor, or ESP32 protocol traffic works.

## Native Validation Checklist

Run these on native Termux:

```sh
./arduino-cli acl transport list
./arduino-cli acl transport diagnose --details
./arduino-cli acl transport acquire --device <device-path>
./arduino-cli acl transport probe-fd --device <device-path> --details
./arduino-cli acl transport stream-status --device <device-path> --details
```

Interpretation:

- `state=experimental` means the fd handoff was observed, but stream behavior
  is still provisional
- `state=failed` means the helper or handoff path failed before stream behavior
  could be observed
- `state=unavailable` means the transport cannot currently expose a stream

## Future Consumers

The stream layer is meant to be consumed by:

- upload workflows
- serial monitor workflows
- diagnostics workflows
- future workspace/GUI tools
- future desktop validation providers

Those consumers should depend on `TransportStream`, not on Android USB APIs.

