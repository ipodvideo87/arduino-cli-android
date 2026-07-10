# Development Workflow

This is the primary day-to-day front door for engineering work in this
repository.

If you have a task, start here. Do not try to assemble the operating model from
memory.
This workflow operates under the Engineering Laws at the top of `AGENTS.md`.

For this repository, the canonical working copy is
`/data/data/com.termux/files/home/Development/GitHub/arduino-cli-android`.
If you land in another checkout, switch there before reading, editing, or
running repo commands.

The non-negotiable process invariants are defined in
[ENGINEERING_INVARIANTS.md](ENGINEERING_INVARIANTS.md).

## Session Start Checklist

Before doing new work:

- confirm the canonical repo path and current branch
- read `AGENTS.md`
- read `docs/android/TASK_RECOVERY.md` if the task was interrupted or resumed
- if the task is likely to recur or may need deeper checks, read
  `docs/android/SELF_IMPROVING_ENGINEERING_FRAMEWORK.md` and
  `docs/android/GOVERNANCE_COVERAGE_MATRIX.md`
- identify the task objective and expected validation level
- decide whether native diagnostics will use the structured evidence collector

## Session Closeout Checklist

Before closing the task:

- record the evidence collected
- validate the claim at the highest supported level
- determine whether validated repository state changed and record one of:
  current-state synchronization required and completed, current-state
  synchronization not required with reasoning, or current-state synchronization
  required but deferred with an explicit documented blocker and uncertainty
- prepare a closeout report that follows
  `docs/android/CLOSEOUT_REPORTING_STANDARD.md`
- ensure every consequential judgment in the closeout report follows
  `docs/android/ENGINEERING_JUDGMENT_STANDARD.md`
- if the same class of problem appeared twice, perform root-cause analysis,
  classify the root cause, and record the smallest durable improvement before
  closing
- update the canonical docs that own the result
- if the same problem appeared twice, record the process bug in the debt
  register or the knowledge docs before closing
- confirm `STATUS.md` still owns current state and `docs/android/ROADMAP.md`
  still owns future ordering, or record a documented deferral
- commit the change set if requested
- push the branch if requested
- reset `docs/android/TASK_RECOVERY.md` after the task is fully complete

### Canonical Document Change Review

Before editing an important canonical document:

- identify the target document
- state the purpose of the change
- name the document's current canonical role
- list the validated engineering behaviors affected
- list the dependent documents affected
- cite the evidence authorizing the change
- decide whether current-state synchronization may be required

After editing an important canonical document:

- classify the behavior change as preserved, strengthened, weakened, replaced,
  removed, contradiction introduced, or contradiction resolved
- note the routing or discoverability effect
- note the ownership effect
- state the current-state synchronization result
- state the historical/current classification result
- cite evidence authorizing any strengthening, weakening, replacement, or
  removal
- give a final compliance judgment

## 1. Understand The Task

First identify what kind of work it is:

- documentation only
- research only
- code inspection
- implementation
- validation
- review or audit
- cross-subsystem architecture work

Then state the objective in plain terms.

## 2. Read The Right Canonical Docs

Use the smallest set of canonical docs that owns the problem:

- `AGENTS.md`
- `docs/android/TASK_RECOVERY.md` if the task was interrupted or you are
  resuming existing work
- `docs/android/SELF_IMPROVING_ENGINEERING_FRAMEWORK.md` when the task may
  recur, needs a process-bug response, or should trigger deeper checks
- `docs/android/GOVERNANCE_COVERAGE_MATRIX.md` when you need to see which rules
  are already covered by policy, validation, or tooling
- `docs/android/ENGINEERING_DEBT_REGISTER.md` when the task is accumulating
  repeatable process debt that should be tracked explicitly
- `docs/android/CODEX_OPERATING_MODEL.md`
- `docs/android/ENGINEERING_PRINCIPLES.md`
- `docs/android/DECISION_FRAMEWORK.md`
- `docs/android/ENGINEERING_KNOWLEDGE_FRAMEWORK.md`
- `docs/android/ENGINEERING_LIFECYCLE.md`
- `docs/android/CONFIDENCE_MODEL.md`
- `docs/android/DOCUMENTATION_ARCHITECTURE.md`
- `docs/android/ENGINEERING_METHODOLOGY.md`
- `docs/android/REPOSITORY_GOVERNANCE.md`
- `docs/android/ARCHITECTURE_REVIEW_PROCESS.md`
- `docs/android/TECHNICAL_DEBT_POLICY.md`
- `docs/android/INTERFACE_STABILITY_POLICY.md`
- `STATUS.md`
- `docs/android/ROADMAP.md`

Read additional subsystem docs only when the task needs them.

If a task has failed twice in the same way, or you are deciding whether to run
an additional check, consult the self-improving engineering framework and the
coverage matrix before choosing the next step.

## 3. Engineering Preview

Before working, write down the preview:

- objective
- expected outcome
- known assumptions
- open unknowns
- expected validation level
- architecture impact
- documentation impact
- expected risks

If you are resuming interrupted work, refresh `docs/android/TASK_RECOVERY.md`
first, then use it as the working snapshot before you revise the preview. The
recovery file should carry the full current task plan, not a short summary.

### Native Validation Package

If native Android/Termux validation is required, expand the preview into a
Native Validation Package before running any device commands. This is the
canonical pre-execution plan for native validation work.

The package must answer:

- the engineering question being answered
- why native validation is required
- exactly what evidence is needed from the device or from the user
- the exact commands to run, ordered from safest to most invasive
- what each command is expected to prove
- the expected successful outcome for each command
- the expected failure outcome for each command
- the technical interpretation of each possible result
- how each outcome changes confidence
- how each outcome changes validation level
- how each outcome affects architecture
- the remaining uncertainty after each outcome
- the decision matrix for the next milestone
- the structured evidence artifact to collect when available
- the exact output to paste back if manual copy is still required
- the completion criteria for honest milestone closeout

When the structured evidence collector is available, prefer it over manual
copy/paste logs for native diagnostics. Use manual pasteback only when a
structured artifact cannot be produced.

Do not start native validation until this package exists.

If the task is architecture-heavy, validation-heavy, or spans subsystems, do the
architecture review before implementation.

### Task Recovery Protocol

Keep a live recovery snapshot at `docs/android/TASK_RECOVERY.md` for any task
that may be interrupted.

The recovery file must contain:

- the full objective and scope currently being worked
- constraints and explicit non-goals
- the intended step sequence
- progress made so far
- files touched or intended to be touched
- validation status and evidence collected so far
- remaining work
- the safest next action

Update the recovery file whenever the task meaningfully changes or reaches a
new checkpoint.

If the task is interrupted, the next session should read the recovery file
first and continue from there without guessing.

When the task is fully completed, validated, committed, and pushed, clear the
contents of the recovery file but keep the file itself in place for the next
task.

Closeout must follow this chain:

evidence -> validation -> documentation -> commit -> push -> recovery reset ->
next milestone

## 4. Research When Required

Research before implementing when:

- Android or Termux behavior may differ from host behavior
- evidence is incomplete
- project history contains prior findings or rejected approaches
- the request could encode an assumption as fact
- multiple approaches are technically viable

Prefer evidence over intuition. If the evidence is not enough, stop and compare
alternatives before acting.

## 5. Decide And Plan

Compare only the approaches that the evidence and architecture make plausible.
Prefer the option that is most durable, least duplicative, and easiest to
validate in native Termux.

If the right direction is still unclear, ask the user before implementing.

## 6. Implement In Small Milestones

- Make the smallest change that resolves the current uncertainty.
- Prefer reusable ACL infrastructure over one-off fixes.
- Keep architecture decisions documented as they land.
- Use explicit installation for validation bootstraps. The default action is
  inspect/report only.

## 7. Validate

Use the lightest validation level that can answer the question:

- static review for structure and ownership
- host validation for local logic
- native Termux for Android behavior
- real hardware for upload, flash, and runtime claims

For native validation work, execute the commands in the Native Validation
Package order and do not add extra device tests unless the evidence changes the
decision.

Do not claim a higher validation level than the evidence supports.

## 8. Record Knowledge

Capture the result in the smallest canonical artifact that owns it:

- evidence -> `docs/android/VALIDATED_FINDINGS.md`
- temporary uncertainty -> `docs/android/UNCERTAINTY_REGISTER.md`
- task-level decision -> `docs/android/DECISION_LOG.md`
- durable architecture decision -> `docs/android/ENGINEERING_DECISIONS.md`
- reusable lesson -> `docs/android/LESSONS_LEARNED.md`
- confidence change -> `docs/android/CONFIDENCE_MODEL.md`
- recurring process debt -> `docs/android/ENGINEERING_DEBT_REGISTER.md`

## 9. Update Docs And Status

Update `STATUS.md` and `docs/android/ROADMAP.md` only when the change affects
project state or milestone sequencing.

Use the synchronization rules in `STATUS.md` and `docs/android/ROADMAP.md`:

- `STATUS.md` owns the current snapshot of work, blockers, validation state,
  and the next engineering milestone.
- `ROADMAP.md` owns the ordered list of future milestones and the long-range
  direction.
- Update both when a validated change changes current state and also changes
  future sequencing.
- If they conflict, `STATUS.md` wins for the current snapshot and `ROADMAP.md`
  wins for future ordering.

## 10. Engineering Review And Closeout

Every completed task should answer:

- Was implementation completed?
- Was validation performed?
- Was evidence recorded?
- Were assumptions confirmed or invalidated?
- Were uncertainties updated?
- Was confidence updated?
- Were lessons captured?
- Was documentation updated?
- If the same class of problem recurred, was root-cause analysis performed and
  the smallest durable improvement recorded?
- If recurrence was external or one-off, was that boundary documented instead
  of forcing automation?
- Does `STATUS.md` require updating?
- Does `ROADMAP.md` require updating?
- Was technical debt introduced?
- Was architecture affected?

For native validation tasks, the review should also confirm that the Native
Validation Package evidence, outcomes, and decision matrix were captured in the
appropriate canonical artifacts.

If any answer is still no, the task is not closed.

## Working Practices

- Keep changes modular, maintainable, and reviewable.
- Do not use `git add .` blindly when untracked or WIP files exist.
- Stage only the files in scope for the change.
- Keep commits small enough that the evidence and intent are readable later.
- Push after a completion so the remote branch reflects the current validated
  state.

## Validation Expectations

- Do not merge to `main` until real Android validation is complete.
- Keep preflight, emulated, native, and hardware evidence distinct.
- Do not claim Android success from emulation.
- Do not claim upload, flashing, or monitor success without the required
  validation level.
