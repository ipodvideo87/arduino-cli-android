# Flash Plan Architecture

The flash plan is the single source of truth for flash offsets and flashable
files.

It lives in `internal/acl/firmware`.

## Design Rules

- Do not spread offsets across UI code, install code, and flashing code.
- The flash plan owns the offsets.
- The build manifest owns the artifact metadata.
- The validator checks that the two agree.

## Shape

A flash plan is a list of entries.

Each entry records:

- offset
- artifact kind
- file path
- whether the entry is required
- human-readable description

## Required Artifacts

The default required flash artifacts are:

- application binary
- bootloader binary
- partition table binary

`boot_app0` can be included when a board or target requires it.

## UI Behavior

Beginner mode should use the flash plan automatically.

Professional mode can inspect and edit the plan later, but the default plan
remains the source of truth.

## Validation

The flash plan validator checks:

- that the plan is not empty
- that offsets are sane and non-duplicated
- that required artifacts are present
- that plan paths match the package manifest

## Current Implementation

The compile path now derives a stable flash plan from actual build artifacts
and the board upload pattern when available.

For ESP32-style builds the current implementation can populate:

- application binary
- bootloader binary
- partition table binary
- `boot_app0` when present

The plan is still stored as data, not hardcoded into UI code or Arduino CLI
transport logic.

## Related Code

- `internal/acl/firmware`
- [Firmware Package Architecture](FIRMWARE_PACKAGE_ARCHITECTURE.md)
