# Living Institutional Memory

This repository uses documentation as living institutional memory.

Important project knowledge should live in version-controlled docs, not only in
chat history or commit history. The goal is for future humans and future AI agents
to recover context quickly and accurately.

## What The Docs Must Preserve

- What was built.
- Why it was built that way.
- What alternatives were considered.
- What has been validated.
- What is still unknown.
- What future agents should not accidentally undo.
- What validation provider was used and what confidence boundary it proves.
- How future contributors should reason, decide, document, and report in this
  repository.
- How task-level questions, decisions, uncertainty, confidence, and lessons are
  preserved in the engineering knowledge framework.
- How the repository should evolve over time without accumulating avoidable
  architectural drift.
- How governance batches and documentation-ownership rules should be preserved
  as durable project memory.

## Operating Rules

- Update the relevant docs when architecture, assumptions, or validation evidence
  changes.
- Treat documentation as part of the product, not paperwork.
- Prefer explicit evidence over implied understanding.
- Record unknowns clearly instead of silently filling gaps with assumptions.
- Keep `STATUS.md` aligned with the current validated state.
- Keep validation provider reports, scripts, and the research log in versioned
  docs so future contributors do not have to reconstruct the evidence chain from
  chat history.
- Preserve governance frameworks and documentation-ownership rules in versioned
  docs so later work can build on the same decision model instead of recreating
  it.
- Preserve the engineering knowledge framework in versioned docs so future work
  can inherit the same question -> evidence -> decision -> lesson lifecycle.
- Preserve repository-governance, debt, and interface-stability guidance in
  versioned docs so future evolution remains deliberate rather than accidental.
- Preserve batch-level governance changes in status and roadmap docs so future
  contributors can see how the documentation system evolved.
