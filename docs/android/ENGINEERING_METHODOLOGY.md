# Engineering Methodology

This repository is engineered for sustained correctness, not just fast feature
throughput.

The goal is to make future work easier to reason about than the work that came
before it. That means the project is optimized for long-lived maintainability,
evidence-based decisions, and architectural clarity rather than for whichever
change is quickest in the moment.

## Why This Repository Is Built This Way

Android and Termux are not just another deployment target here. They impose
their own filesystem, loader, permission, USB, and runtime boundaries. If the
project is going to survive on Android, it cannot rely on assumptions borrowed
from desktop Linux.

That is why the repository values evidence before confidence and research
before assumptions. A change that feels plausible is not enough when the target
environment can invalidate the intuition. The cost of being wrong is usually
not a small local bug; it is a misleading architecture that later work has to
unwind.

The project therefore prefers to understand a problem deeply before solving it.
That sounds slower in the short term, but it reduces the total amount of work
the repository will need over time. The real goal is to prevent a class of
problems from recurring, not to keep fixing one instance of the same mistake.

## Maintenance Over Velocity

Short-term velocity is useful only if it does not create a larger future burden.
This repository is structured to avoid shipping local patches that later become
hidden obligations.

When a task is handled well, it should leave behind:

- a clearer boundary
- a more durable interface
- a better validation story
- a better doc trail
- a smaller cognitive load for the next contributor

This is why the project cares so much about architecture, reporting structure,
and documentation ownership. A system that teaches future contributors how to
work is easier to maintain than a system that requires every person to
rediscover the rules.

## Evidence As An Engineering Input

Evidence is not a postscript. It is part of the design process.

Claims about Android behavior, validation, or architecture should be grounded in
observable evidence rather than confidence alone. This keeps the repository from
accidentally turning speculation into policy.

The same principle applies to documentation. A document should explain what the
repository proves, what it believes, and what remains unknown. That makes the
docs useful as engineering assets instead of as narrative summaries.

## Automation As A Maintenance Strategy

Manual repetition scales poorly. If a step has to be repeated often, the
repository should ask whether it can be automated, formalized, or encoded as a
workflow.

That preference is not about eliminating human judgment. It is about reserving
human judgment for the parts of the problem that actually need it, while
mechanizing the routine parts that would otherwise accumulate mistakes.

## Reduce Cognitive Load

Future contributors should not need to hold the entire project in memory to make
a safe change.

The repository reduces cognitive load by:

- separating concerns
- keeping canonical docs distinct from supporting context
- naming contracts consistently
- making validation scope explicit
- preserving historical evidence without confusing it for current policy

This is also why small readable summaries are allowed. Local context helps
readability, but only if it does not compete with the canonical source of truth.

## Teaching Through The System

The best engineering system teaches by example.

If a contributor reads the docs, inspects the code, and looks at validation
artifacts, they should be able to infer:

- how decisions are made
- what evidence matters
- which layers own which responsibilities
- how to evolve the system without breaking it

That is the purpose of this methodology: to make the repository more legible as
it grows.
