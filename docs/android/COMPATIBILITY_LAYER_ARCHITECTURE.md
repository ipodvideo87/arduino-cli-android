# Compatibility Layer Architecture

The compatibility layer detects and manages compatibility issues across:

- runtime
- libraries
- firmware artifacts
- transports

It is a decision layer, not a patching layer.

## Goals

- detect incompatible library/core combinations
- prefer compatible library selection over source patching
- support documented version-aware patches when they are explicitly defined
- record compatibility decisions in build and installation reports
- surface beginner-friendly messages and professional-level detail

## Layer Placement

The compatibility layer belongs below the UI and above the install or build
actions.

It should answer:

- is this combination compatible?
- if not, is there a compatible alternative?
- if not, is there a documented patch?
- if not, what diagnostic should the user see?

It should not:

- mutate source trees directly
- hardcode board-specific logic into the UI
- replace the install pipeline

## Current Package Shape

The foundation lives in `internal/acl/compatibility`.

Current report types include:

- `Decision`
- `Report`
- `CompatibilityReport`
- `InstallationReport`

These are intentionally reusable across build, install, runtime, and transport
decision paths.

## Example Rule Shape

The observed ESPAsyncWebServer issue is the model example:

- ESPAsyncWebServer 3.1.0 is not compatible with ESP32 Core 3.3.10 because it
  references old mbedTLS symbols
- a newer compatible ESP Async WebServer release should be selected first
- a documented version-aware patch is a fallback, not the primary strategy

## UI Policy

Beginner mode should show:

- a concise compatibility verdict
- a clear fix suggestion

Professional mode should show:

- the rule ID
- the selected version
- alternatives
- evidence
- any documented patch strategy

## Current Implementation

Platform install and library install paths now consult the resolver before
committing to a version, and build / install reports can carry the resulting
compatibility decisions. The resolver is still rule-based and generic; it does
not patch sources itself.

## Related Code

- `internal/acl/compatibility`
- `internal/acl/firmware`
- `internal/acl/install`
- [Firmware Package Architecture](FIRMWARE_PACKAGE_ARCHITECTURE.md)
- [Android Install Patch Pipeline](ANDROID_INSTALL_PATCH_PIPELINE.md)
