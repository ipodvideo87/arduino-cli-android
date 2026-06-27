# Repository Governance

This document defines how the repository should evolve over time.

It is a governance guide for new architecture, interfaces, schemas,
documentation, and implementations.

## Governing Principle

The repository should evolve deliberately, with clear ownership and explicit
evidence, rather than by layering unrelated fixes until the architecture becomes
hard to reason about.

## Introducing New Subsystems

Before adding a subsystem, determine:

- what problem it solves
- whether an existing subsystem already owns the responsibility
- what its inputs and outputs are
- what subsystem boundaries it creates or crosses
- what validation evidence will be needed
- whether the subsystem should be reusable or board-specific

New subsystems should have a narrowly defined responsibility and a clear
relationship to the rest of the architecture. They should not quietly absorb
responsibilities that belong elsewhere.

## Introducing New Interfaces

Interfaces should exist to stabilize a meaningful boundary, not to create
abstraction for its own sake.

When introducing a new interface, decide:

- who owns it
- whether it is public, internal, or experimental
- what downstream consumers depend on it
- how it can evolve without forcing unnecessary breakage
- what validation will prove it is fit for use

New interfaces should be additive by default. If an interface replaces an old
one, the migration path should be explicit and documented.

## Deprecating Old Systems

Deprecation should be intentional and visible.

Before retiring an old system:

- preserve the useful evidence or historical context
- identify what replaces it
- explain the migration path
- note any compatibility risk
- avoid leaving two competing authorities in active use

If a retired system still contains useful reasoning, preserve it as historical
context rather than letting it remain a second live source of truth.

## Architectural Review Expectations

Any change that affects architecture should be reviewed for:

- subsystem impact
- interface impact
- duplication risk
- Android-specific constraints
- documentation impact
- validation impact
- maintenance cost

Architectural review should happen before implementation when the change could
reshape ownership or create future drift.

## ADR Expectations

Create an ADR when a decision is:

- long-lived
- hard to reverse
- architecture-shaping
- likely to have tradeoffs that will matter later
- likely to be re-discussed if not recorded

ADRs should record the decision, alternatives, rationale, and consequences. They
should not be used for routine implementation notes.

## Ownership Of Long-Lived Decisions

Long-lived decisions should have one canonical home.

The owning document should be the place where future contributors look first for
the authoritative version of the rule, contract, or policy. Other documents may
summarize it for local readability, but they should not redefine it.

## Backwards Compatibility Expectations

Compatibility should be preserved when practical.

Breaking changes should be avoided unless they clearly improve the architecture
or are required to fix a real contract problem. When breakage is unavoidable:

- document why it is necessary
- describe the migration path
- call out affected consumers
- record the validation evidence needed before the change can be considered
  stable

## Schema Evolution

Schemas should evolve in a version-aware way.

Prefer additive fields and compatibility-preserving changes. If a schema change
requires a migration, document:

- the old shape
- the new shape
- what consumers must do
- whether both shapes will be supported during transition

Treat schema evolution as a maintenance decision, not just a serialization
detail.

## API Evolution

APIs should evolve conservatively and predictably.

Prefer additive changes and compatibility aliases where possible. When a public
or internal API changes:

- assess whether consumers can stay source-compatible
- preserve names where doing so reduces confusion
- avoid changing contracts casually
- validate the new shape before removing the old one

## Migration Guidance

Migration should be planned, not improvised.

Good migrations make the old and new worlds easy to distinguish. They explain:

- what changes
- what stays supported temporarily
- what must be updated
- when the transition is complete

## Repository Health

Repository health means more than green checks.

A healthy repository has:

- clear ownership
- clear boundaries
- evidence-backed claims
- controlled terminology
- documented migration paths
- a manageable amount of debt
- no hidden competing authorities

Health should be judged by whether future work can proceed safely and
predictably.

## Preventing Architectural Drift

Architectural drift happens when new work quietly bypasses existing contracts.

To prevent it:

- review the current architecture before adding behavior
- prefer existing boundaries over new ad hoc ones
- keep docs aligned with implementation
- retire obsolete paths explicitly
- preserve evidence about why decisions were made

The test for good governance is whether the next contributor can tell which
document owns the decision and why.
