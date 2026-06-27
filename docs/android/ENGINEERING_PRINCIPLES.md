# Engineering Principles

These are the durable engineering principles for this repository.

## Principles

- Android-first development
  - Treat Android and native Termux as the primary design target.

- Native Termux as source of truth
  - When Android behavior and host behavior differ, native Termux wins.

- Evidence before claims
  - Do not elevate a conclusion above the evidence that supports it.

- Architecture before implementation
  - Understand subsystem boundaries and contracts before adding behavior.

- Research before assumptions
  - Investigate project history, current docs, and target-environment behavior
    before guessing.

- Interfaces before concrete implementations
  - Define stable boundaries first so implementations can evolve behind them.

- Production-quality architecture over quick fixes
  - Prefer durable structure over the fastest local patch.

- Separation of concerns
  - Keep discovery, permission, connection, protocol, upload, monitor,
    diagnostics, and UI responsibilities distinct.

- Single source of truth where practical
  - Canonical definitions, policies, standards, and contracts should have one
    home.

- Local context is allowed for readability
  - Small summaries and reminders are good when they help a reader stay on the
    page.

- Automation over repeated manual steps
  - Convert recurring operational work into reusable workflows, tests, or
    scripts.

- Reduce future maintenance burden
  - Prefer changes that make the next change simpler and safer.

- If a mistake can happen twice, make it harder to happen a third time
  - Add structure, validation, tests, or documentation that prevents the same
    failure mode from repeating.

- Every task should leave the repository easier to understand, validate, or
  maintain
  - Even small changes should improve clarity, evidence, or boundaries.

- Preserve compatibility when practical
  - Avoid unnecessary breaking changes when additive evolution is possible.

- Prefer board-agnostic and descriptor-driven designs
  - Do not hardcode device-family assumptions when the problem can be modeled
    generically.

- Document what the repository proves, not what it merely hopes to prove
  - Keep planned and experimental behavior labeled as such.
