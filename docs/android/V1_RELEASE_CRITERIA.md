# V1 Release Criteria

This document is the engineering release checklist for the Android-first
platform. It is not a marketing roadmap.

Status values:

- Planned
- Research
- In Progress
- Implemented
- Native Validated
- Production Ready

## Criteria

- `Implemented` means the code path exists.
- `Native Validated` means the behavior has been reproduced on native Termux
  or real hardware, depending on the subsystem.
- `Production Ready` means the subsystem is the default path for the release
  target and has the validation evidence that target requires.

## Subsystem Status

| Subsystem | Status | Notes |
| --- | --- | --- |
| ACL Engine | Implemented | Orchestrates ordered workflows and structured reports. |
| Workflow Engine | Implemented | Workflow reports and events exist. |
| Firmware Packaging | Native Validated | Package generation, flash plan, analysis, validation report, and README are in place. |
| Validation Engine | In Progress | Unit tests and native validation policies exist; provider coverage is still expanding. |
| Android Transport | Native Validated | Discovery, permission, diagnostics, and fd-handoff evidence are validated on native Termux. |
| Transport Stream Foundation | Implemented | Bounded stream contracts and diagnostics exist; native byte-stream validation is still pending. |
| Transport API | Stabilizing | Provider, manager, session, and stream contracts are acceptable for upload-engine foundation work; breaking changes are not expected. |
| Upload Engine | In Progress | The planner plus prepare-only executor exist, the positional CLI contract is validated, and real upload execution must remain transport-neutral until transport proof exists. |
| Flash Engine | Planned | Must remain transport-based and device-agnostic. |
| Serial Monitor | Planned | Must consume the same transport stream contract. |
| Device Manager | Planned | Should present transport and hardware state through reusable ACL data. |
| Board Manager | Planned | Should build on transport and compatibility infrastructure. |
| Library Manager | Planned | Should remain compatible with the ACL/runtime model. |
| IDE | Planned | Future UI layer over the same engine and reports. |

## Release Gates

Before claiming V1 release readiness:

1. Native Termux compile and workflow claims must be validated.
2. Native Termux USB discovery, permission, and transport diagnostics must be
   validated.
3. Real hardware upload and runtime claims must be validated on target boards.
4. The transport stream layer must have native validation for bounded byte
   stream behavior if upload or monitor features depend on it.
5. Documentation and validation findings must match the implemented behavior.
6. The transport API should remain additive unless a concrete contract bug makes a
   breaking change necessary.
7. The upload engine foundation must remain prepare-only until real transport
   execution is validated on native Termux and real hardware.
8. The upload CLI contract must stay explicit about positional package usage
   unless the architecture and validation docs are intentionally updated.

## Current Non-Goals

- USB flashing is not yet a release claim.
- Upload execution is not yet a release claim.
- Serial monitor behavior is not yet a release claim.
- Transport stream byte-read/byte-write behavior is still experimental.
