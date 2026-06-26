# Diagnostics Workflow

The diagnostics workflow is the structured progress model used for USB,
build, and toolchain work.

It lives in `internal/acl/diagnostics`.

## Status Vocabulary

The workflow uses a shared status vocabulary:

- pending
- running
- passed
- warning
- failed
- skipped

These statuses are intentionally generic so they can drive future progress bars
and log views.

## Workflow Shape

A workflow is a named ordered set of steps.

Each step can carry:

- status
- message
- evidence
- metadata
- timestamps

## Use Cases

The same workflow model can describe:

- USB acquisition
- build stages
- patch pipeline stages
- binary validation
- toolchain readiness

## Current Implementation

The firmware package validator, Android patch pipeline, and compatibility
reports already use this shared status vocabulary in code. The UI can consume
the same shape later without changing the backend model.

## UI Implication

The UI can render the same workflow in three ways:

- beginner: a condensed status line
- advanced: a step list with warnings
- professional: a full diagnostic trace

## Related Code

- `internal/acl/diagnostics`
- `internal/acl/install`
- `internal/acl/firmware`
