# Project North Star

This project is not just "Arduino CLI on Android".

The long-term goal is an Android-first embedded development platform. Termux may
be the current backend, but normal users should not need to understand Termux in
order to compile, inspect, package, validate, or eventually flash firmware.

## Vision

- Android should offer professional embedded development, not a reduced mobile toy.
- Beginner, Advanced, and Professional are UI layers over the same engine.
- Beginner is not a lesser product. It is Professional mode with automation and
  progressive disclosure.
- The system should reduce complexity without reducing capability.
- The validation system should also be layered: static analysis, host smoke
  tests, emulated preflight, native Termux, and real hardware each have a
  defined confidence boundary.
- Every validation provider should emit a human-readable summary and a
  machine-readable report so the ACL Engine and future GUIs can compare results
  without re-discovering context.
- The transport system should follow the same philosophy: discovery,
  permission, connection, protocol, upload, monitor, and diagnostics should be
  separate reusable layers.
- Future GUI/workspaces should talk to transport providers and workflows, not
  directly to Android USB APIs.
- ESP32-S3 is the first validation target, not the architecture.
- The architecture should scale to more boards, transports, and frontends.

## Implications

- The UI should consume high-level workflows and package outputs, not Arduino CLI
  internals.
- The backend should remain reusable across future workspaces and applications.
- The platform should make the same engine useful to both beginners and experts.
