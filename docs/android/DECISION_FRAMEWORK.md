# Decision Framework

This document defines how engineering decisions should be made in this
repository.

## Decision Inputs

Every meaningful decision should consider:

- user intent
- current mission and status
- canonical architecture docs
- the engineering knowledge framework
- validation policy and diagnostic reporting rules
- existing code and interfaces
- project history and prior evidence
- affected subsystems
- maintenance cost
- validation cost
- future extension points

## Decision Confidence

Decision confidence is defined in [CONFIDENCE_MODEL.md](CONFIDENCE_MODEL.md).
Use that document as the canonical source for confidence terminology and
confidence-change rules.

## Decision Flow

```text
Task received
↓
Clarify objective
↓
Read canonical docs
↓
Inspect existing architecture and code
↓
Research if evidence is incomplete or Android behavior may differ
↓
Compare feasible approaches
↓
Ask questions if user intent or tradeoffs are unclear
↓
Plan
↓
Implement
↓
Validate
↓
Report evidence
↓
Update docs and status
```

## When To Research

Research when:

- Android or Termux behavior may differ from host behavior
- the repository history contains prior findings or rejected approaches
- a target API, device, or transport is not well understood
- the change touches a contract that can evolve independently
- the decision might otherwise encode an assumption as fact

## When To Stop And Ask Questions

Stop and ask when:

- user intent is ambiguous
- two or more approaches are technically viable
- the choice changes architecture, naming, ownership, or validation scope
- the change is expensive or hard to reverse
- the project would benefit from a deliberate direction choice

If the choice is mostly a strategy question rather than a requirement question,
compare the options first, then ask the user to confirm the intended direction.

## When To Proceed

Proceed when:

- the objective is clear
- the canonical docs agree on the direction
- the code and evidence do not contradict the planned change
- the affected subsystems are identified
- the long-term maintenance cost is acceptable
- the validation path is known

## How To Compare Approaches

Compare candidate approaches by asking:

- Which option best matches the architecture?
- Which option keeps subsystem boundaries clean?
- Which option is easiest to validate in native Termux?
- Which option is easiest to maintain?
- Which option reduces future duplication?
- Which option preserves extensibility for future transports, workflows, or
  devices?

Prefer the approach that is most durable and least likely to create another
cleanup task later.

## How To Evaluate Maintenance Cost

Estimate maintenance cost by looking at:

- how many files would need to change later
- whether the change creates a new ownership boundary
- whether the change duplicates an existing concept
- whether the change requires special-case logic
- whether the change is board-specific or reusable
- whether the change will be hard to validate again

The right answer is often the one that makes future changes smaller.

## How To Identify Affected Subsystems

Trace the change through:

- canonical docs
- adjacent workflows
- validation docs
- implementation packages
- reporting surfaces
- user-facing commands
- future GUI or workspace dependencies

If a change crosses subsystem boundaries, treat it as an architecture decision,
not just a local code edit.

## How To Avoid Architecture Shortcuts

Avoid shortcuts that:

- hardcode a specific board or transport
- collapse separate layers into one function
- duplicate logic in multiple owners
- encode a temporary workaround as a contract
- hide uncertainty behind vague language
- bypass the canonical validation path

## User Intent Versus Implementation Strategy

User intent is the outcome the user wants.
Implementation strategy is one way to reach that outcome.

Do not assume the first implementation strategy is the intended one.
If project history or architecture suggests a better strategy, compare it and
confirm before implementing.

## Decision Output

A good decision record should say:

- what was chosen
- why it was chosen
- what alternatives were considered
- what evidence supported the choice
- what confidence level applies
- what remains to be validated
- what docs need to be updated

For task-level provenance, also record the question and uncertainty in the
engineering knowledge framework.
