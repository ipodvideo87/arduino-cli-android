# Interface Stability Policy

This document defines how interfaces should evolve without breaking downstream
consumers unnecessarily.

It applies to public interfaces, internal interfaces, schemas, serialization,
firmware package evolution, transport interfaces, provider contracts, and
diagnostics contracts.

## Stability Principle

Interfaces should be stable enough that downstream consumers can rely on them,
but flexible enough that the repository can still evolve.

Stability does not mean freezing everything forever. It means changing things
carefully, with a clear migration path and a deliberate reason.

## Public Interfaces

Public interfaces should change conservatively.

Prefer:

- additive fields
- additive methods
- compatibility aliases
- explicit versioning when needed

Avoid breaking changes unless the existing shape is actively preventing the
architecture from moving forward.

## Internal Interfaces

Internal interfaces can evolve more quickly than public ones, but they still
deserve discipline.

Even internal contracts should be:

- named clearly
- owned by a specific layer
- documented enough to avoid guesswork
- validated against the behavior they describe

Internal does not mean disposable.

## Schema Compatibility

Schemas should evolve in a compatible way whenever practical.

Preferred patterns:

- add new optional fields
- keep old fields during transition
- version the schema when shape changes matter
- preserve parseability across a migration window

When compatibility cannot be preserved, document the break and the migration
path.

## Serialization

Serialized data should be treated as an interface, not just an implementation
detail.

The repository should avoid changing serialized shape casually because that can
break tools, reports, or workflows that depend on it.

If serialization changes, document:

- the old shape
- the new shape
- the compatibility plan
- the consumers that need attention

## Firmware Package Evolution

Firmware package formats should evolve with version awareness.

Changes should prefer:

- metadata additions over shape changes
- backward-compatible manifest updates
- explicit version fields when the package contract changes

If a package format change affects build outputs, install flows, or validation
reports, it should be treated as an interface change, not just a file update.

## Transport Interfaces

Transport interfaces should remain board-agnostic and capability-based.

They should evolve by:

- adding capabilities rather than hardcoding special cases
- preserving diagnostics during transitions
- keeping provider selection and stream semantics explicit

Changes should not collapse discovery, permission, session, stream, endpoint,
and diagnostics into one opaque abstraction.

## Provider Contracts

Provider contracts should be stable enough for orchestration layers and GUI
layers to depend on them.

Provider evolution should preserve:

- selection reason
- capability description
- environment context
- limitations and warnings
- lifecycle state

## Diagnostics Contracts

Diagnostics contracts are part of the user and tooling interface.

Changes to diagnostic reports should be treated as compatibility-relevant when
they affect:

- machine parsing
- user-facing summaries
- workflow comparisons
- validation records

## Evolution Rules

When evolving any interface:

1. Prefer additive changes.
2. Preserve old consumers during transition when practical.
3. Update the canonical docs.
4. Add or update validation.
5. Remove old shapes only after the migration path is complete.

## Stability Heuristic

If a change would require several unrelated consumers to update at once, the
interface is probably not yet stable enough to change casually.

If a change can be introduced additively, that is usually the safer path.
