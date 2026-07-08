# Self-Improving Engineering Framework

This document defines the smallest read-only framework that helps Codex decide
when to run or recommend deeper engineering checks.

It does not define a new subsystem, a broad rule engine, or a mutation tool.
It defines when to ask for more evidence, when to validate policy, and when to
treat repetition as a process bug.

## Purpose

The repository already has policy, validation, and knowledge documents. This
framework connects them so repeated work does not stay invisible.

The goal is to make Codex do three things consistently:

- run cheap, deterministic checks automatically when they are clearly
  warranted
- recommend deeper checks when the task is cross-cutting or uncertain
- record recurring friction as process debt instead of fixing the same class of
  problem twice

## Core Rule

If the same problem appears twice, treat it as a process bug.

At that point, Codex should recommend one of the following:

- an invariant update
- a workflow update
- a validator check
- an automation opportunity
- an explicit reason not to automate

Do not keep applying the same one-off fix without recording why the system is
still allowing the same class of issue to recur.

## Trigger Catalog

### At Session Start

Run or recommend checks when the task is likely to benefit from extra
structure:

- if the task resumes after interruption, read the recovery snapshot first
- if the task touches governance, docs, validation claims, or repo hygiene,
  recommend `acl governance validate`
- if the task makes or revises claims, recommend checking evidence support
- if the task spans multiple subsystems or has repeated friction, recommend a
  broader health review and consult the coverage matrix

### Before Closeout

Before closing a task:

- run governance validation when the task changed repo policy, documentation
  boundaries, or current-state reporting
- recommend an evidence review when claims, docs, or validation results changed
- record recurring friction in the debt register or the knowledge docs
- if a problem appeared twice, require a process-level follow-up before the
  task is treated as finished

### After Failed Validation

After a failed validation:

- classify the failure by layer before changing anything
- recommend the narrowest next check that can distinguish evidence from
  assumption
- if the failure is repeated or preventable, log it as debt or a recurring
  process issue
- do not expand the scope unless the evidence forces it

### When Docs And Evidence Disagree

When a document says one thing and evidence says another:

- treat the evidence as authoritative
- mark the document as stale, incomplete, or overclaimed
- record the contradiction in the appropriate knowledge artifact
- do not normalize the discrepancy by repeating the unsupported claim

### When A Rule Depends On Human Memory

If a rule is only being followed because someone remembers it:

- prefer to move the rule into a document, coverage matrix, or validator
- if it cannot be automated, label it as manual and explain why
- if the same memory-dependent rule appears twice, treat it as debt

### When A New Validator Or Tool Is Added

Whenever a validator, check, or diagnostic tool is added:

- update the governance coverage matrix
- decide whether the check should run automatically, be recommended, or remain
  manual
- add the smallest test coverage that proves the new contract
- document any gap that remains intentionally manual

## Action Levels

Keep the response levels small and explicit:

- Auto-run: cheap, read-only, deterministic, and already well understood
- Recommend: useful, but cross-cutting, slower, or dependent on judgment
- Manual only: expensive, ambiguous, or likely to overstep current evidence

## What This Framework Is Not

- It is not a general-purpose rules engine.
- It is not a replacement for the validation policy.
- It is not a new status system.
- It is not a command runner.
- It is not permission to automate every recurring annoyance.

## Related Canonical Docs

- `docs/android/DEVELOPMENT_WORKFLOW.md`
- `docs/android/GOVERNANCE_COVERAGE_MATRIX.md`
- `docs/android/ENGINEERING_DEBT_REGISTER.md`
- `docs/android/ENGINEERING_KNOWLEDGE_FRAMEWORK.md`
- `docs/android/REPOSITORY_GOVERNANCE.md`
- `docs/android/VALIDATION_POLICY.md`
- `docs/android/TECHNICAL_DEBT_POLICY.md`
