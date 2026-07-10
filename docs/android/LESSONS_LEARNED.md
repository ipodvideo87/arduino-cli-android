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

## Treat FD Handoff And Stream Validation As Separate Milestones

- Lesson: a valid `TERMUX_USB_FD` handoff does not prove generic POSIX
  byte-stream behavior. FD handoff validation and stream validation are
  separate milestones.
- Why it matters: Android USB transport work can look superficially ready after
  the helper handoff succeeds, but read/write behavior still needs its own
  evidence before any stream-capable claims are made.
- Evidence source: native Termux stream-boundary validation where the fd
  handoff remained valid and inspectable, read probing returned `EOF`, and
  write probing returned `invalid argument`.
- When to apply it: when planning or reviewing Android USB transport work that
  uses `TERMUX_USB_FD`, helper handoff, or stream-boundary diagnostics.
- What not to infer from it: do not assume read/write support, claim/release
  readiness, or payload transfer capability from fd handoff alone.

## Canonical Docs Need Synchronization And Review Enforcement

- Lesson: canonical ownership rules are necessary but not sufficient; when
  validated repository state changes, the owning current-state documents must
  be synchronized and important canonical edits must be reviewed for behavior
  preservation, replacement, weakening, or contradiction before closeout.
- Why it matters: structural governance validation can still pass while a
  status file remains stale or a historical summary looks current.
- Evidence source: the Phase 6 after-state audit where `STATUS.md` still
  reflected the pre-Phase-6 snapshot and `ENGINEERING_MILESTONE_SUMMARY.md`
  needed explicit historical labeling even though governance validation passed.
- When to apply it: whenever a milestone changes validated repository state or
  edits an important canonical document.
- What not to infer from it: do not turn this into a general semantic policy
  engine or a license to auto-edit canonical documents.

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
