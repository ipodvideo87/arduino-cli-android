# USB Transport Architecture

This document defines the target architecture for the Android USB transport
subsystem.

It is a design record, not an implementation report.

## High-Level Flow

```text
Future GUI / Workspaces
  ↓
ACL Engine
  ↓
Transport-aware Workflows
  ↓
TransportManager / Provider Model
  ↓
Discovery / Permission / Connection / Protocol / Endpoint / Diagnostics
  ↓
Android USB Host / Termux USB / Native Serial / RFC2217 / Future Transports
  ↓
Hardware
```

## Design Goals

- be board-agnostic
- support serial today and non-serial transports later
- keep Android-specific code inside ACL
- keep Arduino CLI unaware of Android USB internals
- make diagnostics and selection machine-readable
- preserve beginner, advanced, and professional views over the same backend

## Layered Responsibilities

### 1. Transport Providers

Providers represent transport capabilities, not just devices.

They answer:

- what transport kinds exist?
- what capabilities does each one expose?
- is it currently available?
- what does it need in order to open a session?

### 2. TransportManager

`TransportManager` is the capability-based selector and registry.

It should:

- filter unavailable providers
- rank providers by capability and policy
- retain alternatives for diagnostics
- avoid board-specific assumptions

It should not:

- identify a board
- flash firmware
- own the GUI state
- talk directly to Android USB APIs

Current repository state:

- `internal/acl/transport` already provides a selector/registry core plus
  compile-safe provider, session, endpoint-export, permission, and diagnostics
  contracts.
- Fake providers and tests exercise the boundary without real Android USB
  access.
- `internal/acl/transport/termuxusb` now provides the first real provider
  implementation boundary, including command-trace diagnostics and safe
  `acl transport` CLI entry points, but it remains diagnostic-only until native
  Termux USB validation proves the acquisition and fd-handoff path.

### 3. Discovery

Discovery finds candidate devices and transport shapes.

Discovery must record:

- device identity
- interface layout
- endpoint layout
- source of discovery
- raw descriptor evidence

Discovery should not require a predeclared board profile.

### 4. Permission Acquisition

Android permission is separate from discovery and separate from connection.

The framework should record whether access came from:

- a permission dialog
- a cached grant
- an attach intent
- a Termux-native file descriptor handoff

### 5. Connection

Connection owns the live session.

Responsibilities:

- open and close the connection
- claim and release interfaces
- perform control, bulk, and interrupt transfers
- preserve session lifecycle events

### 6. Protocol Adaptation

Protocol adaptation is where the transport becomes useful to tools.

Examples:

- serial line coding
- DTR/RTS handling
- DFU transaction shaping
- HID packet handling
- JTAG/SWD probe protocol handling
- network socket adaptation

This layer is distinct from discovery and from endpoint export.

### 7. Endpoint Export

The tool-facing contract should be serial-shaped when possible.

Preferred exports:

- PTY
- PTY-compatible serial socket
- RFC2217-compatible socket
- future device-specific adapters when a serial endpoint is not enough

### 8. Diagnostics

Diagnostics must be first-class and machine-readable.

Every transport session should emit:

- environment details
- transport kind
- device identity
- interface and endpoint summary
- permission source
- selected protocol adapter
- alternative candidates
- warnings
- limitations
- disconnection / re-enumeration events

## Separation Rules

The framework must keep these concerns separate:

- discovery
- permission
- connection
- protocol
- upload
- monitor
- diagnostics

That separation prevents Android USB details from leaking into upload or
monitor code.

## ACL Integration

### Workflow Engine

The ACL Engine should drive transport-aware workflows and consume structured
transport reports.

### Firmware Package

`FirmwarePackage` may carry transport hints and transport-related diagnostics,
but it should not own USB lifetime.

### Validation Providers

Validation providers should be able to report transport readiness and transport
failure modes separately from upload success.

### Compatibility Layer

Transport compatibility decisions should live in the reusable compatibility
layer, not in ad hoc board code.

### Future Validation Engine

Future validation engines should be able to compare transport readiness across
providers and environments.

### Future GUI / Workspaces

The GUI should display transport providers, selection reasons, warnings, and
alternative candidates without having to parse Android USB APIs.

## Implementation Boundaries

This milestone does not implement:

- USB flashing
- upload
- serial monitor
- protocol-specific device control

Those belong to future milestones that consume the transport framework.

## Transport Kinds and Capabilities

The initial provider model should be able to express:

- native serial
- Android USB file descriptor
- PTY
- RFC2217
- future

Capability examples:

- serial I/O
- descriptor discovery
- USB handle access
- PTY export
- RFC2217 compatibility
- line coding
- modem control
- protocol translation

## Current CLI Surface

The ACL CLI exposes the transport boundary through safe, non-flashing
diagnostic commands:

- `arduino-cli acl transport list`
- `arduino-cli acl transport diagnose`
- `arduino-cli acl transport acquire --device <path>`

These commands are intended to surface discovery, permission, and endpoint
metadata, not to perform upload or monitor operations.

## Stable Design Conclusion

The architecture is stable if the system can answer these questions cleanly:

- what transports exist?
- what can each one do?
- how was permission acquired?
- what endpoint was exported?
- why was this transport selected?
- what alternatives were available?
- what happened when the session failed?

That is the reusable foundation for every future hardware interaction in this
project.
