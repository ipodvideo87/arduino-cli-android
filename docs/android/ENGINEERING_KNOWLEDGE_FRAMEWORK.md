# Engineering Knowledge Framework

This document defines how engineering work becomes permanent repository
knowledge.

The framework is meant to make every completed task answer the same set of
questions in a durable, version-controlled form:

- What question were we trying to answer?
- What evidence was collected?
- What engineering decision was made?
- What uncertainty was eliminated?
- What uncertainty remains?
- What confidence changed?
- What lesson was learned?
- Which documentation was updated?
- How did this influence the roadmap?

## Purpose

The repository should not only accumulate implementation. It should accumulate
repeatable engineering knowledge that makes the next decision easier to make
and harder to repeat incorrectly.

This framework exists to:

- preserve evidence with context
- keep uncertainty visible instead of implicit
- make decisions traceable
- separate validated knowledge from speculation
- promote durable lessons into canonical docs
- reduce repeated investigation

## Canonical Ownership

The engineering knowledge system has multiple canonical homes with distinct
responsibilities:

- `ENGINEERING_DECISIONS.md` owns long-lived ADR-style decisions
- `VALIDATED_FINDINGS.md` owns evidence-backed findings and validation notes
- `DECISION_LOG.md` owns task-level decision provenance
- `UNCERTAINTY_REGISTER.md` owns open questions and closure criteria
- `CONFIDENCE_MODEL.md` owns confidence semantics and change rules
- `LESSONS_LEARNED.md` owns durable lessons distilled from repeated evidence

This document owns the policy that connects those artifacts into one lifecycle.
It does not replace their individual responsibilities.

## Required Closeout Questions

Any substantive engineering task should leave behind enough material to answer
the following:

1. What question were we trying to answer?
2. What evidence did we collect?
3. What decision did we make?
4. What uncertainty was removed?
5. What uncertainty remains?
6. What confidence changed?
7. What lesson did we learn?
8. What documentation changed?
9. What roadmap impact followed?

If the task cannot answer these questions, the task is not fully closed from a
knowledge perspective.

## Artifact Routing

Use the smallest artifact that owns the knowledge:

- short-lived uncertainty belongs in `UNCERTAINTY_REGISTER.md`
- a concrete task decision belongs in `DECISION_LOG.md`
- a durable architecture decision belongs in `ENGINEERING_DECISIONS.md`
- evidence and validation outcome belong in `VALIDATED_FINDINGS.md`
- reusable guidance belongs in `LESSONS_LEARNED.md`
- confidence interpretation belongs in `CONFIDENCE_MODEL.md`

## Promotion Rules

Knowledge should move through the repository deliberately:

1. Capture the active question and the current uncertainty.
2. Gather evidence in the appropriate validation environment.
3. Record the decision and confidence change.
4. Close or narrow the uncertainty.
5. Promote any durable lesson into canonical guidance.
6. Update status and roadmap only when the change affects the project view.

Do not promote speculative reasoning directly into a canonical rule.

## Operating Rule

Every completed engineering task should leave the repository with:

- one more verified fact
- one fewer unresolved question, or a clearly narrowed one
- one explicit decision trail
- one clearer confidence boundary
- one updated doc trail

That is the mechanism by which repository knowledge compounds over time.

