# ADR 0001: Generic Android USB Serial Bridge

## Status

Proposed

## Context

Native Android USB access is app-scoped and permission-gated.

The current project needs a reusable transport that works for Arduino CLI,
esptool, and serial monitor tooling without teaching those tools about Android
internals.

The evidence also shows that existing Android USB tools commonly use generic
serial abstractions rather than hardcoding a single board family.

## Decision

Implement a reusable ACL USB serial bridge with these properties:

- descriptor-driven discovery
- permission acquisition separated from transport logic
- interface claim and endpoint selection separated from discovery
- PTY or PTY-compatible serial endpoint export
- optional RFC2217-compatible fallback when a PTY is not enough
- generic line coding and modem-control translation where supported
- no ESP32-specific VID/PID or upload-sequence hardcoding in the bridge layer

ACL owns the bridge.

Arduino CLI stays unaware of Android internals and consumes only the exposed
serial-like endpoint.

## Consequences

Positive:

- the same transport can serve upload, monitor, and diagnostics
- the implementation can support multiple boards and serial adapters
- Android-specific logic stays isolated in ACL

Negative:

- the bridge is more complex than a direct `/dev/tty*` path
- some tools may require RFC2217 or a side channel for full modem-control
  behavior
- native validation is required before the design can be considered proven

## Notes

This ADR does not claim upload success.

It records the transport shape that should be implemented next.
