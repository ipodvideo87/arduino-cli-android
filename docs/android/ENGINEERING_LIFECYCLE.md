# Engineering Lifecycle

This document defines the lifecycle that turns an engineering task into durable
knowledge.

It is operational, not archival. It describes the sequence that should happen
when work is planned, validated, decided, and closed.

## Lifecycle Stages

| Stage | Purpose | Primary Output |
| --- | --- | --- |
| Frame the question | Define the problem and the uncertainty | Task question and scope |
| Inspect the evidence base | Read the relevant docs and code | Evidence inventory |
| Gather evidence | Run tests, builds, or native validation | Observations and artifacts |
| Compare approaches | Evaluate architecture and maintenance tradeoffs | Recommended option |
| Decide | Choose the path forward | Decision record |
| Record uncertainty | Capture what remains unknown | Uncertainty register entry |
| Update confidence | State how the evidence changed confidence | Confidence note |
| Capture lesson | Distill durable guidance | Lesson entry |
| Update docs | Align the canonical repository memory | Doc updates |
| Close the task | State what changed and what remains | Closeout summary |

In day-to-day use, the early stages are usually captured as an engineering
preview, and the later stages are usually captured as an engineering review.
Those are lightweight wrappers around the same lifecycle, not additional
approval gates.

Interrupted work should also carry a live snapshot in
`docs/android/TASK_RECOVERY.md`. That file is the first place to resume from
after a pause, before any new planning or re-derivation.

## Stage Guidance

### Frame the question

Start by stating the question the task is trying to answer.

This should be phrased as an engineering question, not as a desired outcome.

### Inspect the evidence base

Read the canonical docs and inspect code before deciding.

The goal is to avoid encoding a guess as a contract.

### Gather evidence

Use the lightest validation level that can answer the question:

- static review for structure and ownership
- host validation for local logic
- native Termux for Android behavior
- real hardware for upload, flash, and runtime claims

When native Termux evidence is required, the evidence plan should already be
written as a Native Validation Package in the development workflow. That
package should order commands from safest to most invasive and define how each
result changes the next step.

### Compare approaches

Compare only approaches that the evidence and architecture make plausible.

Prefer the option that is most durable and least likely to require rework.

### Decide

Write down the decision, the alternatives, and why the selected option wins.

If the work changes an enduring contract, record it in `ENGINEERING_DECISIONS.md`
as well as the task-level log.

### Record uncertainty

Anything that remains unknown should be recorded before the task closes.

Unknowns should be specific enough that the next validation step is obvious.

### Update confidence

State the confidence level before and after the work.

Confidence is separate from validation level. Validation is evidence scope;
confidence is the strength of the inference from that evidence.

### Capture lesson

Promote only durable lessons.

If the insight is likely to matter again, move it into `LESSONS_LEARNED.md`.
Otherwise keep it in the decision or findings log.

### Update docs

Update the smallest canonical set of docs that now own the new knowledge.

Do not copy the same rule into multiple places.

### Close the task

Close the task only when the repository can answer:

- what was learned
- what remains uncertain
- what changed in the docs
- what future work this enables or blocks

For native validation tasks, close the task only after the Native Validation
Package completion criteria are satisfied or explicitly narrowed by the
observed evidence.
