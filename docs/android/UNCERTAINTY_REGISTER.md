# Uncertainty Register

This document tracks open engineering questions that matter to the repository.

It is the canonical place to keep uncertainty visible until evidence resolves
it or narrows it enough to stop blocking progress.

## Purpose

The register exists so that the repository does not silently forget what it
does not yet know.

Each entry should make the next validation step obvious.

## What To Track

Track uncertainty that is:

- architecture-shaping
- validation-blocking
- likely to be revisited
- expensive to rediscover
- relevant to roadmap sequencing

## Recommended Entry Shape

```md
## Question ID - short question

- Question: ...
- Why it matters: ...
- Current hypothesis: ...
- Evidence needed: ...
- Owner: ...
- Next validation: ...
- Target close condition: ...
- Related decision log: ...
- Related findings: ...
```

## Status Values

- open
- narrowed
- waiting on evidence
- resolved
- retired

## Closure Rules

Close a question only when one of the following is true:

- the evidence answers it
- the question is no longer relevant
- the question was replaced by a better question
- the uncertainty belongs in a different document

If the uncertainty remains but no longer blocks work, mark it as narrowed and
state the remaining boundary clearly.

## Ownership Rules

- One question should have one primary owner.
- If multiple tasks depend on the same question, keep one shared entry instead
  of duplicating it.
- Use the decision log to record the decision that followed from the question.

