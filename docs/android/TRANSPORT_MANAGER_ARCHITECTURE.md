# Transport Manager Architecture

The transport manager is the capability-based selection layer for ACL.

The deeper provider model is documented in
[Transport Provider Model](TRANSPORT_PROVIDER_MODEL.md).

It lives in `internal/acl/transport` and is responsible for answering one
question:

What transport is available that satisfies the requested capabilities?

## Design Rules

- Do not ask whether the host is Android.
- Ask what transports exist and what each one can do.
- Treat native serial, Android USB fd/handle, PTY, RFC2217, and future
  transports as peer options.
- Select by capabilities and policy, not by board family.

## Current Transport Kinds

- native serial
- Android USB fd/handle
- PTY
- RFC2217
- future

## Capability Model

Capabilities describe what a transport can provide.

Examples:

- serial I/O
- descriptor discovery
- USB handle access
- PTY export
- RFC2217 compatibility
- line coding
- modem control

## Service Boundary

`TransportManager` is a selector and registry.

It does not:

- identify a board
- choose an ESP32-specific path
- perform flashing
- own UI state

The transport layer underneath the manager will later provide the concrete open,
read, write, and close behavior.

Current repository shape:

- `internal/acl/transport` now includes compile-safe provider/session/
  diagnostics contracts in addition to the selector core.
- Fake providers and manager tests cover registration, selection, permission,
  diagnostics, and session-opening behavior without touching real Android USB.

## Selection Policy

Selection should be capability-first.

The default policy is:

1. filter out unavailable transports
2. discard transports that cannot satisfy the required capabilities
3. rank the remaining candidates by policy and priority
4. keep alternatives for diagnostics and UI

## UI Implication

Beginner mode should not expose the transport choice unless there is an error.

Professional mode can show:

- available transports
- why one transport was selected
- the alternative candidates

## Related Code

- `internal/acl/transport`
- [Android USB Transport Framework](ANDROID_USB_TRANSPORT_FRAMEWORK.md)
- [Transport Provider Model](TRANSPORT_PROVIDER_MODEL.md)
