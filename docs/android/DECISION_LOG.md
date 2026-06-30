# Decision Log

This document is the task-level decision provenance log for the repository.

It records the path from question to evidence to decision so future contributors
can understand why a change was made without reconstructing the full chat or
history.

## Purpose

Use this log for decisions that are important enough to keep, but narrower than
an ADR.

Examples:

- choosing between two validation strategies
- selecting a next milestone
- narrowing a transport boundary
- resolving an uncertainty that shaped implementation

## What A Log Entry Must Answer

Every entry should make the following obvious:

- the question being answered
- the evidence used
- the decision made
- the alternatives considered
- the confidence change
- the uncertainty that remains
- the docs that were updated
- the roadmap effect

## Recommended Entry Shape

```md
## YYYY-MM-DD - short title

- Question: ...
- Evidence: ...
- Decision: ...
- Alternatives: ...
- Confidence: before -> after
- Uncertainty removed: ...
- Uncertainty remaining: ...
- Docs updated: ...
- Roadmap impact: ...
- Follow-up: ...
```

## Relationship To Other Records

- Use `ENGINEERING_DECISIONS.md` for durable architecture decisions.
- Use `VALIDATED_FINDINGS.md` for evidence-backed findings.
- Use `UNCERTAINTY_REGISTER.md` for open questions.
- Use `LESSONS_LEARNED.md` for distilled reusable lessons.

The decision log connects those records but does not replace them.

## Entry Rules

- Keep entries chronological.
- Keep entries short enough to scan.
- Do not duplicate a full ADR.
- Do not omit uncertainty just because the decision was made.
- Link out to the evidence source when the full detail belongs elsewhere.

## Closure Rule

When a task is closed, the decision log entry should explain why the remaining
uncertainty is acceptable, or where it was moved if it is still active.

