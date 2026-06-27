# Architecture Review Process

This document defines how architectural work should be reviewed before
implementation.

The purpose of review is to make architecture decisions deliberate rather than
accidental.

## Review Inputs

Before review begins, identify:

- the goal
- the affected subsystem or subsystems
- the proposed boundary or contract change
- the validation environment likely to matter
- the docs that already own related decisions

## Review Steps

### 1. Subsystem Identification

Determine which existing subsystem owns the problem and whether the proposed
change introduces a new one.

### 2. Dependency Analysis

Map what the change depends on and what depends on it.

This includes code dependencies, documentation dependencies, and workflow
dependencies.

### 3. Boundary Review

Check whether the change crosses or weakens an existing boundary.

Prefer keeping boundaries explicit rather than blending responsibilities into a
single layer.

### 4. Interface Review

Review all affected interfaces, including:

- public APIs
- internal APIs
- schemas
- report structures
- provider contracts
- diagnostics surfaces

Ask whether the new shape can evolve without forcing unnecessary breakage.

### 5. Duplication Review

Check whether the proposal duplicates responsibilities that already exist.

If duplication is deliberate, define why. If it is accidental, remove it before
implementation.

### 6. Android-Specific Impact

Assess whether the change is sensitive to:

- native Termux behavior
- Android permissions
- Android filesystem behavior
- Android loader behavior
- USB host mediation
- Bionic versus glibc differences

If yes, native Termux evidence should be part of the review plan.

### 7. Documentation Impact

Identify which documents own the current architecture, decision, or policy.

Review whether the change requires:

- a doc update
- a new ADR
- a reference update
- a historical note

### 8. Validation Impact

Decide what evidence will be needed to support the change.

Validation should match the claim being made. Host validation is not enough for
Android claims, and emulation is not enough for native Termux claims.

### 9. Testing Expectations

Define what tests should exist before or after implementation.

Tests should cover the contract that changed, not just the obvious happy path.

### 10. Long-Term Maintenance Review

Ask whether the change:

- reduces future cognitive load
- introduces extra maintenance burden
- creates a new special case
- increases the number of places a future fix would be required

If the long-term cost is unclear, stop and compare alternatives before
implementing.

## Review Outputs

A good architecture review should produce:

- a clear decision or recommendation
- the affected documents
- the affected code areas
- the validation path
- the known tradeoffs
- any open questions
- the next action

## When To Stop

Pause review and ask for confirmation when:

- the architecture could go in more than one valid direction
- the choice changes ownership boundaries
- the decision is hard to reverse
- the requested change looks like it might solve the wrong problem
- project history suggests a better path than the first obvious one

## Review Outcome

Architecture review is successful when the team can explain:

- what is changing
- why it is changing
- which subsystem owns it
- how the change will be validated
- what should remain stable
- what future contributors should not accidentally undo
