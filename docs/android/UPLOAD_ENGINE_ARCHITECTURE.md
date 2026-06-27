# Upload Engine Architecture

This document defines the ACL-side upload boundary that sits between firmware
packaging and future transport execution.

It is transport-neutral, board-neutral, and currently prepare-only.

## Current Contract

The upload planner consumes a firmware package and derives an ordered upload
plan. The upload executor then validates the package, plan, artifacts, hashes,
and transport readiness up to the point where bytes would be written, then
stops. It does not open real transport streams, send bytes, reset hardware, or
call `esptool`.

Current CLI surface:

```text
arduino-cli acl workflow upload <firmware-package>
```

The command is positional and prepare-only.

- `--dry-run` is not part of the contract
- `--package` is not part of the contract
- the command validates the package and reports the execution-ready plan
  instead of executing it

## Report Shape

The upload execution report should expose one canonical view for the GUI:

- firmware package metadata
- flash-plan metadata
- validation and diagnostics metadata
- prepare-only execution result summary
- progress events
- beginner summary
- professional details without duplicate copies of the same information

The workflow layer may still wrap the upload execution report for orchestration,
but the canonical upload details should live in the execution report itself.

## Validation Boundary

Prepare-only validation proves:

- the package exists
- the manifest is readable
- the flash plan is readable
- required artifacts exist
- an ordered upload plan can be derived

Prepare-only validation does not prove:

- transport availability
- byte-stream support
- upload success
- flashing success
- serial monitor support

## Design Intent

The upload stack is intentionally small and reusable so future work can add:

- real transport execution
- transport selection
- flash backend integration
- post-upload verification
- GUI presentation layers

without reworking the package and planning layer.
