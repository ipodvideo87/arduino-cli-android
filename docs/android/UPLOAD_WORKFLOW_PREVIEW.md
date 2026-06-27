# Upload Workflow Preview

This is a preview of the future upload workflow that will consume the Android
USB transport framework.

It is a reference and operational preview, not a policy document.

It is not an implementation.
It does not perform upload.
It does not perform USB flashing.

The underlying provider/session/endpoint contracts now exist as compile-safe
ACL infrastructure, but the workflow itself remains future work.

The prepare-only contract and evidence-reporting rules are defined in
[UPLOAD_ENGINE_ARCHITECTURE.md](UPLOAD_ENGINE_ARCHITECTURE.md),
[VALIDATION_POLICY.md](VALIDATION_POLICY.md), and
[DIAGNOSTIC_VALIDATION_STANDARD.md](DIAGNOSTIC_VALIDATION_STANDARD.md).

## Goal

The upload workflow should let ACL and future GUI layers ask one question:

What transport is available, what can it do, and how do we use it safely?

Current implementation note:

- the ACL upload engine foundation now exists as a planner plus a prepare-only
  executor
- it consumes firmware packages, flash plans, and validation metadata
- it is exposed as `acl workflow upload <firmware-package>` and remains
  positional only
- `--dry-run` and `--package` are not part of the workflow contract
- the canonical executor report keeps one package section, one plan section,
  one diagnostics section, one result section, and one progress array instead
  of duplicating the same details in nested layers
- it does not open real transport streams or send bytes yet

## Intended Flow

```text
Transport discovery
  ↓
Permission acquisition
  ↓
Session open
  ↓
Protocol identification
  ↓
Endpoint / adapter selection
  ↓
Upload backend execution
  ↓
Post-upload validation
  ↓
Workflow report
```

## Transport Examples

The upload workflow should be able to work with multiple transport families.

Examples:

- USB serial upload to a serial-flash backend
- CDC ACM upload to a serial-flash backend
- DFU upload to a DFU backend
- HID upload to a HID-capable backend
- CMSIS-DAP upload to a debug-probe backend
- JTAG/SWD upload via a probe backend
- network transport upload through a bridge or remote endpoint

## Workflow Responsibilities

The upload workflow should:

- choose a provider by capability
- record how permission was acquired
- record the selected device, interface, and endpoint
- record alternatives that were available
- hand off to a protocol-specific upload backend
- keep Android USB details out of Arduino CLI command code
- emit structured diagnostics for beginner, advanced, and professional views

## What Upload Is Not

Upload is not:

- discovery
- permission acquisition
- generic USB connection management
- board-specific transport logic
- serial monitor streaming

Those are separate layers.

## Failure Modes To Model

The preview should be able to express:

- no transport available
- permission denied
- device detached during selection
- interface mismatch
- endpoint mismatch
- backend protocol unsupported
- upload backend failure
- post-upload verification failure

## Report Expectations

The upload report should include:

- selected provider
- alternatives
- device identity
- permission source
- selected protocol
- upload backend chosen
- warnings
- limitations
- validation level
- human-readable beginner summary
- professional diagnostic details

The exact evidence and claim structure should follow the diagnostic validation
standard.

## ACL Integration

The upload workflow should live behind the ACL Engine so the GUI can call a
workflow instead of direct Android USB APIs.

The workflow should be able to consume:

- `TransportManager` selection results
- compatibility decisions
- firmware package metadata
- validation provider evidence

## Practical Conclusion

The architecture should be transport-driven, provider-based, and protocol-aware.
The upload workflow should not be the layer that knows Android USB details.
