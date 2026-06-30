# Lessons Learned

This document collects durable engineering lessons that are worth preserving as
general guidance.

It is not a decision log and not an ADR. It is the distilled memory of what the
repository has learned repeatedly enough to be worth saying plainly.

## Purpose

Use this document for guidance that should influence future work across tasks
and milestones.

Good lessons are:

- reusable
- validated by evidence or repeated project history
- specific enough to be actionable
- broad enough to matter again

## Promotion Criteria

Move an insight into this document when it:

- has been validated more than once, or
- prevents a high-cost mistake from recurring, or
- explains a pattern that future tasks are likely to encounter again

Do not promote a one-off observation that belongs only in a decision log or
findings note.

## Recommended Entry Shape

```md
## Lesson title

- Lesson: ...
- Why it matters: ...
- Evidence source: ...
- When to apply it: ...
- What not to infer from it: ...
```

## Relationship To Other Docs

- `VALIDATED_FINDINGS.md` records the evidence.
- `DECISION_LOG.md` records the decision that followed.
- `ENGINEERING_DECISIONS.md` records durable architecture decisions.
- `UNCERTAINTY_REGISTER.md` records what still needs proof.

This file records the reusable lesson that survives across tasks.

## Caution

Lessons should not become a second source of policy.

If a lesson changes a rule, promote the rule to the appropriate canonical
document and leave the lesson here as supporting context.

