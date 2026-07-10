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

## D-0002 - canonical documentation drift

- Class: documentation
- Root cause class: workflow
- Reason: validated repository state can change while current-state documents
  remain stale or historical evidence can keep looking current unless the
  workflow forces synchronization and explicit classification.
- Impact: canonical docs can drift from the validated repository state, which
  weakens auditability and can mislead later reviews about what is current.
- Evidence: the Phase 6 after-state audit found `STATUS.md` stale after the
  Phase 6 kernel commit, while `ENGINEERING_MILESTONE_SUMMARY.md` retained
  repository-state details that needed explicit historical labeling.
- Smallest durable improvement: require a current-state synchronization
  determination during closeout, require explicit historical classification for
  archival docs, and extend governance validation with bounded deterministic
  checks for routing, ownership, stale-path hygiene, and historical labels.
- Exit criteria: current-state synchronization determination is required at
  closeout; historical documents are explicitly classified; deterministic
  checks catch the defined stale-state failures; and at least one later complete
  milestone cycle closes without recurrence.
- Owner: Codex governance workflow
- Related docs: `docs/android/DEVELOPMENT_WORKFLOW.md`,
  `docs/android/DOCUMENTATION_ARCHITECTURE.md`,
  `docs/android/GOVERNANCE_COVERAGE_MATRIX.md`

## Guidance

- Keep entries short and evidence-linked.
- Prefer one entry per recurring class of problem.
- Do not use this file as a substitute for `STATUS.md`.
- Do not put implementation tasks here; keep it focused on process debt and the
  exit condition for retiring it.
