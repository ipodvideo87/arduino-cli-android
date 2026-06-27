# Upload Engine Architecture

This document defines the ACL-side upload engine boundary that sits between
firmware packaging and future transport execution.

It is transport-neutral, board-neutral, and currently dry-run only.

## Current Contract

The upload engine consumes a firmware package and derives an ordered upload
plan. It does not open real transport streams, send bytes, reset hardware, or
call `esptool`.

Current CLI surface:

```text
arduino-cli acl workflow upload <firmware-package>
```

The command is positional and dry-run only.

- `--dry-run` is not part of the contract
- `--package` is not part of the contract
- the command validates the package and reports the plan instead of executing it

## Report Shape

The upload report should expose one canonical view for the GUI:

- firmware package metadata
- flash-plan metadata
- validation and diagnostics metadata
- planned upload result summary
- progress events
- beginner summary
- professional details without duplicate copies of the same information

The workflow layer may still wrap the upload report for orchestration, but the
canonical upload details should live in the upload report itself.

## Validation Boundary

Dry-run validation proves:

- the package exists
- the manifest is readable
- the flash plan is readable
- required artifacts exist
- an ordered upload plan can be derived

Dry-run validation does not prove:

- transport availability
- byte-stream support
- upload success
- flashing success
- serial monitor support

## Design Intent

The upload engine is intentionally small and reusable so future work can add:

- real transport execution
- transport selection
- flash backend integration
- post-upload verification
- GUI presentation layers

without reworking the package and planning layer.
