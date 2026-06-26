# Transport Provider Model

This document defines the provider model that sits under the Android USB
transport architecture.

It is intentionally broader than USB serial because future transports may be
DFU, HID, CMSIS-DAP, JTAG, SWD, network-based, or Bluetooth-based.

## Purpose

The provider model gives ACL a reusable way to:

- discover available transports
- describe transport capabilities
- rank transport options
- preserve alternatives for diagnostics
- keep transport selection separate from protocol-specific behavior

## Current Repository Shape

`internal/acl/transport` already implements the selector/registry core and the
compile-safe provider boundary:

- `TransportManager`
- `TransportDescriptor`
- `SelectionRequest`
- `TransportSelection`
- provider/session/permission/endpoint/diagnostics contracts
- fake-provider tests

That is the first layer of the provider model.

The remaining transport lifecycle behavior should be layered above it, not
forced into the selector.

## Provider Categories

### Discovery Provider

Responsibilities:

- enumerate candidate devices
- expose device identity and descriptor evidence
- report availability

### Permission Provider

Responsibilities:

- request or confirm access
- record how permission was acquired
- report denial or revocation

### Session Provider

Responsibilities:

- open the live transport session
- hold the connection
- close the connection
- emit lifecycle events

### Protocol Adapter

Responsibilities:

- interpret protocol-specific behavior
- adapt raw transport semantics to a higher-level protocol

Examples:

- USB serial
- CDC ACM
- DFU
- HID
- CMSIS-DAP
- JTAG/SWD

### Endpoint Provider

Responsibilities:

- export a PTY, socket, or serial-compatible endpoint
- translate tool-facing control expectations when possible

### Diagnostics Provider

Responsibilities:

- emit machine-readable session reports
- preserve warnings and limitations
- record alternatives and selection reasons

## Selection Policy

Selection should be capability-first.

Recommended order:

1. filter unavailable providers
2. filter providers missing required capabilities
3. score by preference and environment suitability
4. preserve alternatives for diagnostics
5. expose selection reason to the UI and workflow reports

This is consistent with the current `TransportManager` behavior.

## Capability Model

Transport capabilities should be descriptive, not board-specific.

Examples:

- serial I/O
- descriptor discovery
- USB handle access
- PTY export
- RFC2217 compatibility
- line coding
- modem control
- hotplug awareness
- reconnect awareness
- protocol translation

## Report Model

Every provider should be able to report:

- provider kind
- provider name
- availability
- capability list
- priority
- environment details
- device identity
- selected interface
- selected endpoint(s)
- permission source
- warnings
- limitations
- selection reason

That report surface should be machine-readable and stable enough for the ACL
Engine and future GUI layers.

## Future Transport Families

The provider model must be able to represent:

- USB serial
- DFU
- HID
- CMSIS-DAP
- JTAG
- SWD
- network transports
- Bluetooth transports
- future wireless transports

## Integration Rules

- Transport providers should not know about GUI state.
- Transport providers should not be board-specific unless a board family truly
  requires it.
- Upload and monitor workflows should consume providers, not Android USB APIs.
- Compatibility decisions should remain in the compatibility layer.

## Current Status

The provider model is partially implemented as selection infrastructure and
documented as a reusable architecture.

The actual open/close/protocol/endpoint providers remain future implementation.
