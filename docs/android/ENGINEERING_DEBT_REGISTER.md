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
- Root cause class: process | documentation | workflow | tooling | validation |
  architecture | external | unknown
- Reason: why this debt exists
- Impact: what it costs or risks
- Evidence: links to validation, findings, or concrete examples
- Smallest durable improvement: the specific prevention change that removes the
  recurrence path
- Exit criteria: what must be true before the debt can be retired
- Owner: who is tracking it
- Related docs: links to the relevant governance or knowledge docs
```

## Register Status

Current entries:

## D-0001 - vague closeout reporting

- Class: process
- Root cause class: workflow
- Reason: closeout reports can be technically complete but still too vague to
  satisfy the repository's evidence-before-claims and auditability standards.
- Impact: manual follow-up is needed after closeout, which slows review and
  weakens traceability.
- Evidence: repeated closeout reports that named completion without fully
  surfacing repository path, file-level changes, validation detail, and final
  verdict.
- Smallest durable improvement: require `docs/android/CLOSEOUT_REPORTING_STANDARD.md`
  in `docs/android/DEVELOPMENT_WORKFLOW.md` and use it as the required report
  shape.
- Exit criteria: closeout reports consistently follow
  `docs/android/CLOSEOUT_REPORTING_STANDARD.md` and include the required
  fields without manual prompting.
- Owner: Codex closeout workflow
- Related docs: `docs/android/DEVELOPMENT_WORKFLOW.md`,
  `docs/android/CLOSEOUT_REPORTING_STANDARD.md`

## Guidance

- Keep entries short and evidence-linked.
- Prefer one entry per recurring class of problem.
- Do not use this file as a substitute for `STATUS.md`.
- Do not put implementation tasks here; keep it focused on process debt and the
  exit condition for retiring it.
