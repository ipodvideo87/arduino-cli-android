# Transport API Stabilization

This document records the current stabilization result for the transport-layer
API surface used by ACL, the Termux USB provider, and future upload/monitor
workflows.

## Status

Transport API status:

- **Stabilizing**
- breaking changes are not expected before upload work depends on the API
- additive improvements are still welcome
- byte-stream behavior remains experimental until native Termux proves read/write
  behavior on real hardware

## What Is Stabilized

The current transport boundary is acceptable for the next layer of work:

- `TransportProvider`
- `TransportManager`
- `TransportSession`
- `TransportStream`
- transport capability and diagnostics models
- Termux USB discovery, permission, diagnostics, and fd-handoff reporting

The API already expresses the core lifecycle questions:

- what provider exists
- what capabilities it exposes
- how permission was acquired
- what session was opened
- what stream state exists
- why a session or stream failed
- what alternatives were available

## What Is Still Experimental

The following remain intentionally experimental or unproven:

- byte-stream read/write behavior on native Termux
- upload workflows built on top of the stream boundary
- serial monitor workflows built on top of the stream boundary
- protocol adapters for CDC ACM, DFU, HID, CMSIS-DAP, and other transports

## Compatibility Policy

The project should avoid unnecessary breaking changes to the transport contract.
Prefer:

1. additive fields
2. additive interfaces
3. compatibility aliases
4. diagnostic refinements

Only make breaking changes if they clearly prevent future architecture problems.

## Naming Guidance

- `StreamSession` is the preferred name for a session that can expose a byte
  stream.
- `ByteStreamSession` is retained only as a compatibility alias.
- `TransportStream` is the preferred byte-oriented boundary for future upload
  and monitor consumers.

## Validation Evidence

Current evidence supporting stabilization:

- unit tests cover provider registration, selection, diagnostics, and stream
  wrapper behavior
- CLI tests cover the safe transport diagnostic surface
- native Termux validation has confirmed discovery, permission acquisition,
  `TERMUX_USB_FD` handoff, and stream-diagnostics reporting
- the stream wrapper exists and remains bounded and diagnostics-first

## Decision

The transport API is stable enough for the Upload Engine foundation and future
GUI plumbing, with the byte-stream implementation itself still marked
experimental until native Termux proves read/write behavior.
