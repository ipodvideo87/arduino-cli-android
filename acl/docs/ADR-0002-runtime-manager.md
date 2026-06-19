# ADR 0002: ACL Runtime Manager

## Status

Accepted

## Context

Sprint 1 confirmed that the copied Termux/glibc runtime is useful for investigation
but not relocatable as-is. Hardcoded paths and loader assumptions mean ACL needs its
own runtime lifecycle instead of treating the copied tree as production input.

## Decision

ACL will manage runtimes through a purpose-built runtime manager with:

- an ACL-owned runtime root
- one directory per installed runtime
- a `manifest.json` file per runtime
- an `active.json` selection record at the root
- validation before install, select, or activate
- compatibility reporting without execution

Runtime packages are expected to be produced for ACL, not copied from Termux.
The copied runtime remains an investigation reference only.

## Consequences

- Runtime layout is independent of the original Termux/glibc tree.
- Runtime builders can evolve separately from execution logic.
- Integrity checks can be performed before any future launch step.
- The manager can later support multiple runtime versions and compatibility levels.

## Non-Goals

- Runtime execution.
- Automatic runtime patching.
- Reuse of the copied Termux runtime as a production package format.

## Follow-Up

The next sprint should add a runtime builder that emits ACL-native packages matching
this manifest and directory contract.
