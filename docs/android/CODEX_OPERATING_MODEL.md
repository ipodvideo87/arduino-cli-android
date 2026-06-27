# Codex Operating Model

This document defines how Codex should operate in this repository.

It is an execution model, not a product policy. It complements
[AGENTS.md](/root/arduino-cli-android/AGENTS.md), the engineering principles,
the decision framework, and the diagnostic validation standard.

## Default Starting Sequence

When a task arrives, start by identifying what kind of work it is:

- documentation only
- research only
- code inspection
- implementation
- validation
- review or audit
- cross-subsystem architecture work

Then:

1. Clarify the objective in plain terms.
2. Read the canonical docs first.
3. Inspect code before planning when the task may affect runtime behavior,
   architecture, validation claims, interfaces, or documentation accuracy.
4. Research before implementing when Android behavior, external APIs, device
   behavior, or project history is uncertain.
5. Compare plausible approaches before choosing one.
6. Ask questions when user intent, tradeoffs, or environment assumptions are
   unclear.
7. Plan before acting when the task is multi-step or spans subsystems.

## What To Read First

The exact reading order depends on the task, but the usual first pass is:

1. `MISSION.md`
2. `STATUS.md`
3. `AGENTS.md`
4. `docs/android/ENGINEERING_PRINCIPLES.md`
5. `docs/android/DECISION_FRAMEWORK.md`
6. `docs/android/DOCUMENTATION_ARCHITECTURE.md`
7. `docs/android/VALIDATION_POLICY.md`
8. `docs/android/DIAGNOSTIC_VALIDATION_STANDARD.md`
9. The most relevant subsystem docs for the task

For Android-specific or architecture-level work, also inspect the relevant code
paths before deciding on a direction.

## When To Inspect Code

Inspect code before planning when:

- the task may change runtime behavior
- a document might be stale or ambiguous
- an interface contract could be involved
- the requested work spans multiple subsystems
- validation claims must be compared against implementation
- existing behavior needs to be preserved while adding new work

Do not guess about current behavior if the code can be checked directly.

## When To Research

Research before implementing when:

- the project history contains prior evidence or rejected approaches
- Android or Termux behavior may differ from host behavior
- there is any meaningful chance the current intuition is incomplete
- an external API, device behavior, or validation environment matters
- multiple approaches appear viable and their tradeoffs matter

Research should reduce uncertainty, not delay indefinitely.

## When To Ask Clarifying Questions

Ask the user when:

- the objective can be interpreted in more than one valid way
- multiple approaches are technically viable but imply different tradeoffs
- the desired user experience is unclear
- a change would be hard to reverse
- the request conflicts with existing architecture or evidence
- the task depends on an environment or device choice that is not obvious

When multiple valid approaches exist, compare them first, then ask the user to
confirm the intended direction before implementing.

## When To Use Plan Mode

Use Plan Mode when the work is:

- multi-step
- cross-subsystem
- architecture-heavy
- validation-heavy
- likely to require tradeoff analysis
- likely to need checkpoints before implementation

Plan Mode should improve coordination and reduce mistakes. It should not be
used to avoid making a decision that the evidence already supports.

## When Not To Implement

Do not implement when:

- the user explicitly requested audit, documentation, or research only
- the evidence is insufficient to justify a change
- the change would encode a guess as a contract
- the task should stop at analysis or comparison
- the requested solution is the first idea but not yet the best supported one

If a better architecture is suggested by the docs, code, or project history,
compare it before acting.

## How To Classify Uncertainty

Label uncertainty explicitly:

- Confirmed: directly supported by repository evidence or validation
- Inferred: strongly suggested by evidence, but not directly proven
- Hypothesis: plausible but not yet supported enough to treat as likely
- Unknown: not enough evidence to decide

Do not collapse these categories into a single vague statement.

## How To Choose Between Approaches

Compare candidate approaches against:

- architectural fit
- separation of concerns
- compatibility with native Termux and Android constraints
- future maintenance cost
- extensibility
- validation cost
- risk of duplication
- risk of future confusion

Prefer the option that creates the strongest long-term structure, not the one
that merely satisfies the immediate request fastest.

If multiple options remain viable after comparison, ask the user to confirm the
direction.

## How To Avoid Overclaiming

Use the repository's diagnostic validation standard for every completion report
and milestone summary.

Avoid words like `works`, `passed`, and `complete` unless they are immediately
qualified by:

- the environment
- the executed commands
- the evidence collected
- the validation level achieved
- what the evidence does not prove

## Completion Expectations

Every completion report should make it easy for the next person to answer:

- what changed
- why this direction was chosen
- what evidence supports the claim
- what remains unknown
- what should be validated next

The report should leave the repository easier to maintain than before by:

- tightening docs that drifted
- preserving useful evidence
- adding missing tests or validation notes when appropriate
- keeping ownership boundaries clear
- recording open questions explicitly
