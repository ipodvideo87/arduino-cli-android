# Engineering Debt Register

This document is the lightweight canonical place to track recurring
engineering-process debt.

Use it when the same class of problem appears more than once and the correct
response is not just to fix the immediate instance again.

This is a template and a starting point, not a large workflow system.

## When To Add An Entry

Add an entry when a problem:

- recurs twice or more
- requires repeated manual cleanup
- exposes a missing invariant or workflow step
- would benefit from a validator or automation but does not yet have one

## Entry Template

```md
## D-0001 - short debt title

- Class: architecture | validation | documentation | automation | process
- Reason: why this debt exists
- Impact: what it costs or risks
- Evidence: links to validation, findings, or concrete examples
- Exit criteria: what must be true before the debt can be retired
- Owner: who is tracking it
- Related docs: links to the relevant governance or knowledge docs
```

## Register Status

The register is currently empty.

That is intentional for Phase 6A. The file exists so the recurrence rule has a
canonical place to land once a real process bug has been observed twice.

## Guidance

- Keep entries short and evidence-linked.
- Prefer one entry per recurring class of problem.
- Do not use this file as a substitute for `STATUS.md`.
- Do not put implementation tasks here; keep it focused on process debt and the
  exit condition for retiring it.
