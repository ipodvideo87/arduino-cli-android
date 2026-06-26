# Android Install Patch Pipeline

The Android install patch pipeline formalizes the lifecycle used after packages
are downloaded and extracted.

It lives in `internal/acl/install`.

## Lifecycle

`Download -> Extract -> Android Patch -> Permission / Runtime Fixes -> Executable Validation -> Register -> Self-Test -> Ready`

## Design Rules

- The pipeline is automatic.
- The user should not have to repair permissions by hand.
- The pipeline should not be tied to a specific board family.
- The pipeline should carry patch-manifest state so later UI and diagnostics can
  explain what happened.

## Patch Manifest

The patch manifest records:

- package identity
- stage states
- applied fixes
- compatibility decisions
- status
- timestamps
- metadata

This is where runtime permission issues belong.

The builtin loader case that produced `.acl/runtime/ld-linux-aarch64.so.1
permission denied` should be tracked as a runtime fix, not as an invisible
manual repair.

Compatibility decisions belong here too when the install layer is selecting
between library versions or explaining why a version-aware patch is required.

## Current Implementation

The platform and tool install paths now invoke this shared pipeline after
extraction and before registration. The ACL executor wraps the existing Android
patcher so the pipeline can record the lifecycle without hardcoding board
specific behavior into Arduino CLI.

## Stage Model

The canonical stages are:

- download
- extract
- android-patch
- permission-runtime-fixes
- executable-validation
- register
- self-test
- ready

Each stage can be:

- pending
- running
- passed
- warning
- failed
- skipped

## Install Coverage

The shared stage should be invoked from:

- core install
- tool install
- any future package install path that carries executables

Source-only libraries should not be forced through ELF repair unless they
contain executable payloads.

## Related Code

- `internal/acl/install`
- `internal/acl/compatibility`
- [Transport Manager Architecture](TRANSPORT_MANAGER_ARCHITECTURE.md)
