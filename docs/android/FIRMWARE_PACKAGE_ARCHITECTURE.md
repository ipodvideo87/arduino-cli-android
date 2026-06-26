# Firmware Package Architecture

The firmware package is the first-class build artifact for the Android-first
workflow.

It lives in `internal/acl/firmware`.

The goal is simple:

Normal users should interact with a firmware package, not a pile of loose `.bin`
files.

## Package Contents

A firmware package includes:

- application binary
- bootloader binary
- partition table binary
- `boot_app0`
- ELF
- MAP
- build manifest
- flash plan
- validation report

## Build Manifest

`BuildManifest` carries the build metadata needed to understand and reproduce a
firmware package.

It records:

- board
- FQBN
- core version
- library list
- toolchain version
- artifact paths
- artifact hashes
- memory usage
- target or chip metadata when known

## Package Boundary

The firmware package is the unit the UI should present.

Beginner mode:

- shows the package as a ready-to-flash result
- hides the artifact clutter unless a failure occurs

Advanced mode:

- shows the package contents and validation state

Professional mode:

- exposes every artifact
- allows inspection of manifest, flash plan, ELF, MAP, hashes, and validation
  details

## Current Implementation

Successful compiles now create a stable firmware package in the sketch build
tree instead of leaving users to inspect `~/.cache/arduino/sketches` directly.

The package writer copies artifacts into a stable output directory and writes:

- `manifest.json`
- `flash-plan.json`
- `validation-report.json`

The compile path is still Android-agnostic at the Arduino CLI boundary; ACL
owns the packaging details underneath it.

## Compatibility Decisions

The build manifest can also carry compatibility decisions.

That lets the package record:

- incompatible library/core combinations
- selected compatible library versions
- fallback documented patches
- transport compatibility notes when they affect package readiness

The manifest should be the durable record for why the package was built the way
it was.

## Related Code

- `internal/acl/firmware`
- `internal/acl/compatibility`
- [Flash Plan Architecture](FLASH_PLAN_ARCHITECTURE.md)
- [Diagnostics Workflow](DIAGNOSTICS_WORKFLOW.md)
