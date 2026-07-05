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

## Solve Recurring Friction at the Class Level

- Lesson: when a problem recurs or creates significant friction, identify the
  class of problem and make the smallest evidence-based improvement that
  prevents the same class of problem from wasting time again.
- Why it matters: repeated manual fixes and repeated diagnostic churn slow the
  repository down more than the original bug does.
- Evidence source: stale binary/source mismatch, the native evidence collector
  bridge, `target_chip` freshness checks, and the canonical worktree rule.
- When to apply it: when the same failure pattern, copy/paste burden, or sync
  confusion shows up more than once.
- What not to infer from it: this is not permission to automate everything or
  add complexity without proof that the pain is real.

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
