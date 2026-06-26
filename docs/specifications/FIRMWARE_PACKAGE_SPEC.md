# Firmware Package Specification

`FirmwarePackage` is the stable boundary between compile backends and UI layers.

## Directory Structure

```text
firmware-package/
├── artifacts/
├── manifest.json
├── flash-plan.json
├── validation-report.json
├── analysis.json
└── README_FLASHING.txt
```

## Package Modes

- `full-flash`
  - intended for boards where the metadata exposes a complete flash layout
  - should include bootloader, partitions, boot_app0, and application artifacts
- `app-only`
  - intended for cases where metadata is incomplete or the board only needs the
    application image

## Required JSON Files

- `manifest.json`
  - package identity, build metadata, artifact list, and compatibility data
- `flash-plan.json`
  - offsets and artifact mapping for flash operations
- `validation-report.json`
  - package readiness, warnings, errors, and validation checks
- `analysis.json`
  - GUI-facing machine-readable build analysis surface

## Artifact Expectations

- `bootloader.bin`
- `partitions.bin`
- `boot_app0.bin`
- `application.bin`
- `firmware.elf`
- `firmware.map`

Full-flash packages should include the bootloader-related artifacts when the build
metadata exposes them. App-only packages may omit them.

## Schema Versioning

- Each JSON document should carry a schema version where appropriate.
- New fields should be additive and backward compatible.
- Existing fields should not be repurposed without a version bump or compatibility
  note.

## GUI Consumption Expectations

- Beginner views should primarily use `manifest.json`, `validation-report.json`, and
  `README_FLASHING.txt`.
- Advanced views should also use `flash-plan.json` and package metadata.
- Professional views should additionally use `analysis.json`, `firmware.elf`,
  `firmware.map`, and detailed validation diagnostics.

## Contract Notes

- `analysis.json` is the GUI-facing analysis source of truth.
- `firmware.elf` and `firmware.map` remain raw authoritative artifacts.
- The package should remain the same boundary whether it was produced by the CLI or
  by the ACL workflow.

