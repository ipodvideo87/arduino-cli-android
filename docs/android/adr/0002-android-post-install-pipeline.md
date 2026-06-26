# ADR 0002: Android Post-Install Pipeline

## Status

Proposed

## Context

Android-specific package repair currently exists in a few installer paths, but
the work should be automatic and consistent.

The repo already patches platforms and tools after install, but the intended
behavior is broader:

- extract the package
- repair Android-specific issues
- validate the installed payload
- register the toolchain
- run a self-test when practical

The user should not have to manually patch packages after install.

## Decision

Introduce a shared Android post-install stage and call it from the relevant
installers.

Recommended behavior:

- core install invokes the shared stage for platforms and required tools
- tool install invokes it directly
- board installation remains orchestration only
- library install invokes it only when the library package contains executable
  payloads or other runtime-facing artifacts

The shared stage should handle:

- Android-specific patching
- executable-mode repair
- runtime fixups
- executable validation
- post-install self-test

## Consequences

Positive:

- Android install behavior becomes consistent across package types
- manual repair steps disappear
- install-time validation is centralized

Negative:

- the installer code path becomes more opinionated
- pure source-only library installs should not be forced through ELF repair
- the shared stage will need careful test coverage to avoid regressions

## Notes

This ADR does not claim the shared pipeline is implemented yet.

It records the install behavior that the Android docs and status should track.
